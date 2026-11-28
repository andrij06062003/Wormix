package app

import (
	"fmt"
	"testing"
)

// game/end_battle answers the awards it gave (the 1.61.7 client shows and adds only these): a boss won (1)
// lists its first-win money, experience and weapons; a lost one nothing.
func TestEndBattleAwards(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	h := map[string]string{"X-TCP-SESSION": alice.Tcp, "X-SESSION-KEY": alice.SessionKey}
	d.Exec(`update wormix_worms_data set level = 10 where owner_id = 1`)
	d.Exec(`update wormix_users_battle_info set battles_count = 5, last_mission_id = 0 where user_id = 1`)
	fight := func(mission, result int) []any {
		t.Helper()
		st, j := call(t, srv, "POST", "/api/internal/game/start_battle", map[string]any{"MissionId": mission}, h)
		if st != 200 {
			t.Fatalf("start %d: %d %v", mission, st, j)
		}
		bid := int(j["data"].(map[string]any)["BattleId"].(float64))
		st, j = call(t, srv, "POST", "/api/internal/game/end_battle", map[string]any{"Result": bid + result, "Type": 0, "ExpBonus": 0,
			"BattleId": bid, "MissionId": mission, "Items": []any{}, "CollectedReagents": []any{}}, h)
		if st != 200 || fmt.Sprint(j["Status"]) != "OK" {
			t.Fatalf("end %d: %d %v", mission, st, j)
		}
		awards, _ := j["Awards"].([]any)
		return awards
	}
	if a := fight(1, -1); len(a) != 0 {
		t.Errorf("boss lost: awards %v", a)
	}
	got := map[string]bool{}
	for _, a := range fight(1, 1) {
		m := a.(map[string]any)
		got[fmt.Sprintf("%v/%v/%v", m["Kind"], m["Count"], m["ItemId"])] = true
	}
	// Mission 1's first win: 90 money, 50 experience and its weapons (missions_awards.json).
	for _, want := range []string{"1/90/0", "11/50/0"} {
		if !got[want] {
			t.Errorf("boss won: no award %s in %v", want, got)
		}
	}
	weapons := 0
	for k := range got {
		if k[:2] == "5/" {
			weapons++
		}
	}
	if weapons == 0 {
		t.Errorf("boss won: no weapon awards in %v", got)
	}
}

// The 1.61.7 bosses (bosses161): a 1.61.7 client's requests ("Client": 161) get its bosses, 28-31 included
// (wormix_missions has the 1.25.0 ones, 1-27), and the awards it shows: the Devotee's (31) first win is 10
// rubies, 190 experience, weapons and its hat 1109 (a hat, so not in the client's list), a win after that 190
// money and 95 experience.
func TestBosses161(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	h := map[string]string{"X-TCP-SESSION": alice.Tcp, "X-SESSION-KEY": alice.SessionKey}
	d.Exec(`update wormix_worms_data set level = 26 where owner_id = 1`)
	d.Exec(`update wormix_users_battle_info set battles_count = 5, last_mission_id = 29 where user_id = 1`)
	d.Exec(`update wormix_user_profiles set real_money = 100 where user_id = 1`)
	// Without the marker (1.25.0) boss 31 doesn't exist; a 1.61.7 client unlocks it for 10 rubies.
	if st, _ := call(t, srv, "POST", "/api/internal/account/buy/mission", map[string]any{"MissionId": 31}, h); st == 200 {
		t.Errorf("1.25.0 unlock of 31: %d, want refused", st)
	}
	st, j := call(t, srv, "POST", "/api/internal/account/buy/mission", map[string]any{"MissionId": 31, "Client": 161}, h)
	if res := fmt.Sprint(j["data"].(map[string]any)["Result"]); st != 200 || res != "0" {
		t.Fatalf("unlock 31: %d %v", st, j)
	}
	win := func() map[string]bool {
		t.Helper()
		st, j := call(t, srv, "POST", "/api/internal/game/start_battle", map[string]any{"MissionId": 31, "Client": 161}, h)
		if st != 200 {
			t.Fatalf("start 31: %d %v", st, j)
		}
		bid := int(j["data"].(map[string]any)["BattleId"].(float64))
		st, j = call(t, srv, "POST", "/api/internal/game/end_battle", map[string]any{"Result": bid + 1, "Type": 0, "ExpBonus": 0,
			"BattleId": bid, "MissionId": 31, "Items": []any{}, "CollectedReagents": []any{}, "Client": 161}, h)
		if st != 200 || fmt.Sprint(j["Status"]) != "OK" {
			t.Fatalf("end 31: %d %v", st, j)
		}
		got := map[string]bool{}
		for _, a := range j["Awards"].([]any) {
			m := a.(map[string]any)
			got[fmt.Sprintf("%v/%v/%v", m["Kind"], m["Count"], m["ItemId"])] = true
		}
		return got
	}
	first := win()
	for _, want := range []string{"2/10/0", "11/190/0", "5/6/72", "5/4/130"} {
		if !first[want] {
			t.Errorf("first win: no award %s in %v", want, first)
		}
	}
	var hat int
	if err := d.QueryRow(`select count(*) from wormix_users_weapons where owner_id = 1 and weapon_id = 1109`).Scan(&hat); err != nil || hat != 1 {
		t.Errorf("first win: hat 1109 not given (%d, %v)", hat, err)
	}
	if again := win(); !again["1/190/0"] || !again["11/95/0"] {
		t.Errorf("second win: %v", again)
	}
}
