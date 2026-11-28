package app

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/andrij06062003/Wormix/services/api/internal/assets"
)

// Every weapon upgrade can be made once its base weapon and the upgrades before it are owned: the base weapons
// new in the 1.61.7 client exist, and the recipes chain up.
func TestUpgradeEveryRecipe161(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	h := map[string]string{"X-TCP-SESSION": alice.Tcp, "X-SESSION-KEY": alice.SessionKey}
	d.Exec(`update wormix_worms_data set level = 30 where owner_id = 1`)

	type recipe struct{ id, prev, upgrade int64 }
	var all []recipe
	byUpgrade := map[int64]recipe{}
	rows, err := d.Query(`select id, prev_upgrade_id, upgrade_id from wormix_craft order by id`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var r recipe
		rows.Scan(&r.id, &r.prev, &r.upgrade)
		all = append(all, r)
		byUpgrade[r.upgrade] = r
	}
	rows.Close()
	if len(all) != 159 {
		t.Fatalf("recipes %d, want 159", len(all))
	}
	for _, r := range all {
		// The upgrades before it, down to the base weapon.
		before := []int64{} // "[]", not "null": the column is never null
		base := r.prev
		for base > 300 {
			p, ok := byUpgrade[base]
			if !ok {
				t.Fatalf("recipe %d: no recipe makes %d", r.id, base)
			}
			before = append([]int64{p.id}, before...)
			base = p.prev
		}
		var known int
		d.QueryRow(`select count(*) from wormix_weapons where id = ?`, base).Scan(&known)
		if known != 1 {
			t.Errorf("recipe %d: base weapon %d unknown", r.id, base)
			continue
		}
		b, _ := json.Marshal(before)
		d.Exec(`update wormix_user_profiles set real_money = 100000, recipes = ? where user_id = 1`, string(b))
		d.Exec(`delete from wormix_users_weapons where owner_id = 1 and weapon_id = ?`, base)
		d.Exec(`insert into wormix_users_weapons (owner_id, weapon_id, count) values (1, ?, -1)`, base)
		st, j := call(t, srv, "POST", "/api/internal/craft/upgrade_weapon", map[string]any{"RecipeId": r.id}, h)
		if st != 200 || fmt.Sprint(j["data"].(map[string]any)["Result"]) != "0" {
			t.Errorf("recipe %d (base %d, after %v): %d %v", r.id, base, before, st, j)
		}
	}
}

// The 1.61.7 client buys the ice bazooka (a complex weapon: 4 levels, a shot a battle each) by the level: each
// costs its price and is kept as a level (-10-L, complex.go), and upgrading it makes it the top one.
func TestBuyShotsAndUpgrade161(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	h := map[string]string{"X-TCP-SESSION": alice.Tcp, "X-SESSION-KEY": alice.SessionKey}
	d.Exec(`update wormix_user_profiles set real_money = 1000, recipes = '[]' where user_id = 1`)
	d.Exec(`update wormix_worms_data set level = 30 where owner_id = 1`)
	d.Exec(`delete from wormix_users_weapons where owner_id = 1 and weapon_id = 54`)
	buy := func(count int) {
		t.Helper()
		st, j := call(t, srv, "POST", "/api/internal/account/buy/items", map[string]any{"ShopItems": []any{map[string]any{"Id": 54, "Count": count, "MoneyType": 0}}}, h)
		if st != 200 || fmt.Sprint(j["data"].(map[string]any)["Result"]) != "0" {
			t.Fatalf("buy %d: %d %v", count, st, j)
		}
	}
	buy(2)
	buy(1)
	var real, count int64
	d.QueryRow(`select real_money from wormix_user_profiles where user_id = 1`).Scan(&real)
	d.QueryRow(`select count from wormix_users_weapons where owner_id = 1 and weapon_id = 54`).Scan(&count)
	if real != 1000-3*8 || count != -13 {
		t.Fatalf("after buying 3 levels: rubies %d, count %d, want -13", real, count)
	}
	st, j := call(t, srv, "POST", "/api/internal/craft/upgrade_weapon", map[string]any{"RecipeId": 91}, h)
	if st != 200 || fmt.Sprint(j["data"].(map[string]any)["Result"]) != "0" {
		t.Fatalf("upgrade 91: %d %v", st, j)
	}
	d.QueryRow(`select count from wormix_users_weapons where owner_id = 1 and weapon_id = 54`).Scan(&count)
	if count != -1 {
		t.Fatalf("after the upgrade: count %d, want -1", count)
	}
	// At its top level there is nothing more to buy: refused, and it costs nothing.
	d.QueryRow(`select real_money from wormix_user_profiles where user_id = 1`).Scan(&real)
	st, j = call(t, srv, "POST", "/api/internal/account/buy/items", map[string]any{"ShopItems": []any{map[string]any{"Id": 54, "Count": 1, "MoneyType": 0}}}, h)
	if fmt.Sprint(j["data"].(map[string]any)["Result"]) == "0" {
		t.Fatalf("a level past the top: %d %v", st, j)
	}
	var after int64
	d.QueryRow(`select real_money from wormix_user_profiles where user_id = 1`).Scan(&after)
	d.QueryRow(`select count from wormix_users_weapons where owner_id = 1 and weapon_id = 54`).Scan(&count)
	if after != real || count != -1 {
		t.Fatalf("a level past the top: rubies %d -> %d, count %d", real, after, count)
	}
}

// Every item the 1.61.7 shop sells can be bought, at its price.
func TestBuyEveryItem161(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	h := map[string]string{"X-TCP-SESSION": alice.Tcp, "X-SESSION-KEY": alice.SessionKey}
	var shop struct {
		Items []struct {
			ID        int64 `json:"id"`
			Price     int64 `json:"price"`
			RealPrice int64 `json:"realprice"`
			Hidden    bool  `json:"hideInShop"`
			Infinite  bool  `json:"infinite"`
		} `json:"items"`
	}
	b, err := assets.Game("shop161.json")
	if err != nil || json.Unmarshal(b, &shop) != nil {
		t.Fatalf("shop161.json: %v", err)
	}
	sold := 0
	for _, it := range shop.Items {
		if it.Hidden {
			continue
		}
		count := 1
		if it.ID >= stuffStartIndex || it.Infinite {
			count = -1
		}
		for moneyType, price := range []int64{it.RealPrice, it.Price} {
			if price == 0 {
				continue
			}
			sold++
			d.Exec(`update wormix_user_profiles set money = 1000000, real_money = 1000000 where user_id = 1`)
			d.Exec(`delete from wormix_users_weapons where owner_id = 1 and weapon_id = ?`, it.ID)
			st, j := call(t, srv, "POST", "/api/internal/account/buy/items", map[string]any{"ShopItems": []any{map[string]any{"Id": it.ID, "Count": count, "MoneyType": moneyType}}}, h)
			if st != 200 || fmt.Sprint(j["data"].(map[string]any)["Result"]) != "0" {
				t.Errorf("buy %d (money type %d): %d %v", it.ID, moneyType, st, j)
				continue
			}
			var money, real int64
			d.QueryRow(`select money, real_money from wormix_user_profiles where user_id = 1`).Scan(&money, &real)
			if spent := []int64{1000000 - real, 1000000 - money}[moneyType]; spent != price {
				t.Errorf("buy %d (money type %d): spent %d, want %d", it.ID, moneyType, spent, price)
			}
		}
	}
	if sold < 200 {
		t.Fatalf("only %d purchases tried", sold)
	}
}
