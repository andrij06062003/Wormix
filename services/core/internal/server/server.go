// Package server runs the TCP servers: main (game commands), PvP (battles), achievements and the Flash
// domain-policy server. Port of the C# Server/ and Session/ directories.
package server

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/andrij06062003/Wormix/services/core/internal/cache"
	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

// Kind of server.
type Kind string

const (
	Main         Kind = "main"
	Pvp          Kind = "pvp"
	Achievements Kind = "achievements"
	Policy       Kind = "policy"
)

// Cache is the per-session scratch store shared by all main-server sessions.
var Cache = cache.New()

// Server is one listening TCP server and its sessions.
type Server struct {
	Kind Kind
	Addr string

	ln       net.Listener
	mu       sync.Mutex
	sessions map[string]session
	wg       sync.WaitGroup
	closing  atomic.Bool
}

type session interface {
	id() string
	run()
	stop()
}

// New creates a server of the given kind; nil for an unknown kind (ignored, like the C# config parser).
func New(kind string, addr string) *Server {
	switch Kind(kind) {
	case Main, Pvp, Achievements, Policy:
		return &Server{Kind: Kind(kind), Addr: addr, sessions: map[string]session{}}
	}
	return nil
}

// Start listens and accepts connections in the background.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	s.ln = ln
	s.wg.Add(1)
	go s.acceptLoop()
	return nil
}

// ListenAddr is the address the server listens on (useful with port 0).
func (s *Server) ListenAddr() string { return s.ln.Addr().String() }

// Close stops listening and closes all connections.
func (s *Server) Close() {
	s.closing.Store(true)
	if s.ln != nil {
		s.ln.Close()
	}
	s.mu.Lock()
	all := make([]session, 0, len(s.sessions))
	for _, x := range s.sessions {
		all = append(all, x)
	}
	s.mu.Unlock()
	for _, x := range all {
		x.stop()
	}
	s.wg.Wait()
}

func (s *Server) acceptLoop() {
	defer s.wg.Done()
	for {
		c, err := s.ln.Accept()
		if err != nil {
			if s.closing.Load() || errors.Is(err, net.ErrClosed) {
				return
			}
			log.Printf("[%s] accept error: %v", s.Kind, err)
			continue
		}
		if tc, ok := c.(*net.TCPConn); ok {
			tc.SetNoDelay(true)
		}
		base := newConn(s, c)
		var sess session
		switch s.Kind {
		case Main:
			sess = newMainSession(base)
		case Pvp:
			sess = newPvpSession(base)
		case Achievements:
			sess = &rawSession{conn: base, policy: false}
		case Policy:
			sess = &rawSession{conn: base, policy: true}
		}
		s.mu.Lock()
		s.sessions[base.sid] = sess
		s.mu.Unlock()
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			sess.run()
			sess.stop()
			s.mu.Lock()
			delete(s.sessions, base.sid)
			s.mu.Unlock()
		}()
	}
}

func (s *Server) find(id string) session {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessions[id]
}

// ---- conn: the common part of a session ------------------------------------------------------------------

type conn struct {
	srv    *Server
	sid    string
	c      net.Conn
	sendMu sync.Mutex

	// pending holds bytes written to the session stream that don't form a complete message yet
	// (the C# MessageWriteStream).
	pendMu  sync.Mutex
	pending []byte

	stopped atomic.Bool
}

func newConn(s *Server, c net.Conn) *conn {
	return &conn{srv: s, sid: newGUID(), c: c}
}

func (c *conn) id() string { return c.sid }

func (c *conn) short() string { return c.sid[:8] }

// SendMessage sends one or more complete messages. Safe to call from any goroutine.
func (c *conn) SendMessage(msg []byte) bool {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	_, err := c.c.Write(msg)
	return err == nil
}

// write is the session stream the serializers write to: it collects the pieces and sends each complete
// message (by the length in its header) in one piece; an incomplete tail waits for the next write.
func (c *conn) write(b []byte) {
	c.pendMu.Lock()
	defer c.pendMu.Unlock()
	c.pending = append(c.pending, b...)
	complete := 0
	for len(c.pending)-complete >= wire.HeaderSize {
		n := wire.FrameLength(c.pending[complete:])
		if len(c.pending)-complete < wire.HeaderSize+n {
			break
		}
		complete += wire.HeaderSize + n
	}
	if complete == 0 {
		return
	}
	c.SendMessage(c.pending[:complete])
	c.pending = append([]byte(nil), c.pending[complete:]...)
}

// Close closes the socket; the session's goroutine then ends and disconnects the session.
func (c *conn) Close() { c.c.Close() }

// ---- achievements and policy -------------------------------------------------------------------------------

const domainPolicy = "<?xml version=\"1.0\"?>\n<cross-domain-policy>\n" +
	"<site-control permitted-cross-domain-policies=\"all\"/>\n" +
	"<allow-http-request-headers-from domain=\"*\" headers=\"*\" secure=\"false\"/>\n" +
	"<allow-access-from domain=\"*\" to-ports=\"*\" secure=\"false\"/>\n</cross-domain-policy>\x00"

// rawSession is a policy-server connection (answers the Flash policy request and closes) or an
// achievements-server connection (the C# server registered no handlers there: data is ignored).
type rawSession struct {
	*conn
	policy bool
}

func (r *rawSession) run() {
	buf := make([]byte, 4096)
	for {
		n, err := r.c.Read(buf)
		if n > 0 && r.policy {
			r.SendMessage([]byte(domainPolicy))
			sleep1s()
			return
		}
		if err != nil {
			return
		}
	}
}

func (r *rawSession) stop() {
	if r.stopped.Swap(true) {
		return
	}
	r.c.Close()
}

// ---- ids ---------------------------------------------------------------------------------------------------

// newGUID returns a random GUID in .NET's "D" format (lowercase).
func newGUID() string {
	var b [16]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic(err)
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	h := hex.EncodeToString(b[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}

// parseGUID is Guid.Parse for the "D", "N", "B" and "P" formats; returns the "D" form.
func parseGUID(s string) (string, error) {
	s = strings.TrimSpace(s)
	if len(s) == 38 && (s[0] == '{' && s[37] == '}' || s[0] == '(' && s[37] == ')') {
		s = s[1:37]
	}
	var h string
	switch len(s) {
	case 36:
		if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
			return "", fmt.Errorf("unrecognized GUID format")
		}
		h = s[0:8] + s[9:13] + s[14:18] + s[19:23] + s[24:36]
	case 32:
		h = s
	default:
		return "", fmt.Errorf("unrecognized GUID format")
	}
	if _, err := hex.DecodeString(h); err != nil {
		return "", fmt.Errorf("unrecognized GUID format")
	}
	h = strings.ToLower(h)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32], nil
}
