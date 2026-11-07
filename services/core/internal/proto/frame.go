package proto

import (
	"github.com/andrij06062003/Wormix/services/core/internal/csx"
	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

// Out receives the bytes a serializer writes to a session's output stream. A serializer may call it several
// times for one message (header, then body), like the C# serializers writing to the session stream.
type Out func([]byte)

// Header returns a message header, panicking with an ArgumentException for values a header can't hold.
func Header(cmd uint32, length uint32) []byte {
	h, err := wire.AppendHeader(nil, cmd, length)
	if err != nil {
		csx.Throw(csx.Argument, "%v", err)
	}
	return h
}

// Frame builds a complete message: header + body (+ MD5 suffix if secure). This is the PvP Wire.Message.
func Frame(cmd uint32, body []byte, secure bool) []byte {
	n := len(body)
	if secure {
		n += 16
	}
	out := Header(cmd, uint32(n))
	out = append(out, body...)
	if secure {
		out = append(out, wire.Secure(body)...)
	}
	return out
}

// emitStreaming writes a header with the announced size and then the body as written, which is what the
// C# serializers that wrote straight to the session stream did. The announced size comes from GetSize and
// counts strings in UTF-16 units, so for non-ASCII text it is smaller than the body, as in the original.
// If writing the body fails, what was written so far still goes to the stream.
func emitStreaming(out Out, cmd uint32, size uint32, write func(*wire.Writer)) {
	out(Header(cmd, size))
	w := &wire.Writer{}
	defer func() {
		if r := recover(); r != nil {
			if len(w.B) > 0 {
				out(w.B)
			}
			panic(r)
		}
	}()
	write(w)
	out(w.B)
}

// emitFixed serializes the body into a buffer of exactly size bytes (a non-expandable MemoryStream in C#):
// a longer body is an exception, a shorter one is padded with zeros. With secure, the MD5 of the buffer
// follows it and the header announces size + 16.
func emitFixed(out Out, cmd uint32, size uint32, secure bool, write func(*wire.Writer)) {
	if size > 0x3FFFFFFF {
		csx.Throw(csx.Argument, "illegal header values")
	}
	w := &wire.Writer{B: make([]byte, 0, size)}
	write(w)
	if uint32(len(w.B)) > size {
		csx.Throw(csx.Generic, "Memory stream is not expandable.")
	}
	payload := w.B[:size]
	for i := len(w.B); i < int(size); i++ {
		payload[i] = 0
	}
	length := size
	if secure {
		length += 16
	}
	msg := Header(cmd, length)
	msg = append(msg, payload...)
	if secure {
		msg = append(msg, wire.Secure(payload)...)
	}
	out(msg)
}
