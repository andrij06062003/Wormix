package wire

import (
	"crypto/md5"
	"encoding/binary"
	"strings"
	"unicode/utf8"

	"github.com/andrij06062003/Wormix/services/core/internal/csx"
)

// SecretKey is appended to a payload before hashing it for the MD5 "security" suffix.
const SecretKey = "ygfA69i3Uz6bDO8"

// Secure returns the MD5 suffix the client checks on some messages: md5(payload + SecretKey).
func Secure(payload []byte) []byte {
	h := md5.New()
	h.Write(payload)
	h.Write([]byte(SecretKey))
	return h.Sum(nil)
}

// SecretKey125 is the salt of the 1.25.0 client (the same MD5 suffix, another key).
const SecretKey125 = "qlLZmjqXfZiKcJVj"

// SecureWith is Secure with another salt.
func SecureWith(payload []byte, key string) []byte {
	h := md5.New()
	h.Write(payload)
	h.Write([]byte(key))
	return h.Sum(nil)
}

// IsSecureWith reports whether body ends with a valid MD5 suffix for this salt.
func IsSecureWith(body []byte, key string) bool {
	if len(body) < 16 {
		return false
	}
	return string(SecureWith(body[:len(body)-16], key)) == string(body[len(body)-16:])
}

// IsSecure reports whether body ends with a valid MD5 suffix.
func IsSecure(body []byte) bool {
	if len(body) < 16 {
		return false
	}
	sum := Secure(body[:len(body)-16])
	return string(sum) == string(body[len(body)-16:])
}

// Writer builds a big-endian message body.
type Writer struct {
	B []byte
}

func (w *Writer) U8(v byte) { w.B = append(w.B, v) }
func (w *Writer) Bool(v bool) {
	if v {
		w.B = append(w.B, 1)
	} else {
		w.B = append(w.B, 0)
	}
}
func (w *Writer) U16(v uint16) { w.B = binary.BigEndian.AppendUint16(w.B, v) }
func (w *Writer) U32(v uint32) { w.B = binary.BigEndian.AppendUint32(w.B, v) }

// I16 writes the low 16 bits of v, like C# (ushort)(short)value.
func (w *Writer) I16(v int) { w.U16(uint16(int16(v))) }

// I32 writes v as 4 bytes.
func (w *Writer) I32(v int32) { w.U32(uint32(v)) }

// Bytes appends raw bytes.
func (w *Writer) Bytes(b []byte) { w.B = append(w.B, b...) }

// UTF writes a string as a 2-byte byte length followed by its UTF-8 bytes (C# WriteUTF8). Like the
// original, a length over 65535 is truncated to 16 bits but all bytes are written.
func (w *Writer) UTF(s string) {
	w.U16(uint16(len(s)))
	w.B = append(w.B, s...)
}

// Reader reads a message body with the semantics of a C# BinaryReader over a MemoryStream: fixed-size reads
// past the end throw EndOfStreamException, ReadBytes returns fewer bytes.
type Reader struct {
	data []byte
	pos  int
}

// NewReader returns a reader over data.
func NewReader(data []byte) *Reader { return &Reader{data: data} }

// Remaining is the number of unread bytes.
func (r *Reader) Remaining() int { return len(r.data) - r.pos }

func (r *Reader) take(n int) []byte {
	if r.pos+n > len(r.data) {
		r.pos = len(r.data)
		csx.Throw(csx.EndOfStream, "Unable to read beyond the end of the stream.")
	}
	b := r.data[r.pos : r.pos+n]
	r.pos += n
	return b
}

func (r *Reader) U8() byte      { return r.take(1)[0] }
func (r *Reader) Bool() bool    { return r.U8() != 0 }
func (r *Reader) U16() uint16   { return binary.BigEndian.Uint16(r.take(2)) }
func (r *Reader) U32() uint32   { return binary.BigEndian.Uint32(r.take(4)) }
func (r *Reader) U16LE() uint16 { return binary.LittleEndian.Uint16(r.take(2)) }
func (r *Reader) U32LE() uint32 { return binary.LittleEndian.Uint32(r.take(4)) }

// BytesLenient reads up to n bytes (C# BinaryReader.ReadBytes: no exception at the end of the data).
func (r *Reader) BytesLenient(n int) []byte {
	if n > r.Remaining() {
		n = r.Remaining()
	}
	b := r.data[r.pos : r.pos+n]
	r.pos += n
	return b
}

// UTF reads a 2-byte length and that many bytes (fewer if the data ends) as UTF-8 (C# ReadUTF8).
func (r *Reader) UTF() string {
	n := r.U16()
	if n == 0 {
		return ""
	}
	return DecodeUTF8(r.BytesLenient(int(n)))
}

// UTFStrict reads a 2-byte length and exactly that many bytes (the PvP WireReader).
func (r *Reader) UTFStrict() string {
	n := r.U16()
	return DecodeUTF8(r.take(int(n)))
}

// U32Array reads a 2-byte count and that many big-endian uint32 values.
func (r *Reader) U32Array() []uint32 {
	n := int(r.U16())
	out := make([]uint32, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, r.U32())
	}
	return out
}

// DecodeUTF8 decodes bytes like .NET's Encoding.UTF8.GetString: each maximal invalid subsequence becomes
// one U+FFFD.
func DecodeUTF8(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	var sb strings.Builder
	for i := 0; i < len(b); {
		c := b[i]
		if c < 0x80 {
			sb.WriteByte(c)
			i++
			continue
		}
		n := 0
		lo, hi := byte(0x80), byte(0xBF)
		switch {
		case c >= 0xC2 && c <= 0xDF:
			n = 2
		case c == 0xE0:
			n, lo = 3, 0xA0
		case c >= 0xE1 && c <= 0xEC, c == 0xEE, c == 0xEF:
			n = 3
		case c == 0xED:
			n, hi = 3, 0x9F
		case c == 0xF0:
			n, lo = 4, 0x90
		case c >= 0xF1 && c <= 0xF3:
			n = 4
		case c == 0xF4:
			n, hi = 4, 0x8F
		default:
			sb.WriteRune(utf8.RuneError)
			i++
			continue
		}
		j := i + 1
		ok := true
		for k := 1; k < n; k++ {
			if j >= len(b) {
				ok = false
				break
			}
			l, h := byte(0x80), byte(0xBF)
			if k == 1 {
				l, h = lo, hi
			}
			if b[j] < l || b[j] > h {
				ok = false
				break
			}
			j++
		}
		if ok {
			sb.Write(b[i:j])
		} else {
			sb.WriteRune(utf8.RuneError)
		}
		i = j
	}
	return sb.String()
}

// UTF16Len is the length of s in UTF-16 code units (C# string.Length), which the original server used as
// the encoded size of a string.
func UTF16Len(s string) int {
	n := 0
	for _, r := range s {
		if r >= 0x10000 {
			n += 2
		} else {
			n++
		}
	}
	return n
}
