package jsonx

import (
	"bytes"
	"testing"

	"github.com/andrij06062003/Wormix/services/core/internal/csx"
)

func mustParse(t *testing.T, s string) *Value {
	t.Helper()
	v, err := Parse([]byte(s))
	if err != nil {
		t.Fatalf("Parse(%s): %v", s, err)
	}
	return v
}

func TestParse(t *testing.T) {
	v := mustParse(t, `{"a":1,"b":[true,null,"x",1.5,12345678901234567890],"a":2}`)
	if v.Kind != Object || len(v.Obj.Keys) != 2 || v.Obj.Keys[0] != "a" || v.Obj.Get("a").I != 2 {
		t.Fatalf("duplicate key handling: %+v", v.Obj)
	}
	arr := v.Obj.Get("b").Arr
	if arr[0].Kind != Bool || arr[1].Kind != Null || arr[2].S != "x" || arr[3].Kind != Float || arr[4].Big == nil {
		t.Fatalf("array %+v", arr)
	}
	for _, bad := range []string{"", "  ", "{} x", "{", "[1,]", "<html>"} {
		if _, err := Parse([]byte(bad)); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
	if mustParse(t, "null").Kind != Null {
		t.Error("null")
	}
}

func conv(f func()) bool { return csx.Catch(f) == nil }

func TestConversions(t *testing.T) {
	v := func(s string) *Value { return mustParse(t, s) }
	var i32 int32
	var u32 uint32
	var i16 int16
	var b byte
	cases := []struct {
		name string
		ok   bool
		f    func()
		chk  func() bool
	}{
		{"int from int", true, func() { i32 = v("-5").Int32() }, func() bool { return i32 == -5 }},
		{"int from string", true, func() { i32 = v(`" -3 "`).Int32() }, func() bool { return i32 == -3 }},
		{"int from float rounds to even", true, func() { i32 = v("-5.5").Int32() }, func() bool { return i32 == -6 }},
		{"int from float 2.5", true, func() { i32 = v("2.5").Int32() }, func() bool { return i32 == 2 }},
		{"int from bool", false, func() { v("true").Int32() }, nil},
		{"int from null", false, func() { v("null").Int32() }, nil},
		{"int from empty string", false, func() { v(`""`).Int32() }, nil},
		{"int overflow", false, func() { v("2147483648").Int32() }, nil},
		{"uint from bool", true, func() { u32 = v("true").Uint32() }, func() bool { return u32 == 1 }},
		{"uint from +5", true, func() { u32 = v(`"+5"`).Uint32() }, func() bool { return u32 == 5 }},
		{"uint negative", false, func() { v("-1").Uint32() }, nil},
		{"uint from float", true, func() { u32 = v("2.5").Uint32() }, func() bool { return u32 == 2 }},
		{"short from bool", false, func() { v("true").Int16() }, nil},
		{"short overflow", false, func() { v("40000").Int16() }, nil},
		{"short from string", true, func() { i16 = v(`"12"`).Int16() }, func() bool { return i16 == 12 }},
		{"byte from bool", false, func() { v("false").Byte() }, nil},
		{"byte 255", true, func() { b = v("255").Byte() }, func() bool { return b == 255 }},
		{"byte 256", false, func() { v("256").Byte() }, nil},
		{"big integer", false, func() { v("123456789012345678901234567890").Uint32() }, nil},
		{"bool from string", true, func() { _ = v(`" TRUE "`).Bool() }, nil},
		{"bool from bad string", false, func() { v(`"yes"`).Bool() }, nil},
		{"bool from null", false, func() { v("null").Bool() }, nil},
		{"string from object", false, func() { v("{}").StringPtr() }, nil},
		{"struct from null", false, func() { v("null").Struct("X") }, nil},
		{"list from object", false, func() { v("{}").List("L", func(*Value) {}) }, nil},
	}
	for _, c := range cases {
		ok := conv(c.f)
		if ok != c.ok {
			t.Errorf("%s: ok=%v, want %v", c.name, ok, c.ok)
			continue
		}
		if ok && c.chk != nil && !c.chk() {
			t.Errorf("%s: wrong value", c.name)
		}
	}
	if s := v("null").StringPtr(); s != nil {
		t.Error("null string not nil")
	}
	if s := v("123").StringPtr(); *s != "123" {
		t.Errorf("string from int %q", *s)
	}
	if s := v("true").StringPtr(); *s != "True" {
		t.Errorf("string from bool %q", *s)
	}
	if v("null").List("L", func(*Value) {}) {
		t.Error("null list")
	}
}

func TestFormatDouble(t *testing.T) {
	for f, want := range map[float64]string{1.5: "1.5", 1e20: "1E+20", 1e15: "1E+15", 1e14: "100000000000000",
		0.0001: "0.0001", 0.00001: "1E-05", -2.5e-7: "-2.5E-07", 123.456: "123.456", 0: "0", 100: "100"} {
		if got := FormatDouble(f); got != want {
			t.Errorf("FormatDouble(%v) = %s, want %s", f, got, want)
		}
	}
}

func TestMembersCaseInsensitive(t *testing.T) {
	o := mustParse(t, `{"money":1,"Money":2,"MONEY":3,"other":4}`).Obj
	var got []int64
	o.Members([]string{"Money"}, func(_ int, x *Value) { got = append(got, x.I) })
	if len(got) != 3 || got[2] != 3 {
		t.Errorf("members %v", got)
	}
}

func TestWriter(t *testing.T) {
	var buf bytes.Buffer
	WriteString(&buf, "a\"\\\b\f\n\r\t\x01\x1f\u0085  <>&é😀\x7f")
	want := `"a\"\\\b\f\n\r\t\u0001\u001f\u0085` + "\\u2028\\u2029<>&é😀\x7f\""
	if buf.String() != want {
		t.Errorf("escaped %s\n   want %s", buf.String(), want)
	}
	o := NewObject().Int("Id", 1).Str("K", "v").Bool("B", true).Ints("L", []int64{1, -2}).Ints("N", nil).
		Objects("O", []*ObjectWriter{NewObject().Int("x", 1)})
	if got := string(o.Bytes()); got != `{"Id":1,"K":"v","B":true,"L":[1,-2],"N":null,"O":[{"x":1}]}` {
		t.Errorf("object %s", got)
	}
	if got := string(NewObject().Bytes()); got != "{}" {
		t.Errorf("empty %s", got)
	}
}

func TestParseNetUint32(t *testing.T) {
	for s, want := range map[string]uint32{"5": 5, " 42 ": 42, "+7": 7, "-0": 0, "4294967295": 4294967295} {
		if v, ok := ParseNetUint32(s); !ok || v != want {
			t.Errorf("%q: %d %v", s, v, ok)
		}
	}
	for _, s := range []string{"", "-1", "4294967296", "x", "1 2", "0x10"} {
		if _, ok := ParseNetUint32(s); ok {
			t.Errorf("accepted %q", s)
		}
	}
}
