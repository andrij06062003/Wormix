package main

// WebSocket <-> TCP relay. Browsers can't open raw TCP sockets, so Ruffle sends the game's
// connections over a WebSocket (its "socketProxy" setting); they are forwarded to wormix-core:
//
//	ws://<host>:32700/49197  ->  tcp <Target>:49197

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
)

const (
	keepAliveInterval = 30 * time.Second
	relayBufferSize   = 64 * 1024
	messageHeaderSize = 7
)

// parsePortSegment matches the {port:int} route constraint: an optional sign and digits that fit
// an Int32.
func parsePortSegment(seg string) (int, bool) {
	if seg == "" {
		return 0, false
	}
	digits := strings.TrimLeft(seg, "+-")
	if len(seg)-len(digits) > 1 || digits == "" {
		return 0, false
	}
	for _, c := range digits {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	n, err := strconv.ParseInt(seg, 10, 32)
	if err != nil {
		return 0, false
	}
	return int(n), true
}

// isWebSocketRequest is ASP.NET's HttpContext.WebSockets.IsWebSocketRequest.
func isWebSocketRequest(r *http.Request) bool {
	if r.Method != http.MethodGet || r.ProtoMajor != 1 {
		return false
	}
	hasToken := func(header, token string) bool {
		for _, v := range r.Header.Values(header) {
			for _, t := range strings.Split(v, ",") {
				if strings.EqualFold(strings.TrimSpace(t), token) {
					return true
				}
			}
		}
		return false
	}
	if !hasToken("Connection", "upgrade") || !hasToken("Upgrade", "websocket") {
		return false
	}
	if strings.TrimSpace(r.Header.Get("Sec-WebSocket-Version")) != "13" {
		return false
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(r.Header.Get("Sec-WebSocket-Key")))
	return err == nil && len(key) == 16
}

func writeJSONString(w http.ResponseWriter, status int, s string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	io.WriteString(w, strconv.Quote(s))
}

func (s *server) relay(w http.ResponseWriter, r *http.Request, port int) {
	if !isWebSocketRequest(r) {
		writeJSONString(w, http.StatusBadRequest, "WebSocket expected")
		return
	}
	if !slices.Contains(s.allowedPorts, port) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	addr := net.JoinHostPort(s.target, strconv.Itoa(port))
	var d net.Dialer
	tcp, err := d.DialContext(r.Context(), "tcp", addr)
	if err != nil {
		s.log.Warn("Can't reach " + addr + ": " + err.Error())
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer tcp.Close()

	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // ASP.NET accepts any Origin
		CompressionMode:    websocket.CompressionDisabled,
	})
	if err != nil {
		s.log.Warn("WebSocket handshake failed: " + err.Error())
		return
	}
	ws.SetReadLimit(-1)

	who := remoteIP(r) + " -> " + addr
	s.log.Info("Open   " + who)
	s.sessions.Add(1)
	defer s.sessions.Done()

	// The session ends when either side stops, or the relay shuts down for good.
	ctx, cancel := context.WithCancel(s.relayCtx)
	defer cancel()
	done := make(chan struct{}, 3)

	go func() { // game -> server
		defer func() { done <- struct{}{} }()
		buf := make([]byte, relayBufferSize)
		for {
			_, msg, err := ws.Reader(ctx)
			if err != nil {
				return
			}
			if _, err := io.CopyBuffer(tcp, onlyReader{msg}, buf); err != nil {
				return
			}
		}
	}()
	go func() { // server -> game, whole game messages only
		defer func() { done <- struct{}{} }()
		reassemble(tcp, func(msg []byte) error {
			return ws.Write(ctx, websocket.MessageBinary, msg)
		})
	}()
	go func() { // keepalive
		t := time.NewTicker(keepAliveInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				pctx, pcancel := context.WithTimeout(ctx, keepAliveInterval)
				ws.Ping(pctx) // a dead connection is noticed by the reader
				pcancel()
			}
		}
	}()

	select {
	case <-done:
	case <-ctx.Done():
	}
	if s.relayCtx.Err() != nil {
		ws.Close(websocket.StatusGoingAway, "server shutting down")
	} else {
		ws.Close(websocket.StatusNormalClosure, "")
	}
	tcp.Close()
	cancel()
	s.log.Info("Closed " + who)
}

// onlyReader hides WriterTo/ReaderFrom so io.CopyBuffer uses the given buffer.
type onlyReader struct{ io.Reader }

// reassemble reads from the game server and calls send with complete game messages only.
//
// wormix-core writes each message in several small pieces (flag byte, command id, length, body).
// The game's 2012 network code breaks if a message *header* arrives split across deliveries
// (it throws "End of file" / "invalid command header" and drops the data), so we reassemble here.
// Message format: 7-byte header = 1 flag byte, 2-byte command id, 4-byte big-endian length
// (lower 30 bits = body size), then the body. Everything complete so far goes out in one
// WebSocket message.
func reassemble(src io.Reader, send func([]byte) error) error {
	buf := make([]byte, relayBufferSize)
	var pending []byte
	for {
		n, err := src.Read(buf)
		if n > 0 {
			pending = append(pending, buf[:n]...)
			complete := 0
			for len(pending)-complete >= messageHeaderSize {
				h := pending[complete:]
				bodyLength := int(uint32(h[3])<<24|uint32(h[4])<<16|uint32(h[5])<<8|uint32(h[6])) & 0x3FFFFFFF
				if len(pending)-complete < messageHeaderSize+bodyLength {
					break
				}
				complete += messageHeaderSize + bodyLength
			}
			if complete > 0 {
				if serr := send(pending[:complete]); serr != nil {
					return serr
				}
				// Keep the incomplete tail in a fresh slice: send may still hold the old one.
				pending = append([]byte(nil), pending[complete:]...)
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}
