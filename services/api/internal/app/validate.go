package app

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/andrij06062003/Wormix/services/api/internal/php"
)

// Laravel 11 default English validation messages used by the API.
var defaultMessages = map[string]string{
	"array":           "The :attribute field must be an array.",
	"boolean":         "The :attribute field must be true or false.",
	"different":       "The :attribute field and :other must be different.",
	"dimensions":      "The :attribute field has invalid image dimensions.",
	"exists":          "The selected :attribute is invalid.",
	"image":           "The :attribute field must be an image.",
	"integer":         "The :attribute field must be an integer.",
	"mimes":           "The :attribute field must be a file of type: :values.",
	"required":        "The :attribute field is required.",
	"required_unless": "The :attribute field is required unless :other is in :values.",
	"same":            "The :attribute field must match :other.",
	"string":          "The :attribute field must be a string.",
	"unique":          "The :attribute has already been taken.",
	"uuid":            "The :attribute field must be a valid UUID.",
	"between.array":   "The :attribute field must have between :min and :max items.",
	"between.file":    "The :attribute field must be between :min and :max kilobytes.",
	"between.numeric": "The :attribute field must be between :min and :max.",
	"between.string":  "The :attribute field must be between :min and :max characters.",
	"max.array":       "The :attribute field must not have more than :max items.",
	"max.file":        "The :attribute field must not be greater than :max kilobytes.",
	"max.numeric":     "The :attribute field must not be greater than :max.",
	"max.string":      "The :attribute field must not be greater than :max characters.",
	"min.array":       "The :attribute field must have at least :min items.",
	"min.file":        "The :attribute field must be at least :min kilobytes.",
	"min.numeric":     "The :attribute field must be at least :min.",
	"min.string":      "The :attribute field must be at least :min characters.",
}

var implicitRules = map[string]bool{"required": true, "required_unless": true}
var sizeRules = map[string]bool{"between": true, "max": true, "min": true, "size": true}

// R is one "attribute => rules" entry.
type R struct{ Attr, Rules string }

// Closure is a closure validation rule (runs after the string rules of its attribute):
// it returns the failure message, or "" when the value passes.
type Closure func(v any) string

// M is one custom "attribute.rule => message" entry.
type M struct{ Key, Msg string }

type rule struct {
	name   string
	params []string
	fn     func(v any) string
}

type attrRules struct {
	attr  string
	rules []rule
}

// ValidationError carries the 422 payload.
type ValidationError struct {
	Message string
	Errors  *php.Arr
}

func (e *ValidationError) Error() string { return e.Message }

type validator struct {
	c        *Ctx
	data     *php.Arr
	rules    []attrRules
	implicit map[string]bool // expanded wildcard attributes
	messages []M
	errs     *php.Arr
	failed   map[string]map[string]bool
}

func parseRules(s string) []rule {
	var out []rule
	for _, part := range strings.Split(s, "|") {
		name, params, _ := strings.Cut(part, ":")
		r := rule{name: strings.ToLower(strings.TrimSpace(name))}
		if params != "" {
			r.params = strings.Split(params, ",")
		}
		out = append(out, r)
	}
	return out
}

// Validate runs a FormRequest validation over $request->all(); on failure it panics
// with a *ValidationError (rendered as Laravel's 422 JSON).
func (c *Ctx) Validate(rules []R, messages ...M) { c.ValidateWith(rules, nil, messages...) }

// ValidateWith is Validate with closure rules keyed by attribute.
func (c *Ctx) ValidateWith(rules []R, closures map[string]Closure, messages ...M) {
	v := &validator{c: c, data: c.req.all(), implicit: map[string]bool{}, messages: messages,
		errs: php.NewArr(), failed: map[string]map[string]bool{}}
	var wild []attrRules
	for _, r := range rules {
		rs := parseRules(r.Rules)
		if fn := closures[r.Attr]; fn != nil {
			rs = append(rs, rule{name: "closure", fn: fn})
		}
		if strings.Contains(r.Attr, "*") {
			for _, a := range v.expand(r.Attr) {
				v.implicit[a] = true
				wild = append(wild, attrRules{a, rs})
			}
			continue
		}
		v.rules = append(v.rules, attrRules{r.Attr, rs})
	}
	// mergeRules: an expanded key that already exists gets the rules merged in place.
	for _, w := range wild {
		merged := false
		for i := range v.rules {
			if v.rules[i].attr == w.attr {
				v.rules[i].rules = append(v.rules[i].rules, w.rules...)
				merged = true
				break
			}
		}
		if !merged {
			v.rules = append(v.rules, w)
		}
	}
	for _, ar := range v.rules {
		for _, r := range ar.rules {
			v.validateAttribute(ar.attr, r)
			if v.shouldStop(ar.attr, ar.rules) {
				break
			}
		}
	}
	if v.errs.Len() == 0 {
		return
	}
	var all []string
	for _, k := range v.errs.Keys() {
		for _, m := range v.errs.Val(k).(*php.Arr).Values() {
			all = append(all, m.(string))
		}
	}
	msg := all[0]
	if n := len(all) - 1; n > 0 {
		w := "errors"
		if n == 1 {
			w = "error"
		}
		msg += fmt.Sprintf(" (and %d more %s)", n, w)
	}
	panic(&ValidationError{Message: msg, Errors: v.errs})
}

func (v *validator) shouldStop(attr string, rules []rule) bool {
	f := v.failed[attr]
	if f == nil {
		return false
	}
	for _, r := range rules {
		if implicitRules[r.name] {
			for name := range f {
				if implicitRules[name] {
					return true
				}
			}
			return false
		}
	}
	return false
}

// expand resolves a wildcard attribute against the data (ValidationData::initializeAndGatherData).
func (v *validator) expand(attr string) []string {
	explicit := attr[:strings.Index(attr, "*")]
	explicit = strings.TrimSuffix(explicit, ".")
	// extractDataFromPath
	data := php.NewArr()
	if val, ok := php.GetPath(v.data, explicit); ok {
		setPath(data, php.SplitDots(explicit), val)
	}
	data = data.Clone()
	if !strings.HasSuffix(attr, "*") {
		dataSetInit(data, php.SplitDots(attr))
	}
	flat := php.NewArr()
	dotFlatten(flat, data, "")
	pattern := regexp.MustCompile(`^` + strings.ReplaceAll(regexp.QuoteMeta(attr), `\*`, `[^\.]+`))
	extra := php.NewArr()
	for _, k := range flat.Keys() {
		if m := pattern.FindString(php.ToString(k)); m != "" {
			if !extra.Has(m) {
				extra.Set(m, true)
			}
		}
	}
	for _, k := range extra.Keys() {
		val, _ := php.GetPath(v.data, php.ToString(k))
		flat.Set(k, val)
	}
	full := regexp.MustCompile(`^` + strings.ReplaceAll(regexp.QuoteMeta(attr), `\*`, `[^\.]*`) + `$`)
	var out []string
	for _, k := range flat.Keys() {
		ks := php.ToString(k)
		if strings.HasPrefix(ks, attr) || full.MatchString(ks) {
			out = append(out, ks)
		}
	}
	return out
}

func setPath(a *php.Arr, segs []string, val any) {
	for i, s := range segs {
		if i == len(segs)-1 {
			a.Set(s, val)
			return
		}
		next, ok := a.Val(s).(*php.Arr)
		if !ok {
			next = php.NewArr()
			a.Set(s, next)
		}
		a = next
	}
}

// dataSetInit is data_set($data, $attr, null, false) on a fresh structure.
func dataSetInit(target any, segs []string) any {
	if len(segs) == 0 {
		return target
	}
	seg, rest := segs[0], segs[1:]
	if seg == "*" {
		a, ok := target.(*php.Arr)
		if !ok {
			a = php.NewArr()
		}
		if len(rest) > 0 {
			for _, k := range a.Keys() {
				a.Set(k, dataSetInit(a.Val(k), rest))
			}
		}
		return a
	}
	if a, ok := target.(*php.Arr); ok {
		if len(rest) > 0 {
			if !a.Has(seg) {
				a.Set(seg, php.NewArr())
			}
			a.Set(seg, dataSetInit(a.Val(seg), rest))
		} else if !a.Has(seg) {
			a.Set(seg, nil)
		}
		return a
	}
	// Observed Laravel 11.10 behaviour: a scalar element is replaced by [seg => null].
	a := php.NewArr()
	if len(rest) > 0 {
		a.Set(seg, dataSetInit(nil, rest))
	} else {
		a.Set(seg, nil)
	}
	return a
}

func dotFlatten(out *php.Arr, a *php.Arr, prefix string) {
	for _, k := range a.Keys() {
		key := prefix + php.ToString(k)
		val := a.Val(k)
		if sub, ok := val.(*php.Arr); ok && sub.Len() > 0 {
			dotFlatten(out, sub, key+".")
			continue
		}
		out.Set(key, val)
	}
}

func (v *validator) value(attr string) any {
	val, _ := php.GetPath(v.data, attr)
	return val
}

func (v *validator) present(attr string) bool {
	_, ok := php.GetPath(v.data, attr)
	return ok
}

func (v *validator) hasRule(attr string, names ...string) bool {
	for _, ar := range v.rules {
		if ar.attr == attr {
			for _, r := range ar.rules {
				for _, n := range names {
					if r.name == n {
						return true
					}
				}
			}
		}
	}
	return false
}

func (v *validator) validateAttribute(attr string, r rule) {
	if r.name == "" {
		return
	}
	val := v.value(attr)
	// isValidatable
	implicit := implicitRules[r.name]
	if s, ok := val.(string); ok && php.Trim(s) == "" {
		if !implicit {
			return
		}
	} else if !v.present(attr) && !implicit {
		return
	}
	if !implicit && v.hasRule(attr, "nullable") && val == nil {
		return
	}
	if (r.name == "exists" || r.name == "unique") && v.errs.Has(attr) {
		return
	}
	if r.fn != nil {
		if msg := r.fn(val); msg != "" {
			v.addMessage(attr, r.name, msg)
		}
		return
	}
	if !v.passes(attr, val, r) {
		v.addFailure(attr, r)
	}
}

func isFile(v any) (*UploadedFile, bool) {
	f, ok := v.(*UploadedFile)
	return f, ok
}

func (v *validator) passes(attr string, val any, r rule) bool {
	switch r.name {
	case "nullable":
		return true
	case "required":
		return validateRequired(val)
	case "required_unless":
		other, _ := php.GetPath(v.data, r.params[0])
		values := r.params[1:]
		vals := make([]any, len(values))
		for i, x := range values {
			vals[i] = x
			if other == nil && x == "null" {
				vals[i] = nil
			}
		}
		strict := false
		if _, ok := other.(bool); ok || other == nil {
			strict = true
		}
		if !php.InArray(other, php.List(vals...), strict) {
			return validateRequired(val)
		}
		return true
	case "string":
		_, ok := val.(string)
		return ok
	case "integer":
		return php.FilterInt(val)
	case "boolean":
		switch t := val.(type) {
		case bool:
			return true
		case int64:
			return t == 0 || t == 1
		case string:
			return t == "0" || t == "1"
		}
		return false
	case "array":
		_, ok := val.(*php.Arr)
		return ok
	case "uuid":
		return php.IsUUID(val)
	case "min", "max", "between":
		if f, ok := isFile(val); ok && f == nil {
			return false
		}
		size := v.size(attr, val)
		p0, _ := php.NumericString(strings.TrimSpace(r.params[0]))
		switch r.name {
		case "min":
			return php.Ge(size, p0)
		case "max":
			return php.Le(size, p0)
		default:
			p1, _ := php.NumericString(strings.TrimSpace(r.params[1]))
			return php.Ge(size, p0) && php.Le(size, p1)
		}
	case "same":
		other, _ := php.GetPath(v.data, r.params[0])
		return php.StrictEq(val, other)
	case "different":
		for _, p := range r.params {
			if other, ok := php.GetPath(v.data, p); ok {
				if php.StrictEq(val, other) {
					return false
				}
			}
		}
		return true
	case "exists", "unique":
		table, col := r.params[0], r.params[1]
		if a, ok := val.(*php.Arr); ok {
			uniq := php.NewArr()
			for _, x := range a.Values() {
				if _, isArr := x.(*php.Arr); isArr {
					php.Fail("Array to string conversion")
				}
				if !php.InArray(x, uniq, false) {
					uniq.Append(x)
				}
			}
			if uniq.Len() == 0 {
				return r.name == "exists"
			}
			n := v.c.Count(`select count(distinct "`+col+`") as aggregate from "`+table+`" where "`+col+`" in (`+placeholders(uniq.Len())+`)`, uniq.Values()...)
			if r.name == "exists" {
				return n >= int64(uniq.Len())
			}
			return n == 0
		}
		n := v.c.Count(`select count(*) as aggregate from "`+table+`" where "`+col+`" = ?`, val)
		if r.name == "exists" {
			return n >= 1
		}
		return n == 0
	case "image":
		return validateMimes(val, []string{"jpg", "jpeg", "png", "gif", "bmp", "svg", "webp"})
	case "mimes":
		return validateMimes(val, r.params)
	case "dimensions":
		f, ok := isFile(val)
		if !ok || f == nil {
			return false
		}
		cfg, _, err := image.DecodeConfig(bytes.NewReader(f.Data))
		if err != nil {
			return false
		}
		for _, p := range r.params {
			k, x, _ := strings.Cut(p, "=")
			n, _ := strconv.Atoi(x)
			switch k {
			case "min_width":
				if cfg.Width < n {
					return false
				}
			case "min_height":
				if cfg.Height < n {
					return false
				}
			case "max_width":
				if cfg.Width > n {
					return false
				}
			case "max_height":
				if cfg.Height > n {
					return false
				}
			}
		}
		return true
	}
	return true
}

func validateRequired(val any) bool {
	switch t := val.(type) {
	case nil:
		return false
	case string:
		return php.Trim(t) != ""
	case *php.Arr:
		return t.Len() > 0
	case *UploadedFile:
		return t != nil
	}
	return true
}

// guessExtension maps the sniffed MIME type to an extension (Symfony MimeTypes).
func guessExtension(data []byte) string {
	switch mimeType(data) {
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "image/bmp":
		return "bmp"
	case "image/webp":
		return "webp"
	case "image/svg+xml":
		return "svg"
	}
	return ""
}

func mimeType(data []byte) string {
	t := http.DetectContentType(data)
	t, _, _ = strings.Cut(t, ";")
	switch t {
	case "text/xml", "text/plain":
		if bytes.Contains(data[:min(len(data), 1024)], []byte("<svg")) {
			return "image/svg+xml"
		}
	}
	return t
}

func validateMimes(val any, params []string) bool {
	f, ok := isFile(val)
	if !ok || f == nil {
		return false
	}
	ext := strings.ToLower(strings.TrimSpace(fileExt(f.ClientName)))
	switch ext {
	case "php", "php3", "php4", "php5", "php7", "php8", "phtml", "phar":
		return false
	}
	ps := append([]string(nil), params...)
	for _, p := range params {
		if p == "jpg" || p == "jpeg" {
			ps = append(ps, "jpg", "jpeg")
			break
		}
	}
	g := guessExtension(f.Data)
	for _, p := range ps {
		if p == g {
			return true
		}
	}
	return false
}

func fileExt(name string) string {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[i+1:]
	}
	return ""
}

func (v *validator) size(attr string, val any) any {
	if v.hasRule(attr, "integer", "numeric", "decimal") && php.IsNumeric(val) {
		if s, ok := val.(string); ok {
			n, _ := php.NumericString(s)
			return n
		}
		return val
	}
	switch t := val.(type) {
	case *php.Arr:
		return int64(t.Len())
	case *UploadedFile:
		return float64(len(t.Data)) / 1024
	}
	return int64(len([]rune(php.ToString(val))))
}

func (v *validator) attrType(attr string) string {
	switch {
	case v.hasRule(attr, "integer", "numeric", "decimal"):
		return "numeric"
	case v.hasRule(attr, "array"):
		return "array"
	}
	if _, ok := isFile(v.value(attr)); ok {
		return "file"
	}
	return "string"
}

func (v *validator) display(attr string) string {
	if v.implicit[attr] {
		return attr
	}
	return strings.ReplaceAll(php.Snake(attr), "_", " ")
}

func (v *validator) inlineMessage(attr, rule string) (string, bool) {
	for _, key := range []string{attr + "." + rule, rule, attr} {
		for _, m := range v.messages {
			if strings.Contains(m.Key, "*") {
				pat := `^` + strings.ReplaceAll(regexp.QuoteMeta(m.Key), `\*`, `([^.]*)`) + `$`
				if regexp.MustCompile(pat).MatchString(key) {
					return m.Msg, true
				}
				continue
			}
			if m.Key == key {
				return m.Msg, true
			}
		}
	}
	return "", false
}

func (v *validator) addFailure(attr string, r rule) {
	if v.failed[attr] == nil {
		v.failed[attr] = map[string]bool{}
	}
	v.failed[attr][r.name] = true
	msg, ok := v.inlineMessage(attr, r.name)
	if !ok {
		if sizeRules[r.name] {
			msg = defaultMessages[r.name+"."+v.attrType(attr)]
		} else {
			msg = defaultMessages[r.name]
		}
	}
	disp := v.display(attr)
	msg = strings.NewReplacer(":attribute", disp, ":ATTRIBUTE", strings.ToUpper(disp), ":Attribute", ucfirst(disp)).Replace(msg)
	switch r.name {
	case "min":
		msg = strings.ReplaceAll(msg, ":min", r.params[0])
	case "max":
		msg = strings.ReplaceAll(msg, ":max", r.params[0])
	case "between":
		msg = strings.NewReplacer(":min", r.params[0], ":max", r.params[1]).Replace(msg)
	case "same", "different":
		msg = strings.ReplaceAll(msg, ":other", v.display(r.params[0]))
	case "required_unless":
		msg = strings.NewReplacer(":other", v.display(r.params[0]), ":values", strings.Join(r.params[1:], ", ")).Replace(msg)
	case "mimes":
		msg = strings.ReplaceAll(msg, ":values", strings.Join(r.params, ", "))
	}
	v.addMessage(attr, r.name, msg)
}

func (v *validator) addMessage(attr, ruleName, msg string) {
	if v.failed[attr] == nil {
		v.failed[attr] = map[string]bool{}
	}
	v.failed[attr][ruleName] = true
	list, _ := v.errs.Val(attr).(*php.Arr)
	if list == nil {
		list = php.NewArr()
		v.errs.Set(attr, list)
	}
	list.Append(msg)
}

func ucfirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

var _ = sort.Strings
