package pvp

import (
	"testing"
	"time"

	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

type pveFixture struct {
	t     *testing.T
	b     *Battle
	conns []*fakeConn
}

// newPveFixture: a co-op boss battle of players 1 and 2 (numbers 0 and 1), both logged in.
func newPveFixture(t *testing.T) *pveFixture {
	t.Helper()
	ps := []*player{{userID: 1}, {userID: 2}}
	f := &pveFixture{t: t, b: newBattle(88, 0, ps, ps)}
	f.b.v125, f.b.pve = true, true
	for _, p := range ps {
		c := &fakeConn{}
		f.conns = append(f.conns, c)
		if !f.b.Login(c, p.userID) {
			t.Fatalf("login of %d refused", p.userID)
		}
	}
	return f
}

func (f *pveFixture) action(num int) {
	var w wire.Writer
	w.U32(0)
	w.U32(30)
	w.U16(uint16(f.b.turn))
	w.U16(1)
	w.U32(f.b.ID)
	w.U8(byte(num))
	f.b.OnMessage(f.conns[num], CmdAction, w.B)
}

// end: player num's EndTurn (or EndTurnResponse) of the current turn eliminating these squads.
func (f *pveFixture) end(num int, response bool, elim ...byte) {
	var w wire.Writer
	w.U16(uint16(f.b.turn))
	w.U16(2)
	w.U32(f.b.ID)
	cmd := uint32(CmdEndTurnResponse)
	if !response {
		w.U8(byte(num))
		cmd = CmdEndTurn
	}
	w.U16(uint16(len(elim)))
	for _, x := range elim {
		w.U8(x)
	}
	w.U16(0)
	w.U16(0)
	w.I32(5)
	f.b.OnMessage(f.conns[num], cmd, w.B)
}

func (f *pveFixture) startTurns() (int, int) {
	return f.conns[0].count(CmdStartTurn), f.conns[1].count(CmdStartTurn)
}

func (f *pveFixture) results() map[uint32]int {
	f.b.mu.Lock()
	defer f.b.mu.Unlock()
	return f.b.results
}

// Either player may move (the client picks), the AI's turns only have answers, and the players lose when
// both squads are eliminated.
func TestCoopBattleTurns(t *testing.T) {
	f := newPveFixture(t)

	f.action(0)
	f.end(0, false)
	if f.conns[1].count(CmdAction) != 1 || f.conns[1].count(CmdEndTurn) != 1 || f.conns[0].count(CmdEndTurn) != 0 {
		t.Fatalf("moves of player 0 not forwarded to player 1 only: %v / %v", f.conns[0].cmds, f.conns[1].cmds)
	}
	f.end(1, true)
	if a, b := f.startTurns(); a != 1 || b != 1 || f.b.turn != 2 {
		t.Fatalf("turn 1 not over: StartTurn %d/%d, turn %d", a, b, f.b.turn)
	}

	// Player 1 moves; player 0's moves of the same turn are ignored.
	f.action(1)
	f.action(0)
	if f.conns[1].count(CmdAction) != 1 {
		t.Fatal("a second mover's Action forwarded")
	}
	f.end(1, false)
	f.end(0, true)
	if f.b.turn != 3 {
		t.Fatalf("turn 2 not over: turn %d", f.b.turn)
	}

	// An AI turn: both answer, one AI squad (number 2) is eliminated.
	f.end(0, true, 2)
	if f.b.turn != 3 {
		t.Fatal("AI turn over before everyone answered")
	}
	f.end(1, true, 2)
	if f.b.turn != 4 || f.b.Over() {
		t.Fatalf("AI turn not over: turn %d, over %v", f.b.turn, f.b.Over())
	}

	// The boss kills one squad, then the other.
	f.end(0, true, 0)
	f.end(1, true, 0)
	if f.b.Over() {
		t.Fatal("over with a squad left")
	}
	f.action(0) // eliminated: can't move any more
	if f.b.actor != nil {
		t.Fatal("an eliminated player took the turn")
	}
	f.end(1, true, 1)
	if r := f.results(); !f.b.Over() || r[1] != -1 || r[2] != -1 {
		t.Fatalf("both squads dead: over %v, results %v", f.b.Over(), r)
	}
}

// The boss is dead when the battle goes quiet after a StartTurn.
func TestCoopBattleWonWhenQuiet(t *testing.T) {
	f := newPveFixture(t)
	f.action(0)
	f.end(0, false)
	f.end(1, true)

	f.b.quietSince = time.Now().Add(-PveOverMin)
	f.b.tick(time.Minute)
	if f.b.Over() {
		t.Fatal("won before the quiet limit")
	}
	f.b.OnDisconnected(f.conns[0])
	f.b.quietSince = time.Now().Add(-PveOverTimeout - time.Second)
	f.b.tick(time.Minute)
	if f.b.Over() {
		t.Fatal("won while a player is disconnected")
	}
	f.b.Reconnect(f.conns[0], 1, 2, 0)
	f.b.quietSince = time.Now().Add(-PveOverTimeout - time.Second)
	f.b.tick(time.Minute)
	if r := f.results(); !f.b.Over() || r[1] != 1 || r[2] != 1 {
		t.Fatalf("quiet: over %v, results %v", f.b.Over(), r)
	}
	if f.conns[0].count(CmdEndBattle) != 1 || f.conns[1].count(CmdEndBattle) != 1 {
		t.Fatal("EndBattle not sent to both")
	}
}

// A player who leaves with worms alive cancels the battle: they lose, the partner gets the battle back.
func TestCoopBattleSurrenderCancels(t *testing.T) {
	f := newPveFixture(t)
	var w wire.Writer
	w.U8(1)
	w.U32(f.b.ID)
	w.I16(1)
	f.b.OnMessage(f.conns[1], CmdDropPlayer, w.B)
	if r := f.results(); !f.b.Over() || r[1] != 0 || r[2] != -1 {
		t.Fatalf("surrender: over %v, results %v", f.b.Over(), r)
	}
}
