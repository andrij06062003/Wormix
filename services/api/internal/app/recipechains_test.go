package app

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/andrij06062003/Wormix/services/api/internal/db"
)

// The sniper rifle (23): 13 makes 313, from which 60 makes 360 and 61 makes 361 (two branches); 138 and
// 139 go on from them. 20 is the bazooka's first upgrade.
var sniperLinks = map[int64]db.RecipeLink{
	13: {Prev: 23, Upgrade: 313}, 60: {Prev: 313, Upgrade: 360}, 61: {Prev: 313, Upgrade: 361},
	138: {Prev: 360, Upgrade: 438}, 139: {Prev: 361, Upgrade: 439}, 20: {Prev: 1, Upgrade: 320},
}

func TestConsistentRecipes(t *testing.T) {
	for _, c := range []struct{ crafted, keep, drop []int64 }{
		{[]int64{13, 61, 139}, []int64{13, 61, 139}, nil},
		// both branches: the last crafted one (139, on 61) stays
		{[]int64{13, 60, 61, 139}, []int64{13, 61, 139}, []int64{60}},
		{[]int64{13, 61, 60, 138}, []int64{13, 60, 138}, []int64{61}},
		// a recipe whose upgrade before it is missing
		{[]int64{20, 61}, []int64{20}, []int64{61}},
		// unknown recipes are kept, a repeated one dropped
		{[]int64{999, 20, 20}, []int64{999, 20}, []int64{20}},
	} {
		keep, drop := db.ConsistentRecipes(c.crafted, sniperLinks)
		if !reflect.DeepEqual(keep, c.keep) || !reflect.DeepEqual(drop, c.drop) {
			t.Errorf("%v: keep %v drop %v, want %v %v", c.crafted, keep, drop, c.keep, c.drop)
		}
	}
}

// The other branch of an upgrade is refused (the client shows one branch per weapon), and so is un-crafting
// a recipe a crafted one starts from.
func TestUpgradeBranches161(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	h := map[string]string{"X-TCP-SESSION": alice.Tcp, "X-SESSION-KEY": alice.SessionKey}
	d.Exec(`update wormix_worms_data set level = 30 where owner_id = 1`)
	d.Exec(`update wormix_user_profiles set real_money = 100000, recipes = '[13,60]' where user_id = 1`)
	d.Exec(`delete from wormix_users_weapons where owner_id = 1 and weapon_id = 23`)
	d.Exec(`insert into wormix_users_weapons (owner_id, weapon_id, count) values (1, 23, -1)`)
	result := func(route string, id int) string {
		t.Helper()
		st, j := call(t, srv, "POST", "/api/internal/craft/"+route, map[string]any{"RecipeId": id}, h)
		if st != 200 {
			t.Fatalf("%s %d: %d %v", route, id, st, j)
		}
		return fmt.Sprint(j["data"].(map[string]any)["Result"])
	}
	if r := result("upgrade_weapon", 61); r != "1" {
		t.Errorf("the other branch (61): result %s, want 1", r)
	}
	if r := result("upgrade_weapon", 138); r != "0" {
		t.Errorf("on along the branch (138): result %s, want 0", r)
	}
	if r := result("downgrade_weapon", 60); r != "1" {
		t.Errorf("un-crafting 60 under 138: result %s, want 1", r)
	}
	if r := result("downgrade_weapon", 138); r != "0" {
		t.Errorf("un-crafting the top one (138): result %s, want 0", r)
	}
	var recipes string
	d.QueryRow(`select recipes from wormix_user_profiles where user_id = 1`).Scan(&recipes)
	if recipes != "[13,60]" {
		t.Errorf("recipes %s, want [13,60]", recipes)
	}
}

// The migration keeps one branch of each weapon and gives back the reagents of the recipes it drops.
func TestMigrateRecipeBranches(t *testing.T) {
	_, d, _ := newServer(t)
	d.Exec(`update wormix_user_profiles set recipes = '[13,60,61,139]', reagents = '[]' where user_id = 1`)
	var raw string
	d.QueryRow(`select reagents from wormix_craft where id = 60`).Scan(&raw)
	var cost [][]int64
	if err := json.Unmarshal([]byte(raw), &cost); err != nil || len(cost) == 0 {
		t.Fatalf("recipe 60 reagents %q: %v", raw, err)
	}
	tx, err := d.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.MigrateRecipeBranches(context.Background(), tx); err != nil {
		t.Fatal(err)
	}
	tx.Commit()
	var recipes, reagents string
	d.QueryRow(`select recipes, reagents from wormix_user_profiles where user_id = 1`).Scan(&recipes, &reagents)
	if recipes != "[13,61,139]" {
		t.Errorf("recipes %s, want [13,61,139]", recipes)
	}
	var have []int64
	if err := json.Unmarshal([]byte(reagents), &have); err != nil {
		t.Fatalf("reagents %q: %v", reagents, err)
	}
	for _, r := range cost {
		if int64(len(have)) <= r[0] || have[r[0]] != r[1] {
			t.Errorf("reagent %d: %v, want %d back", r[0], reagents, r[1])
		}
	}
}
