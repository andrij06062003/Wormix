package pvp

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/andrij06062003/Wormix/services/core/internal/csx"
	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

// Battles in progress (and recently finished ones, so a late login still gets the result).

const keepFinished = 2 * time.Minute

// BattleLoginError codes.
const (
	loginErrIncorrectKey  = 1
	loginErrInternalError = 3
)

var (
	battlesMu sync.Mutex
	battles   = map[uint32]*Battle{}
	// Battle ids must not repeat after a restart (the API keeps the last one per player). Mission battles use
	// small ids, so PvP ids start high, from the clock, below int.MaxValue.
	nextID atomic.Int64
)

func init() {
	nextID.Store(1_000_000_000 + time.Now().Unix()%100_000_000)
}

func newBattleID() uint32 { return uint32(nextID.Add(1)) }

func createBattle(id uint32, wager int32, first, second *player) *Battle {
	b := newBattle(id, wager, first, second)
	battlesMu.Lock()
	battles[id] = b
	battlesMu.Unlock()
	logf("[PvP %d] created: %d vs %d, wager %d", id, first.userID, second.userID, wager)
	return b
}

func findBattle(id uint32) *Battle {
	battlesMu.Lock()
	defer battlesMu.Unlock()
	return battles[id]
}

func snapshot() []*Battle {
	battlesMu.Lock()
	defer battlesMu.Unlock()
	out := make([]*Battle, 0, len(battles))
	for _, b := range battles {
		out = append(out, b)
	}
	return out
}

// isInBattle reports whether the player is in a battle that hasn't ended.
func isInBattle(userID uint32) bool {
	for _, b := range snapshot() {
		if !b.Over() && b.HasPlayer(userID) {
			return true
		}
	}
	return false
}

// RunTicker drives battle timeouts and challenge expiry once a second until ctx ends.
func RunTicker(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			tickAll()
		}
	}
}

func tickAll() {
	for _, b := range snapshot() {
		var forget bool
		if exc := csx.Catch(func() { forget = b.tick(keepFinished) }); exc != nil {
			logf("[PvP %d] tick error: %v", b.ID, exc)
			continue
		}
		if forget {
			battlesMu.Lock()
			delete(battles, b.ID)
			battlesMu.Unlock()
		}
	}
	if exc := csx.Catch(lobbyTick); exc != nil {
		logf("[Lobby] tick error: %v", exc)
	}
}

func isEndOfStream(r any) bool {
	e, ok := r.(*csx.Exception)
	return ok && e.Kind == csx.EndOfStream
}

func sendLoginError(c Conn, code int32) {
	c.SendMessage(message(CmdLoginError, false, func(w *wire.Writer) { w.I32(code) }))
}

// HandlePvp handles a command on a PvP-server connection (PvpSession.HandleRaw). Returns false for an unknown
// command outside a battle. A truncated command is logged and ignored.
func HandlePvp(c Conn, cmd uint32, body []byte) (handled bool) {
	defer func() {
		if r := recover(); r != nil {
			if isEndOfStream(r) {
				logf("[PvP] truncated command %d", cmd)
				handled = true
				return
			}
			panic(r)
		}
	}()
	switch cmd {
	case CmdPing:
		c.SendMessage(message(CmdPong, false, func(*wire.Writer) {}))
		return true
	case CmdBattleLogin:
		login(c, body)
		return true
	case CmdBattleReconnect:
		reconnect(c, body)
		return true
	default:
		b := c.Battle()
		if b == nil {
			return false
		}
		b.OnMessage(c, cmd, body)
		return true
	}
}

// login is BattleLogin (1001) on the PvP server.
func login(c Conn, body []byte) {
	// id u32, firstTurn bool, authKey utf, enemyIds u32[], mainHost utf, mainPort i32, battleId u32, socialId u8
	r := wire.NewReader(body)
	userID := r.U32()
	r.Bool()
	authKey := r.UTFStrict()
	r.U32Array()
	r.UTFStrict()
	r.U32()
	battleID := r.U32()
	loginOrReconnect(c, userID, authKey, battleID, func(b *Battle) bool { return b.Login(c, userID) })
}

// reconnect is BattleReconnect (1015) on the PvP server.
func reconnect(c Conn, body []byte) {
	// id u32, authKey utf, mainHost utf, mainPort i32, battleId u32, turnNum u16, playerNum u8, lastCommandNum u16, socialId u8
	r := wire.NewReader(body)
	userID := r.U32()
	authKey := r.UTFStrict()
	r.UTFStrict()
	r.U32()
	battleID := r.U32()
	turn := r.U16()
	r.U8()
	last := r.U16()
	loginOrReconnect(c, userID, authKey, battleID, func(b *Battle) bool { return b.Reconnect(c, userID, int(turn), last) })
}

func loginOrReconnect(c Conn, userID uint32, authKey string, battleID uint32, join func(*Battle) bool) {
	// The player must be logged in to the main server with the same key (the PvP connection carries no other proof).
	main := Find(userID)
	if main == nil || authKey == "" || main.AuthKey() != authKey {
		logf("[PvP %d] login of %d refused: not logged in to the main server with this key", battleID, userID)
		sendLoginError(c, loginErrIncorrectKey)
		return
	}
	b := findBattle(battleID)
	if b == nil || !join(b) {
		logf("[PvP %d] login of %d refused: no such battle for this player", battleID, userID)
		sendLoginError(c, loginErrInternalError)
	}
}

// OnPvpDisconnected is called when a PvP-server connection closes.
func OnPvpDisconnected(c Conn) {
	if b := c.Battle(); b != nil {
		b.OnDisconnected(c)
	}
	c.SetBattle(nil)
}
