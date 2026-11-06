// Package php reproduces the PHP value semantics the Laravel API relied on:
// ordered arrays (lists and maps in one type), json_encode/json_decode,
// serialize/unserialize, loose/strict comparisons and arithmetic.
//
// Values are represented as: nil, bool, int64, float64, string, *Arr.
package php

import (
	"strconv"
)

// Arr is a PHP array: an ordered map whose keys are int64 or string.
type Arr struct {
	keys []any
	vals map[any]any
	next int64
}

// NewArr returns an empty array.
func NewArr() *Arr { return &Arr{vals: map[any]any{}} }

// List builds a list (0..n-1 keys).
func List(vals ...any) *Arr {
	a := NewArr()
	for _, v := range vals {
		a.Append(v)
	}
	return a
}

// Assoc builds an array from key/value pairs: Assoc("a", 1, "b", 2).
func Assoc(kv ...any) *Arr {
	a := NewArr()
	for i := 0; i+1 < len(kv); i += 2 {
		a.Set(kv[i], kv[i+1])
	}
	return a
}

// NormKey converts a key the way PHP does: canonical decimal strings become ints,
// bools/floats become ints, nil becomes "".
func NormKey(k any) any {
	switch t := k.(type) {
	case int:
		return int64(t)
	case int64:
		return t
	case string:
		if n, ok := canonicalIntString(t); ok {
			return n
		}
		return t
	case bool:
		if t {
			return int64(1)
		}
		return int64(0)
	case float64:
		return int64(t)
	case nil:
		return ""
	}
	return k
}

func canonicalIntString(s string) (int64, bool) {
	if s == "" || len(s) > 20 {
		return 0, false
	}
	if s == "0" {
		return 0, true
	}
	i := 0
	if s[0] == '-' {
		i = 1
		if len(s) == 1 {
			return 0, false
		}
	}
	if s[i] < '1' || s[i] > '9' {
		return 0, false
	}
	for j := i; j < len(s); j++ {
		if s[j] < '0' || s[j] > '9' {
			return 0, false
		}
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// Set assigns a[k] = v keeping the position of an existing key.
func (a *Arr) Set(k any, v any) {
	k = NormKey(k)
	if _, ok := a.vals[k]; !ok {
		a.keys = append(a.keys, k)
		if n, ok := k.(int64); ok && n >= a.next {
			a.next = n + 1
		}
	}
	a.vals[k] = v
}

// Append is $a[] = v.
func (a *Arr) Append(v any) {
	a.Set(a.next, v)
}

// Get returns a[k] and whether it exists.
func (a *Arr) Get(k any) (any, bool) {
	if a == nil {
		return nil, false
	}
	v, ok := a.vals[NormKey(k)]
	return v, ok
}

// Val returns a[k] or nil.
func (a *Arr) Val(k any) any {
	v, _ := a.Get(k)
	return v
}

// Has reports whether key k exists.
func (a *Arr) Has(k any) bool {
	_, ok := a.Get(k)
	return ok
}

// Delete is unset($a[k]).
func (a *Arr) Delete(k any) {
	k = NormKey(k)
	if _, ok := a.vals[k]; !ok {
		return
	}
	delete(a.vals, k)
	for i, kk := range a.keys {
		if kk == k {
			a.keys = append(a.keys[:i:i], a.keys[i+1:]...)
			break
		}
	}
}

// Len is count($a).
func (a *Arr) Len() int {
	if a == nil {
		return 0
	}
	return len(a.keys)
}

// Keys returns the keys in order.
func (a *Arr) Keys() []any {
	if a == nil {
		return nil
	}
	return append([]any(nil), a.keys...)
}

// Values returns the values in order (array_values).
func (a *Arr) Values() []any {
	if a == nil {
		return nil
	}
	out := make([]any, len(a.keys))
	for i, k := range a.keys {
		out[i] = a.vals[k]
	}
	return out
}

// Each iterates in order.
func (a *Arr) Each(f func(k, v any)) {
	if a == nil {
		return
	}
	for _, k := range a.Keys() {
		f(k, a.vals[k])
	}
}

// IsList reports whether keys are exactly 0..n-1 in order (json_encode emits a JSON array).
func (a *Arr) IsList() bool {
	for i, k := range a.keys {
		if n, ok := k.(int64); !ok || n != int64(i) {
			return false
		}
	}
	return true
}

// Clone deep-copies nested arrays (PHP arrays are values).
func (a *Arr) Clone() *Arr {
	if a == nil {
		return nil
	}
	b := &Arr{keys: append([]any(nil), a.keys...), vals: make(map[any]any, len(a.vals)), next: a.next}
	for k, v := range a.vals {
		if sub, ok := v.(*Arr); ok {
			v = sub.Clone()
		}
		b.vals[k] = v
	}
	return b
}

// ArrayValues is array_values().
func ArrayValues(a *Arr) *Arr { return List(a.Values()...) }

// GetPath is data_get/Arr::get with dot notation.
func GetPath(v any, path string) (any, bool) {
	if a, ok := v.(*Arr); ok {
		if x, ok := a.Get(path); ok {
			return x, true
		}
	}
	cur := v
	for _, seg := range splitDots(path) {
		a, ok := cur.(*Arr)
		if !ok {
			return nil, false
		}
		x, ok := a.Get(seg)
		if !ok {
			return nil, false
		}
		cur = x
	}
	return cur, true
}

func splitDots(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

// SplitDots splits a dotted path.
func SplitDots(s string) []string { return splitDots(s) }

// ToArr returns v as *Arr if it is one.
func ToArr(v any) (*Arr, bool) {
	a, ok := v.(*Arr)
	return a, ok
}

// InArray is in_array($needle, $haystack, $strict).
func InArray(needle any, hay *Arr, strict bool) bool {
	if hay == nil {
		return false
	}
	for _, v := range hay.Values() {
		if strict {
			if StrictEq(needle, v) {
				return true
			}
		} else if LooseEq(needle, v) {
			return true
		}
	}
	return false
}
