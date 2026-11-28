package app

import "testing"

// A wager battle won pays the wager back and the award its start gave (a 1.61.7 wager's, as its client shows
// it), or 80% of the wager without one (the 1.25.0 wagers).
func TestWagerAward(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	h := map[string]string{"X-TCP-SESSION": alice.Tcp, "X-SESSION-KEY": alice.SessionKey}
	for i, c := range []struct {
		award any
		want  int64
	}{{120, 1000 + 120}, {nil, 1000 + 56}} {
		d.Exec(`update wormix_user_profiles set money = 1000 where user_id = 1`)
		battle := 900 + i
		body := map[string]any{"BattleId": battle, "Wager": 70, "Reagents": []int{}}
		if c.award != nil {
			body["Award"] = c.award
		}
		if st, j := call(t, srv, "POST", "/api/internal/pvp/start_battle", body, h); st != 200 {
			t.Fatalf("start_battle: %d %v", st, j)
		}
		if st, j := call(t, srv, "POST", "/api/internal/pvp/end_battle", map[string]any{"BattleId": battle, "Result": 1}, h); st != 200 {
			t.Fatalf("end_battle: %d %v", st, j)
		}
		var money int64
		d.QueryRow(`select money from wormix_user_profiles where user_id = 1`).Scan(&money)
		if money != c.want {
			t.Errorf("award %v: money %d, want %d", c.award, money, c.want)
		}
	}
}
