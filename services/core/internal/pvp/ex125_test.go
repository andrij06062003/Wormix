package pvp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/andrij06062003/Wormix/services/core/internal/api"
	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

// player125 is a main-server session of a 1.25.0 client.
type player125 struct {
	id  uint32
	box *msgBox
}

func (p *player125) UserID() uint32 {
	return p.id
}

func (p *player125) AuthKey() string {
	return "key"
}

func (p *player125) Token() string {
	return string(rune('0'+p.id)) + ".k"
}

func (p *player125) SendMessage(m []byte) bool {
	return p.box.SendMessage(m)
}

func (p *player125) V125() bool {
	return true
}

type sent struct {
	cmd  uint32
	body []byte
}

// msgBox records what is sent to a connection.
type msgBox struct {
	mu     sync.Mutex
	msgs   []sent
	battle *Battle
}

func (c *msgBox) SendMessage(msg []byte) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	cmd, _, err := wire.ParseHeader(msg[:wire.HeaderSize])
	if err != nil {
		panic(err)
	}
	c.msgs = append(c.msgs, sent{cmd, msg[wire.HeaderSize:]})
	return true
}
func (c *msgBox) Close() {}

func (c *msgBox) Battle() *Battle {
	return c.battle
}

func (c *msgBox) SetBattle(b *Battle) {
	c.battle = b
}

// last returns the body of the last message with this id (nil if none).
func (c *msgBox) last(cmd uint32) []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := len(c.msgs) - 1; i >= 0; i-- {
		if c.msgs[i].cmd == cmd {
			return c.msgs[i].body
		}
	}
	return nil
}

// TestMain starts the fake API before any test: battles record their results in the background, after
// their test may have ended, so api.BaseURL must not change while tests run.
func TestMain(m *testing.M) {
	startFakeAPI125()
	os.Exit(m.Run())
}

func withAPI125(t *testing.T, ids ...uint32) map[uint32]*player125 {
	t.Helper()
	ps := map[uint32]*player125{}
	for _, id := range ids {
		ps[id] = &player125{id: id, box: &msgBox{}}
		Register(ps[id])
	}
	t.Cleanup(func() {
		for _, p := range ps {
			Unregister(p)
		}
		exMu.Lock()
		exRooms, exQueue = map[uint32]*exRoom{}, nil
		exMu.Unlock()
	})
	return ps
}

// startFakeAPI125 answers the PvP routes for any player.
func startFakeAPI125() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.ReadAll(r.Body)
		var id int
		json.Unmarshal([]byte(strings.Split(r.Header.Get("X-SESSION-KEY"), ".")[0]), &id)
		profile := map[string]any{"Id": id, "SocialId": "s", "Money": 450, "RealMoney": 0, "Rating": 0,
			"WormsGroup": []any{}, "WeaponRecordList": []any{}, "Stuff": []any{}, "ReactionRate": 0, "Recipes": []any{}}
		var out any
		switch strings.TrimPrefix(r.URL.Path, "/api/internal/") {
		case "pvp/profile":
			out = map[string]any{"data": profile}
		case "pvp/start_battle":
			out = map[string]any{"data": map[string]any{"UserProfileStructure": profile, "ReagentsForBattle": []int{}}}
		default:
			out = map[string]any{"Status": "OK"}
		}
		json.NewEncoder(w).Encode(out)
	}))
	api.BaseURL = srv.URL + "/api/internal/"
}

func signed125(w *wire.Writer) []byte {
	return append(append([]byte(nil), w.B...), wire.SecureWith(w.B, wire.SecretKey125)...)
}

func exLoginPart(w *wire.Writer, id uint32) {
	w.U32(id)
	w.UTF("s")
	w.U8(1)
	w.UTF("key")
	w.UTF("{}")
	w.U16(1)
}

func createBattle125(id uint32, typ int, wagerID int, ids []uint32, teams []byte) []byte {
	var w wire.Writer
	exLoginPart(&w, id)
	w.I16(typ)
	w.U32(7) // map
	if wagerID != 0 {
		writeI16s(&w, []int16{int16(wagerID)})
	} else {
		writeI16s(&w, []int16{0})
	}
	writeI16s(&w, nil)
	w.U16(uint16(len(ids)))
	for _, x := range ids {
		w.U32(x)
	}
	w.U16(uint16(len(teams)))
	for _, x := range teams {
		w.U8(x)
	}
	w.Bool(false)
	w.Bool(false)
	return signed125(&w)
}

func joinOrReject125(id, battleID uint32, reject bool) []byte {
	var w wire.Writer
	exLoginPart(&w, id)
	w.U32(battleID)
	if reject {
		w.Bool(true)
	}
	return signed125(&w)
}

func ready125(battleID uint32) []byte {
	var w wire.Writer
	w.U32(battleID)
	return signed125(&w)
}

// endTurn125 is PvpEndTurn (response false) or PvpEndTurnResponse of turn 1 eliminating these player numbers.
func endTurn125(battleID uint32, response bool, eliminated ...byte) []byte {
	var w wire.Writer
	w.U16(1)
	w.U16(1)
	w.U32(battleID)
	if !response {
		w.U8(0)
	}
	w.U16(uint16(len(eliminated)))
	for _, x := range eliminated {
		w.U8(x)
	}
	w.U16(0) // collected reagents
	w.U16(1) // items: one WeaponStructure
	w.U16(8)
	w.U32(5)
	w.U32(1)
	w.U32(0) // battleState
	if !response {
		w.Bool(false)
		w.I16(0)
		w.UTF("")
	}
	return signed125(&w)
}

func mustHandle(t *testing.T, c Conn, cmd uint32, body []byte) {
	t.Helper()
	if !HandlePvp(c, cmd, body) {
		t.Fatalf("command %d not handled", cmd)
	}
}

// battleIDOf reads the battle id of a CallToBattle.
func battleIDOf(body []byte) uint32 {
	r := wire.NewReader(body)
	r.U32()
	r.U16()
	return r.U32()
}

func TestFriendDuel125(t *testing.T) {
	ps := withAPI125(t, 1, 2)
	a, b := &msgBox{}, &msgBox{}

	mustHandle(t, a, cmdExCreateBattle, createBattle125(1, exFriendPvp, 0, []uint32{1, 2}, []byte{0, 1}))
	call := a.last(cmdExCallToBattle)
	if call == nil || ps[2].box.last(cmdExCallToBattle) == nil {
		t.Fatalf("CallToBattle not sent to both: caller %v, friend %v", a.msgs, ps[2].box.msgs)
	}
	id := battleIDOf(call)

	mustHandle(t, b, cmdExJoin, joinOrReject125(2, id, false))
	if a.last(CmdSystemMessage) == nil {
		t.Fatalf("caller not told the friend joined: %v", a.msgs)
	}
	for _, c := range []*msgBox{a, b} {
		m := c.last(cmdExBattleCreated)
		if m == nil || !wire.IsSecureWith(m, wire.SecretKey125) {
			t.Fatalf("no signed BattleCreated: %v", c.msgs)
		}
	}

	mustHandle(t, a, cmdExReady, ready125(id))
	if a.last(cmdExStartBattle) != nil {
		t.Fatal("StartBattle before everyone is ready")
	}
	mustHandle(t, b, cmdExReady, ready125(id))
	for i, c := range []*msgBox{a, b} {
		m := c.last(cmdExStartBattle)
		if m == nil {
			t.Fatalf("no StartBattle for player %d", i)
		}
		r := wire.NewReader(m)
		if r.U32() != id || r.UTF() != "" || int(r.U8()) != i {
			t.Fatalf("StartBattle of player %d: % x", i, m)
		}
	}

	// Player 0 moves first and eliminates player 1; player 1 agrees.
	mustHandle(t, a, CmdEndTurn, endTurn125(id, false, 1))
	if b.last(CmdEndTurn) == nil {
		t.Fatal("EndTurn not forwarded")
	}
	mustHandle(t, b, CmdEndTurnResponse, endTurn125(id, true, 1))
	for i, c := range []*msgBox{a, b} {
		m := c.last(CmdEndBattle)
		if m == nil || !wire.IsSecureWith(m, wire.SecretKey125) {
			t.Fatalf("no signed EndBattle for player %d", i)
		}
		r := wire.NewReader(m)
		r.U32()
		r.U16()
		r.U16()
		if r.U16() != 2 || r.U8() != 0 || r.U8() != 1 {
			t.Fatalf("EndBattle players: % x", m)
		}
		if r.U16() != 2 || r.U8() != 0 || r.U8() != 1 || !r.Bool() {
			t.Fatalf("EndBattle results (player 0 wins, confirmed): % x", m)
		}
	}
}

func TestFriendRejects125(t *testing.T) {
	withAPI125(t, 1, 2)
	a, b := &msgBox{}, &msgBox{}
	mustHandle(t, a, cmdExCreateBattle, createBattle125(1, exFriendPvp, 0, []uint32{1, 2}, []byte{0, 1}))
	id := battleIDOf(a.last(cmdExCallToBattle))
	mustHandle(t, b, cmdExReject, joinOrReject125(2, id, true))
	m := a.last(cmdExCreationFailure)
	if m == nil {
		t.Fatal("caller not told")
	}
	r := wire.NewReader(m)
	if r.U32() != id || r.U32() != 2 || r.U8() != 1 || r.U16() != exRejected {
		t.Fatalf("BattleCreationFailure: % x", m)
	}
	if len(exRooms) != 0 {
		t.Fatal("room left behind")
	}
}

func TestWagerDuel125(t *testing.T) {
	withAPI125(t, 1, 2)
	a, b := &msgBox{}, &msgBox{}
	mustHandle(t, a, cmdExCreateBattle, createBattle125(1, exWagerDuel, 1, []uint32{1}, []byte{0}))
	if a.last(cmdExBattleCreated) != nil || len(exQueue) != 1 {
		t.Fatal("first player should wait")
	}
	mustHandle(t, b, cmdExCreateBattle, createBattle125(2, exWagerDuel, 1, []uint32{2}, []byte{0}))
	for _, c := range []*msgBox{a, b} {
		m := c.last(cmdExBattleCreated)
		if m == nil {
			t.Fatalf("no BattleCreated: %v", c.msgs)
		}
		r := wire.NewReader(m)
		if r.U16() != exWagerDuel {
			t.Fatalf("battle type: % x", m)
		}
		r.U32()
		if r.U16() != 1 {
			t.Fatalf("wager id: % x", m)
		}
	}
}

func TestCreateBattle105StillLogin(t *testing.T) {
	// An unsigned 1001 is the 1.05.0 BattleLogin, not a CreateBattleRequest.
	var w wire.Writer
	w.U32(1)
	w.Bool(true)
	w.UTF("key")
	w.U16(0)
	w.UTF("h")
	w.U32(0)
	w.U32(42)
	if handleEx(&msgBox{}, cmdExCreateBattle, w.B) {
		t.Fatal("1.05.0 BattleLogin taken for a 1.25.0 request")
	}
}
