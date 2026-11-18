package pvp

import (
	"testing"

	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

// player161 is a main-server session of a 1.61.7 client.
type player161 struct{ player125 }

func (p *player161) V161() bool { return true }

// box161 is a PvP connection that records whether it was marked as a 1.61.7 client's.
type box161 struct {
	msgBox
	v161 bool
}

func (c *box161) SetV161() { c.v161 = true }

func withPlayers161(t *testing.T, ids ...uint32) map[uint32]*player161 {
	t.Helper()
	ps := map[uint32]*player161{}
	for _, id := range ids {
		ps[id] = &player161{player125{id: id, box: &msgBox{}}}
		Register(ps[id])
	}
	t.Cleanup(func() {
		for _, p := range ps {
			Unregister(p)
		}
		exMu.Lock()
		exRooms, exQueue = map[uint32]*exRoom{}, nil
		exMu.Unlock()
		// A battle a test launched must not make its players busy in the next test.
		battlesMu.Lock()
		battles = map[uint32]*Battle{}
		battlesMu.Unlock()
	})
	return ps
}

func signed161(w *wire.Writer) []byte {
	return append(append([]byte(nil), w.B...), wire.SecureWith(w.B, wire.SecretKey161)...)
}

// createBattle161 is the 1.61.7 CreateBattleRequest of a co-op boss (wager 12) or friend battle (15).
func createBattle161(id uint32, wager int16, missions []int16, ids []uint32, teams []byte) []byte {
	var w wire.Writer
	exLoginPart(&w, id)
	w.U32(7) // map
	w.I16(int(wager))
	writeI16s(&w, missions)
	w.U16(uint16(len(ids)))
	for _, x := range ids {
		w.U32(x)
	}
	w.U16(uint16(len(teams)))
	for _, x := range teams {
		w.U8(x)
	}
	w.UTF(`{"p":1}`)
	w.U8(1) // friendSocialNetId
	return signed161(&w)
}

func join161(id, battleID uint32) []byte {
	var w wire.Writer
	exLoginPart(&w, id)
	w.U32(battleID)
	w.U16(0) // backpackConf
	return signed161(&w)
}

// skipSized skips a list of sized structures.
func skipSized(r *wire.Reader) int {
	n := int(r.U16())
	for i := 0; i < n; i++ {
		for size := int(r.U16()); size > 0; size-- {
			r.U8()
		}
	}
	return n
}

// A co-op boss of two 1.61.7 players: CallToBattle and BattleCreated in the 1.61.7 layout (its wager id, its
// params given back), the connections marked as 1.61.7 ones, and the battle started for both.
func TestCoopInvite161(t *testing.T) {
	ps := withPlayers161(t, 1, 2)
	a, b := &box161{}, &box161{}

	mustHandle(t, a, cmdExCreateBattle, createBattle161(1, 12, []int16{101}, []uint32{1, 2}, []byte{0, 1}))
	if !a.v161 {
		t.Fatal("caller's connection not marked as 1.61.7")
	}
	call := a.last(cmdExCallToBattle)
	if call == nil || ps[2].box.last(cmdExCallToBattle) == nil {
		t.Fatalf("CallToBattle not sent to both: caller %v, friend %v", a.msgs, ps[2].box.msgs)
	}
	r := wire.NewReader(call)
	if r.U32() != 1 || r.U16() != 12 {
		t.Fatalf("CallToBattle caller/wager: % x", call)
	}
	id := r.U32()
	r.U32()
	if m := readI16s(r); len(m) != 1 || m[0] != 101 {
		t.Fatalf("CallToBattle missions %v", m)
	}
	if n := skipSized(r); n != 2 {
		t.Fatalf("CallToBattle profiles %d", n)
	}
	if p := r.UTF(); p != `{"p":1}` || r.Remaining() != 0 {
		t.Fatalf("CallToBattle params %q, %d bytes left", p, r.Remaining())
	}

	mustHandle(t, b, cmdExJoin, join161(2, id))
	if !b.v161 {
		t.Fatal("friend's connection not marked as 1.61.7")
	}
	for _, c := range []*box161{a, b} {
		m := c.last(cmdExBattleCreated)
		if m == nil || !wire.IsSecureWith(m, wire.SecretKey125) {
			t.Fatalf("no signed BattleCreated: %v", c.msgs)
		}
		r := wire.NewReader(m)
		if r.U16() != exPveFriend || r.U32() != id || r.U16() != 12 {
			t.Fatalf("BattleCreated type/id/wager: % x", m)
		}
		r.U32()
		readI16s(r)
		if n := skipSized(r); n != 2 {
			t.Fatalf("BattleCreated profiles %d", n)
		}
		// The seed, then the reagents.
		r.U32()
		for n := r.U16(); n > 0; n-- {
			r.U8()
		}
		r.U16() // questId
		r.U16()
		for range 3 {
			r.U16()
			r.U32()
			r.U32()
		}
		r.U32() // serverTime
		if r.Remaining() != 16 {
			t.Fatalf("BattleCreated: %d bytes after serverTime, want the 16 of the MD5", r.Remaining())
		}
	}

	mustHandle(t, a, cmdExReady, ready125(id))
	mustHandle(t, b, cmdExReady, ready125(id))
	bt := findBattle(id)
	if bt == nil || !bt.v161 || !bt.pve {
		t.Fatalf("battle %v not a 1.61.7 co-op one", bt)
	}
	for i, c := range []*box161{a, b} {
		if c.last(cmdExStartBattle) == nil {
			t.Fatalf("no StartBattle for player %d", i)
		}
	}
}

// A 1.61.7 player can't call a 1.25.0 one (their games can't play together): the friend is "offline".
func TestCoopInvite161Not125(t *testing.T) {
	withPlayers161(t, 1)
	withAPI125(t, 2)
	a := &box161{}
	mustHandle(t, a, cmdExCreateBattle, createBattle161(1, 12, []int16{101}, []uint32{1, 2}, []byte{0, 0}))
	m := a.last(cmdExCreationFailure)
	if m == nil {
		t.Fatalf("no BattleCreationFailure: %v", a.msgs)
	}
	r := wire.NewReader(m)
	r.U32()
	r.U32()
	r.U8()
	if r.U16() != exOffline {
		t.Fatalf("BattleCreationFailure: % x", m)
	}
}

// The 1.61.7 PvpEndBattle tells each player its own result: here player 2 surrenders (loses) and player 1
// gets its battle back (a draw for the client).
func TestEndBattle161PerPlayer(t *testing.T) {
	ps := []*player{{userID: 1}, {userID: 2}}
	bt := newBattle(89, 0, ps, ps)
	bt.v125, bt.v161, bt.pve = true, true, true
	conns := []*msgBox{{}, {}}
	for i, p := range ps {
		if !bt.Login(conns[i], p.userID) {
			t.Fatalf("login of %d refused", p.userID)
		}
	}
	var w wire.Writer
	w.U8(1)
	w.U32(bt.ID)
	w.I16(1)
	bt.OnMessage(conns[1], CmdDropPlayer, w.B)
	want := []int{2, 1}
	for i, c := range conns {
		if i == 1 {
			continue // it left
		}
		m := c.last(CmdEndBattle)
		if m == nil {
			t.Fatalf("no EndBattle for player %d", i)
		}
		r := wire.NewReader(m)
		if r.U32() != 89 {
			t.Fatalf("EndBattle battle id: % x", m)
		}
		r.U16()
		r.U16()
		if got := int(int16(r.U16())); got != want[i] {
			t.Fatalf("player %d result %d, want %d", i, got, want[i])
		}
		r.U32()
		r.U32()
		if r.U16() != 0 || r.UTF() != "" {
			t.Fatalf("EndBattle awards/note: % x", m)
		}
		r.Bool()
		if r.Remaining() != 16 {
			t.Fatalf("EndBattle: %d bytes after confirmed", r.Remaining())
		}
	}
}
