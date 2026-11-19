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
