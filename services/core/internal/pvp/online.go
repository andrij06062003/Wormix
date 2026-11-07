// Package pvp implements PvP: finding an opponent on the main server (Lobby) and running battles on the PvP
// server (Battle). Port of the C# Pvp/ directory.
package pvp

import (
	"log"
	"sync"

	"github.com/andrij06062003/Wormix/services/core/internal/proto"
	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

// Player is a logged-in main-server connection.
type Player interface {
	// UserID is the game id of the logged-in player (0 until login succeeds).
	UserID() uint32
	// AuthKey is the auth key the player logged in with.
	AuthKey() string
	// Token is the main-server session token ("{userId}.{key}") used for API calls.
	Token() string
	// SendMessage sends complete messages; false if the connection is gone.
	SendMessage(msg []byte) bool
}

// Conn is a connection to the PvP server.
type Conn interface {
	SendMessage(msg []byte) bool
	// Close closes the socket; the connection's goroutine then disconnects it.
	Close()
	Battle() *Battle
	SetBattle(b *Battle)
}

// ---- Online ------------------------------------------------------------------------------------------------

var (
	onlineMu sync.RWMutex
	online   = map[uint32]Player{}
)

// Register records a player as logged in (replacing an older session of the same user).
func Register(p Player) {
	onlineMu.Lock()
	online[p.UserID()] = p
	onlineMu.Unlock()
}

// Unregister removes the player, unless the user has logged in again from a newer session meanwhile.
func Unregister(p Player) {
	id := p.UserID()
	if id == 0 {
		return
	}
	onlineMu.Lock()
	if online[id] == p {
		delete(online, id)
	}
	onlineMu.Unlock()
}

// Find returns the logged-in session of a user, or nil.
func Find(userID uint32) Player {
	onlineMu.RLock()
	defer onlineMu.RUnlock()
	return online[userID]
}

// ---- helpers -----------------------------------------------------------------------------------------------

func logf(format string, args ...any) { log.Printf(format, args...) }

// writeProfile writes a UserProfileStructure as the client reads it inside PvP messages: a 2-byte length (the
// real encoded size), then the structure.
func writeProfile(w *wire.Writer, p *proto.UserProfile) {
	var body wire.Writer
	p.Write(&body)
	w.U16(uint16(len(body.B)))
	w.Bytes(body.B)
}

// message is Wire.Message: header + body (+ MD5 suffix).
func message(cmd uint32, secure bool, body func(w *wire.Writer)) []byte {
	var w wire.Writer
	body(&w)
	return proto.Frame(cmd, w.B, secure)
}
