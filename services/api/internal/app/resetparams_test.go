package app

import (
	"fmt"
	"testing"
)

// The attack and armor reset is free below level 10 (as the 1.61.7 client shows it), 300 money from it.
func TestResetParameters(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	h := map[string]string{"X-TCP-SESSION": alice.Tcp, "X-SESSION-KEY": alice.SessionKey}
	reset := func() string {
		t.Helper()
		st, j := call(t, srv, "POST", "/api/internal/account/reset/parameters", map[string]any{"MoneyType": 1}, h)
		if st != 200 {
			t.Fatalf("reset: %d %v", st, j)
		}
		return fmt.Sprint(j["data"].(map[string]any)["Result"])
	}
	state := func() (money, attack, armor int64) {
		d.QueryRow(`select money from wormix_user_profiles where user_id = 1`).Scan(&money)
		d.QueryRow(`select attack, armor from wormix_worms_data where owner_id = 1`).Scan(&attack, &armor)
		return
	}
	d.Exec(`update wormix_user_profiles set money = 50 where user_id = 1`)
	d.Exec(`update wormix_worms_data set level = 5, attack = 3, armor = 2 where owner_id = 1`)
	if r := reset(); r != "0" {
		t.Fatalf("level 5 with 50 money: result %s, want 0 (free)", r)
	}
	if m, at, ar := state(); m != 50 || at != 0 || ar != 0 {
		t.Fatalf("level 5: money %d attack %d armor %d", m, at, ar)
	}
	d.Exec(`update wormix_worms_data set level = 10, attack = 3 where owner_id = 1`)
	if r := reset(); r != "3" {
		t.Fatalf("level 10 with 50 money: result %s, want 3", r)
	}
	d.Exec(`update wormix_user_profiles set money = 400 where user_id = 1`)
	if r := reset(); r != "0" {
		t.Fatalf("level 10 with 400 money: result %s", r)
	}
	if m, at, _ := state(); m != 100 || at != 0 {
		t.Fatalf("level 10: money %d attack %d", m, at)
	}
}
