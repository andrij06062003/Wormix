package php

import (
	"testing"
)

// Expected values were produced by PHP 8.3 (php -r) in the wormix-api:ref image.

func TestJSONEncodeMatchesPHP(t *testing.T) {
	v := Assoc(
		"a/b", "Вася/<&>\"\\",
		"e", NewArr(),
		"f", 1.0,
		"g", 0.1,
		"h", 1e25,
		"i", Assoc(int64(1), int64(1)),
		"j", "\U0001F600\t",
		"k", negZero(),
		"l", 1.5e-7,
		"m", 123456789012345678.0,
	)
	got, err := JSONEncode(v)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"a\/b":"\u0412\u0430\u0441\u044f\/<&>\"\\","e":[],"f":1,"g":0.1,"h":1.0e+25,"i":{"1":1},"j":"\ud83d\ude00\t","k":-0,"l":1.5e-7,"m":1.2345678901234568e+17}`
	if string(got) != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func negZero() float64 {
	z := 0.0
	return -z
}

func TestJSONDecode(t *testing.T) {
	v, ok := JSONDecode([]byte(`{"5":1,"05":2,"a":[1,2.0,"x",null,true],"big":99999999999999999999}`))
	if !ok {
		t.Fatal("decode failed")
	}
	a := v.(*Arr)
	if _, ok := a.Get(int64(5)); !ok {
		t.Error("numeric string key should become int key")
	}
	if _, ok := a.Get("05"); !ok {
		t.Error(`"05" stays a string key`)
	}
	list := a.Val("a").(*Arr)
	if _, ok := list.Val(1).(float64); !ok {
		t.Error("2.0 decodes to float")
	}
	if _, ok := a.Val("big").(float64); !ok {
		t.Error("big int decodes to float")
	}
	if _, ok := JSONDecode([]byte("{bad")); ok {
		t.Error("invalid json must fail")
	}
}

func TestSerializeMatchesPHP(t *testing.T) {
	if got := Serialize(`{"id":5,"auth_key":"ab","logged_in":true}`); got != `s:41:"{"id":5,"auth_key":"ab","logged_in":true}";` {
		t.Fatalf("string: %s", got)
	}
	v := Assoc("user_id", int64(1), "last_seq", int64(1700000000), "last_result",
		Assoc("indexes", List(int64(0), int64(100)), "values", List(int64(60), int64(2)), "x", nil, "f", 1.5, "b", false, "s", "é"))
	want := `a:3:{s:7:"user_id";i:1;s:8:"last_seq";i:1700000000;s:11:"last_result";a:6:{s:7:"indexes";a:2:{i:0;i:0;i:1;i:100;}s:6:"values";a:2:{i:0;i:60;i:1;i:2;}s:1:"x";N;s:1:"f";d:1.5;s:1:"b";b:0;s:1:"s";s:2:"é";}}`
	if got := Serialize(v); got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
	back, err := Unserialize(want)
	if err != nil {
		t.Fatal(err)
	}
	if !StrictEq(back, v) {
		t.Fatal("round trip differs")
	}
}

func TestComparisons(t *testing.T) {
	cases := []struct {
		a, b any
		eq   bool
	}{
		{int64(5), "5", true},
		{int64(0), "a", false}, // PHP 8
		{nil, int64(0), true},
		{nil, "0", false},
		{"abc", "ABC", false},
		{"1e3", "1000", true},
		{true, "x", true},
		{1.0, int64(1), true},
	}
	for _, c := range cases {
		if LooseEq(c.a, c.b) != c.eq {
			t.Errorf("%#v == %#v should be %v", c.a, c.b, c.eq)
		}
	}
	if StrictEq(int64(1), 1.0) || !StrictEq(int64(1), int64(1)) {
		t.Error("strict")
	}
	if !Lt(int64(-1), "0") || !Gt("10", "9") {
		t.Error("numeric string ordering")
	}
}

func TestArithmetic(t *testing.T) {
	if Div(int64(9), int64(3)) != int64(3) {
		t.Error("exact int division stays int")
	}
	if Div(int64(10), int64(4)) != 2.5 {
		t.Error("inexact division gives float")
	}
	if Add(int64(1), "2") != int64(3) {
		t.Error("numeric string operand")
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("non-numeric operand must fail")
			}
		}()
		Add(int64(1), "abc")
	}()
	if ToInt("12abc") != 12 || ToInt(" 7") != 7 || ToInt(3.9) != 3 {
		t.Error("(int) cast")
	}
}

func TestFilterInt(t *testing.T) {
	ok := []any{int64(5), "5", " 5 ", "-3", "+3", true, 5.0, "0"}
	bad := []any{"05", "5.0", 5.5, "abc", false, nil, NewArr(), ""}
	for _, v := range ok {
		if !FilterInt(v) {
			t.Errorf("%#v should be an integer", v)
		}
	}
	for _, v := range bad {
		if FilterInt(v) {
			t.Errorf("%#v should not be an integer", v)
		}
	}
}

func TestSnake(t *testing.T) {
	for in, want := range map[string]string{
		"internal_user_id":           "internal_user_id",
		"AuthKey":                    "auth_key",
		"ShopItems":                  "shop_items",
		"password_confirmation":      "password_confirmation",
		"social_data.first_name":     "social_data.first_name",
		"user.password_confirmation": "user.password_confirmation",
		"Id":                         "id",
	} {
		if got := Snake(in); got != want {
			t.Errorf("Snake(%q) = %q, want %q", in, got, want)
		}
	}
}
