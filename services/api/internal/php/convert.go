package php

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Error is a PHP runtime error (TypeError, warning promoted to ErrorException, ...).
// Handlers panic with it; the HTTP layer turns it into a 500 like Laravel does.
type Error struct{ Msg string }

func (e *Error) Error() string { return e.Msg }

// Fail panics with a PHP runtime error.
func Fail(format string, args ...any) {
	panic(&Error{Msg: fmt.Sprintf(format, args...)})
}

// TypeName mirrors get_debug_type for messages.
func TypeName(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "bool"
	case int64, int:
		return "int"
	case float64:
		return "float"
	case string:
		return "string"
	case *Arr:
		return "array"
	}
	return fmt.Sprintf("%T", v)
}

func isSpaceByte(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f'
}

// NumericString parses a PHP numeric string (leading and trailing whitespace allowed).
// Returns int64 or float64.
func NumericString(s string) (any, bool) {
	t := strings.TrimLeft(s, " \t\n\r\v\f")
	t = strings.TrimRight(t, " \t\n\r\v\f")
	if t == "" {
		return nil, false
	}
	end, isInt := scanNumber(t)
	if end != len(t) {
		return nil, false
	}
	return parseNum(t, isInt), true
}

// scanNumber returns the length of the longest numeric prefix of s and whether it is an integer.
func scanNumber(s string) (int, bool) {
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	digits := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
		digits++
	}
	isInt := true
	if i < len(s) && s[i] == '.' {
		j := i + 1
		fd := 0
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
			fd++
		}
		if digits+fd > 0 {
			i = j
			digits += fd
			isInt = false
		}
	}
	if digits == 0 {
		return 0, true
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		j := i + 1
		if j < len(s) && (s[j] == '+' || s[j] == '-') {
			j++
		}
		ed := 0
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
			ed++
		}
		if ed > 0 {
			i = j
			isInt = false
		}
	}
	return i, isInt
}

func parseNum(t string, isInt bool) any {
	if isInt {
		if n, err := strconv.ParseInt(t, 10, 64); err == nil {
			return n
		}
	}
	f, _ := strconv.ParseFloat(t, 64)
	return f
}

// IsNumeric is is_numeric().
func IsNumeric(v any) bool {
	switch t := v.(type) {
	case int64, int, float64:
		return true
	case string:
		_, ok := NumericString(t)
		return ok
	}
	return false
}

// ToNumber converts an operand for arithmetic like PHP 8 does; non-numeric strings,
// arrays and leading-numeric strings (warning -> ErrorException in Laravel) fail.
func ToNumber(v any) any {
	switch t := v.(type) {
	case nil:
		return int64(0)
	case bool:
		if t {
			return int64(1)
		}
		return int64(0)
	case int:
		return int64(t)
	case int64, float64:
		return t
	case string:
		if n, ok := NumericString(t); ok {
			return n
		}
		Fail("A non-numeric value encountered")
	}
	Fail("Unsupported operand types: %s", TypeName(v))
	return nil
}

func asFloat(v any) float64 {
	switch t := v.(type) {
	case int64:
		return float64(t)
	case float64:
		return t
	}
	return 0
}

// Add is $a + $b.
func Add(a, b any) any {
	x, y := ToNumber(a), ToNumber(b)
	xi, ok1 := x.(int64)
	yi, ok2 := y.(int64)
	if ok1 && ok2 {
		s := xi + yi
		if (s > xi) == (yi > 0) {
			return s
		}
		return float64(xi) + float64(yi)
	}
	return asFloat(x) + asFloat(y)
}

// Sub is $a - $b.
func Sub(a, b any) any {
	x, y := ToNumber(a), ToNumber(b)
	xi, ok1 := x.(int64)
	yi, ok2 := y.(int64)
	if ok1 && ok2 {
		s := xi - yi
		if (s < xi) == (yi > 0) {
			return s
		}
		return float64(xi) - float64(yi)
	}
	return asFloat(x) - asFloat(y)
}

// Mul is $a * $b.
func Mul(a, b any) any {
	x, y := ToNumber(a), ToNumber(b)
	xi, ok1 := x.(int64)
	yi, ok2 := y.(int64)
	if ok1 && ok2 {
		if xi == 0 || yi == 0 {
			return int64(0)
		}
		p := xi * yi
		if p/yi == xi && !(xi == -1 && yi == math.MinInt64) && !(yi == -1 && xi == math.MinInt64) {
			return p
		}
		return float64(xi) * float64(yi)
	}
	return asFloat(x) * asFloat(y)
}

// Div is $a / $b.
func Div(a, b any) any {
	x, y := ToNumber(a), ToNumber(b)
	if asFloat(y) == 0 {
		panic(&Error{Msg: "Division by zero"})
	}
	xi, ok1 := x.(int64)
	yi, ok2 := y.(int64)
	if ok1 && ok2 && xi%yi == 0 {
		return xi / yi
	}
	return asFloat(x) / asFloat(y)
}

// Mod is $a % $b (operands converted to int).
func Mod(a, b any) any {
	x, y := ToIntStrict(a), ToIntStrict(b)
	if y == 0 {
		panic(&Error{Msg: "Modulo by zero"})
	}
	if y == -1 {
		return int64(0)
	}
	return x % y
}

// ToIntStrict converts an arithmetic operand to int (for %).
func ToIntStrict(v any) int64 {
	n := ToNumber(v)
	if f, ok := n.(float64); ok {
		return floatToInt(f)
	}
	return n.(int64)
}

func floatToInt(f float64) int64 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	if f >= -9.223372036854775808e18 && f < 9.223372036854775808e18 {
		return int64(f)
	}
	// PHP 8 on 64-bit: out-of-range is UB-ish (modular); rare, return 0.
	return 0
}

// ToInt is the (int) cast.
func ToInt(v any) int64 {
	switch t := v.(type) {
	case nil:
		return 0
	case bool:
		if t {
			return 1
		}
		return 0
	case int:
		return int64(t)
	case int64:
		return t
	case float64:
		return floatToInt(t)
	case string:
		s := strings.TrimLeft(t, " \t\n\r\v\f")
		end, isInt := scanNumber(s)
		if end == 0 {
			return 0
		}
		n := parseNum(s[:end], isInt)
		if f, ok := n.(float64); ok {
			if isInt { // integer overflow saturates for strings
				if f > 0 {
					return math.MaxInt64
				}
				return math.MinInt64
			}
			return floatToInt(f)
		}
		return n.(int64)
	case *Arr:
		if t.Len() > 0 {
			return 1
		}
		return 0
	}
	return 0
}

// ToBool is the (bool) cast.
func ToBool(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case int:
		return t != 0
	case int64:
		return t != 0
	case float64:
		return t != 0
	case string:
		return t != "" && t != "0"
	case *Arr:
		return t.Len() > 0
	}
	return true
}

// ToString is the (string) cast (floats with precision=14).
func ToString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case bool:
		if t {
			return "1"
		}
		return ""
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return FloatString(t)
	case string:
		return t
	case *Arr:
		return "Array"
	}
	return fmt.Sprint(v)
}

// FloatString formats like PHP's (string)$float (precision ini = 14).
func FloatString(f float64) string {
	if math.IsInf(f, 1) {
		return "INF"
	}
	if math.IsInf(f, -1) {
		return "-INF"
	}
	if math.IsNaN(f) {
		return "NAN"
	}
	if f == 0 {
		if math.Signbit(f) {
			return "-0"
		}
		return "0"
	}
	s := strconv.FormatFloat(f, 'g', 14, 64)
	// Go 'g' uses e+XX; PHP uses E+XX and trims zeros; 'g' already trims.
	if strings.Contains(s, "e") {
		mant, ex, _ := strings.Cut(s, "e")
		if !strings.Contains(mant, ".") {
			mant += ".0"
		}
		sign := ex[0]
		digits := strings.TrimLeft(ex[1:], "0")
		return mant + "E" + string(sign) + digits
	}
	return s
}

// StrictEq is ===.
func StrictEq(a, b any) bool {
	a, b = normInt(a), normInt(b)
	switch x := a.(type) {
	case nil:
		return b == nil
	case bool:
		y, ok := b.(bool)
		return ok && x == y
	case int64:
		y, ok := b.(int64)
		return ok && x == y
	case float64:
		y, ok := b.(float64)
		return ok && x == y
	case string:
		y, ok := b.(string)
		return ok && x == y
	case *Arr:
		y, ok := b.(*Arr)
		if !ok || x.Len() != y.Len() {
			return false
		}
		for i, k := range x.keys {
			if y.keys[i] != k || !StrictEq(x.vals[k], y.vals[k]) {
				return false
			}
		}
		return true
	}
	return false
}

func normInt(v any) any {
	if i, ok := v.(int); ok {
		return int64(i)
	}
	return v
}

// LooseEq is == (PHP 8 semantics).
func LooseEq(a, b any) bool {
	return Compare(a, b) == 0
}

// Compare is the spaceship operator <=> with PHP 8 semantics (returns -1, 0, 1).
// Uncomparable array/scalar pairs compare as array greater.
func Compare(a, b any) int {
	a, b = normInt(a), normInt(b)
	switch x := a.(type) {
	case nil:
		switch y := b.(type) {
		case nil:
			return 0
		case string:
			return cmpStr("", y)
		case *Arr:
			return boolCmp(false, y.Len() > 0)
		default:
			return boolCmp(false, ToBool(y))
		}
	case bool:
		return boolCmp(x, ToBool(b))
	case *Arr:
		switch y := b.(type) {
		case *Arr:
			if x.Len() != y.Len() {
				return sign(x.Len() - y.Len())
			}
			for _, k := range x.keys {
				yv, ok := y.vals[k]
				if !ok {
					return 1
				}
				if c := Compare(x.vals[k], yv); c != 0 {
					return c
				}
			}
			return 0
		case nil:
			return boolCmp(x.Len() > 0, false)
		case bool:
			return boolCmp(x.Len() > 0, y)
		default:
			return 1
		}
	}
	switch y := b.(type) {
	case nil:
		if s, ok := a.(string); ok {
			return cmpStr(s, "")
		}
		return boolCmp(ToBool(a), false)
	case bool:
		return boolCmp(ToBool(a), y)
	case *Arr:
		return -1
	}
	// a, b are int64/float64/string
	as, aStr := a.(string)
	bs, bStr := b.(string)
	if aStr && bStr {
		an, aok := NumericString(as)
		bn, bok := NumericString(bs)
		if aok && bok {
			return numCmp(an, bn)
		}
		return cmpStr(as, bs)
	}
	if aStr {
		if n, ok := NumericString(as); ok {
			return numCmp(n, b)
		}
		return cmpStr(as, ToString(b))
	}
	if bStr {
		if n, ok := NumericString(bs); ok {
			return numCmp(a, n)
		}
		return cmpStr(ToString(a), bs)
	}
	return numCmp(a, b)
}

func numCmp(a, b any) int {
	ai, ok1 := a.(int64)
	bi, ok2 := b.(int64)
	if ok1 && ok2 {
		switch {
		case ai < bi:
			return -1
		case ai > bi:
			return 1
		}
		return 0
	}
	af, bf := asFloat(a), asFloat(b)
	switch {
	case af < bf:
		return -1
	case af > bf:
		return 1
	}
	return 0
}

func cmpStr(a, b string) int {
	return sign(strings.Compare(a, b))
}

func boolCmp(a, b bool) int {
	if a == b {
		return 0
	}
	if !a {
		return -1
	}
	return 1
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}

// Lt etc. are the comparison operators.
func Lt(a, b any) bool  { return Compare(a, b) < 0 }
func Le(a, b any) bool  { return Compare(a, b) <= 0 }
func Gt(a, b any) bool  { return Compare(a, b) > 0 }
func Ge(a, b any) bool  { return Compare(a, b) >= 0 }
func Neq(a, b any) bool { return Compare(a, b) != 0 }

// IntArg coerces a value passed to a user function with an `int` parameter
// (coercive typing mode): int, integral float, numeric string with integral value, bool.
func IntArg(v any) int64 {
	switch t := v.(type) {
	case int:
		return int64(t)
	case int64:
		return t
	case bool:
		if t {
			return 1
		}
		return 0
	case float64:
		if t == math.Trunc(t) && !math.IsInf(t, 0) {
			return int64(t)
		}
		// fractional floats are deprecated but accepted (truncated)
		return floatToInt(t)
	case string:
		if n, ok := NumericString(t); ok {
			if i, ok := n.(int64); ok {
				return i
			}
			return floatToInt(n.(float64))
		}
	}
	Fail("must be of type int, %s given", TypeName(v))
	return 0
}

// FilterInt is filter_var($v, FILTER_VALIDATE_INT) !== false.
func FilterInt(v any) bool {
	var s string
	switch t := v.(type) {
	case int64, int:
		return true
	case bool:
		return t
	case float64:
		s = FloatString(t)
	case string:
		s = t
	default:
		return false
	}
	s = strings.Trim(s, " \t\n\r\v\x00")
	if s == "" {
		return false
	}
	i := 0
	if s[0] == '+' || s[0] == '-' {
		i = 1
	}
	if i >= len(s) {
		return false
	}
	if s[i] == '0' && len(s) > i+1 {
		return false
	}
	for j := i; j < len(s); j++ {
		if s[j] < '0' || s[j] > '9' {
			return false
		}
	}
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}
