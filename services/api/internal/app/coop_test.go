package app

import (
	"fmt"
	"testing"
	"time"
)

// A co-op boss of the 1.25.0 client (101+): open from the required level, one at a time; a win gives the
// first-win awards and moves the co-op progress on, a repeat gives the repeat awards, a cancelled battle
// gives the battle back.
func TestCoopBoss125(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	h := map[string]string{"X-TCP-SESSION": alice.Tcp, "X-SESSION-KEY": alice.SessionKey}
	now := time.Now().Unix()
	d.Exec(`update wormix_users_battle_info set battles_count = 3, last_battle_time = ? where user_id = 1`, now)
	d.Exec(`update wormix_user_profiles set money = 1000, real_money = 10 where user_id = 1`)
	d.Exec(`update wormix_worms_data set level = 10 where owner_id = 1`)

	state := func(mid int) string {
		t.Helper()
		st, j := call(t, srv, "POST", "/api/internal/pvp/coop_state", map[string]any{"MissionId": mid}, h)
		if st != 200 {
			t.Fatalf("coop_state: %d %v", st, j)
		}
		return fmt.Sprint(j["data"].(map[string]any)["State"])
	}
	if s := state(101); s != "2" {
		t.Fatalf("level 10: state %s, want 2 (locked)", s)
	}
	d.Exec(`update wormix_worms_data set level = 12 where owner_id = 1`)
	if s := state(101); s != "0" {
		t.Fatalf("level 12: state %s, want 0", s)
	}
	if s := state(102); s != "2" {
		t.Fatalf("second boss before the first: state %s, want 2", s)
	}

	start := func(battle, mid int) int {
		t.Helper()
		st, _ := call(t, srv, "POST", "/api/internal/pvp/start_battle", map[string]any{"BattleId": battle, "Wager": 0, "Reagents": []int{1, 2}, "MissionId": mid}, h)
		return st
	}
	end := func(battle, result int) string {
		t.Helper()
		st, j := call(t, srv, "POST", "/api/internal/pvp/end_battle", map[string]any{"BattleId": battle, "Result": result}, h)
		if st != 200 {
			t.Fatalf("end_battle: %d %v", st, j)
		}
		return fmt.Sprint(j["Status"])
	}
	var battles, last, money, real int64
	read := func() {
		d.QueryRow(`select battles_count, last_coop_mission_id from wormix_users_battle_info where user_id = 1`).Scan(&battles, &last)
		d.QueryRow(`select money, real_money from wormix_user_profiles where user_id = 1`).Scan(&money, &real)
	}

	if st := start(501, 102); st != 422 {
		t.Fatalf("start of a locked boss: %d", st)
	}
	if st := start(501, 101); st != 200 {
		t.Fatalf("start: %d", st)
	}
	read()
	if battles != 2 {
		t.Fatalf("battles %d after the start, want 2", battles)
	}
	if s := end(501, 1); s != "OK" {
		t.Fatalf("end: %s", s)
	}
	if s := end(501, 1); s != "IGNORED" {
		t.Fatalf("second end: %s", s)
	}
	read()
	if last != 101 || money != 1000 || real != 11 {
		t.Fatalf("first win: last %d, money %d, real money %d; want 101, 1000, 11", last, money, real)
	}
	var hat int
	d.QueryRow(`select count(*) from wormix_users_weapons where owner_id = 1 and weapon_id = 1068`).Scan(&hat)
	if hat != 1 {
		t.Fatalf("boss hat not given")
	}

	// A repeat: the repeat awards (80 fuzy), the progress stays.
	if st := start(502, 101); st != 200 {
		t.Fatalf("repeat start: %d", st)
	}
	end(502, 1)
	read()
	if last != 101 || money != 1080 || real != 11 {
		t.Fatalf("repeat: last %d, money %d, real money %d; want 101, 1080, 11", last, money, real)
	}

	// Cancelled: the battle comes back.
	start(503, 102)
	read()
	before := battles
	end(503, 0)
	read()
	if battles != before+1 || last != 101 {
		t.Fatalf("cancelled: battles %d (was %d), last %d", battles, before, last)
	}

	st, j := call(t, srv, "POST", "/api/internal/game/get_arena", map[string]any{"ReturnUsersProfiles": false}, h)
	if st != 200 || fmt.Sprint(j["data"].(map[string]any)["CurrentCoopMission"]) != "101" {
		t.Fatalf("get_arena: %d %v", st, j)
	}

	// No battles left.
	d.Exec(`update wormix_users_battle_info set battles_count = 0, last_battle_time = ? where user_id = 1`, now)
	if s := state(101); s != "1" {
		t.Fatalf("no battles: state %s, want 1", s)
	}
}
