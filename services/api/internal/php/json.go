package php

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// JSONDecode is json_decode($s, true). ok=false when PHP would return null because of an error.
func JSONDecode(data []byte) (v any, ok bool) {
	if !utf8.Valid(data) {
		return nil, false
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	v, err := decodeValue(dec, 0)
	if err != nil {
		return nil, false
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, false
	}
	return v, true
}

var errDepth = errors.New("depth")

func decodeValue(dec *json.Decoder, depth int) (any, error) {
	if depth > 512 {
		return nil, errDepth
	}
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '[':
			a := NewArr()
			for dec.More() {
				v, err := decodeValue(dec, depth+1)
				if err != nil {
					return nil, err
				}
				a.Append(v)
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return a, nil
		case '{':
			a := NewArr()
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, err
				}
				k, _ := kt.(string)
				v, err := decodeValue(dec, depth+1)
				if err != nil {
					return nil, err
				}
				a.Set(k, v)
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return a, nil
		}
		return nil, errors.New("bad delim")
	case json.Number:
		s := string(t)
		if !strings.ContainsAny(s, ".eE") {
			if n, err := strconv.ParseInt(s, 10, 64); err == nil {
				return n, nil
			}
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			// out of range: PHP yields INF
			if strings.HasPrefix(s, "-") {
				return math.Inf(-1), nil
			}
			return math.Inf(1), nil
		}
		return f, nil
	case string, bool, nil:
		return t, nil
	}
	return nil, errors.New("bad token")
}

// ErrEncode is returned when json_encode would fail (invalid UTF-8, INF/NAN).
var ErrEncode = errors.New("json_encode failed")

// JSONEncode is json_encode($v) with default flags (escaped slashes and unicode).
func JSONEncode(v any) ([]byte, error) {
	var b bytes.Buffer
	if err := encode(&b, v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// MustJSON encodes or panics with a PHP error.
func MustJSON(v any) string {
	b, err := JSONEncode(v)
	if err != nil {
		panic(&Error{Msg: "Malformed UTF-8 characters, possibly incorrectly encoded"})
	}
	return string(b)
}

func encode(b *bytes.Buffer, v any) error {
	switch t := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		if t {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case int:
		b.WriteString(strconv.FormatInt(int64(t), 10))
	case int64:
		b.WriteString(strconv.FormatInt(t, 10))
	case float64:
		if math.IsInf(t, 0) || math.IsNaN(t) {
			return ErrEncode
		}
		b.WriteString(FloatJSON(t))
	case string:
		return encodeString(b, t)
	case *Arr:
		if t == nil {
			b.WriteString("null")
			return nil
		}
		if t.IsList() {
			b.WriteByte('[')
			for i, k := range t.keys {
				if i > 0 {
					b.WriteByte(',')
				}
				if err := encode(b, t.vals[k]); err != nil {
					return err
				}
			}
			b.WriteByte(']')
			return nil
		}
		b.WriteByte('{')
		for i, k := range t.keys {
			if i > 0 {
				b.WriteByte(',')
			}
			var ks string
			switch kk := k.(type) {
			case int64:
				ks = strconv.FormatInt(kk, 10)
			case string:
				ks = kk
			}
			if err := encodeString(b, ks); err != nil {
				return err
			}
			b.WriteByte(':')
			if err := encode(b, t.vals[k]); err != nil {
				return err
			}
		}
		b.WriteByte('}')
	case []any:
		return encode(b, List(t...))
	case []int64:
		a := NewArr()
		for _, x := range t {
			a.Append(x)
		}
		return encode(b, a)
	default:
		return errors.New("unsupported type")
	}
	return nil
}

const hexd = "0123456789abcdef"

func encodeString(b *bytes.Buffer, s string) error {
	if !utf8.ValidString(s) {
		return ErrEncode
	}
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '/':
			b.WriteString(`\/`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 || r >= 0x80 {
				if r >= 0x10000 {
					r1, r2 := utf16.EncodeRune(r)
					writeU(b, r1)
					writeU(b, r2)
				} else {
					writeU(b, r)
				}
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return nil
}

func writeU(b *bytes.Buffer, r rune) {
	b.WriteString(`\u`)
	b.WriteByte(hexd[(r>>12)&0xf])
	b.WriteByte(hexd[(r>>8)&0xf])
	b.WriteByte(hexd[(r>>4)&0xf])
	b.WriteByte(hexd[r&0xf])
}

// FloatJSON formats like json_encode with serialize_precision=-1
// (shortest round-trip digits, php_gcvt layout, no forced ".0").
func FloatJSON(f float64) string { return phpGcvt(f, 'e') }

// phpGcvt is smart_str_append_double(precision=17 shortest, zero_frac=false).
func phpGcvt(f float64, expChar byte) string {
	if f == 0 {
		if math.Signbit(f) {
			return "-0"
		}
		return "0"
	}
	e := strconv.FormatFloat(f, 'e', -1, 64) // d.ddde±XX
	mant, ex, _ := strings.Cut(e, "e")
	exp, _ := strconv.Atoi(ex)
	decpt := exp + 1
	if decpt < -3 || decpt > 17 {
		if !strings.Contains(mant, ".") {
			mant += ".0"
		}
		sign := "+"
		if exp < 0 {
			sign = "-"
			exp = -exp
		}
		return mant + string(expChar) + sign + strconv.Itoa(exp)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}
