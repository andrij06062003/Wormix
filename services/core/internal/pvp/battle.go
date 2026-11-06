package pvp

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

// Command ids of the PvP connection (from the 1.05.0 client).
const (
	// client -> server
	CmdPing               = 16
	CmdChat               = 17 // also server -> client (forwarded)
	CmdBattleLogin        = 1001
	CmdDropPlayer         = 1006
	CmdAction             = 1007 // also server -> client (forwarded)
	CmdEndTurn            = 1014 // also server -> client (forwarded)
	CmdBattleReconnect    = 1015
	CmdEndTurnResponse    = 1017
	CmdRetryRequestClient = 1018

	// server -> client
	CmdPong               = 10017
	CmdRetryRequestServer = 1016
	CmdStartTurn          = 1019
	CmdEndBattle          = 1021
	CmdLoginSuccess       = 2001
	CmdLoginError         = 2004
	CmdSystemMessage      = 2021
)

// PvpSystemMessage types.
const (
	sysDroppedByReconnectionTimeout = 1
	sysDroppedByCommandTimeout      = 2
	sysDroppedByResponseTimeout     = 3
	sysSurrendered                  = 4
	sysCheater                      = 5
	sysDisconnected                 = 6
	sysReconnected                  = 7
)

// PvpDropPlayer types.
const (
	dropIAmCheater = 4
	dropDesync     = 5
)

// Timeouts of a battle.
var (
	NoShowTimeout         = 60 * time.Second
	ReconnectTimeout      = 45 * time.Second
	TurnTimeout           = 150 * time.Second
	ResponseTimeout       = 120 * time.Second
	EndTurnResendInterval = 5 * time.Second
)

// player is one player of a battle. Fields other than the first three are guarded by the battle's lock.
type player struct {
	userID    uint32
	firstTurn bool
	token     string // main-server session token, for recording the result in the API

	session        Conn
	everConnected  bool
	disconnectedAt *time.Time
	left           bool // ended the battle on their side (surrender etc.): don't wait for them any more
}

func (p *player) num() byte {
	if p.firstTurn {
		return 1
	}
	return 2
}

func (p *player) connected() bool { return p.session != nil }

type turnCommand struct {
	num uint16
	msg []byte
}

type turnEnd struct {
	eliminated []uint32
	state      int32
}

// Battle is a battle between two players on the PvP server.
//
// The game simulation runs in both clients; the server relays the active player's moves to the other player
// and referees the turns:
//  1. the active player sends moves (Action) and finally EndTurn: both are forwarded to the other player;
//  2. the other player replays them and answers EndTurnResponse;
//  3. when both agree on who is eliminated, the server either ends the battle (EndBattle) or starts the
//     next turn for the other player (StartTurn to both).
//
// It also handles reconnects (resending what a player missed), surrender, disconnects and inactivity.
type Battle struct {
	ID      uint32
	Wager   int32
	players [2]*player
	created time.Time

	mu             sync.Mutex
	turn           int
	active         int // index of the player whose turn it is
	turnCommands   []turnCommand
	lastCommandNum uint16
	activeEnd      *turnEnd
	passiveEnd     *turnEnd
	activeEndAt    time.Time
	lastEndTurn    time.Time
	lastActivity   time.Time
	lastStartTurn  []byte
	endBattle      []byte
	overAt         time.Time
	results        map[uint32]int // 1 won, 0 draw/cancelled, -1 lost

	over atomic.Bool
}

func newBattle(id uint32, wager int32, first, second *player) *Battle {
	now := time.Now()
	b := &Battle{ID: id, Wager: wager, players: [2]*player{first, second}, created: now, turn: 1, lastActivity: now,
		results: map[uint32]int{}}
	if !first.firstTurn {
		b.active = 1
	}
	return b
}

// Over reports whether the battle has ended.
func (b *Battle) Over() bool { return b.over.Load() }

// HasPlayer reports whether the user plays in this battle.
func (b *Battle) HasPlayer(userID uint32) bool {
	return b.players[0].userID == userID || b.players[1].userID == userID
}

func (b *Battle) playerOf(c Conn) *player {
	for _, p := range b.players {
		if p.session != nil && p.session == c {
			return p
		}
	}
	return nil
}

func (b *Battle) playerByID(userID uint32) *player {
	for _, p := range b.players {
		if p.userID == userID {
			return p
		}
	}
	return nil
}

func (b *Battle) other(p *player) *player {
	if b.players[0] == p {
		return b.players[1]
	}
	return b.players[0]
}

func (b *Battle) activePlayer() *player  { return b.players[b.active] }
func (b *Battle) passivePlayer() *player { return b.players[1-b.active] }

func (b *Battle) logf(format string, args ...any) {
	logf("[PvP %d] %s", b.ID, fmt.Sprintf(format, args...))
}

func send(p *player, msg []byte) {
	if p.session != nil {
		p.session.SendMessage(msg)
	}
}

func systemMessage(typ int, profileID uint32) []byte {
	return message(CmdSystemMessage, false, func(w *wire.Writer) {
		w.I16(typ)
		w.U32(profileID)
	})
}

// ---- Login / reconnect -------------------------------------------------------------------------------------

// Login is the first login of a player (BattleLogin). Returns false if the player isn't part of this battle.
func (b *Battle) Login(c Conn, userID uint32) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	p := b.playerByID(userID)
	if p == nil {
		return false
	}
	reconnect := p.everConnected
	b.attach(p, c)
	if reconnect {
		b.logf("%d logged in again", userID)
	} else {
		b.logf("%d logged in", userID)
	}
	if b.Over() {
		b.sendEndBattle(p)
		return true
	}
	if reconnect {
		send(b.other(p), systemMessage(sysReconnected, p.userID))
	}
	if p == b.passivePlayer() {
		b.resendTurnCommands(p, 0) // moves the active player made before this player connected
	}
	return true
}

// Reconnect: a player whose PvP connection dropped comes back (BattleReconnect) and says how far it got.
func (b *Battle) Reconnect(c Conn, userID uint32, turnNum int, lastCommandNum uint16) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	p := b.playerByID(userID)
	if p == nil {
		return false
	}
	b.attach(p, c)
	b.logf("%d reconnected at turn %d (server turn %d), last command %d", userID, turnNum, b.turn, lastCommandNum)
	if b.Over() {
		b.sendEndBattle(p)
		return true
	}
	send(b.other(p), systemMessage(sysReconnected, p.userID))

	switch {
	case turnNum < b.turn && b.lastStartTurn != nil:
		send(p, b.lastStartTurn) // missed the start of the current turn
	case p == b.passivePlayer():
		if turnNum == b.turn {
			b.resendTurnCommands(p, int(lastCommandNum))
		} else if turnNum == b.turn+1 && b.activeEnd != nil && b.passiveEnd == nil {
			// It already answered EndTurn but the answer was lost with the connection.
			// Both had the same moves, so go with the active player's view.
			b.passiveEnd = b.activeEnd
			b.tryFinishTurn()
		}
	case b.activeEnd == nil && (turnNum == b.turn || turnNum == b.turn+1):
		// The active player may have sent moves (or its EndTurn) that never arrived: ask for everything after
		// the last one we have.
		from := int(b.lastCommandNum) + 1
		if minFrom := map[bool]int{true: 1, false: 2}[b.turn == 1]; minFrom > from {
			from = minFrom
		}
		send(p, message(CmdRetryRequestServer, true, func(w *wire.Writer) {
			w.U32(b.ID)
			w.U16(2)
			w.I16(from)
			w.I16(-1)
			w.U16(uint16(b.turn))
		}))
	}
	return true
}

func (b *Battle) attach(p *player, c Conn) {
	if p.session != nil && p.session != c {
		old := p.session
		old.SetBattle(nil)
		old.Close()
	}
	p.session = c
	p.everConnected = true
	p.disconnectedAt = nil
	c.SetBattle(b)
	c.SendMessage(message(CmdLoginSuccess, false, func(w *wire.Writer) {
		w.UTF("")
		w.U8(p.num())
	}))
}

// OnDisconnected is called when a player's PvP connection closes.
func (b *Battle) OnDisconnected(c Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	p := b.playerOf(c)
	if p == nil {
		return
	}
	p.session = nil
	if b.Over() || p.left {
		return
	}
	now := time.Now()
	p.disconnectedAt = &now
	b.logf("%d disconnected", p.userID)
	send(b.other(p), systemMessage(sysDisconnected, p.userID))
}

// ---- Messages during the battle ----------------------------------------------------------------------------

// OnMessage handles a battle command from a player's PvP connection. A truncated command panics with an
// EndOfStream exception after the lock is released.
func (b *Battle) OnMessage(c Conn, cmd uint32, body []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	p := b.playerOf(c)
	if p == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			if isEndOfStream(r) {
				b.logf("truncated command %d from %d", cmd, p.userID)
				return
			}
			panic(r)
		}
	}()
	switch cmd {
	case CmdAction:
		b.onAction(p, body)
	case CmdEndTurn:
		b.onEndTurn(p, body)
	case CmdEndTurnResponse:
		b.onEndTurnResponse(p, body)
	case CmdRetryRequestClient:
		b.onRetryRequest(p, body)
	case CmdDropPlayer:
		b.onDropPlayer(p, body)
	case CmdChat:
		if len(body) <= 1024 {
			send(b.other(p), message(CmdChat, false, func(w *wire.Writer) { w.Bytes(body) }))
		}
	default:
		b.logf("unexpected command %d from %d", cmd, p.userID)
	}
}

func (b *Battle) onAction(p *player, body []byte) {
	// firstFrame u32, lastFrame u32, turnNum u16, commandNum u16, battleId u32, playerNum u8, actions...
	r := wire.NewReader(body)
	r.U32()
	r.U32()
	turn := r.U16()
	num := r.U16()
	if b.Over() || p != b.activePlayer() || int(turn) != b.turn || b.activeEnd != nil {
		return
	}
	b.store(num, message(CmdAction, false, func(w *wire.Writer) { w.Bytes(body) }))
}

func (b *Battle) onEndTurn(p *player, body []byte) {
	// turnNum u16, commandNum u16, battleId i32, playerNum u8, eliminated u32[], battleState i32
	r := wire.NewReader(body)
	turn := r.U16()
	num := r.U16()
	r.U32()
	r.U8()
	eliminated := r.U32Array()
	state := int32(r.U32())
	if b.Over() || p != b.activePlayer() || int(turn) != b.turn || b.activeEnd != nil {
		return
	}
	b.store(num, message(CmdEndTurn, false, func(w *wire.Writer) { w.Bytes(body) }))
	b.activeEnd = &turnEnd{eliminated, state}
	b.activeEndAt = time.Now()
	b.lastEndTurn = b.activeEndAt
	b.tryFinishTurn()
}

func (b *Battle) store(num uint16, msg []byte) {
	b.lastActivity = time.Now()
	for _, c := range b.turnCommands {
		if c.num == num {
			return // a resend of something we already have
		}
	}
	b.turnCommands = append(b.turnCommands, turnCommand{num, msg})
	sort.SliceStable(b.turnCommands, func(i, j int) bool { return b.turnCommands[i].num < b.turnCommands[j].num })
	if num > b.lastCommandNum {
		b.lastCommandNum = num
	}
	send(b.passivePlayer(), msg)
}

func (b *Battle) onEndTurnResponse(p *player, body []byte) {
	// turnNum u16, commandNum u16, battleId i32, eliminated u32[], battleState i32
	r := wire.NewReader(body)
	turn := r.U16()
	r.U16()
	r.U32()
	eliminated := r.U32Array()
	state := int32(r.U32())
	if b.Over() || p != b.passivePlayer() || int(turn) != b.turn || b.passiveEnd != nil {
		return
	}
	b.passiveEnd = &turnEnd{eliminated, state}
	b.tryFinishTurn()
}

func (b *Battle) onRetryRequest(p *player, body []byte) {
	// commandNum u16 (last one it has), turnNum u16, battleId i32
	r := wire.NewReader(body)
	have := r.U16()
	turn := r.U16()
	if !b.Over() && p == b.passivePlayer() && int(turn) == b.turn {
		b.resendTurnCommands(p, int(have))
	}
}

func (b *Battle) resendTurnCommands(p *player, after int) {
	for _, c := range b.turnCommands {
		if int(c.num) > after {
			send(p, c.msg)
		}
	}
}

func (b *Battle) onDropPlayer(p *player, body []byte) {
	// playerId u32, battleId u32, type i32
	r := wire.NewReader(body)
	r.U32()
	r.U32()
	typ := int32(r.U32())
	if b.Over() {
		return
	}
	p.left = true
	other := b.other(p)
	if typ == dropDesync {
		b.logf("%d reports desync: draw", p.userID)
		b.finish([]result{{p.userID, 0}, {other.userID, 0}}, true)
		return
	}
	b.logf("%d left the battle (type %d): %d wins", p.userID, typ, other.userID)
	sys := sysSurrendered
	if typ == dropIAmCheater {
		sys = sysCheater
	}
	send(other, systemMessage(sys, p.userID))
	b.finish([]result{{p.userID, -1}, {other.userID, 1}}, false)
}

func sameSet(a, b []uint32) bool {
	in := func(x uint32, l []uint32) bool { return slices.Contains(l, x) }
	for _, x := range a {
		if !in(x, b) {
			return false
		}
	}
	for _, x := range b {
		if !in(x, a) {
			return false
		}
	}
	return true
}

func joinIDs(l []uint32) string {
	s := make([]string, len(l))
	for i, x := range l {
		s[i] = fmt.Sprint(x)
	}
	return strings.Join(s, ",")
}

func (b *Battle) all(r int) []result {
	return []result{{b.players[0].userID, r}, {b.players[1].userID, r}}
}

func (b *Battle) tryFinishTurn() {
	a, p := b.activeEnd, b.passiveEnd
	if a == nil || p == nil {
		return
	}
	if !sameSet(a.eliminated, p.eliminated) {
		b.logf("desync at turn %d: eliminated [%s] vs [%s]", b.turn, joinIDs(a.eliminated), joinIDs(p.eliminated))
		b.finish(b.all(0), true)
		return
	}
	if a.state != p.state {
		b.logf("state hash differs at turn %d (%d vs %d); continuing", b.turn, a.state, p.state)
	}
	var out []*player
	for _, x := range b.players {
		if slices.Contains(a.eliminated, x.userID) {
			out = append(out, x)
		}
	}
	switch len(out) {
	case 2:
		b.finish(b.all(0), false)
		return
	case 1:
		res := make([]result, 0, 2)
		for _, x := range b.players {
			r := 1
			if x == out[0] {
				r = -1
			}
			res = append(res, result{x.userID, r})
		}
		b.finish(res, false)
		return
	}

	// Next turn: the other player.
	b.turn++
	b.active = 1 - b.active
	b.turnCommands = nil
	b.lastCommandNum = 0
	b.activeEnd, b.passiveEnd = nil, nil
	b.lastActivity = time.Now()
	activeID := b.activePlayer().userID
	b.lastStartTurn = message(CmdStartTurn, true, func(w *wire.Writer) {
		w.U32(b.ID)
		w.U32(activeID)
		w.U16(uint16(b.turn))
		w.U16(0)
		w.U16(0) // dropped players
	})
	for _, x := range b.players {
		send(x, b.lastStartTurn)
	}
}

// ---- Timeouts ----------------------------------------------------------------------------------------------

// tick is called about once a second. Returns true once the battle has been over long enough to forget it.
func (b *Battle) tick(keepFinished time.Duration) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	if b.Over() {
		return now.Sub(b.overAt) > keepFinished
	}

	// Someone never showed up, or didn't come back after a disconnect.
	var gone []*player
	for _, p := range b.players {
		if (!p.everConnected && now.Sub(b.created) > NoShowTimeout) ||
			(p.disconnectedAt != nil && now.Sub(*p.disconnectedAt) > ReconnectTimeout) {
			gone = append(gone, p)
		}
	}
	if len(gone) == 2 {
		b.logf("both players gone: cancelled")
		b.finish(b.all(0), false)
		return false
	}
	// One is gone; the other wins if it's here (if it's away too, wait for it to come back or time out).
	if len(gone) == 1 && b.other(gone[0]).connected() {
		other := b.other(gone[0])
		b.logf("%d didn't come back: %d wins", gone[0].userID, other.userID)
		send(other, systemMessage(sysDroppedByReconnectionTimeout, gone[0].userID))
		b.finish([]result{{gone[0].userID, -1}, {other.userID, 1}}, false)
		return false
	}

	if b.activeEnd == nil {
		// The active player stopped playing (connected, but nothing for a long time).
		if b.activePlayer().connected() && now.Sub(b.lastActivity) > TurnTimeout {
			act, pas := b.activePlayer(), b.passivePlayer()
			b.logf("%d inactive: %d wins", act.userID, pas.userID)
			send(pas, systemMessage(sysDroppedByCommandTimeout, act.userID))
			b.finish([]result{{act.userID, -1}, {pas.userID, 1}}, false)
		}
	} else if b.passiveEnd == nil && b.passivePlayer().connected() {
		if now.Sub(b.activeEndAt) > ResponseTimeout {
			act, pas := b.activePlayer(), b.passivePlayer()
			b.logf("%d never confirmed the turn: %d wins", pas.userID, act.userID)
			send(act, systemMessage(sysDroppedByResponseTimeout, pas.userID))
			b.finish([]result{{pas.userID, -1}, {act.userID, 1}}, false)
		} else if now.Sub(b.lastEndTurn) > EndTurnResendInterval {
			// The client ignores EndTurn until it's ready to replay the turn (e.g. still building the map on
			// turn 1), so repeat it. A client that already has it ignores the copy; one that misses moves asks
			// for them (RetryRequestClient).
			b.lastEndTurn = now
			if n := len(b.turnCommands); n > 0 {
				send(b.passivePlayer(), b.turnCommands[n-1].msg)
			}
		}
	}
	return false
}

// ---- End ---------------------------------------------------------------------------------------------------

type result struct {
	userID uint32
	r      int
}

func (b *Battle) finish(results []result, desync bool) {
	b.over.Store(true)
	b.overAt = time.Now()
	parts := make([]string, len(results))
	for i, r := range results {
		b.results[r.userID] = r.r
		parts[i] = fmt.Sprintf("%d=%d", r.userID, r.r)
	}
	suffix := ""
	if desync {
		suffix = " (desync)"
	}
	b.logf("over: %s%s", strings.Join(parts, ", "), suffix)

	b.endBattle = message(CmdEndBattle, true, func(w *wire.Writer) {
		if desync {
			w.U8(1)
		} else {
			w.U8(0)
		}
		w.U32(b.ID)
		w.U16(uint16(b.turn))
		w.U16(b.lastCommandNum)
		w.U16(uint16(len(b.players)))
		for _, p := range b.players {
			w.U32(p.userID)
		}
		// results: 0 won, 3 draw, 1 lost
		w.U16(uint16(len(b.players)))
		for _, p := range b.players {
			switch b.results[p.userID] {
			case 1:
				w.U8(0)
			case 0:
				w.U8(3)
			default:
				w.U8(1)
			}
		}
		// signatures (not checked by this server): empty
		w.U16(uint16(len(b.players)))
		for range b.players {
			w.U16(0)
			w.U16(0)
		}
	})
	for _, p := range b.players {
		if !p.left {
			b.sendEndBattle(p)
		}
	}

	// Pay out (wager, rating) in the API, off the battle's lock.
	type settle struct {
		userID uint32
		token  string
		result int
	}
	var todo []settle
	for _, p := range b.players {
		token := p.token
		if s := Find(p.userID); s != nil && s.Token() != "" {
			token = s.Token()
		}
		todo = append(todo, settle{p.userID, token, b.results[p.userID]})
	}
	id := b.ID
	go func() {
		for _, s := range todo {
			tryEndBattle(s.token, s.userID, id, s.result)
		}
	}()
}

func (b *Battle) sendEndBattle(p *player) {
	if b.endBattle != nil {
		send(p, b.endBattle)
	}
}
