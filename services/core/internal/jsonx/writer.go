package jsonx

import (
	"bytes"
	"strconv"
)

const hexDigits = "0123456789abcdef"

// WriteString writes s as a JSON string escaped like Newtonsoft.Json's default escaping: control
// characters, quote, backslash, U+0085, U+2028 and U+2029; everything else as UTF-8.
func WriteString(buf *bytes.Buffer, s string) {
	buf.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\b':
			buf.WriteString(`\b`)
		case '\t':
			buf.WriteString(`\t`)
		case '\n':
			buf.WriteString(`\n`)
		case '\f':
			buf.WriteString(`\f`)
		case '\r':
			buf.WriteString(`\r`)
		case 0x85, 0x2028, 0x2029:
			buf.WriteString(`\u`)
			buf.WriteByte(hexDigits[r>>12&0xF])
			buf.WriteByte(hexDigits[r>>8&0xF])
			buf.WriteByte(hexDigits[r>>4&0xF])
			buf.WriteByte(hexDigits[r&0xF])
		default:
			if r < 0x20 {
				buf.WriteString(`\u00`)
				buf.WriteByte(hexDigits[r>>4])
				buf.WriteByte(hexDigits[r&0xF])
			} else {
				buf.WriteRune(r)
			}
		}
	}
	buf.WriteByte('"')
}

// ObjectWriter writes a JSON object with members in the order they are added (Newtonsoft writes fields in
// declaration order), without whitespace (Formatting.None).
type ObjectWriter struct {
	buf   bytes.Buffer
	first bool
}

// NewObject starts an object.
func NewObject() *ObjectWriter {
	w := &ObjectWriter{first: true}
	w.buf.WriteByte('{')
	return w
}

func (w *ObjectWriter) key(k string) {
	if !w.first {
		w.buf.WriteByte(',')
	}
	w.first = false
	WriteString(&w.buf, k)
	w.buf.WriteByte(':')
}

// Int adds an integer member.
func (w *ObjectWriter) Int(k string, v int64) *ObjectWriter {
	w.key(k)
	w.buf.WriteString(strconv.FormatInt(v, 10))
	return w
}

// Bool adds a boolean member.
func (w *ObjectWriter) Bool(k string, v bool) *ObjectWriter {
	w.key(k)
	w.buf.WriteString(strconv.FormatBool(v))
	return w
}

// Str adds a string member.
func (w *ObjectWriter) Str(k string, v string) *ObjectWriter {
	w.key(k)
	WriteString(&w.buf, v)
	return w
}

// Ints adds an array of integers; nil writes null.
func (w *ObjectWriter) Ints(k string, v []int64) *ObjectWriter {
	w.key(k)
	if v == nil {
		w.buf.WriteString("null")
		return w
	}
	w.buf.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			w.buf.WriteByte(',')
		}
		w.buf.WriteString(strconv.FormatInt(x, 10))
	}
	w.buf.WriteByte(']')
	return w
}

// Objects adds an array of objects.
func (w *ObjectWriter) Objects(k string, v []*ObjectWriter) *ObjectWriter {
	w.key(k)
	w.buf.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			w.buf.WriteByte(',')
		}
		w.buf.Write(x.Bytes())
	}
	w.buf.WriteByte(']')
	return w
}

// Bytes closes the object and returns its JSON.
func (w *ObjectWriter) Bytes() []byte {
	b := append([]byte(nil), w.buf.Bytes()...)
	return append(b, '}')
}
