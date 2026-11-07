package php

import (
	"errors"
	"strconv"
	"strings"
)

// Serialize is PHP serialize() for nil/bool/int/float/string/*Arr.
func Serialize(v any) string {
	var b strings.Builder
	ser(&b, v)
	return b.String()
}

func ser(b *strings.Builder, v any) {
	switch t := v.(type) {
	case nil:
		b.WriteString("N;")
	case bool:
		if t {
			b.WriteString("b:1;")
		} else {
			b.WriteString("b:0;")
		}
	case int:
		b.WriteString("i:" + strconv.Itoa(t) + ";")
	case int64:
		b.WriteString("i:" + strconv.FormatInt(t, 10) + ";")
	case float64:
		b.WriteString("d:" + phpGcvt(t, 'E') + ";")
	case string:
		b.WriteString("s:" + strconv.Itoa(len(t)) + ":\"" + t + "\";")
	case *Arr:
		b.WriteString("a:" + strconv.Itoa(t.Len()) + ":{")
		for _, k := range t.keys {
			ser(b, k)
			ser(b, t.vals[k])
		}
		b.WriteString("}")
	}
}

var errUnser = errors.New("unserialize: bad data")

// Unserialize parses PHP serialize() output (scalars and arrays only).
func Unserialize(s string) (any, error) {
	p := &unser{s: s}
	v, err := p.value()
	if err != nil {
		return nil, err
	}
	return v, nil
}

type unser struct {
	s string
	i int
}

func (p *unser) readUntil(c byte) (string, error) {
	j := strings.IndexByte(p.s[p.i:], c)
	if j < 0 {
		return "", errUnser
	}
	out := p.s[p.i : p.i+j]
	p.i += j + 1
	return out, nil
}

func (p *unser) expect(c byte) error {
	if p.i >= len(p.s) || p.s[p.i] != c {
		return errUnser
	}
	p.i++
	return nil
}

func (p *unser) value() (any, error) {
	if p.i+1 >= len(p.s) {
		return nil, errUnser
	}
	t := p.s[p.i]
	switch t {
	case 'N':
		p.i++
		return nil, p.expect(';')
	case 'b', 'i', 'd':
		p.i++
		if err := p.expect(':'); err != nil {
			return nil, err
		}
		raw, err := p.readUntil(';')
		if err != nil {
			return nil, err
		}
		switch t {
		case 'b':
			return raw == "1", nil
		case 'i':
			n, err := strconv.ParseInt(raw, 10, 64)
			return n, err
		default:
			switch raw {
			case "INF":
				raw = "+Inf"
			case "-INF":
				raw = "-Inf"
			}
			return strconv.ParseFloat(raw, 64)
		}
	case 's':
		p.i++
		if err := p.expect(':'); err != nil {
			return nil, err
		}
		ls, err := p.readUntil(':')
		if err != nil {
			return nil, err
		}
		n, err := strconv.Atoi(ls)
		if err != nil || n < 0 {
			return nil, errUnser
		}
		if err := p.expect('"'); err != nil {
			return nil, err
		}
		if p.i+n > len(p.s) {
			return nil, errUnser
		}
		str := p.s[p.i : p.i+n]
		p.i += n
		if err := p.expect('"'); err != nil {
			return nil, err
		}
		return str, p.expect(';')
	case 'a':
		p.i++
		if err := p.expect(':'); err != nil {
			return nil, err
		}
		ls, err := p.readUntil(':')
		if err != nil {
			return nil, err
		}
		n, err := strconv.Atoi(ls)
		if err != nil || n < 0 {
			return nil, errUnser
		}
		if err := p.expect('{'); err != nil {
			return nil, err
		}
		a := NewArr()
		for k := 0; k < n; k++ {
			key, err := p.value()
			if err != nil {
				return nil, err
			}
			val, err := p.value()
			if err != nil {
				return nil, err
			}
			a.Set(key, val)
		}
		return a, p.expect('}')
	}
	return nil, errUnser
}
