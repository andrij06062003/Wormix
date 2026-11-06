package server

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/andrij06062003/Wormix/services/core/internal/api"
	"github.com/andrij06062003/Wormix/services/core/internal/proto"
	"github.com/andrij06062003/Wormix/services/core/internal/pvp"
	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

func TestWriteStreamSplitsFrames(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	c := &conn{c: a, sid: newGUID()}
	got := make(chan []byte, 10)
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := b.Read(buf)
			if err != nil {
				close(got)
				return
			}
			got <- append([]byte(nil), buf[:n]...)
		}
	}()
	h := proto.Header(10013, 2)
	c.write(h[:3]) // incomplete: nothing sent
	c.write(h[3:])
	c.write([]byte{0}) // still one body byte missing
	select {
	case m := <-got:
		t.Fatalf("sent early: %x", m)
	case <-time.After(50 * time.Millisecond):
	}
	// The missing byte plus a whole second message: both go out together.
	second := proto.Header(10017, 0)
	c.write(append([]byte{1}, second...))
	m := <-got
	if len(m) != 9+7 || m[8] != 1 {
		t.Fatalf("got %x", m)
	}
	if len(c.pending) != 0 {
		t.Fatalf("pending %x", c.pending)
	}
}

func TestParseGUID(t *testing.T) {
	want := "0f8fad5b-d9cb-469f-a165-70867728950e"
	for _, s := range []string{want, strings.ToUpper(want), " " + want + " ", "{" + want + "}", "(" + want + ")",
		strings.ReplaceAll(want, "-", "")} {
		if g, err := parseGUID(s); err != nil || g != want {
			t.Errorf("parseGUID(%q) = %q, %v", s, g, err)
		}
	}
	for _, s := range []string{"", "not-a-guid", want[:35], strings.Replace(want, "0", "x", 1)} {
		if _, err := parseGUID(s); err == nil {
			t.Errorf("accepted %q", s)
		}
	}
	if !GUIDRe(newGUID()) {
		t.Error("newGUID format")
	}
}

func GUIDRe(s string) bool {
	g, err := parseGUID(s)
	return err == nil && g == s
}

// ---- end to end, in process ---------------------------------------------------------------------------------

type fakeAPI struct {
	mu       sync.Mutex
	requests []string // "route key body"
}

func (f *fakeAPI) handler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	route := strings.TrimPrefix(r.URL.Path, "/api/internal/")
	f.mu.Lock()
	f.requests = append(f.requests, route+" "+r.Header.Get("X-SESSION-KEY")+" "+string(body))
	f.mu.Unlock()
	profile := func(id int) map[string]any {
		return map[string]any{"Id": id, "SocialId": "s", "Money": 450, "RealMoney": 0, "Rating": 0, "WormsGroup": []any{},
			"WeaponRecordList": []any{}, "Stuff": []any{}, "ReactionRate": 0, "Recipes": []any{}}
	}
	var out any
	switch route {
	case "login":
		var req struct{ Id int }
		json.Unmarshal(body, &req)
		others := []any{} // like the API: every other player, ordered by id
		for id := 1; id <= 4; id++ {
			if id != req.Id {
				others = append(others, profile(id))
			}
		}
		out = map[string]any{"type": "EnterAccount", "data": map[string]any{"UserProfileStructure": profile(req.Id),
			"UserProfileStructures": others, "DailyBonusStructure": map[string]any{}, "Reagents": []any{},
			"SessionKey": strings.Repeat("x", 0) + itoa(req.Id) + ".k"}}
	case "pvp/profile":
		out = map[string]any{"data": profile(idOf(r))}
	case "pvp/start_battle":
		out = map[string]any{"data": map[string]any{"UserProfileStructure": profile(idOf(r)), "ReagentsForBattle": []int{1}}}
	default:
		out = map[string]any{"data": map[string]any{"Result": 0}}
	}
	json.NewEncoder(w).Encode(out)
}

func itoa(i int) string { b, _ := json.Marshal(i); return string(b) }

func idOf(r *http.Request) int {
	var id int
	json.Unmarshal([]byte(strings.Split(r.Header.Get("X-SESSION-KEY"), ".")[0]), &id)
	return id
}

func (f *fakeAPI) seen(prefix string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, r := range f.requests {
		if strings.HasPrefix(r, prefix) {
			out = append(out, r)
		}
	}
	return out
}

type client struct {
	t *testing.T
	c net.Conn
}

func dial(t *testing.T, addr string) *client {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	return &client{t, c}
}

func (c *client) send(cmd uint32, body []byte, secure bool) {
	if _, err := c.c.Write(proto.Frame(cmd, body, secure)); err != nil {
		c.t.Fatal(err)
	}
}

// expect reads messages until one with this command id arrives.
func (c *client) expect(cmd uint32) []byte {
	c.t.Helper()
	c.c.SetReadDeadline(time.Now().Add(10 * time.Second))
	for {
		got, n, err := wire.ReadHeader(c.c)
		if err != nil {
			c.t.Fatalf("waiting for %d: %v", cmd, err)
		}
		body := make([]byte, n)
		if _, err := io.ReadFull(c.c, body); err != nil {
			c.t.Fatal(err)
		}
		if got == cmd {
			return body
		}
	}
}

func body(f func(w *wire.Writer)) []byte {
	var w wire.Writer
	f(&w)
	return w.B
}

func TestEndToEnd(t *testing.T) {
	fa := &fakeAPI{}
	srv := httptest.NewServer(http.HandlerFunc(fa.handler))
	defer srv.Close()
	api.BaseURL = srv.URL + "/api/internal/"
	pvp.ReconnectTimeout = 2 * time.Second
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go pvp.RunTicker(ctx)

	mainSrv, pvpSrv := New("main", "127.0.0.1:0"), New("pvp", "127.0.0.1:0")
	for _, s := range []*Server{mainSrv, pvpSrv} {
		if err := s.Start(); err != nil {
			t.Fatal(err)
		}
		defer s.Close()
	}

	login := func(id uint32) *client {
		c := dial(t, mainSrv.ListenAddr())
		c.send(proto.CmdLogin, body(func(w *wire.Writer) { w.U32(id); w.U32(0); w.UTF("KEY"); w.U16(0); w.U8(1) }), false)
		c.expect(proto.CmdEnterAccount)
		return c
	}
	a, b := login(1), login(2)

	// Ping and an API command with the session key.
	a.send(proto.CmdPing, nil, false)
	a.expect(proto.CmdPong)
	a.send(proto.CmdBuyBattle, body(func(w *wire.Writer) { w.U32(1) }), false)
	if r := a.expect(proto.CmdBuyBattleResult); len(r) != 2 {
		t.Fatalf("buy battle %x", r)
	}
	if got := fa.seen("account/buy/battles 1.k {\"MoneyType\":1}"); len(got) != 1 {
		t.Fatalf("API request: %v", fa.seen("account"))
	}

	// Challenge, accept.
	a.send(1002, body(func(w *wire.Writer) { w.UTF("h"); w.U32(49198); w.U32(3); w.U32(42); w.UTF("2") }), false)
	offer := b.expect(2002)
	battle := binary.BigEndian.Uint32(offer[len(offer)-4:])
	b.send(1003, body(func(w *wire.Writer) { w.U16(0); w.U32(1); w.U32(battle) }), false)
	if r := a.expect(2003); r[1] != 0 {
		t.Fatalf("request result %x", r)
	}

	pvpLogin := func(id uint32, first bool) *client {
		c := dial(t, pvpSrv.ListenAddr())
		c.send(pvp.CmdBattleLogin, body(func(w *wire.Writer) {
			w.U32(id)
			w.Bool(first)
			w.UTF("KEY")
			w.U16(0)
			w.UTF("h")
			w.U32(49197)
			w.U32(battle)
			w.U8(1)
		}), false)
		c.expect(pvp.CmdLoginSuccess)
		return c
	}
	pa, pb := pvpLogin(1, true), pvpLogin(2, false)

	// Turn 1: action relayed, end turn, response -> StartTurn.
	pa.send(pvp.CmdAction, body(func(w *wire.Writer) {
		w.U32(1)
		w.U32(2)
		w.U16(1)
		w.U16(1)
		w.U32(battle)
		w.U8(1)
	}), true)
	pb.expect(pvp.CmdAction)
	pa.send(pvp.CmdEndTurn, body(func(w *wire.Writer) {
		w.U16(1)
		w.U16(2)
		w.U32(battle)
		w.U8(1)
		w.U16(0)
		w.U32(7)
	}), true)
	pb.expect(pvp.CmdEndTurn)
	pb.send(pvp.CmdEndTurnResponse, body(func(w *wire.Writer) { w.U16(1); w.U16(2); w.U32(battle); w.U16(0); w.U32(7) }), true)
	st := pa.expect(pvp.CmdStartTurn)
	if !wire.IsSecure(st) || binary.BigEndian.Uint32(st[4:8]) != 2 {
		t.Fatalf("start turn %x", st)
	}
	pb.expect(pvp.CmdStartTurn)

	// B disconnects and doesn't come back: A wins after the reconnect timeout.
	pb.c.Close()
	pa.expect(pvp.CmdSystemMessage)
	eb := pa.expect(pvp.CmdEndBattle)
	if !wire.IsSecure(eb) {
		t.Fatal("end battle md5")
	}
	deadline := time.Now().Add(8 * time.Second)
	for len(fa.seen("pvp/end_battle")) < 2 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	ends := fa.seen("pvp/end_battle")
	if len(ends) != 2 || !strings.Contains(strings.Join(ends, "\n"), `"Result":1`) {
		t.Fatalf("end battle API calls %v", ends)
	}

	// A wrong key is refused on the PvP server.
	bad := dial(t, pvpSrv.ListenAddr())
	bad.send(pvp.CmdBattleLogin, body(func(w *wire.Writer) {
		w.U32(1)
		w.Bool(true)
		w.UTF("wrong")
		w.U16(0)
		w.UTF("h")
		w.U32(1)
		w.U32(battle)
		w.U8(1)
	}), false)
	if r := bad.expect(pvp.CmdLoginError); binary.BigEndian.Uint32(r) != 1 {
		t.Fatalf("login error %x", r)
	}

	// A bad header closes the main connection; the session's cache entry goes with it.
	before := Cache.Len()
	a.c.Write([]byte{0, 0, 0, 0, 0, 0, 0})
	a.c.SetReadDeadline(time.Now().Add(5 * time.Second))
	if _, err := io.ReadAll(a.c); err != nil {
		t.Fatalf("expected EOF, got %v", err)
	}
	for (Cache.Len() != before-1 || pvp.Find(1) != nil) && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if Cache.Len() != before-1 {
		t.Fatalf("cache %d -> %d", before, Cache.Len())
	}
	if pvp.Find(1) != nil {
		t.Fatal("user 1 still online")
	}
	for _, c := range []*client{b, pa, bad} {
		c.c.Close()
	}
}

// friendsInLogin reads the friends list (ids, in order) and OnlineFriends from an EnterAccount body.
func friendsInLogin(t *testing.T, b []byte) (ids []uint32, online uint16) {
	t.Helper()
	r := wire.NewReader(b)
	r.BytesLenient(int(r.U16())) // own profile
	for n := r.U16(); n > 0; n-- {
		p := wire.NewReader(r.BytesLenient(int(r.U16())))
		ids = append(ids, p.U32())
	}
	r.BytesLenient(int(r.U16())) // daily bonus
	return ids, r.U16()
}

func TestOnlineFriendsFirst(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc((&fakeAPI{}).handler))
	defer srv.Close()
	api.BaseURL = srv.URL + "/api/internal/"
	mainSrv := New("main", "127.0.0.1:0")
	if err := mainSrv.Start(); err != nil {
		t.Fatal(err)
	}
	defer mainSrv.Close()
	login := func(id uint32) (*client, []uint32, uint16) {
		c := dial(t, mainSrv.ListenAddr())
		c.send(proto.CmdLogin, body(func(w *wire.Writer) { w.U32(id); w.U32(0); w.UTF("KEY"); w.U16(0); w.U8(1) }), false)
		ids, online := friendsInLogin(t, c.expect(proto.CmdEnterAccount))
		return c, ids, online
	}
	check := func(who string, ids []uint32, online uint16, wantIDs []uint32, wantOnline uint16) {
		t.Helper()
		if fmt.Sprint(ids) != fmt.Sprint(wantIDs) || online != wantOnline {
			t.Fatalf("%s: friends %v, %d online; want %v, %d online", who, ids, online, wantIDs, wantOnline)
		}
	}

	c3, ids, online := login(3)
	check("first player", ids, online, []uint32{1, 2, 4}, 0)
	c1, ids, online := login(1)
	check("with 3 online", ids, online, []uint32{3, 2, 4}, 1)
	_, ids, online = login(4)
	check("with 1 and 3 online", ids, online, []uint32{1, 3, 2}, 2)

	// Someone who left is offline again.
	c3.c.Close()
	c1.c.Close()
	deadline := time.Now().Add(5 * time.Second)
	for pvp.Find(3) != nil || pvp.Find(1) != nil {
		if time.Now().After(deadline) {
			t.Fatal("players 1 and 3 still online after disconnecting")
		}
		time.Sleep(20 * time.Millisecond)
	}
	_, ids, online = login(2)
	check("after 1 and 3 left", ids, online, []uint32{4, 1, 3}, 1)
}
