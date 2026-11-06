package wire

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/andrij06062003/Wormix/services/core/internal/csx"
)

func TestHeaderRoundTrip(t *testing.T) {
	seen := map[byte]bool{}
	for _, cmd := range []uint32{0, 1, 16, 1022, 10001, 0x7FFF} {
		for _, n := range []uint32{0, 1, 7, 65536, 0x3FFFFFFF} {
			for i := 0; i < 50; i++ {
				h, err := AppendHeader(nil, cmd, n)
				if err != nil {
					t.Fatal(err)
				}
				if len(h) != HeaderSize {
					t.Fatalf("header size %d", len(h))
				}
				c, l, err := ParseHeader(h)
				if err != nil || c != cmd || l != n {
					t.Fatalf("round trip %d/%d: got %d/%d %v", cmd, n, c, l, err)
				}
				if FrameLength(h) != int(n) {
					t.Fatalf("FrameLength %d != %d", FrameLength(h), n)
				}
				seen[h[0]&(128|32|8)] = true
			}
		}
	}
	if len(seen) != 8 {
		t.Errorf("sync choices not randomized: %v", seen)
	}
}

func TestHeaderInvalid(t *testing.T) {
	if _, err := AppendHeader(nil, 0x8000, 0); err == nil {
		t.Error("command id 0x8000 accepted")
	}
	if _, err := AppendHeader(nil, 1, 0x40000000); err == nil {
		t.Error("length 2^30 accepted")
	}
	bad := [][]byte{
		{0, 0, 16, 0, 0, 0, 0},                    // constant flag bits missing
		{87 | 128 | 32 | 8, 0x80, 16, 0, 0, 0, 0}, // B flag and B sync both set
		{87 | 32 | 8, 0, 16, 0, 0, 0, 0},          // neither
		{87 | 128 | 32 | 8, 0, 16, 0x40, 0, 0, 0}, // D both
		{87 | 128 | 32 | 8, 0, 16, 0x80, 0, 0, 0}, // E both
	}
	for _, h := range bad {
		if _, _, err := ParseHeader(h); err == nil {
			t.Errorf("accepted %x", h)
		}
	}
	// Client-style header: all flag bits set, plain values.
	c, l, err := ParseHeader([]byte{87 | 128 | 32 | 8, 0x27, 0x11, 0, 0, 0, 5})
	if err != nil || c != 10001 || l != 5 {
		t.Errorf("got %d %d %v", c, l, err)
	}
	// Sync bits instead of flags.
	c, l, err = ParseHeader([]byte{87, 0x80 | 0x27, 0x11, 0xC0, 0, 0, 5})
	if err != nil || c != 10001 || l != 5 {
		t.Errorf("got %d %d %v", c, l, err)
	}
}

func TestReadHeaderShort(t *testing.T) {
	if _, _, err := ReadHeader(bytes.NewReader([]byte{87, 1})); err == nil {
		t.Error("short header accepted")
	}
}

func TestDecodeUTF8LikeDotNet(t *testing.T) {
	cases := []struct{ in, want string }{
		{"abc", "abc"},
		{"Пётр", "Пётр"},
		{"a\xffb", "a�b"},
		{"\xe2\x82A", "�A"},     // truncated 3-byte sequence: one replacement (maximal subpart)
		{"\xc0\xaf", "��"},      // overlong: C0 is never valid
		{"\xed\xa0\x80", "���"}, // surrogate: ED A0 is not a valid prefix
		{"\xf0\x9f\x98", "�"},   // truncated 4-byte at the end
		{"\xf4\x90\x80\x80", "����"},
		{"x\xf0\x9f\x98\x80y", "x😀y"},
	}
	for _, c := range cases {
		if got := DecodeUTF8([]byte(c.in)); got != c.want {
			t.Errorf("DecodeUTF8(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestUTF16Len(t *testing.T) {
	for s, n := range map[string]int{"": 0, "abc": 3, "Пётр": 4, "😀": 2, "a😀б": 4} {
		if UTF16Len(s) != n {
			t.Errorf("UTF16Len(%q) = %d, want %d", s, UTF16Len(s), n)
		}
	}
}

func TestWriterReader(t *testing.T) {
	var w Writer
	w.U8(0xAB)
	w.Bool(true)
	w.U16(0xBEEF)
	w.U32(0xDEADBEEF)
	w.I16(-2)
	w.I32(-3)
	w.UTF("hé")
	w.U16(2)
	w.U32(7)
	w.U32(9)
	if got := hex.EncodeToString(w.B); got != "ab01beefdeadbeeffffefffffffd000368c3a9000200000007"+"00000009" {
		t.Fatalf("encoded %s", got)
	}
	r := NewReader(w.B)
	if r.U8() != 0xAB || !r.Bool() || r.U16() != 0xBEEF || r.U32() != 0xDEADBEEF || int16(r.U16()) != -2 ||
		int32(r.U32()) != -3 || r.UTF() != "hé" {
		t.Fatal("read back mismatch")
	}
	if a := r.U32Array(); len(a) != 2 || a[0] != 7 || a[1] != 9 {
		t.Fatalf("array %v", a)
	}
	if r.Remaining() != 0 {
		t.Fatal("left over")
	}
}

func TestReaderLikeBinaryReader(t *testing.T) {
	// ReadUTF8 with fewer bytes than announced returns what is there (BinaryReader.ReadBytes).
	r := NewReader([]byte{0, 5, 'a', 'b'})
	if s := r.UTF(); s != "ab" {
		t.Errorf("lenient UTF = %q", s)
	}
	// The strict PvP reader throws instead.
	exc := csx.Catch(func() { NewReader([]byte{0, 5, 'a'}).UTFStrict() })
	if exc == nil || exc.Kind != csx.EndOfStream {
		t.Errorf("strict UTF: %v", exc)
	}
	exc = csx.Catch(func() { NewReader([]byte{1, 2, 3}).U32() })
	if exc == nil || exc.Kind != csx.EndOfStream {
		t.Errorf("U32 past end: %v", exc)
	}
	if b := NewReader([]byte{1, 2}).BytesLenient(16); len(b) != 2 {
		t.Errorf("BytesLenient %v", b)
	}
	if v := NewReader([]byte{1, 0, 0, 0}).U32LE(); v != 1 {
		t.Errorf("U32LE %d", v)
	}
}

func TestSecure(t *testing.T) {
	// ShopResult body from the reference server: payload + md5(payload + key).
	body, _ := hex.DecodeString("0000000100080000000500000002000100096319fea52baa2fc1f6abf3c16d3f5a6d")
	if !IsSecure(body) {
		t.Fatal("reference MD5 not accepted")
	}
	if !bytes.Equal(Secure(body[:len(body)-16]), body[len(body)-16:]) {
		t.Fatal("Secure differs")
	}
	body[0] ^= 1
	if IsSecure(body) || IsSecure([]byte{1, 2}) {
		t.Fatal("bad MD5 accepted")
	}
}
