package main

// /api/... is forwarded to wormix-api, so the game, the relay and the API share one address.

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const apiTimeout = 30 * time.Second

// maxRequestBody is Kestrel's default MaxRequestBodySize.
const maxRequestBody = 30_000_000

// Request and response headers that are not copied (compared case-insensitively).
var skipHeaders = []string{"Host", "Connection", "Transfer-Encoding", "Keep-Alive", "Upgrade", "Content-Length", "Content-Type"}

func skipHeader(name string) bool {
	for _, s := range skipHeaders {
		if strings.EqualFold(s, name) {
			return true
		}
	}
	return false
}

var secretField = regexp.MustCompile(`("(?:password|password_confirmation|auth_token|token)"\s*:\s*")[^"]*"`)

// redact hides passwords and tokens in a logged JSON body.
func redact(text string) string { return secretField.ReplaceAllString(text, `${1}***"`) }

// truncate cuts s to n UTF-16 code units like C#'s s[..n] + "...", on a rune boundary.
func truncate(s string, n int) string {
	units := 0
	for i, r := range s {
		w := 1
		if r >= 0x10000 {
			w = 2
		}
		if units+w > n {
			return s[:i] + "..."
		}
		units += w
	}
	return s
}

// newAPIClient mirrors .NET's HttpClient defaults: no transparent decompression, follows redirects.
func newAPIClient() *http.Client {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DisableCompression = true
	return &http.Client{
		Transport: t,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 50 {
				return errors.New("too many redirects")
			}
			return nil
		},
	}
}

// isInternal reports whether p is /api/internal or below it (segment match, any case).
func isInternal(p string) bool { return hasSegmentPrefix(p, "/api/internal") }

// readBody reads a request body within Kestrel's size limit; ok is false if it answered 413.
func readBody(w http.ResponseWriter, r *http.Request) ([]byte, bool, error) {
	if r.ContentLength > maxRequestBody {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return nil, false, nil
	}
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBody))
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return nil, false, nil
	}
	return b, err == nil, err
}

func (s *server) proxyAPI(w http.ResponseWriter, r *http.Request, p string) {
	// /api/internal/ is for wormix-core only (it changes money, items, battle results).
	// wormix-core calls the API directly, so players never need it: don't expose it.
	if isInternal(p) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	url := s.apiBase + escapePath(p)
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		url += "?" + r.URL.RawQuery
	}

	// Ends the API call after 30 s unless its headers arrived by then (HttpClient.Timeout with
	// ResponseHeadersRead); the body then streams for as long as the player stays connected.
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	timer := time.AfterFunc(apiTimeout, cancel)

	var body io.Reader
	var length int64
	if r.ContentLength > 0 || len(r.TransferEncoding) > 0 {
		b, ok, err := readBody(w, r)
		if !ok {
			timer.Stop()
			if err != nil {
				s.log.Warn("Can't read request body", "path", p, "error", err)
			}
			return
		}
		if s.logAPI {
			s.log.Info("API "+r.Method+" "+p+" "+truncate(redact(string(b)), 300), "method", r.Method, "path", p)
		}
		body, length = bytes.NewReader(b), int64(len(b))
	}
	req, err := http.NewRequestWithContext(ctx, r.Method, url, body)
	if err != nil {
		timer.Stop()
		s.log.Warn("Can't reach API at "+url+": "+err.Error(), "url", url)
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	req.ContentLength = length
	for k, vs := range r.Header {
		if !skipHeader(k) {
			req.Header[k] = append([]string(nil), vs...)
		}
	}
	if body != nil {
		if ct := r.Header.Get("Content-Type"); ct != "" {
			req.Header.Set("Content-Type", ct)
		}
	}
	if ip := remoteIP(r); ip != "" {
		if prev := req.Header.Values("X-Forwarded-For"); len(prev) > 0 {
			req.Header.Set("X-Forwarded-For", strings.Join(prev, ", ")+", "+ip)
		} else {
			req.Header.Set("X-Forwarded-For", ip)
		}
	}
	if _, ok := req.Header["User-Agent"]; !ok {
		req.Header["User-Agent"] = []string{""} // HttpClient sends none; stop Go adding its own
	}

	resp, err := s.http.Do(req)
	if !timer.Stop() && err == nil {
		// Timed out just as the headers arrived: same as a timeout.
		resp.Body.Close()
		err = context.DeadlineExceeded
	}
	if err != nil {
		s.log.Warn("Can't reach API at "+url+": "+err.Error(), "url", url)
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if s.logAPI && (resp.StatusCode < 200 || resp.StatusCode > 299) {
		s.log.Warn("API answered "+resp.Status+" for "+p, "status", resp.StatusCode, "path", p)
	}
	out := w.Header()
	for k, vs := range resp.Header {
		if !skipHeader(k) || k == "Content-Type" {
			out[k] = vs
		}
	}
	w.WriteHeader(resp.StatusCode)
	copyFlushing(w, resp.Body)
}

// copyFlushing streams src to w, flushing after every read so nothing waits in a buffer.
func copyFlushing(w http.ResponseWriter, src io.Reader) {
	rc := http.NewResponseController(w)
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				return
			}
			rc.Flush()
		}
		if err != nil {
			return
		}
	}
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (s *server) debugLog(w http.ResponseWriter, r *http.Request) {
	if !s.debugLogOn {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	b, ok, _ := readBody(w, r)
	if !ok {
		return
	}
	for _, line := range strings.Split(string(b), "\n") {
		if line != "" {
			s.log.Info("GAME " + truncate(line, 1000))
		}
	}
	w.WriteHeader(http.StatusNoContent)
}
