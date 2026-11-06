package pvp

import (
	"bytes"
	"slices"
	"testing"
	"time"

	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

// fakeConn records the command ids sent to a player.
type fakeConn struct {
	cmds   []uint32
	battle *Battle
}

func (c *fakeConn) SendMessage(msg []byte) bool {
	cmd, _, err := wire.ReadHeader(bytes.NewReader(msg))
	if err != nil {
		panic(err)
	}
	c.cmds = append(c.cmds, cmd)
	return true
}
func (c *fakeConn) Close()              {}
func (c *fakeConn) Battle() *Battle     { return c.battle }
func (c *fakeConn) SetBattle(b *Battle) { c.battle = b }

func (c *fakeConn) count(cmd uint32) int {
	n := 0
	for _, x := range c.cmds {
		if x == cmd {
			n++
		}
	}
	return n
}

type teamFixture struct {
	t     *testing.T
	b     *Battle
	conns map[uint32]*fakeConn
}

// newTeamFixture: a 2v2 battle with players 1, 2 (team 1) and 3, 4 (team 2), turn order 1, 3, 2, 4, all
// logged in.
func newTeamFixture(t *testing.T) *teamFixture {
	t.Helper()
	order := []*player{
		{userID: 1, team: 1, firstTurn: true}, {userID: 3, team: 2}, {userID: 2, team: 1}, {userID: 4, team: 2},
	}
	f := &teamFixture{t: t, b: newBattle(77, 0, order, order), conns: map[uint32]*fakeConn{}}
	for _, p := range order {
		c := &fakeConn{}
		f.conns[p.userID] = c
		if !f.b.Login(c, p.userID) {
			t.Fatalf("login of %d refused", p.userID)
		}
	}
	return f
}

func (f *teamFixture) active() uint32 { return f.b.activePlayer().userID }

// playTurn: the active player ends the turn eliminating elim, everyone else who must answer agrees.
func (f *teamFixture) playTurn(elim ...uint32) {
	f.t.Helper()
	b := f.b
	act := b.activePlayer()
	turn := uint16(b.turn)
	num := uint16(1)
	if turn > 1 {
		num = 2
	}
	var w wire.Writer
	w.U16(turn)
	w.U16(num)
	w.U32(b.ID)
	w.U8(act.num())
	w.U16(uint16(len(elim)))
	for _, id := range elim {
		w.U32(id)
	}
	w.I32(5)
	b.OnMessage(f.conns[act.userID], CmdEndTurn, w.B)
	for _, p := range b.responders() {
		f.respond(p.userID, elim...)
	}
}

func (f *teamFixture) respond(userID uint32, elim ...uint32) {
	var w wire.Writer
	w.U16(uint16(f.b.turn))
	w.U16(2)
	w.U32(f.b.ID)
	w.U16(uint16(len(elim)))
	for _, id := range elim {
		w.U32(id)
	}
	w.I32(5)
	f.b.OnMessage(f.conns[userID], CmdEndTurnResponse, w.B)
}

func (f *teamFixture) results() map[uint32]int {
	f.b.mu.Lock()
	defer f.b.mu.Unlock()
	return f.b.results
}

func TestTeamBattleTurnOrderFollowsClient(t *testing.T) {
	f := newTeamFixture(t)
	want := []uint32{1, 3, 2, 4, 1}
	for i, id := range want {
		if got := f.active(); got != id {
			t.Fatalf("turn %d: active %d, want %d", i+1, got, id)
		}
		f.playTurn()
	}
	// Each player saw StartTurn for every turn after the first.
	for id, c := range f.conns {
		if n := c.count(CmdStartTurn); n != len(want) {
			t.Fatalf("player %d got %d StartTurn, want %d", id, n, len(want))
		}
	}

	// Turn 6 is player 3's (index 1). Player 3 eliminates player 1 (index 0): the client drops that squad and
	// keeps the index, so the next one is index 2 of [3, 2, 4]: player 4, not player 2.
	if f.active() != 3 {
		t.Fatalf("active %d, want 3", f.active())
	}
	f.playTurn(1)
	if f.active() != 4 {
		t.Fatalf("after eliminating 1: active %d, want 4", f.active())
	}
	if f.b.Over() {
		t.Fatal("battle over with a player of each team left")
	}
	// Player 1 is out: the turn doesn't wait for them.
	if slices.ContainsFunc(f.b.responders(), func(p *player) bool { return p.userID == 1 }) {
		t.Fatal("eliminated player still has to answer")
	}
	f.playTurn()
	if f.active() != 3 {
		t.Fatalf("active %d, want 3", f.active())
	}
}

func TestTeamBattleWonWhenWholeTeamOut(t *testing.T) {
	f := newTeamFixture(t)
	f.playTurn(3) // player 1 kills player 3
	if f.b.Over() {
		t.Fatal("over after one player of team 2 died")
	}
	f.playTurn(4)
	if !f.b.Over() {
		t.Fatal("not over after team 2 died")
	}
	want := map[uint32]int{1: 1, 2: 1, 3: -1, 4: -1}
	for id, r := range want {
		if got := f.results()[id]; got != r {
			t.Fatalf("result of %d: %d, want %d", id, got, r)
		}
	}
	for id, c := range f.conns {
		if c.count(CmdEndBattle) != 1 {
			t.Fatalf("player %d got no EndBattle", id)
		}
	}
}

func TestTeamBattleWaitsForEveryResponder(t *testing.T) {
	f := newTeamFixture(t)
	var w wire.Writer
	w.U16(1)
	w.U16(1)
	w.U32(77)
	w.U8(1)
	w.U16(0)
	w.I32(5)
	f.b.OnMessage(f.conns[1], CmdEndTurn, w.B)
	f.respond(3)
	f.respond(2)
	if f.b.turn != 1 {
		t.Fatal("turn finished before every player answered")
	}
	f.respond(4)
	if f.b.turn != 2 || f.active() != 3 {
		t.Fatalf("turn %d, active %d; want turn 2, active 3", f.b.turn, f.active())
	}
}

func TestTeamBattleDesync(t *testing.T) {
	f := newTeamFixture(t)
	var w wire.Writer
	w.U16(1)
	w.U16(1)
	w.U32(77)
	w.U8(1)
	w.U16(0)
	w.I32(5)
	f.b.OnMessage(f.conns[1], CmdEndTurn, w.B)
	f.respond(3)
	f.respond(2, 4) // sees player 4 dead, the others don't
	f.respond(4)
	if !f.b.Over() {
		t.Fatal("no desync")
	}
	for id, r := range f.results() {
		if r != 0 {
			t.Fatalf("desync result of %d: %d, want draw", id, r)
		}
	}
}

func dropBody(userID uint32, typ int32) []byte {
	var w wire.Writer
	w.U32(userID)
	w.U32(77)
	w.I32(typ)
	return w.B
}

func TestTeamBattleSurrenderLosesForTeam(t *testing.T) {
	f := newTeamFixture(t)
	f.b.OnMessage(f.conns[2], CmdDropPlayer, dropBody(2, 1))
	if !f.b.Over() {
		t.Fatal("not over after a surrender")
	}
	want := map[uint32]int{1: -1, 2: -1, 3: 1, 4: 1}
	for id, r := range want {
		if got := f.results()[id]; got != r {
			t.Fatalf("result of %d: %d, want %d", id, got, r)
		}
	}
	if f.conns[1].count(CmdSystemMessage) == 0 || f.conns[3].count(CmdSystemMessage) == 0 {
		t.Fatal("the others weren't told about the surrender")
	}
}

func TestTeamBattleEliminatedPlayerMayLeave(t *testing.T) {
	f := newTeamFixture(t)
	f.playTurn(3) // player 3 out; next is index 1 of [1, 2, 4]: player 2
	f.b.OnMessage(f.conns[3], CmdDropPlayer, dropBody(3, 1))
	if f.b.Over() {
		t.Fatal("battle ended when an eliminated player left")
	}
	if f.active() != 2 {
		t.Fatalf("active %d, want 2", f.active())
	}
	f.playTurn()
	if f.active() != 4 {
		t.Fatalf("active %d, want 4", f.active())
	}
}

func TestTeamBattleReconnectTimeoutLosesForTeam(t *testing.T) {
	f := newTeamFixture(t)
	f.b.OnDisconnected(f.conns[4])
	past := time.Now().Add(-ReconnectTimeout - time.Second)
	f.b.mu.Lock()
	f.b.playerByID(4).disconnectedAt = &past
	f.b.mu.Unlock()
	f.b.tick(time.Minute)
	if !f.b.Over() {
		t.Fatal("not over after a player didn't come back")
	}
	if f.results()[3] != -1 || f.results()[1] != 1 {
		t.Fatalf("results %v: team 2 should lose", f.results())
	}
}

func TestDuelUnchanged(t *testing.T) {
	// 1v1 keeps the old rules: the second player given may move first, turns alternate, one dead loses.
	a := &player{userID: 10}
	bp := &player{userID: 20, firstTurn: true}
	b := newDuel(5, 0, a, bp)
	if b.activePlayer() != bp || a.num() != 2 || bp.num() != 1 {
		t.Fatalf("first mover / player numbers wrong")
	}
	if b.players[0] != a {
		t.Fatal("EndBattle order changed")
	}
}
