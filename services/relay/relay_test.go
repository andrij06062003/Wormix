package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// msg builds a game message: 7-byte header (with random-looking flag bits) and body.
func msg(id int, body []byte, highBits uint32) []byte {
	h := make([]byte, 7)
	h[0] = 0x57
	binary.BigEndian.PutUint16(h[1:], uint16(id))
	binary.BigEndian.PutUint32(h[3:], uint32(len(body))|highBits)
	return append(h, body...)
}

// chunkReader returns the data in the given chunk sizes, one Read per chunk.
type chunkReader struct {
	data   []byte
	chunks []int
}

func (c *chunkReader) Read(p []byte) (int, error) {
	if len(c.data) == 0 {
		return 0, io.EOF
	}
	n := len(c.data)
	if len(c.chunks) > 0 {
		n = min(c.chunks[0], n)
		c.chunks = c.chunks[1:]
	}
	n = copy(p, c.data[:n])
	c.data = c.data[n:]
	return n, nil
}

func runReassemble(t *testing.T, data []byte, chunks []int) [][]byte {
	t.Helper()
	var out [][]byte
	err := reassemble(&chunkReader{data: data, chunks: chunks}, func(b []byte) error {
		out = append(out, append([]byte(nil), b...))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestReassembleSplitHeader(t *testing.T) {
	m := msg(5, []byte("hello"), 0xC0000000) // both high bits set: must be ignored for the length
	// Byte by byte: nothing goes out until the whole message is there.
	chunks := make([]int, len(m))
	for i := range chunks {
		chunks[i] = 1
	}
	out := runReassemble(t, m, chunks)
	if len(out) != 1 || !bytes.Equal(out[0], m) {
		t.Fatalf("got %d sends %x, want one %x", len(out), out, m)
	}
	// Header split in the middle of the length field.
	out = runReassemble(t, m, []int{5, 7})
	if len(out) != 1 || !bytes.Equal(out[0], m) {
		t.Fatalf("split header: got %x", out)
	}
}

func TestReassembleMultipleMessagesPerRead(t *testing.T) {
	a, b, c := msg(1, []byte("aa"), 0), msg(2, nil, 0x40000000), msg(3, []byte("cccc"), 0x80000000)
	all := slices.Concat(a, b, c)
	// All three in one read: one WebSocket message.
	out := runReassemble(t, all, nil)
	if len(out) != 1 || !bytes.Equal(out[0], all) {
		t.Fatalf("got %x", out)
	}
	// a, b and half of c in the first read; rest of c later.
	cut := len(a) + len(b) + 3
	out = runReassemble(t, all, []int{cut, len(all) - cut})
	if len(out) != 2 || !bytes.Equal(out[0], slices.Concat(a, b)) || !bytes.Equal(out[1], c) {
		t.Fatalf("got %x", out)
	}
}

func TestReassembleBigBody(t *testing.T) {
	body := bytes.Repeat([]byte{0xAB}, 300_000) // bigger than the 64 KiB read buffer
	m := msg(9, body, 0x40000000)
	next := msg(10, []byte("x"), 0)
	out := runReassemble(t, slices.Concat(m, next[:4]), nil)
	if len(out) != 1 || !bytes.Equal(out[0], m) {
		t.Fatalf("got %d sends, first len %d", len(out), len(out[0]))
	}
}

func TestReassembleDropsIncompleteTailAtEOF(t *testing.T) {
	m := msg(1, []byte("ok"), 0)
	out := runReassemble(t, slices.Concat(m, []byte{0x57, 0, 1}), nil)
	if len(out) != 1 || !bytes.Equal(out[0], m) {
		t.Fatalf("got %x", out)
	}
}

func TestConfigPrecedence(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "appsettings.json"), []byte(`{
		// comments are allowed
		"Urls": "http://127.0.0.1:1", "Target": "json", "ApiUrl": "http://json",
		"AllowedPorts": [ 1, 2, 3 ], "DebugLog": true,
		"Logging": { "LogLevel": { "Default": "Information" } },
	}`), 0o644)
	os.WriteFile(filepath.Join(dir, "appsettings.Staging.json"), []byte(`{"ApiUrl": "http://staging"}`), 0o644)

	cfg, err := LoadConfig(nil, nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	if v := cfg.String("target", ""); v != "json" {
		t.Errorf("Target = %q, want json", v)
	}
	if v := cfg.String("Logging:LogLevel:Default", ""); v != "Information" {
		t.Errorf("nested = %q", v)
	}
	if b, _ := cfg.Bool("DebugLog", false); !b {
		t.Error("DebugLog from JSON not true")
	}
	if p, ok, _ := cfg.Ints("AllowedPorts"); !ok || !slices.Equal(p, []int{1, 2, 3}) {
		t.Errorf("AllowedPorts = %v %v", p, ok)
	}

	env := []string{
		"ASPNETCORE_URLS=http://prefixed", // loses to appsettings.json
		"ASPNETCORE_ENVIRONMENT=Staging",
		"TARGET=env",
		"AllowedPorts__0=10", // overrides one element only, like ASP.NET
		"AllowedPorts__10=11",
		"DebugLog=False",
	}
	cfg, err = LoadConfig(nil, env, dir)
	if err != nil {
		t.Fatal(err)
	}
	if v := cfg.String("Urls", ""); v != "http://127.0.0.1:1" {
		t.Errorf("Urls = %q", v)
	}
	if v := cfg.String("ApiUrl", ""); v != "http://staging" {
		t.Errorf("ApiUrl = %q, want staging file", v)
	}
	if v := cfg.String("Target", ""); v != "env" {
		t.Errorf("Target = %q, want env", v)
	}
	if p, _, _ := cfg.Ints("AllowedPorts"); !slices.Equal(p, []int{10, 2, 3, 11}) {
		t.Errorf("AllowedPorts = %v", p)
	}
	if b, _ := cfg.Bool("DebugLog", true); b {
		t.Error("DebugLog env override not applied")
	}

	args := []string{"--Target", "cli", "--ApiUrl=http://cli", "/Urls", "http://0.0.0.0:2", "LogApi=true", "stray", "--AllowedPorts:1", "20"}
	cfg, err = LoadConfig(args, env, dir)
	if err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]string{"Target": "cli", "ApiUrl": "http://cli", "Urls": "http://0.0.0.0:2", "LogApi": "true"} {
		if v := cfg.String(k, ""); v != want {
			t.Errorf("%s = %q, want %q", k, v, want)
		}
	}
	if p, _, _ := cfg.Ints("AllowedPorts"); !slices.Equal(p, []int{10, 20, 3, 11}) {
		t.Errorf("AllowedPorts = %v", p)
	}

	// Nothing set: defaults apply.
	cfg, _ = LoadConfig(nil, nil, t.TempDir())
	if _, ok, _ := cfg.Ints("AllowedPorts"); ok {
		t.Error("AllowedPorts should be unset")
	}
	if _, err := (&Config{values: map[string]string{"debuglog": "yes"}}).Bool("DebugLog", false); err == nil {
		t.Error("bad bool accepted")
	}
}

func testServer(t *testing.T, apiURL string) (*server, string) {
	t.Helper()
	dir := t.TempDir()
	own, client := filepath.Join(dir, "wwwroot"), filepath.Join(dir, "client")
	for _, d := range []string{own, client, filepath.Join(client, "vk", "1.05.0"), filepath.Join(client, "sub")} {
		os.MkdirAll(d, 0o755)
	}
	write := func(p, s string) { os.WriteFile(p, []byte(s), 0o644) }
	write(filepath.Join(own, "play.html"), "own play")
	write(filepath.Join(client, "play.html"), "client play")
	write(filepath.Join(client, "preloader.swf"), "0123456789")
	write(filepath.Join(client, "vk", "1.05.0", "a.xml"), "<a/>")
	write(filepath.Join(client, "sub", "play.html"), "sub play")
	write(filepath.Join(client, "x.unknownext"), "x")
	write(filepath.Join(client, ".hidden.txt"), "secret")
	write(filepath.Join(dir, "outside.txt"), "outside")
	os.Symlink(filepath.Join(dir, "outside.txt"), filepath.Join(client, "link.txt"))

	cfg := newConfig()
	cfg.set("ClientRoot", client)
	cfg.set("ApiUrl", apiURL)
	cfg.set("Target", "127.0.0.1")
	s, err := newServer(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), dir, dir)
	if err != nil {
		t.Fatal(err)
	}
	return s, dir
}

func rawGet(t *testing.T, h http.Handler, method, target string, hdr map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "http://relay.test/", nil)
	req.URL.Path = target // as the server decodes it, before normalizing
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestStaticFiles(t *testing.T) {
	s, _ := testServer(t, "http://127.0.0.1:1")
	cases := []struct {
		method, path string
		status       int
		body, ctype  string
	}{
		{"GET", "/", 200, "own play", "text/html"},
		{"GET", "/play.html", 200, "own play", "text/html"},
		{"HEAD", "/play.html", 200, "", "text/html"},
		{"GET", "/preloader.swf", 200, "0123456789", "application/x-shockwave-flash"},
		{"GET", "/vk/1.05.0/a.xml", 200, "<a/>", "text/xml"},
		{"GET", "/sub/", 200, "sub play", "text/html"},
		{"GET", "/sub", 301, "", ""},
		{"GET", "/vk/", 404, "", ""},
		{"GET", "/missing.swf", 404, "", ""},
		{"GET", "/x.unknownext", 404, "", ""},
		{"GET", "/.hidden.txt", 404, "", ""},
		{"POST", "/play.html", 404, "", ""},
		// path traversal
		{"GET", "/../outside.txt", 404, "", ""},
		{"GET", "/vk/../../outside.txt", 404, "", ""},
		{"GET", "/../../../../etc/passwd", 404, "", ""},
		{"GET", "/link.txt", 404, "", ""}, // symlink out of the root
		{"GET", "/vk/1.05.0/../../preloader.swf", 200, "0123456789", "application/x-shockwave-flash"},
	}
	for _, c := range cases {
		rec := rawGet(t, s, c.method, c.path, nil)
		if rec.Code != c.status || rec.Body.String() != c.body || (c.ctype != "" && rec.Header().Get("Content-Type") != c.ctype) {
			t.Errorf("%s %s: %d %q %q, want %d %q %q", c.method, c.path, rec.Code, rec.Body.String(), rec.Header().Get("Content-Type"), c.status, c.body, c.ctype)
		}
		if c.status == 200 && rec.Header().Get("Cache-Control") != "no-cache" {
			t.Errorf("%s: no Cache-Control", c.path)
		}
	}
	if loc := rawGet(t, s, "GET", "/sub", nil).Header().Get("Location"); loc != "http://relay.test/sub/" {
		t.Errorf("redirect Location %q", loc)
	}

	rec := rawGet(t, s, "GET", "/preloader.swf", map[string]string{"Range": "bytes=2-4"})
	if rec.Code != 206 || rec.Body.String() != "234" || rec.Header().Get("Content-Range") != "bytes 2-4/10" {
		t.Errorf("range: %d %q %q", rec.Code, rec.Body.String(), rec.Header().Get("Content-Range"))
	}
	if rec := rawGet(t, s, "GET", "/preloader.swf", map[string]string{"Range": "bytes=20-"}); rec.Code != 416 || rec.Header().Get("Content-Range") != "bytes */10" {
		t.Errorf("416: %d", rec.Code)
	}
	etag := rawGet(t, s, "GET", "/preloader.swf", nil).Header().Get("ETag")
	if rec := rawGet(t, s, "GET", "/preloader.swf", map[string]string{"If-None-Match": etag}); rec.Code != 304 {
		t.Errorf("If-None-Match: %d", rec.Code)
	}
	if rec := rawGet(t, s, "GET", "/preloader.swf", map[string]string{"If-Match": `"other"`}); rec.Code != 412 {
		t.Errorf("If-Match: %d", rec.Code)
	}
}

func TestFileETagMatchesASPNET(t *testing.T) {
	// Values seen from the ASP.NET relay: play.html, 9974 bytes, modified 2026-11-05 20:08:26 UTC.
	mod := time.Date(2026, 11, 5, 20, 8, 26, 0, time.UTC)
	if got := fileETag(mod, 9974); got != `"1dd6d616aadbff6"` {
		t.Errorf("etag %s", got)
	}
}

func TestRouting(t *testing.T) {
	var gotPath, gotQuery, gotXFF, gotUA string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery, gotXFF, gotUA = r.URL.EscapedPath(), r.URL.RawQuery, r.Header.Get("X-Forwarded-For"), r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Add("Set-Cookie", "a=1")
		w.Header().Add("Set-Cookie", "b=2")
		w.WriteHeader(418)
		io.WriteString(w, `{"ok":1}`)
	}))
	defer api.Close()
	s, _ := testServer(t, api.URL+"/")

	// /api/internal is blocked whatever the case, dot segments or trailing slash.
	for _, p := range []string{"/api/internal", "/api/internal/", "/API/Internal/login", "/api/x/../internal/login", "/api/./internal/x"} {
		gotPath = ""
		if rec := rawGet(t, s, "GET", p, nil); rec.Code != 404 || gotPath != "" {
			t.Errorf("%s: %d, API saw %q", p, rec.Code, gotPath)
		}
	}
	for _, p := range []string{"/api/internalx", "/api/account/internal"} {
		if rec := rawGet(t, s, "GET", p, nil); rec.Code != 418 || gotPath != p {
			t.Errorf("%s: %d, API saw %q", p, rec.Code, gotPath)
		}
	}

	req := httptest.NewRequest("POST", "http://relay.test/api/account/sign_in?x=1&y=%20", strings.NewReader(`{"password":"p"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "1.1.1.1")
	req.RemoteAddr = "10.0.0.7:5555"
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != 418 || rec.Body.String() != `{"ok":1}` || len(rec.Header().Values("Set-Cookie")) != 2 {
		t.Errorf("proxy: %d %q %v", rec.Code, rec.Body.String(), rec.Header())
	}
	if gotPath != "/api/account/sign_in" || gotQuery != "x=1&y=%20" || gotXFF != "1.1.1.1, 10.0.0.7" || gotUA != "" {
		t.Errorf("API saw %q %q %q %q", gotPath, gotQuery, gotXFF, gotUA)
	}

	for _, c := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/debug-log", 405}, {"POST", "/debug-log", 404}, {"POST", "/Debug-Log/", 404},
		{"GET", "/wormix_achieve", 405}, {"GET", "/wormix_achievements/x", 405},
		{"POST", "/wormix_achieve/", 200}, {"POST", "/WORMIX_ACHIEVEMENTS", 200},
		{"GET", "/49197", 400}, {"GET", "/49197/", 400}, {"GET", "/+5", 400}, {"GET", "/99999999999", 404},
		{"GET", "/nothing", 404},
	} {
		if rec := rawGet(t, s, c.method, c.path, nil); rec.Code != c.status {
			t.Errorf("%s %s: %d, want %d", c.method, c.path, rec.Code, c.status)
		}
	}

	s.apiBase = "http://127.0.0.1:1" // nothing listens there
	if rec := rawGet(t, s, "GET", "/api/x", nil); rec.Code != 502 {
		t.Errorf("unreachable API: %d", rec.Code)
	}
}

func TestRedact(t *testing.T) {
	got := redact(`{"login":"a","password": "hunter2","token":"t","x":"y"}`)
	if got != `{"login":"a","password": "***","token":"***","x":"y"}` {
		t.Error(got)
	}
}

func TestWebSocketRelay(t *testing.T) {
	// A fake game server that echoes with a header split over several writes.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		buf := make([]byte, 5)
		if _, err := io.ReadFull(c, buf); err != nil {
			return
		}
		m := msg(7, buf, 0)
		for _, part := range [][]byte{m[:1], m[1:3], m[3:8], m[8:]} {
			c.Write(part)
			time.Sleep(20 * time.Millisecond)
		}
	}()

	s, _ := testServer(t, "http://127.0.0.1:1")
	s.allowedPorts = []int{port}
	hs := httptest.NewServer(s)
	defer hs.Close()
	wsURL := "ws" + strings.TrimPrefix(hs.URL, "http")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, resp, err := websocket.Dial(ctx, wsURL+"/1", nil); err == nil || resp == nil || resp.StatusCode != 403 {
		t.Errorf("disallowed port: %v %v", err, resp)
	}
	c, _, err := websocket.Dial(ctx, wsURL+"/"+strconv.Itoa(port), &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {"http://elsewhere"}}})
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	if err := c.Write(ctx, websocket.MessageBinary, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	typ, data, err := c.Read(ctx)
	if err != nil || typ != websocket.MessageBinary || !bytes.Equal(data, msg(7, []byte("hello"), 0)) {
		t.Fatalf("got %v %x %v", typ, data, err)
	}
	// The server hung up: the relay closes the WebSocket normally.
	_, _, err = c.Read(ctx)
	if websocket.CloseStatus(err) != websocket.StatusNormalClosure {
		t.Fatalf("close: %v", err)
	}

	// Allowed port, nothing listening: 502.
	ln2, _ := net.Listen("tcp", "127.0.0.1:0")
	dead := ln2.Addr().(*net.TCPAddr).Port
	ln2.Close()
	s.allowedPorts = append(s.allowedPorts, dead)
	if _, resp, err := websocket.Dial(ctx, wsURL+"/"+strconv.Itoa(dead), nil); err == nil || resp == nil || resp.StatusCode != 502 {
		t.Errorf("dead port: %v %v", err, resp)
	}
}
