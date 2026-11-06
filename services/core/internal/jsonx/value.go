// Package jsonx parses API responses into an ordered JSON tree and converts values the way Newtonsoft.Json
// (JToken.Parse + ToObject<T>) did in the C# server, and writes request JSON the way JsonConvert.SerializeObject
// did. Conversion failures panic with a csx exception, like the exceptions Newtonsoft threw.
package jsonx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode"

	"github.com/andrij06062003/Wormix/services/core/internal/csx"
)

// Kind of a JSON token.
type Kind int

const (
	Null Kind = iota
	Bool
	Integer
	Float
	String
	Object
	Array
)

// Value is a parsed JSON value.
type Value struct {
	Kind Kind
	B    bool
	I    int64
	Big  *big.Int // integers that don't fit in int64
	F    float64
	S    string
	Obj  *Obj
	Arr  []*Value
}

// Obj is a JSON object that keeps its member order.
type Obj struct {
	Keys []string
	Vals []*Value
}

// Get returns the member with exactly this name (JObject indexer), or nil.
func (o *Obj) Get(name string) *Value {
	for i, k := range o.Keys {
		if k == name {
			return o.Vals[i]
		}
	}
	return nil
}

func (o *Obj) set(name string, v *Value) {
	for i, k := range o.Keys {
		if k == name {
			o.Vals[i] = v // duplicate property: the last one wins, in the first one's place
			return
		}
	}
	o.Keys = append(o.Keys, name)
	o.Vals = append(o.Vals, v)
}

// Parse parses one JSON document (JToken.Parse): trailing content is an error.
func Parse(data []byte) (*Value, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	v, err := parseValue(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, errors.New("additional text encountered after finished reading JSON content")
	}
	return v, nil
}

func parseValue(dec *json.Decoder) (*Value, error) {
	tok, err := dec.Token()
	if err != nil {
		if err == io.EOF {
			return nil, errors.New("unexpected end of JSON")
		}
		return nil, err
	}
	return fromToken(dec, tok)
}

func fromToken(dec *json.Decoder, tok json.Token) (*Value, error) {
	switch t := tok.(type) {
	case nil:
		return &Value{Kind: Null}, nil
	case bool:
		return &Value{Kind: Bool, B: t}, nil
	case string:
		return &Value{Kind: String, S: t}, nil
	case json.Number:
		return parseNumber(string(t))
	case json.Delim:
		switch t {
		case '{':
			o := &Obj{}
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, ok := kt.(string)
				if !ok {
					return nil, errors.New("invalid object key")
				}
				v, err := parseValue(dec)
				if err != nil {
					return nil, err
				}
				o.set(key, v)
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return &Value{Kind: Object, Obj: o}, nil
		case '[':
			a := []*Value{}
			for dec.More() {
				v, err := parseValue(dec)
				if err != nil {
					return nil, err
				}
				a = append(a, v)
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return &Value{Kind: Array, Arr: a}, nil
		}
	}
	return nil, fmt.Errorf("unexpected JSON token %v", tok)
}

func parseNumber(s string) (*Value, error) {
	if strings.ContainsAny(s, ".eE") {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil && !errors.Is(err, strconv.ErrRange) {
			return nil, err
		}
		return &Value{Kind: Float, F: f}, nil
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return &Value{Kind: Integer, I: i}, nil
	}
	b, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil, fmt.Errorf("invalid number %q", s)
	}
	return &Value{Kind: Integer, Big: b}, nil
}

// ---- conversions (Newtonsoft ToObject semantics) ----------------------------------------------------------

func convErr(v *Value, target string) {
	csx.Throw(csx.Generic, "Error converting value %s to type '%s'", v.describe(), target)
}

func (v *Value) describe() string {
	if v == nil {
		return "{null}"
	}
	switch v.Kind {
	case Null:
		return "{null}"
	case String:
		return strconv.Quote(v.S)
	case Object:
		return "{object}"
	case Array:
		return "[array]"
	default:
		return v.ToString()
	}
}

// IsNull reports whether v is a JSON null.
func (v *Value) IsNull() bool { return v != nil && v.Kind == Null }

// ToString is JToken.ToString() for a value (used to read the response "type").
func (v *Value) ToString() string {
	switch v.Kind {
	case Null:
		return ""
	case Bool:
		if v.B {
			return "True"
		}
		return "False"
	case Integer:
		if v.Big != nil {
			return v.Big.String()
		}
		return strconv.FormatInt(v.I, 10)
	case Float:
		return FormatDouble(v.F)
	case String:
		return v.S
	default:
		var buf bytes.Buffer
		v.writeJSON(&buf)
		return buf.String()
	}
}

func (v *Value) writeJSON(buf *bytes.Buffer) {
	switch v.Kind {
	case Null:
		buf.WriteString("null")
	case Bool:
		if v.B {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case Integer, Float:
		buf.WriteString(v.ToString())
	case String:
		WriteString(buf, v.S)
	case Object:
		buf.WriteByte('{')
		for i, k := range v.Obj.Keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			WriteString(buf, k)
			buf.WriteByte(':')
			v.Obj.Vals[i].writeJSON(buf)
		}
		buf.WriteByte('}')
	case Array:
		buf.WriteByte('[')
		for i, e := range v.Arr {
			if i > 0 {
				buf.WriteByte(',')
			}
			e.writeJSON(buf)
		}
		buf.WriteByte(']')
	}
}

// FormatDouble formats a double like .NET Core's double.ToString(InvariantCulture): the shortest
// round-trippable digits, in scientific notation for exponents >= 15 or < -4.
func FormatDouble(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	}
	e := strconv.FormatFloat(f, 'e', -1, 64) // d.ddde±XX
	mant, expS, _ := strings.Cut(e, "e")
	exp, _ := strconv.Atoi(expS)
	neg := strings.HasPrefix(mant, "-")
	mant = strings.TrimPrefix(mant, "-")
	digits := strings.Replace(mant, ".", "", 1)
	sign := ""
	if neg {
		sign = "-"
	}
	if exp >= 15 || exp < -4 {
		m := digits[:1]
		if len(digits) > 1 {
			m += "." + digits[1:]
		}
		es := "+"
		if exp < 0 {
			es = "-"
			exp = -exp
		}
		return fmt.Sprintf("%s%sE%s%02d", sign, m, es, exp)
	}
	if exp < 0 {
		return sign + "0." + strings.Repeat("0", -exp-1) + digits
	}
	if len(digits) <= exp+1 {
		return sign + digits + strings.Repeat("0", exp+1-len(digits))
	}
	return sign + digits[:exp+1] + "." + digits[exp+1:]
}

// netWhite is the whitespace .NET number parsing skips.
func netWhite(c byte) bool { return c == 0x20 || (c >= 0x09 && c <= 0x0D) }

// parseNetInteger parses s with NumberStyles.Integer (surrounding whitespace, leading sign), like
// int.Parse/uint.Parse with the invariant culture.
func parseNetInteger(s string) (*big.Int, bool) {
	i, j := 0, len(s)
	for j > i && s[j-1] == 0 { // trailing NULs are accepted
		j--
	}
	for i < j && netWhite(s[i]) {
		i++
	}
	for j > i && netWhite(s[j-1]) {
		j--
	}
	neg := false
	if i < j && (s[i] == '+' || s[i] == '-') {
		neg = s[i] == '-'
		i++
	}
	if i >= j {
		return nil, false
	}
	for k := i; k < j; k++ {
		if s[k] < '0' || s[k] > '9' {
			return nil, false
		}
	}
	b, _ := new(big.Int).SetString(s[i:j], 10)
	if neg {
		b.Neg(b)
	}
	return b, true
}

// ParseNetUint32 is uint.TryParse(s) with NumberStyles.Integer.
func ParseNetUint32(s string) (uint32, bool) {
	b, ok := parseNetInteger(s)
	if !ok || !b.IsInt64() || b.Int64() < 0 || b.Int64() > math.MaxUint32 {
		return 0, false
	}
	return uint32(b.Int64()), true
}

// toInteger converts a value to an integer in [lo, hi]. Newtonsoft reads int, short and byte fields with
// ReadAsInt32, which rejects booleans; uint and ushort fields are read as tokens and converted with
// Convert.ChangeType, which accepts them (allowBool).
func (v *Value) toInteger(lo, hi int64, target string, allowBool bool) int64 {
	if v == nil {
		convErr(v, target)
	}
	switch v.Kind {
	case Integer:
		if v.Big == nil && v.I >= lo && v.I <= hi {
			return v.I
		}
	case Float:
		if !math.IsNaN(v.F) {
			r := math.RoundToEven(v.F)
			if r >= float64(lo) && r <= float64(hi) {
				return int64(r)
			}
		}
	case String:
		if b, ok := parseNetInteger(v.S); ok && b.IsInt64() && b.Int64() >= lo && b.Int64() <= hi {
			return b.Int64()
		}
	case Bool:
		if allowBool {
			if v.B {
				return 1
			}
			return 0
		}
	}
	convErr(v, target)
	return 0
}

// Int32 converts like a C# int field.
func (v *Value) Int32() int32 {
	return int32(v.toInteger(math.MinInt32, math.MaxInt32, "System.Int32", false))
}

// Uint32 converts like a C# uint field.
func (v *Value) Uint32() uint32 { return uint32(v.toInteger(0, math.MaxUint32, "System.UInt32", true)) }

// Int16 converts like a C# short field (read with ReadAsInt32, so booleans are rejected).
func (v *Value) Int16() int16 {
	return int16(v.toInteger(math.MinInt16, math.MaxInt16, "System.Int16", false))
}

// Uint16 converts like a C# ushort field.
func (v *Value) Uint16() uint16 { return uint16(v.toInteger(0, math.MaxUint16, "System.UInt16", true)) }

// Byte converts like a C# byte field (read with ReadAsInt32, so booleans are rejected).
func (v *Value) Byte() byte { return byte(v.toInteger(0, math.MaxUint8, "System.Byte", false)) }

// Bool converts like a C# bool field.
func (v *Value) Bool() bool {
	if v != nil {
		switch v.Kind {
		case Bool:
			return v.B
		case Integer:
			if v.Big != nil {
				return v.Big.Sign() != 0
			}
			return v.I != 0
		case Float:
			return v.F != 0
		case String:
			s := strings.TrimFunc(v.S, func(r rune) bool { return unicode.IsSpace(r) || r == 0 })
			if strings.EqualFold(s, "true") {
				return true
			}
			if strings.EqualFold(s, "false") {
				return false
			}
		}
	}
	convErr(v, "System.Boolean")
	return false
}

// StringPtr converts like a C# string field: nil for JSON null.
func (v *Value) StringPtr() *string {
	if v == nil {
		convErr(v, "System.String")
	}
	switch v.Kind {
	case Null:
		return nil
	case Object, Array:
		convErr(v, "System.String")
	}
	s := v.ToString()
	return &s
}

// Members calls set for each member of the object whose name matches one of names: exactly, or else
// ignoring case (Newtonsoft's GetClosestMatchProperty). Unknown members are ignored.
func (o *Obj) Members(names []string, set func(field int, v *Value)) {
	for i, k := range o.Keys {
		idx := -1
		for n, name := range names {
			if name == k {
				idx = n
				break
			}
		}
		if idx < 0 {
			for n, name := range names {
				if strings.EqualFold(name, k) {
					idx = n
					break
				}
			}
		}
		if idx >= 0 {
			set(idx, o.Vals[i])
		}
	}
}

// Struct returns the object to decode into a C# struct: a JSON null or non-object is an error.
func (v *Value) Struct(target string) *Obj {
	if v == nil || v.Kind != Object {
		convErr(v, target)
	}
	return v.Obj
}

// List calls elem for each element of a JSON array and returns false for JSON null (a null list).
// Anything else is an error.
func (v *Value) List(target string, elem func(*Value)) bool {
	if v == nil {
		convErr(v, target)
	}
	switch v.Kind {
	case Null:
		return false
	case Array:
		for _, e := range v.Arr {
			elem(e)
		}
		return true
	}
	convErr(v, target)
	return false
}
