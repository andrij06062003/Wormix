// Game host for running the Wormix Flash client in a modern browser with Ruffle.
//
//  1. Serves the client files (play.html, preloader.swf, vk/1.25.0/..., vk/storage/...) from "ClientRoot".
//
//  2. Relays the game's network connections: browsers can't open raw TCP sockets, so Ruffle sends
//     them over a WebSocket (its "socketProxy" setting). Those are forwarded, byte for byte, to
//     the wormix-core server ports:
//
//     ws://<host>:32700/49197  ->  tcp <Target>:49197
//
//  3. Forwards /api/... to wormix-api, so the game, the relay and the API share one address.
//
//  4. Serves the achievements connection (HTTP, not a socket): see achievements.go.
//
// Only ports listed in "AllowedPorts" can be reached, so the relay can't be used to connect anywhere else.
// Settings: appsettings.json, environment variables or command line, e.g.
// --Urls http://0.0.0.0:32700 --Target 192.168.1.50  (see config.go).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

const shutdownTimeout = 30 * time.Second // ASP.NET's default HostOptions.ShutdownTimeout

var defaultAllowedPorts = []int{49197, 49198, 49199, 49297, 49298, 49299, 49300}

type server struct {
	log          *slog.Logger
	target       string
	allowedPorts []int
	apiBase      string
	http         *http.Client
	logAPI       bool
	debugLogOn   bool
	static       *staticFiles // nil when ClientRoot doesn't exist

	relayCtx    context.Context // cancelled when open game connections must close
	relayCancel context.CancelFunc
	sessions    sync.WaitGroup
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	cfg, err := LoadConfig(os.Args[1:], os.Environ(), cwd)
	if err != nil {
		return err
	}
	exeDir := cwd
	if exe, err := os.Executable(); err == nil {
		exeDir = filepath.Dir(exe)
	}
	s, err := newServer(cfg, logger, exeDir, cwd)
	if err != nil {
		return err
	}

	urls := cfg.String("Urls", "http://127.0.0.1:32700")
	listeners, err := listen(urls)
	if err != nil {
		return err
	}
	srv := &http.Server{
		Handler:           s,
		ReadHeaderTimeout: 30 * time.Second, // Kestrel's RequestHeadersTimeout
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelDebug),
	}
	errc := make(chan error, len(listeners))
	for _, l := range listeners {
		logger.Info("Now listening on: http://" + l.Addr().String())
		go func() { errc <- srv.Serve(l) }()
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, os.Interrupt)
	select {
	case err := <-errc:
		return err
	case <-sig:
	}
	logger.Info("Application is shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	// Stop accepting, let running requests and game connections finish until the timeout,
	// then close the game connections that are still open.
	srvDone := make(chan error, 1)
	go func() { srvDone <- srv.Shutdown(ctx) }()
	sessionsDone := make(chan struct{})
	go func() { s.sessions.Wait(); close(sessionsDone) }()
	select {
	case <-sessionsDone:
	case <-ctx.Done():
	}
	s.relayCancel()
	select {
	case <-sessionsDone:
	case <-time.After(5 * time.Second):
	}
	if err := <-srvDone; err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	srv.Close()
	return nil
}

func newServer(cfg *Config, logger *slog.Logger, exeDir, contentRoot string) (*server, error) {
	s := &server{
		log:     logger.With("category", "wormix-relay"),
		target:  cfg.String("Target", "127.0.0.1"),
		apiBase: strings.TrimRight(cfg.String("ApiUrl", "http://127.0.0.1:8000"), "/"),
		http:    newAPIClient(),
	}
	s.relayCtx, s.relayCancel = context.WithCancel(context.Background())
	ports, ok, err := cfg.Ints("AllowedPorts")
	if err != nil {
		return nil, err
	}
	if !ok {
		ports = defaultAllowedPorts
	}
	s.allowedPorts = ports
	if s.logAPI, err = cfg.Bool("LogApi", false); err != nil {
		return nil, err
	}
	if s.debugLogOn, err = cfg.Bool("DebugLog", false); err != nil {
		return nil, err
	}

	// ClientRoot is relative to the program's folder, else to the working directory.
	root := cfg.String("ClientRoot", "../wormix-client")
	clientRoot := absFrom(exeDir, root)
	if !isDir(clientRoot) {
		clientRoot = absFrom(contentRoot, root)
	}
	if isDir(clientRoot) {
		// This project's own pages (wwwroot/play.html) first, then the game files from ClientRoot.
		dirs := []string{clientRoot}
		if own := filepath.Join(contentRoot, "wwwroot"); isDir(own) {
			dirs = []string{own, clientRoot}
		}
		if s.static, err = newStaticFiles(dirs...); err != nil {
			return nil, err
		}
		s.log.Info("Serving game client from " + clientRoot)
	} else {
		s.log.Warn("Client folder " + clientRoot + " not found: only the relay is running")
	}
	return s, nil
}

func absFrom(base, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(base, p)
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// listen opens a listener per URL in a ';'-separated Urls value, like Kestrel.
func listen(urls string) ([]net.Listener, error) {
	var out []net.Listener
	fail := func(err error) ([]net.Listener, error) {
		for _, l := range out {
			l.Close()
		}
		return nil, err
	}
	for _, raw := range strings.Split(urls, ";") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		scheme, rest, ok := strings.Cut(raw, "://")
		if !ok || !strings.EqualFold(scheme, "http") {
			return fail(fmt.Errorf("unsupported URL %q in Urls: only http:// is supported", raw))
		}
		hostport, _, _ := strings.Cut(rest, "/")
		host, port, err := net.SplitHostPort(hostport)
		if err != nil {
			host, port = hostport, "80"
		}
		var addrs [][2]string // network, address
		switch strings.ToLower(host) {
		case "*", "+", "", "[::]", "::":
			addrs = [][2]string{{"tcp", ":" + port}}
		case "localhost":
			addrs = [][2]string{{"tcp4", "127.0.0.1:" + port}, {"tcp6", "[::1]:" + port}}
		default:
			ip := net.ParseIP(strings.Trim(host, "[]"))
			if ip == nil {
				// Kestrel binds any other host name to all interfaces.
				addrs = [][2]string{{"tcp", ":" + port}}
			} else if ip.To4() != nil {
				addrs = [][2]string{{"tcp4", net.JoinHostPort(ip.String(), port)}}
			} else {
				addrs = [][2]string{{"tcp6", net.JoinHostPort(ip.String(), port)}}
			}
		}
		for i, a := range addrs {
			l, err := net.Listen(a[0], a[1])
			if err != nil {
				if strings.EqualFold(host, "localhost") && i > 0 {
					continue // no IPv6 loopback: IPv4 is enough
				}
				return fail(err)
			}
			out = append(out, l)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no URL to listen on in Urls %q", urls)
	}
	return out, nil
}

// normalizePath removes "." and ".." segments from a decoded path, like Kestrel does.
func normalizePath(p string) string {
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	segs := strings.Split(p[1:], "/")
	out := make([]string, 0, len(segs))
	for i, seg := range segs {
		last := i == len(segs)-1
		switch seg {
		case ".":
			if last {
				out = append(out, "")
			}
		case "..":
			if len(out) > 0 {
				out = out[:len(out)-1]
			}
			if last {
				out = append(out, "")
			}
		default:
			out = append(out, seg)
		}
	}
	return "/" + strings.Join(out, "/")
}

func escapePath(p string) string { return (&url.URL{Path: p}).EscapedPath() }

// hasSegmentPrefix reports whether p is prefix or below it, ignoring case.
func hasSegmentPrefix(p, prefix string) bool {
	return len(p) >= len(prefix) && strings.EqualFold(p[:len(prefix)], prefix) &&
		(len(p) == len(prefix) || p[len(prefix)] == '/')
}

func methodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	w.WriteHeader(http.StatusMethodNotAllowed)
}

// ServeHTTP routes like the ASP.NET endpoints (case-insensitive, an optional trailing slash);
// requests no endpoint matches go to the static files.
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := normalizePath(r.URL.Path)
	route := p
	if len(route) > 1 {
		route = strings.TrimSuffix(route, "/")
	}
	switch {
	case strings.EqualFold(route, "/debug-log"):
		if r.Method != http.MethodPost {
			methodNotAllowed(w, "POST")
			return
		}
		s.debugLog(w, r)
		return
	case hasSegmentPrefix(p, "/api"):
		s.proxyAPI(w, r, p)
		return
	}
	for _, prefix := range achievementPrefixes {
		if hasSegmentPrefix(p, prefix) {
			if r.Method != http.MethodPost {
				methodNotAllowed(w, "POST")
				return
			}
			s.serveAchievements(w, r)
			return
		}
	}
	if seg := route[1:]; !strings.Contains(seg, "/") {
		if port, ok := parsePortSegment(seg); ok {
			s.relay(w, r, port)
			return
		}
	}
	if s.static != nil && s.static.serve(w, r, p) {
		return
	}
	w.WriteHeader(http.StatusNotFound)
}
