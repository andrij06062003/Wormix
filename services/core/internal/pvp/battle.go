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

// player is one player of a battle. Fields other than the first four are guarded by the battle's lock.
type player struct {
	userID    uint32
	firstTurn bool
	team      int    // players of the same team fight together (1v1: each player is its own team)
	token     string // main-server session token, for recording the result in the API

	seat           int // place in the turn order (0 moves first)
	session        Conn
	everConnected  bool
	disconnectedAt *time.Time
	left           bool     // ended the battle on their side (surrender etc.): don't wait for them any more
	out            bool     // all their worms are dead: they only watch now
	end            *turnEnd // their EndTurnResponse for the current turn (when not the active player)
}

// num is the player number the client puts in its commands: its place in the turn order, from 1.
func (p *player) num() byte { return byte(p.seat + 1) }

func (p *player) connected() bool { return p.session != nil }

type turnCommand struct {
	num uint16
	msg []byte
}

type turnEnd struct {
	eliminated []uint32
	state      int32
}

// Battle is a battle on the PvP server: two players (1v1), or two teams of two (2v2).
//
// The game simulation runs in every client; the server relays the active player's moves to the others and
// referees the turns:
//  1. the active player sends moves (Action) and finally EndTurn: both are forwarded to the other players;
//  2. the other players replay them and answer EndTurnResponse;
//  3. when all agree on who is eliminated, the server either ends the battle (EndBattle) or starts the
//     next turn (StartTurn to all).
//
// Turns go round the squads still in the game in the order the client's GameMaster keeps them (see
// tryFinishTurn). A battle ends when only one team has worms left.
//
// It also handles reconnects (resending what a player missed), surrender, disconnects and inactivity. A
// player who leaves before their worms are all dead loses the battle for their team: the other games can't
// go on without them.
type Battle struct {
	ID      uint32
	Wager   int32
	players []*player // in the order they were given (EndBattle lists them in this order)
	created time.Time

	mu             sync.Mutex
	turn           int
	order          []*player // squads still in the game, in turn order
	active         int       // index in order of the player whose turn it is
	turnCommands   []turnCommand
	lastCommandNum uint16
	activeEnd      *turnEnd
	activeEndAt    time.Time
	lastEndTurn    time.Time
	lastActivity   time.Time
	lastStartTurn  []byte
	endBattle      []byte
	overAt         time.Time
	results        map[uint32]int // 1 won, 0 draw/cancelled, -1 lost

	// v125: a battle of 1.25.0 clients (ex125.go). Players are told apart by their number (their place in
	// players, which is also the turn order) instead of their id, and messages are signed with its salt.
	v125 bool

	over atomic.Bool
}

// newBattle: players in any order, order is the turn order (the same players).
func newBattle(id uint32, wager int32, players, order []*player) *Battle {
	now := time.Now()
	b := &Battle{ID: id, Wager: wager, players: players, order: slices.Clone(order), created: now, turn: 1,
		lastActivity: now, results: map[uint32]int{}}
	for i, p := range order {
		p.seat = i
	}
	return b
}

// newDuel is a 1v1 battle: the player with firstTurn moves first.
func newDuel(id uint32, wager int32, first, second *player) *Battle {
	first.team, second.team = 1, 2
	order := []*player{first, second}
	if !first.firstTurn {
		order = []*player{second, first}
	}
	return newBattle(id, wager, []*player{first, second}, order)
}

// Over reports whether the battle has ended.
func (b *Battle) Over() bool { return b.over.Load() }

// HasPlayer reports whether the user plays in this battle.
func (b *Battle) HasPlayer(userID uint32) bool { return b.playerByID(userID) != nil }

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

func (b *Battle) activePlayer() *player { return b.order[b.active] }

// sendOthers sends to every player but p who is still in the battle.
func (b *Battle) sendOthers(p *player, msg []byte) {
	for _, x := range b.players {
		if x != p && !x.left {
			send(x, msg)
		}
	}
}

// responders are the players whose EndTurnResponse the turn waits for: everyone but the active player who
// still has worms and hasn't left.
func (b *Battle) responders() []*player {
	var out []*player
	act := b.activePlayer()
	for _, x := range b.players {
		if x != act && !x.left && !x.out {
			out = append(out, x)
		}
	}
	return out
}

// teamLoses: results where p's team loses and everyone else wins.
func (b *Battle) teamLoses(p *player) []result {
	res := make([]result, 0, len(b.players))
	for _, x := range b.players {
		r := 1
		if x.team == p.team {
			r = -1
		}
		res = append(res, result{x.userID, r})
	}
	return res
}

// opponentConnected reports whether some player of another team is still here.
func (b *Battle) opponentConnected(team int) bool {
	for _, x := range b.players {
		if x.team != team && !x.left && x.connected() {
			return true
		}
	}
	return false
}

func (b *Battle) logf(format string, args ...any) {
	logf("[PvP %d] %s", b.ID, fmt.Sprintf(format, args...))
}

func send(p *player, msg []byte) {
	if p.session != nil {
		p.session.SendMessage(msg)
	}
}

// msg is message, signed for the battle's clients.
func (b *Battle) msg(cmd uint32, secure bool, body func(w *wire.Writer)) []byte {
	if b.v125 {
		return message125(cmd, secure, body)
	}
	return message(cmd, secure, body)
}

// sysMsg is a PvpSystemMessage about p.
func (b *Battle) sysMsg(typ int, p *player) []byte {
	if b.v125 {
		return message125(CmdSystemMessage, false, func(w *wire.Writer) {
			w.I16(typ)
			w.U8(byte(p.seat))
		})
	}
	return systemMessage(typ, p.userID)
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
		b.sendOthers(p, b.sysMsg(sysReconnected, p))
	}
	if p != b.activePlayer() {
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
	b.sendOthers(p, b.sysMsg(sysReconnected, p))

	switch {
	case turnNum < b.turn && b.lastStartTurn != nil:
		send(p, b.lastStartTurn) // missed the start of the current turn
	case p != b.activePlayer():
		if turnNum == b.turn {
			b.resendTurnCommands(p, int(lastCommandNum))
		} else if turnNum == b.turn+1 && b.activeEnd != nil && p.end == nil {
			// It already answered EndTurn but the answer was lost with the connection.
			// Both had the same moves, so go with the active player's view.
			p.end = b.activeEnd
			b.tryFinishTurn()
		}
	case b.activeEnd == nil && (turnNum == b.turn || turnNum == b.turn+1):
		// The active player may have sent moves (or its EndTurn) that never arrived: ask for everything after
		// the last one we have.
		from := int(b.lastCommandNum) + 1
		if minFrom := map[bool]int{true: 1, false: 2}[b.turn == 1]; minFrom > from {
			from = minFrom
		}
		send(p, b.msg(CmdRetryRequestServer, true, func(w *wire.Writer) {
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
	if b.v125 {
		// StartBattle: the 1.25.0 client starts the battle with it (it also answers a reconnect).
		c.SendMessage(message125(cmdExStartBattle, false, func(w *wire.Writer) {
			w.U32(b.ID)
			w.UTF("")
			w.U8(byte(p.seat))
		}))
		return
	}
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
	b.sendOthers(p, b.sysMsg(sysDisconnected, p))
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
		if len(body) > 1024 {
			break
		}
		msg := message(CmdChat, false, func(w *wire.Writer) { w.Bytes(body) })
		if b.v125 {
			// playerNum u8, message utf, battleId u32, isTeamMsg bool, MD5
			r := wire.NewReader(body)
			r.U8()
			r.UTFStrict()
			r.U32()
			if r.Bool() {
				for _, x := range b.players {
					if x != p && !x.left && x.team == p.team {
						send(x, msg)
					}
				}
				break
			}
		}
		b.sendOthers(p, msg)
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
	// turnNum u16, commandNum u16, battleId i32, playerNum u8, eliminated, (1.25.0: collected reagents, items,)
	// battleState i32, ...
	r := wire.NewReader(body)
	turn := r.U16()
	num := r.U16()
	r.U32()
	r.U8()
	eliminated := b.readEliminated(r)
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

// readEliminated reads the squads a turn eliminated, as player ids: 1.05.0 sends ids (u32[]), 1.25.0 player
// numbers (u8[]) followed by the reagents and items the turn collected (skipped).
func (b *Battle) readEliminated(r *wire.Reader) []uint32 {
	if !b.v125 {
		return r.U32Array()
	}
	n := int(r.U16())
	ids := make([]uint32, 0, n)
	for i := 0; i < n; i++ {
		if num := int(r.U8()); num < len(b.players) {
			ids = append(ids, b.players[num].userID)
		}
	}
	for n = int(r.U16()); n > 0; n-- { // collected reagents
		r.U8()
	}
	for n = int(r.U16()); n > 0; n-- { // items: size u16 + WeaponStructure
		for size := int(r.U16()); size > 0; size-- {
			r.U8()
		}
	}
	return ids
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
	b.sendOthers(b.activePlayer(), msg)
}

func (b *Battle) onEndTurnResponse(p *player, body []byte) {
	// turnNum u16, commandNum u16, battleId i32, eliminated, (1.25.0: collected reagents, items,) battleState i32
	r := wire.NewReader(body)
	turn := r.U16()
	r.U16()
	r.U32()
	eliminated := b.readEliminated(r)
	state := int32(r.U32())
	if b.Over() || p == b.activePlayer() || int(turn) != b.turn || p.end != nil {
		return
	}
	p.end = &turnEnd{eliminated, state}
	b.tryFinishTurn()
}

func (b *Battle) onRetryRequest(p *player, body []byte) {
	// commandNum u16 (last one it has), turnNum u16, battleId i32
	r := wire.NewReader(body)
	have := r.U16()
	turn := r.U16()
	if !b.Over() && p != b.activePlayer() && int(turn) == b.turn {
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
	// playerId u32, battleId u32, type i32 (1.25.0: playerNum u8, battleId u32, type i16)
	r := wire.NewReader(body)
	var typ int32
	if b.v125 {
		r.U8()
		r.U32()
		typ = int32(int16(r.U16()))
	} else {
		r.U32()
		r.U32()
		typ = int32(r.U32())
	}
	if b.Over() {
		return
	}
	p.left = true
	if typ == dropDesync {
		b.logf("%d reports desync: draw", p.userID)
		b.finish(b.all(0), true)
		return
	}
	if p.out {
		// Their worms are all dead already: they stop watching, the others play on.
		b.logf("%d left the battle (type %d) after being eliminated", p.userID, typ)
		b.tryFinishTurn()
		return
	}
	b.logf("%d left the battle (type %d): team %d loses", p.userID, typ, p.team)
	sys := sysSurrendered
	if typ == dropIAmCheater {
		sys = sysCheater
	}
	b.sendOthers(p, b.sysMsg(sys, p))
	b.finish(b.teamLoses(p), false)
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
	res := make([]result, len(b.players))
	for i, p := range b.players {
		res[i] = result{p.userID, r}
	}
	return res
}

func (b *Battle) tryFinishTurn() {
	a := b.activeEnd
	if a == nil || b.Over() {
		return
	}
	resp := b.responders()
	for _, p := range resp {
		if p.end == nil {
			return
		}
	}
	for _, p := range resp {
		if !sameSet(a.eliminated, p.end.eliminated) {
			b.logf("desync at turn %d: eliminated [%s] vs [%s] (%d)", b.turn, joinIDs(a.eliminated),
				joinIDs(p.end.eliminated), p.userID)
			b.finish(b.all(0), true)
			return
		}
		if a.state != p.end.state {
			b.logf("state hash differs at turn %d (%d vs %d); continuing", b.turn, a.state, p.end.state)
		}
	}

	// Drop the eliminated squads the way the client's GameMaster does: it removes them from its list and keeps
	// the index of the squad that just moved, then moves on to the next index.
	for i := 0; i < len(b.order); i++ {
		if slices.Contains(a.eliminated, b.order[i].userID) {
			b.order[i].out = true
			b.order = slices.Delete(b.order, i, i+1)
			i--
		}
	}
	var teams []int
	for _, x := range b.order {
		if !slices.Contains(teams, x.team) {
			teams = append(teams, x.team)
		}
	}
	switch len(teams) {
	case 0:
		b.finish(b.all(0), false)
		return
	case 1:
		res := make([]result, 0, len(b.players))
		for _, x := range b.players {
			r := -1
			if x.team == teams[0] {
				r = 1
			}
			res = append(res, result{x.userID, r})
		}
		b.finish(res, false)
		return
	}

	// Next turn.
	b.turn++
	b.active = (b.active + 1) % len(b.order)
	b.turnCommands = nil
	b.lastCommandNum = 0
	b.activeEnd = nil
	for _, x := range b.players {
		x.end = nil
	}
	b.lastActivity = time.Now()
	act := b.activePlayer()
	b.lastStartTurn = b.msg(CmdStartTurn, true, func(w *wire.Writer) {
		w.U32(b.ID)
		if b.v125 {
			w.U8(byte(act.seat))
		} else {
			w.U32(act.userID)
		}
		w.U16(uint16(b.turn))
		w.U16(0)
		w.U16(0) // dropped players
	})
	for _, x := range b.players {
		if !x.left {
			send(x, b.lastStartTurn)
		}
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
		if !p.left && ((!p.everConnected && now.Sub(b.created) > NoShowTimeout) ||
			(p.disconnectedAt != nil && now.Sub(*p.disconnectedAt) > ReconnectTimeout)) {
			if p.out {
				// Eliminated already: nobody waits for them.
				b.logf("%d (eliminated) didn't come back", p.userID)
				p.left = true
				b.tryFinishTurn()
				continue
			}
			gone = append(gone, p)
		}
	}
	if b.Over() {
		return false
	}
	var goneTeams []int
	for _, p := range gone {
		if !slices.Contains(goneTeams, p.team) {
			goneTeams = append(goneTeams, p.team)
		}
	}
	if len(goneTeams) > 1 {
		b.logf("players of both sides gone: cancelled")
		b.finish(b.all(0), false)
		return false
	}
	// One side lost a player; the other side wins if it's here (if it's away too, wait for it to come back or
	// time out).
	if len(gone) > 0 && b.opponentConnected(gone[0].team) {
		b.logf("%d didn't come back: team %d loses", gone[0].userID, gone[0].team)
		b.sendOthers(gone[0], b.sysMsg(sysDroppedByReconnectionTimeout, gone[0]))
		b.finish(b.teamLoses(gone[0]), false)
		return false
	}

	if b.activeEnd == nil {
		// The active player stopped playing (connected, but nothing for a long time).
		if act := b.activePlayer(); act.connected() && now.Sub(b.lastActivity) > TurnTimeout {
			b.logf("%d inactive: team %d loses", act.userID, act.team)
			b.sendOthers(act, b.sysMsg(sysDroppedByCommandTimeout, act))
			b.finish(b.teamLoses(act), false)
		}
		return false
	}
	var waiting []*player
	for _, p := range b.responders() {
		if p.end == nil && p.connected() {
			waiting = append(waiting, p)
		}
	}
	if len(waiting) == 0 {
		return false
	}
	if now.Sub(b.activeEndAt) > ResponseTimeout {
		pas := waiting[0]
		b.logf("%d never confirmed the turn: team %d loses", pas.userID, pas.team)
		b.sendOthers(pas, b.sysMsg(sysDroppedByResponseTimeout, pas))
		b.finish(b.teamLoses(pas), false)
	} else if now.Sub(b.lastEndTurn) > EndTurnResendInterval {
		// The client ignores EndTurn until it's ready to replay the turn (e.g. still building the map on
		// turn 1), so repeat it. A client that already has it ignores the copy; one that misses moves asks
		// for them (RetryRequestClient).
		b.lastEndTurn = now
		if n := len(b.turnCommands); n > 0 {
			for _, p := range waiting {
				send(p, b.turnCommands[n-1].msg)
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

	if b.v125 {
		b.endBattle = b.endBattle125(desync)
	} else {
		b.endBattle = b.endBattle105(desync)
	}
	for _, p := range b.players {
		if !p.left {
			b.sendEndBattle(p)
		}
	}
	b.settle()
}

// endBattle105 is the 1.05.0 PvpEndBattle.
func (b *Battle) endBattle105(desync bool) []byte {
	return message(CmdEndBattle, true, func(w *wire.Writer) {
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
}

// endBattle125 is the 1.25.0 PvpEndBattle: player numbers and their results (0 won, 1 lost, 2 draw), and
// whether the result is confirmed (not a desync).
func (b *Battle) endBattle125(desync bool) []byte {
	return message125(CmdEndBattle, true, func(w *wire.Writer) {
		w.U32(b.ID)
		w.U16(uint16(b.turn))
		w.U16(b.lastCommandNum)
		w.U16(uint16(len(b.players)))
		for _, p := range b.players {
			w.U8(byte(p.seat))
		}
		w.U16(uint16(len(b.players)))
		for _, p := range b.players {
			switch b.results[p.userID] {
			case 1:
				w.U8(0)
			case 0:
				w.U8(2)
			default:
				w.U8(1)
			}
		}
		w.Bool(!desync)
	})
}

// settle pays out the result (wager, rating) in the API, off the battle's lock.
func (b *Battle) settle() {
	type payout struct {
		userID uint32
		token  string
		result int
	}
	var todo []payout
	for _, p := range b.players {
		token := p.token
		if s := Find(p.userID); s != nil && s.Token() != "" {
			token = s.Token()
		}
		todo = append(todo, payout{p.userID, token, b.results[p.userID]})
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
