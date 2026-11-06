// Package wire implements the Flox binary framing used by the Wormix 1.05.0 client and the
// C#-compatible primitive encodings (big-endian integers, UTF-8 strings with a 2-byte length).
package wire

import (
	"encoding/binary"
	"errors"
	"io"
	"math/rand/v2"
)

// HeaderSize is the size of a message header: 1 flag byte, 2-byte command id, 4-byte length.
const HeaderSize = 7

const (
	constFlags    = 87
	bFlag         = 128
	dFlag         = 32
	eFlag         = 8
	bSync         = 0x8000
	dSync         = 0x40000000
	eSync         = 0x80000000
	cleanCmd      = 0x7FFF
	cleanLength   = 0x3FFFFFFF
	maxCommandID  = 0x7FFF
	maxLengthBody = 0x3FFFFFFF
)

// ErrIllegalHeader is returned for a header whose sync bits don't match its flags.
var ErrIllegalHeader = errors.New("illegal command header")

// ErrIllegalValues is returned when a header can't encode the command id or length.
var ErrIllegalValues = errors.New("illegal header values")

// ParseHeader validates a raw 7-byte header and returns the command id and body length.
func ParseHeader(h []byte) (cmd uint32, length uint32, err error) {
	flags := uint32(h[0])
	rawCmd := uint32(binary.BigEndian.Uint16(h[1:3]))
	rawLen := binary.BigEndian.Uint32(h[3:7])
	ok := flags&constFlags == constFlags &&
		(flags&bFlag != 0) != (rawCmd&bSync != 0) &&
		(flags&dFlag != 0) != (rawLen&dSync != 0) &&
		(flags&eFlag != 0) != (rawLen&eSync != 0)
	if !ok {
		return 0, 0, ErrIllegalHeader
	}
	return rawCmd & cleanCmd, rawLen & cleanLength, nil
}

// ReadHeader reads and validates one header from r.
func ReadHeader(r io.Reader) (cmd uint32, length uint32, err error) {
	var h [HeaderSize]byte
	if _, err := io.ReadFull(r, h[:]); err != nil {
		return 0, 0, err
	}
	return ParseHeader(h[:])
}

// AppendHeader appends a header for the given command and body length. Like the C# server, each of the
// three sync choices (flag bit or high bit of the field) is made at random.
func AppendHeader(dst []byte, cmd uint32, length uint32) ([]byte, error) {
	if cmd > maxCommandID || length > maxLengthBody {
		return dst, ErrIllegalValues
	}
	flags := uint32(constFlags)
	rawCmd := cmd
	rawLen := length
	if rand.IntN(2) == 0 {
		flags |= bFlag
	} else {
		rawCmd |= bSync
	}
	if rand.IntN(2) == 0 {
		flags |= dFlag
	} else {
		rawLen |= dSync
	}
	if rand.IntN(2) == 0 {
		flags |= eFlag
	} else {
		rawLen |= eSync
	}
	dst = append(dst, byte(flags))
	dst = binary.BigEndian.AppendUint16(dst, uint16(rawCmd))
	dst = binary.BigEndian.AppendUint32(dst, rawLen)
	return dst, nil
}

// FrameLength returns the body length announced by a (possibly unvalidated) header: the low 30 bits of the
// length field. Used to split a byte stream into messages the way the C# MessageWriteStream does.
func FrameLength(h []byte) int {
	return int(binary.BigEndian.Uint32(h[3:7]) & cleanLength)
}
