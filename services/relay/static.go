package main

// Static files, matching ASP.NET Core's UseDefaultFiles + UseStaticFiles over a CompositeFileProvider:
// wwwroot first, then ClientRoot; "play.html" is the default document of any folder; only known
// extensions are served; conditional requests and single byte ranges work like StaticFileContext.

import (
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultDocument = "play.html"

type staticFiles struct {
	roots []*os.Root // searched in order
}

func newStaticFiles(dirs ...string) (*staticFiles, error) {
	s := &staticFiles{}
	for _, d := range dirs {
		r, err := os.OpenRoot(d)
		if err != nil {
			return nil, err
		}
		s.roots = append(s.roots, r)
	}
	return s, nil
}

// rel turns a cleaned URL path into a path relative to a root, or "" if it can't name a file.
func rel(p string) (string, bool) {
	p = strings.TrimLeft(p, "/")
	if strings.ContainsRune(p, 0) {
		return "", false
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return "", false // never happens after normalizePath, but be sure
		}
	}
	return p, true
}

// openFile finds a regular, non-hidden file in the first root that has it.
func (s *staticFiles) openFile(p string) (*os.File, fs.FileInfo) {
	name, ok := rel(p)
	if !ok || name == "" {
		return nil, nil
	}
	base := name[strings.LastIndexByte(name, '/')+1:]
	if base == "" || strings.HasPrefix(base, ".") {
		return nil, nil // PhysicalFileProvider excludes dot-prefixed files
	}
	for _, r := range s.roots {
		f, err := r.Open(name)
		if err != nil {
			continue
		}
		st, err := f.Stat()
		if err != nil || !st.Mode().IsRegular() {
			f.Close()
			continue
		}
		return f, st
	}
	return nil, nil
}

func (s *staticFiles) isDir(p string) bool {
	name, ok := rel(p)
	if !ok {
		return false
	}
	if name == "" {
		name = "."
	}
	for _, r := range s.roots {
		if st, err := r.Stat(strings.TrimSuffix(name, "/")); err == nil && st.IsDir() {
			return true
		}
	}
	return false
}

// serve handles a request that no route matched. It returns false if the request is not for a
// static file, so the caller answers 404.
func (s *staticFiles) serve(w http.ResponseWriter, r *http.Request, p string) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	// UseDefaultFiles: a folder containing play.html.
	if s.isDir(p) {
		if f, _ := s.openFile(strings.TrimSuffix(p, "/") + "/" + defaultDocument); f != nil {
			f.Close()
			if !strings.HasSuffix(p, "/") {
				// Redirect to the folder URL with a trailing slash, like ASP.NET.
				loc := "http://" + r.Host + escapePath(p) + "/"
				if r.URL.RawQuery != "" {
					loc += "?" + r.URL.RawQuery
				}
				w.Header().Set("Location", loc)
				w.WriteHeader(http.StatusMovedPermanently)
				return true
			}
			p += defaultDocument
		}
	}
	// UseStaticFiles. The content type comes from the text after the path's last '.', exactly
	// like FileExtensionContentTypeProvider.
	dot := strings.LastIndexByte(p, '.')
	if dot < 0 {
		return false
	}
	ctype, ok := aspnetContentTypes[strings.ToLower(p[dot:])]
	if !ok {
		return false
	}
	f, st := s.openFile(p)
	if f == nil {
		return false
	}
	defer f.Close()
	serveFile(w, r, f, st, ctype)
	return true
}

// Precondition states, in the order StaticFileContext takes the maximum of them.
const (
	preUnspecified = iota
	preNotModified
	preShouldProcess
	preFailed
)

func serveFile(w http.ResponseWriter, r *http.Request, f io.ReadSeeker, st fs.FileInfo, ctype string) {
	length := st.Size()
	lastModified := st.ModTime().UTC().Truncate(time.Second)
	etag := fileETag(lastModified, length)
	h := r.Header
	now := time.Now()

	state := preUnspecified
	max := func(s int) {
		if s > state {
			state = s
		}
	}
	// If-Match / If-Unmodified-Since
	if tags := parseETags(h.Values("If-Match")); len(tags) > 0 {
		s := preFailed
		for _, t := range tags {
			if t.any || (!t.weak && t.tag == etag) {
				s = preShouldProcess
				break
			}
		}
		max(s)
	}
	if t, ok := httpDate(h.Get("If-Unmodified-Since")); ok && !t.After(now) {
		if !t.Before(lastModified) {
			max(preShouldProcess)
		} else {
			max(preFailed)
		}
	}
	// If-None-Match / If-Modified-Since
	if tags := parseETags(h.Values("If-None-Match")); len(tags) > 0 {
		s := preShouldProcess
		for _, t := range tags {
			if t.any || t.tag == etag {
				s = preNotModified
				break
			}
		}
		max(s)
	}
	if t, ok := httpDate(h.Get("If-Modified-Since")); ok && !t.After(now) {
		if t.Before(lastModified) {
			max(preShouldProcess)
		} else {
			max(preNotModified)
		}
	}

	// Range (GET only), unless If-Range says the file changed.
	isRange, start, end := false, int64(0), int64(-1)
	if r.Method == http.MethodGet {
		useRange := true
		if ir := strings.TrimSpace(h.Get("If-Range")); ir != "" {
			if t, ok := httpDate(ir); ok {
				useRange = !lastModified.After(t)
			} else if tags := parseETags([]string{ir}); len(tags) == 1 && !tags[0].any {
				useRange = !tags[0].weak && tags[0].tag == etag
			}
		}
		if useRange {
			isRange, start, end = parseRange(h.Values("Range"), length)
		}
	}

	hdr := w.Header()
	apply := func(status int) {
		if status < 400 {
			hdr.Set("Content-Type", ctype)
			hdr.Set("Last-Modified", lastModified.Format(http.TimeFormat))
			hdr.Set("ETag", etag)
			hdr.Set("Accept-Ranges", "bytes")
		}
		if status == http.StatusOK {
			hdr.Set("Content-Length", strconv.FormatInt(length, 10))
		}
		hdr.Set("Cache-Control", "no-cache") // OnPrepareResponse
		w.WriteHeader(status)
	}

	switch state {
	case preNotModified:
		apply(http.StatusNotModified)
		return
	case preFailed:
		apply(http.StatusPreconditionFailed)
		return
	}
	if r.Method == http.MethodHead {
		apply(http.StatusOK)
		return
	}
	if isRange {
		if end < 0 {
			hdr.Set("Content-Range", "bytes */"+strconv.FormatInt(length, 10))
			apply(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		n := end - start + 1
		hdr.Set("Content-Range", "bytes "+strconv.FormatInt(start, 10)+"-"+strconv.FormatInt(end, 10)+"/"+strconv.FormatInt(length, 10))
		hdr.Set("Content-Length", strconv.FormatInt(n, 10))
		apply(http.StatusPartialContent)
		if _, err := f.Seek(start, io.SeekStart); err == nil {
			io.CopyN(w, f, n)
		}
		return
	}
	apply(http.StatusOK)
	io.Copy(w, f)
}

// fileETag is StaticFileContext's ETag: hex of (last modified as a Windows file time) XOR length.
func fileETag(lastModified time.Time, length int64) string {
	const ticksTo1970 = 116444736000000000 // 100 ns ticks from 1601-01-01 to 1970-01-01
	fileTime := lastModified.Unix()*10_000_000 + ticksTo1970
	return `"` + strconv.FormatUint(uint64(fileTime^length), 16) + `"`
}

type entityTag struct {
	tag  string // with quotes
	weak bool
	any  bool
}

func parseETags(values []string) []entityTag {
	var tags []entityTag
	for _, v := range values {
		for _, part := range strings.Split(v, ",") {
			part = strings.TrimSpace(part)
			switch {
			case part == "":
			case part == "*":
				tags = append(tags, entityTag{any: true})
			default:
				weak := false
				if strings.HasPrefix(part, "W/") || strings.HasPrefix(part, "w/") {
					weak, part = true, part[2:]
				}
				if len(part) >= 2 && part[0] == '"' && part[len(part)-1] == '"' {
					tags = append(tags, entityTag{tag: part, weak: weak})
				}
			}
		}
	}
	return tags
}

func httpDate(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	t, err := http.ParseTime(strings.TrimSpace(s))
	return t, err == nil
}

// parseRange follows ASP.NET's RangeHelper: only one "bytes=" range is honored (several ranges or
// a malformed header serve the whole file). end < 0 with isRange means 416.
func parseRange(values []string, length int64) (isRange bool, start, end int64) {
	if len(values) == 0 || (len(values) == 1 && values[0] == "") {
		return false, 0, -1
	}
	if len(values) > 1 || strings.Contains(values[0], ",") {
		return false, 0, -1
	}
	unit, spec, ok := strings.Cut(values[0], "=")
	if !ok || !strings.EqualFold(strings.TrimSpace(unit), "bytes") {
		return false, 0, -1
	}
	from, to, ok := strings.Cut(strings.TrimSpace(spec), "-")
	if !ok {
		return false, 0, -1
	}
	from, to = strings.TrimSpace(from), strings.TrimSpace(to)
	parse := func(s string) (int64, bool, error) {
		if s == "" {
			return 0, false, nil
		}
		for _, c := range s {
			if c < '0' || c > '9' {
				return 0, false, errors.New("bad")
			}
		}
		n, err := strconv.ParseInt(s, 10, 64)
		return n, true, err
	}
	f, hasFrom, err1 := parse(from)
	t, hasTo, err2 := parse(to)
	if err1 != nil || err2 != nil || (!hasFrom && !hasTo) || (hasFrom && hasTo && f > t) {
		return false, 0, -1
	}
	if length == 0 {
		return true, 0, -1
	}
	if hasFrom {
		if f >= length {
			return true, 0, -1
		}
		if !hasTo || t >= length {
			t = length - 1
		}
		return true, f, t
	}
	if t == 0 {
		return true, 0, -1
	}
	n := min(t, length)
	return true, length - n, length - 1
}
