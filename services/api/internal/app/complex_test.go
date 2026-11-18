package app

import (
	"context"
	"fmt"
	"testing"

	"github.com/andrij06062003/Wormix/services/api/internal/db"
)

// A complex weapon (the sniper rifle) bought by the level stays the player's: levels add up to the top one
// (-1), and a battle that uses it takes nothing, so it stays in the arsenal.
func TestComplexWeaponLevels(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	h := map[string]string{"X-TCP-SESSION": alice.Tcp, "X-SESSION-KEY": alice.SessionKey}
	d.Exec(`update wormix_user_profiles set real_money = 1000, money = 100000 where user_id = 1`)
	d.Exec(`update wormix_worms_data set level = 30 where owner_id = 1`)
	d.Exec(`update wormix_users_battle_info set battles_count = 5 where user_id = 1`)
	d.Exec(`delete from wormix_users_weapons where owner_id = 1 and weapon_id in (23, 54)`)
	count := func(id int) (n int64) {
		t.Helper()
		if err := d.QueryRow(`select count from wormix_users_weapons where owner_id = 1 and weapon_id = ?`, id).Scan(&n); err != nil {
			t.Fatalf("weapon %d: %v", id, err)
		}
		return n
	}
	buy := func(id, levels int) {
		t.Helper()
		st, j := call(t, srv, "POST", "/api/internal/account/buy/items", map[string]any{"ShopItems": []any{map[string]any{"Id": id, "Count": levels, "MoneyType": 1}}}, h)
		data := j["data"].(map[string]any)
		if st != 200 || fmt.Sprint(data["Result"]) != "0" {
			t.Fatalf("buy %d x%d: %d %v", id, levels, st, j)
		}
		// the answer has the levels bought, which the client adds to the ones it shows
		if w := data["Weapons"].([]any); len(w) != 1 || fmt.Sprint(w[0].(map[string]any)["Count"]) != fmt.Sprint(levels) {
			t.Fatalf("buy %d x%d: answered %v", id, levels, w)
		}
	}
	buy(23, 1)
	if n := count(23); n != -11 {
		t.Fatalf("sniper rifle level 1: count %d, want -11", n)
	}

	// A battle that fired it 3 times.
	st, j := call(t, srv, "POST", "/api/internal/game/start_battle", map[string]any{"MissionId": 0}, h)
	if st != 200 {
		t.Fatalf("start: %d %v", st, j)
	}
	bid, _ := j["data"].(map[string]any)["BattleId"].(float64)
	st, j = call(t, srv, "POST", "/api/internal/game/end_battle", map[string]any{"Result": int(bid) + 1, "Type": 0, "ExpBonus": 0,
		"BattleId": int(bid), "MissionId": 0, "Items": []any{map[string]any{"Id": 23, "Count": 3}}, "CollectedReagents": []any{}}, h)
	if st != 200 || fmt.Sprint(j["Status"]) != "OK" {
		t.Fatalf("end: %d %v", st, j)
	}
	if n := count(23); n != -11 {
		t.Fatalf("after a battle: count %d, want -11", n)
	}

	buy(23, 2)
	if n := count(23); n != -13 {
		t.Fatalf("sniper rifle level 3: count %d, want -13", n)
	}
	buy(23, 5)
	if n := count(23); n != -1 {
		t.Fatalf("sniper rifle past the top: count %d, want -1", n)
	}
}

// The migration gives back the levels battles used up: a positive count is that level (the top one at most),
// 0 level 1; other weapons, and the ones owned for good, keep theirs.
func TestMigrateComplexWeapons(t *testing.T) {
	_, d, _ := newServer(t)
	d.Exec(`delete from wormix_users_weapons where owner_id = 1 and weapon_id in (23, 54, 41, 72)`)
	for _, r := range [][2]int{{23, 0}, {54, 2}, {41, 7}, {72, 5}} {
		d.Exec(`insert into wormix_users_weapons (owner_id, weapon_id, count) values (1, ?, ?)`, r[0], r[1])
	}
	tx, err := d.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.MigrateComplexWeapons(context.Background(), tx); err != nil {
		t.Fatal(err)
	}
	tx.Commit()
	for id, want := range map[int]int64{23: -11, 54: -12, 41: -1, 72: 5} {
		var n int64
		d.QueryRow(`select count from wormix_users_weapons where owner_id = 1 and weapon_id = ?`, id).Scan(&n)
		if n != want {
			t.Errorf("weapon %d: count %d, want %d", id, n, want)
		}
	}
}
