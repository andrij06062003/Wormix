package app

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/andrij06062003/Wormix/services/api/internal/php"
)

// UploadedFile is a file from a multipart request.
type UploadedFile struct {
	ClientName string
	Data       []byte
}

// parsedRequest holds the Laravel request bags.
type parsedRequest struct {
	isJSON  bool
	query   *php.Arr
	form    *php.Arr // Symfony "request" bag for non-JSON bodies
	json    *php.Arr // $request->json()
	files   *php.Arr
	body    []byte
	method  string
	rawPath string
}

const maxBody = 64 << 20

func parseRequest(r *http.Request) *parsedRequest {
	p := &parsedRequest{query: php.NewArr(), form: php.NewArr(), files: php.NewArr()}
	ct := r.Header.Get("Content-Type")
	p.isJSON = strings.Contains(ct, "/json") || strings.Contains(ct, "+json")
	parsePHPQuery(p.query, r.URL.RawQuery)

	mediatype, params, _ := mime.ParseMediaType(ct)
	if mediatype == "multipart/form-data" && r.Method == http.MethodPost {
		parseMultipart(p, r.Body, params["boundary"])
	} else {
		p.body, _ = io.ReadAll(io.LimitReader(r.Body, maxBody))
		if mediatype == "application/x-www-form-urlencoded" && r.Method != http.MethodGet && r.Method != http.MethodHead {
			parsePHPQuery(p.form, string(p.body))
		}
	}

	// $request->json(): (array) json_decode($content ?: '[]', true)
	body := p.body
	if len(body) == 0 {
		body = []byte("[]")
	}
	p.json = php.NewArr()
	if v, ok := php.JSONDecode(body); ok {
		switch t := v.(type) {
		case *php.Arr:
			p.json = t
		case nil:
		default:
			p.json.Append(t)
		}
	}

	// Symfony method override (_method / X-HTTP-Method-Override) for POST.
	p.method = r.Method
	if p.method == http.MethodHead {
		p.method = http.MethodGet
	}
	if r.Method == http.MethodPost {
		ov := r.Header.Get("X-Http-Method-Override")
		if ov == "" {
			if v, ok := p.form.Get("_method"); ok {
				ov = php.ToString(v)
			} else if v, ok := p.query.Get("_method"); ok {
				ov = php.ToString(v)
			}
		}
		if ov != "" {
			m := strings.ToUpper(ov)
			if isTokenMethod(m) {
				p.method = m
			}
		}
	}

	// Global middleware TrimStrings + ConvertEmptyStringsToNull.
	cleanBag(p.query, "")
	if p.isJSON {
		cleanBag(p.json, "")
	} else {
		cleanBag(p.form, "")
	}
	return p
}

func isTokenMethod(m string) bool {
	switch m {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD", "TRACE", "CONNECT", "PURGE":
		return true
	}
	return false
}

var neverTrim = map[string]bool{"current_password": true, "password": true, "password_confirmation": true}

func cleanBag(a *php.Arr, prefix string) {
	for _, k := range a.Keys() {
		key := prefix + php.ToString(k)
		v := a.Val(k)
		switch t := v.(type) {
		case *php.Arr:
			cleanBag(t, key+".")
		case string:
			if !neverTrim[key] {
				t = php.LaravelTrim(t)
			}
			if t == "" {
				a.Set(k, nil)
			} else {
				a.Set(k, t)
			}
		}
	}
}

// inputSource is getInputSource(): json for JSON requests, query for GET, form otherwise.
func (p *parsedRequest) inputSource() *php.Arr {
	if p.isJSON {
		return p.json
	}
	if p.method == http.MethodGet {
		return p.query
	}
	return p.form
}

// all() = input() + query, with files merged (array_replace_recursive).
func (p *parsedRequest) all() *php.Arr {
	out := p.query.Clone()
	src := p.inputSource()
	// input() = source + query: source keys first, then query keys not in source.
	res := php.NewArr()
	for _, k := range src.Keys() {
		res.Set(k, src.Val(k))
	}
	for _, k := range out.Keys() {
		if !res.Has(k) {
			res.Set(k, out.Val(k))
		}
	}
	replaceRecursive(res, p.files)
	return res
}

func replaceRecursive(dst, src *php.Arr) {
	for _, k := range src.Keys() {
		sv := src.Val(k)
		if sa, ok := sv.(*php.Arr); ok {
			if da, ok := dst.Val(k).(*php.Arr); ok {
				da = da.Clone()
				replaceRecursive(da, sa)
				dst.Set(k, da)
				continue
			}
		}
		dst.Set(k, sv)
	}
}

// parsePHPQuery is parse_str() into arr (keeps order, supports a[b][]=c).
func parsePHPQuery(arr *php.Arr, raw string) {
	for _, pair := range strings.FieldsFunc(raw, func(r rune) bool { return r == '&' }) {
		k, v, _ := strings.Cut(pair, "=")
		key, err := url.QueryUnescape(k)
		if err != nil {
			continue
		}
		val, err := url.QueryUnescape(v)
		if err != nil {
			continue
		}
		phpSetVar(arr, key, val)
	}
}

func phpSetVar(arr *php.Arr, name string, value any) {
	base := name
	rest := ""
	if i := strings.IndexByte(name, '['); i > 0 && strings.Contains(name[i:], "]") {
		base, rest = name[:i], name[i:]
	}
	base = strings.TrimLeft(base, " ")
	if base == "" {
		return
	}
	base = strings.Map(func(r rune) rune {
		if r == ' ' || r == '.' || r == '[' {
			return '_'
		}
		return r
	}, base)
	var segs []string
	for strings.HasPrefix(rest, "[") {
		j := strings.IndexByte(rest, ']')
		if j < 0 {
			break
		}
		segs = append(segs, rest[1:j])
		rest = rest[j+1:]
	}
	if len(segs) == 0 {
		arr.Set(base, value)
		return
	}
	cur := arr
	key := any(base)
	for _, s := range segs {
		next, ok := cur.Val(key).(*php.Arr)
		if !ok {
			next = php.NewArr()
			cur.Set(key, next)
		}
		cur = next
		if s == "" {
			key = nil
		} else {
			key = s
		}
	}
	if key == nil {
		cur.Append(value)
	} else {
		cur.Set(key, value)
	}
}

func parseMultipart(p *parsedRequest, body io.Reader, boundary string) {
	if boundary == "" {
		return
	}
	mr := multipart.NewReader(io.LimitReader(body, maxBody), boundary)
	for {
		part, err := mr.NextPart()
		if err != nil {
			return
		}
		name := part.FormName()
		data, _ := io.ReadAll(part)
		if name == "" {
			continue
		}
		if part.FileName() != "" || part.Header.Get("Content-Type") != "" && part.FileName() != "" {
			if len(data) == 0 {
				phpSetVar(p.files, name, nil)
			} else {
				phpSetVar(p.files, name, &UploadedFile{ClientName: part.FileName(), Data: bytes.Clone(data)})
			}
			continue
		}
		phpSetVar(p.form, name, string(data))
	}
}
