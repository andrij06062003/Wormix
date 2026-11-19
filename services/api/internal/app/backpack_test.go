package app

import (
	"fmt"
	"testing"
)

// The arsenal's layouts and hotkeys are kept as the client sends them; ones it can't keep are refused.
func TestBackpack(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	h := map[string]string{"X-TCP-SESSION": alice.Tcp, "X-SESSION-KEY": alice.SessionKey}
	post := func(route string, body map[string]any) string {
		t.Helper()
		st, j := call(t, srv, "POST", "/api/internal/"+route, body, h)
		if st != 200 {
			t.Fatalf("%s: %d %v", route, st, j)
		}
		return fmt.Sprint(j["data"].(map[string]any)["Result"])
	}
	if r := post("account/backpack", map[string]any{"Configs": "[[1,2,-3],[],[32767]]", "Active": 2}); r != "0" {
		t.Fatalf("backpack: result %s", r)
	}
	if r := post("account/hotkeys", map[string]any{"Hotkeys": "[5,6]"}); r != "0" {
		t.Fatalf("hotkeys: result %s", r)
	}
	var confs, keys string
	var active int
	d.QueryRow(`select backpack_confs, active_backpack_conf, hotkeys from wormix_user_profiles where user_id = 1`).Scan(&confs, &active, &keys)
	if confs != "[[1,2,-3],[],[32767]]" || active != 2 || keys != "[5,6]" {
		t.Fatalf("saved %s %d %s", confs, active, keys)
	}
	for _, bad := range []string{"[[40000]]", "{}", "null", "[1]"} {
		if r := post("account/backpack", map[string]any{"Configs": bad, "Active": 0}); r != "1" {
			t.Errorf("backpack %s: result %s, want 1", bad, r)
		}
	}
	d.QueryRow(`select backpack_confs from wormix_user_profiles where user_id = 1`).Scan(&confs)
	if confs != "[[1,2,-3],[],[32767]]" {
		t.Errorf("a refused layout changed the saved one: %s", confs)
	}
}

// A layout with two cells of one weapon family (the bazooka 1 and its upgrades 320, 321, saved while the client
// split the family) keeps the first: saved so, and given so at login for layouts saved before.
func TestBackpackOneCellPerFamily(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	h := map[string]string{"X-TCP-SESSION": alice.Tcp, "X-SESSION-KEY": alice.SessionKey}
	if st, j := call(t, srv, "POST", "/api/internal/account/backpack", map[string]any{"Configs": "[[320,2,1,321],[1]]", "Active": 0}, h); st != 200 {
		t.Fatalf("backpack: %d %v", st, j)
	}
	var confs string
	d.QueryRow(`select backpack_confs from wormix_user_profiles where user_id = 1`).Scan(&confs)
	if confs != "[[320,2,0,0],[1]]" {
		t.Errorf("saved %s, want [[320,2,0,0],[1]]", confs)
	}
	base := map[int64]int64{320: 1, 321: 1}
	if got := oneCellPerFamily("[[1,0,321,-3]]", base); got != "[[1,0,0,-3]]" {
		t.Errorf("login: %s", got)
	}
	if got := oneCellPerFamily("[[1,2]]", base); got != "[[1,2]]" {
		t.Errorf("unchanged layout: %s", got)
	}
}
