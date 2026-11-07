package php

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Snake is Laravel's Str::snake($value).
func Snake(value string) string {
	if isCtypeLower(value) {
		return value
	}
	v := ucwords(value)
	v = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, v)
	// preg_replace('/(.)(?=[A-Z])/u', '$1_', $value)
	var b strings.Builder
	rs := []rune(v)
	for i, r := range rs {
		b.WriteRune(r)
		if i+1 < len(rs) && rs[i+1] >= 'A' && rs[i+1] <= 'Z' {
			b.WriteByte('_')
		}
	}
	return strings.ToLower(b.String())
}

func isCtypeLower(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 'a' || s[i] > 'z' {
			return false
		}
	}
	return true
}

func ucwords(s string) string {
	b := []byte(s)
	up := true
	for i, c := range b {
		if up && c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
		up = c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\f' || c == '\v'
	}
	return string(b)
}

// LaravelTrim is the TrimStrings middleware transformation.
func LaravelTrim(s string) string {
	isTrim := func(r rune) bool {
		return unicode.IsSpace(r) || r == 0xFEFF || r == 0x200B || r == 0x200E
	}
	if !utf8.ValidString(s) {
		return strings.Trim(s, " \t\n\r\x00\x0B")
	}
	return strings.TrimFunc(s, isTrim)
}

// Trim is PHP trim().
func Trim(s string) string { return strings.Trim(s, " \t\n\r\x00\x0B") }

const alnum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// RandomString is Str::random($n).
func RandomString(n int) string {
	b := make([]byte, n)
	max := big.NewInt(int64(len(alnum)))
	for i := range b {
		x, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(err)
		}
		b[i] = alnum[x.Int64()]
	}
	return string(b)
}

// RandomBytes returns n cryptographically random bytes.
func RandomBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b
}

// UUID4 returns Str::uuid() (v4, lowercase).
func UUID4() string {
	b := RandomBytes(16)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}

var uuidRe = regexp.MustCompile(`^[\da-fA-F]{8}-[\da-fA-F]{4}-[\da-fA-F]{4}-[\da-fA-F]{4}-[\da-fA-F]{12}$`)

// IsUUID is Str::isUuid.
func IsUUID(v any) bool {
	s, ok := v.(string)
	return ok && uuidRe.MatchString(s)
}
