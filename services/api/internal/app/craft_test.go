package app

import (
	"context"
	"fmt"
	"testing"

	"github.com/andrij06062003/Wormix/services/api/internal/php"
)

func TestCraftedHatPacking(t *testing.T) {
	for _, c := range [][2]int64{{2, 0}, {2, 1005}, {3, 1499}, {2, 1501}, {5, 1647}, {0, 1500}} {
		race, hat := c[0], c[1]
		packed := hatByRaceAndHat(hat, race)
		if r, h := raceAndHat(packed); r != race || h != hat || packed > 32767 {
			t.Fatalf("race %d hat %d: packed %d, back %d %d", race, hat, packed, r, h)
		}
	}
}

func TestAssembleEquip(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	h := map[string]string{"X-TCP-SESSION": alice.Tcp, "X-SESSION-KEY": alice.SessionKey}
	assemble := func(family, moneyType int) (string, int64) {
		t.Helper()
		st, j := call(t, srv, "POST", "/api/internal/craft/assemble_equip", map[string]any{"FamilyId": family, "MoneyType": moneyType}, h)
		if st != 200 {
			t.Fatalf("assemble_equip: %d %v", st, j)
		}
		data := j["data"].(map[string]any)
		return fmt.Sprint(data["Result"]), int64(data["EquipId"].(float64))
	}
	owns := func(id int64) bool {
		var n int
		d.QueryRow(`select count(*) from wormix_users_weapons where owner_id = 1 and weapon_id = ?`, id).Scan(&n)
		return n > 0
	}
	money := func() (int64, int64) {
		var m, r int64
		d.QueryRow(`select money, real_money from wormix_user_profiles where user_id = 1`).Scan(&m, &r)
		return m, r
	}
	defer func(r func(int64) int64) { craftRoll = r }(craftRoll)
	craftRoll = func(int64) int64 { return 0 } // the first prefix

	// Jawblade (2540): artifacts 2021 and 2012 and 9 badges, 2000 fuzy or 20 rubies.
	d.Exec(`update wormix_user_profiles set money = 0, real_money = 1000, reagents = '[]' where user_id = 1`)
	if r, _ := assemble(2540, 0); r != "1" {
		t.Fatalf("without the items: result %s, want 1", r)
	}
	d.Exec(`insert into wormix_users_weapons (owner_id, weapon_id) values (1, 2021), (1, 2012)`)
	d.Exec(`update wormix_worms_data set artifact = 2021 where owner_id = 1`)
	if r, _ := assemble(2540, 1); r != "4" {
		t.Fatalf("fuzy without badges: result %s, want 4", r)
	}
	r, id := assemble(2540, 0)
	if r != "0" || id != 2541 || !owns(2541) || owns(2021) || owns(2012) {
		t.Fatalf("craft: result %s id %d", r, id)
	}
	if _, real := money(); real != 1000-20-9*5 {
		t.Fatalf("rubies %d, want %d (20 + 9 badges at 5)", real, 1000-20-45)
	}
	var artifact int64
	d.QueryRow(`select artifact from wormix_worms_data where owner_id = 1`).Scan(&artifact)
	if artifact != 2541 {
		t.Fatalf("worn artifact %d, want 2541", artifact)
	}

	// Remake: 70 rubies + 10 badges (the player has 4), the old one is replaced.
	d.Exec(`update wormix_user_profiles set real_money = 1000, reagents = ? where user_id = 1`, reagentsJSON(50, 4))
	craftRoll = func(n int64) int64 { return n - 1 } // the best prefix
	r, id = assemble(2540, 0)
	if r != "0" || id != 2546 || owns(2541) || !owns(2546) {
		t.Fatalf("remake: result %s id %d", r, id)
	}
	if _, real := money(); real != 1000-70-6*5 {
		t.Fatalf("rubies after remake %d, want %d", real, 1000-70-30)
	}
	if r, _ := assemble(2540, 0); r != "1" {
		t.Fatalf("remake of the best: result %s, want 1", r)
	}

	// Samurai helm (1500), a hat: hat 1011 and 2 badges, paid with fuzy.
	d.Exec(`insert into wormix_users_weapons (owner_id, weapon_id) values (1, 1011)`)
	d.Exec(`update wormix_user_profiles set money = 800, reagents = ? where user_id = 1`, reagentsJSON(50, 2))
	r, id = assemble(1500, 1)
	if m, _ := money(); r != "0" || id != 1505 || m != 0 || owns(1011) {
		t.Fatalf("hat: result %s id %d money %d", r, id, m)
	}
	var hat int64
	d.QueryRow(`select hat from wormix_worms_data where owner_id = 1`).Scan(&hat)
	if _, h := raceAndHat(hat); h != 1505 {
		t.Fatalf("worn hat %d (%d), want 1505", h, hat)
	}
}

// reagentsJSON is a reagents column with count n of reagent id.
func reagentsJSON(id, n int) string {
	s := "["
	for i := 0; i <= id; i++ {
		if i > 0 {
			s += ","
		}
		if i == id {
			s += fmt.Sprint(n)
		} else {
			s += "0"
		}
	}
	return s + "]"
}

// The 1.25.0 client's shop sells hats the 1.05.0 one didn't have, some of them craft ingredients:
// Gladiator helmet (1064) is one of the Nordic helm's.
func TestBuyHat125(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	h := map[string]string{"X-TCP-SESSION": alice.Tcp, "X-SESSION-KEY": alice.SessionKey}
	d.Exec(`update wormix_user_profiles set money = 0, real_money = 100 where user_id = 1`)
	buy := func(id int64) string {
		t.Helper()
		st, j := call(t, srv, "POST", "/api/internal/account/buy/items", map[string]any{"ShopItems": []any{map[string]any{"Id": id, "Count": -1, "MoneyType": 0}}}, h)
		if st != 200 {
			t.Fatalf("buy %d: %d %v", id, st, j)
		}
		return fmt.Sprint(j["data"].(map[string]any)["Result"])
	}
	if r := buy(1064); r != "0" {
		t.Fatalf("buy 1064: result %s, want 0", r)
	}
	var n, real int64
	d.QueryRow(`select count(*) from wormix_users_weapons where owner_id = 1 and weapon_id = 1064`).Scan(&n)
	d.QueryRow(`select real_money from wormix_user_profiles where user_id = 1`).Scan(&real)
	if n != 1 || real != 100-42 {
		t.Fatalf("after buying 1064: owned %d, rubies %d", n, real)
	}
	// The Ice Shard is sold in the 1.25.0 shop.
	if r := buy(2019); r != "0" {
		t.Fatalf("buy 2019: result %s, want 0", r)
	}
	// A boss hat is hidden in the shop.
	if r := buy(1096); r != "1" {
		t.Fatalf("buy 1096: result %s, want 1", r)
	}
}

// The 1.25.0 client's award tiers have more items: the atomic bomb (77) is in the fourth.
func TestChooseBonus125(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	choose := func(item int) string {
		t.Helper()
		st, j := call(t, srv, "POST", "/api/internal/achievements/choose_bonus", map[string]any{"session_id": alice.AchSession, "item_id": item}, map[string]string{"X-TCP-SESSION": "achievements"})
		if st != 200 {
			t.Fatalf("choose_bonus %d: %d %v", item, st, j)
		}
		return fmt.Sprint(j["result"])
	}
	// Every achievement at its top level: plenty of award points.
	cfg := achievementsConfig()
	for _, id := range cfg.achievements.Keys() {
		levels := cfg.levels(id)
		top := levels[len(levels)-1].(*php.Arr).Val("max")
		d.Exec(`insert into wormix_users_achievements (user_id, achievement_id, value) values (1, ?, ?)
			on conflict (user_id, achievement_id) do update set value = excluded.value`, id, top)
	}
	d.Exec(`delete from wormix_users_weapons where owner_id = 1 and weapon_id in (77, 78, 2022)`)
	d.Exec(`update wormix_user_profiles set invested_award_points = 2 where user_id = 1`)
	if r := choose(77); r != "2" {
		t.Fatalf("atomic bomb before the fourth tier: result %s, want 2", r)
	}
	if r := choose(78); r != "0" {
		t.Fatalf("third tier 78: result %s, want 0", r)
	}
	if r := choose(77); r != "0" {
		t.Fatalf("atomic bomb: result %s, want 0", r)
	}
}

// The 1.25.0 wardrobe puts a hat on and takes it off (StuffId 0); the race stays.
func TestSelectHat125(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	h := map[string]string{"X-TCP-SESSION": alice.Tcp, "X-SESSION-KEY": alice.SessionKey}
	d.Exec(`update wormix_user_profiles set real_money = 100 where user_id = 1`)
	if st, j := call(t, srv, "POST", "/api/internal/account/buy/items", map[string]any{"ShopItems": []any{map[string]any{"Id": 1064, "Count": -1, "MoneyType": 0}}}, h); st != 200 {
		t.Fatalf("buy: %d %v", st, j)
	}
	d.Exec(`update wormix_worms_data set hat = 2 where owner_id = 1`)
	sel := func(id int) int64 {
		t.Helper()
		st, j := call(t, srv, "POST", "/api/internal/account/select_stuff", map[string]any{"StuffId": id}, h)
		if st != 200 || fmt.Sprint(j["data"].(map[string]any)["Result"]) != "0" {
			t.Fatalf("select %d: %d %v", id, st, j)
		}
		var hat int64
		d.QueryRow(`select hat from wormix_worms_data where owner_id = 1`).Scan(&hat)
		return hat
	}
	if hat := sel(1064); hat != hatByRaceAndHat(1064, 2) {
		t.Fatalf("hat on: %d", hat)
	}
	if hat := sel(0); hat != 2 {
		t.Fatalf("hat off: %d", hat)
	}
}

// The weapon upgrades new in 1.25.0 (here the sheep's) can be made.
func TestUpgradeSheep125(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	h := map[string]string{"X-TCP-SESSION": alice.Tcp, "X-SESSION-KEY": alice.SessionKey}
	d.Exec(`update wormix_user_profiles set real_money = 1000, recipes = '[]' where user_id = 1`)
	d.Exec(`update wormix_worms_data set level = 30 where owner_id = 1`)
	d.Exec(`delete from wormix_users_weapons where owner_id = 1 and weapon_id = 40`)
	d.Exec(`insert into wormix_users_weapons (owner_id, weapon_id, count) values (1, 40, -1)`)
	st, j := call(t, srv, "POST", "/api/internal/craft/upgrade_weapon", map[string]any{"RecipeId": 47}, h)
	if st != 200 || fmt.Sprint(j["data"].(map[string]any)["Result"]) != "0" {
		t.Fatalf("upgrade: %d %v", st, j)
	}
	var recipes string
	d.QueryRow(`select recipes from wormix_user_profiles where user_id = 1`).Scan(&recipes)
	if recipes != "[47]" {
		t.Fatalf("recipes %s", recipes)
	}
}

// The races new in 1.25.0 (here the dragon) can be bought, and the hat stays on.
func TestBuyDragon125(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	h := map[string]string{"X-TCP-SESSION": alice.Tcp, "X-SESSION-KEY": alice.SessionKey}
	d.Exec(`update wormix_user_profiles set real_money = 100 where user_id = 1`)
	d.Exec(`update wormix_worms_data set level = 30, hat = 1003 where owner_id = 1`)
	st, j := call(t, srv, "POST", "/api/internal/account/buy/race", map[string]any{"RaceId": 7, "MoneyType": 0}, h)
	if st != 200 || fmt.Sprint(j["data"].(map[string]any)["Result"]) != "0" {
		t.Fatalf("buy: %d %v", st, j)
	}
	var hat, real int64
	d.QueryRow(`select hat from wormix_worms_data where owner_id = 1`).Scan(&hat)
	d.QueryRow(`select real_money from wormix_user_profiles where user_id = 1`).Scan(&real)
	if hat != hatByRaceAndHat(1003, 7) || real != 74 {
		t.Fatalf("hat %d, real money %d", hat, real)
	}
}

// The weapons new in the 1.25.0 shop (here the laser gun) can be bought.
func TestBuyWeapon125(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	h := map[string]string{"X-TCP-SESSION": alice.Tcp, "X-SESSION-KEY": alice.SessionKey}
	d.Exec(`update wormix_user_profiles set money = 5000 where user_id = 1`)
	st, j := call(t, srv, "POST", "/api/internal/account/buy/items", map[string]any{"ShopItems": []any{map[string]any{"Id": 94, "Count": -1, "MoneyType": 1}}}, h)
	if st != 200 || fmt.Sprint(j["data"].(map[string]any)["Result"]) != "0" {
		t.Fatalf("buy: %d %v", st, j)
	}
	var money, count int64
	d.QueryRow(`select money from wormix_user_profiles where user_id = 1`).Scan(&money)
	d.QueryRow(`select count from wormix_users_weapons where owner_id = 1 and weapon_id = 94`).Scan(&count)
	if money != 800 || count != -1 {
		t.Fatalf("money %d, count %d", money, count)
	}
}

// The 1.25.0 client leaves a friend of the team out of battles, and it stays so.
func TestToggleTeammate125(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	h := map[string]string{"X-TCP-SESSION": alice.Tcp, "X-SESSION-KEY": alice.SessionKey}
	d.Exec(`insert into wormix_users_teams (user_id, teammate_id, "order") values (1, 2, 1)`)
	toggle := func(id int, active bool) string {
		t.Helper()
		st, j := call(t, srv, "POST", "/api/internal/team/toggle", map[string]any{"ProfileId": id, "Active": active}, h)
		if st != 200 {
			t.Fatalf("toggle %d: %d %v", id, st, j)
		}
		return fmt.Sprint(j["data"].(map[string]any)["Result"])
	}
	inactive := func() int {
		var n int
		d.QueryRow(`select inactive from wormix_users_teams where user_id = 1 and teammate_id = 2`).Scan(&n)
		return n
	}
	if r := toggle(2, false); r != "0" || inactive() != 1 {
		t.Fatalf("off: result %s, inactive %d", r, inactive())
	}
	if r := toggle(2, true); r != "0" || inactive() != 0 {
		t.Fatalf("on: result %s, inactive %d", r, inactive())
	}
	if r := toggle(3, false); r != "1" {
		t.Fatalf("not in the team: result %s", r)
	}
}

// The owner's grant: fuzes and rubies added, the best of every crafted hat, a worn lesser one made the best.
func TestGrantOwnerCraftedHats(t *testing.T) {
	d := openDB(t, copyFixture(t))
	ctx := context.Background()
	if _, err := d.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	// bob (2) becomes the owner, with a Samurai helm 1502 worn by a race-1 worm.
	for _, q := range []string{
		`update users set login = 'andrii' where id = 2`,
		`insert into wormix_users_weapons (owner_id, weapon_id, count, expire_at) values (2, 1502, -1, -1)`,
		`update wormix_worms_data set hat = 20202 where owner_id = 2`,
		`delete from migrations where migration = '2026_11_10_130000_grant_owner_crafted_hats'`,
	} {
		if _, err := d.Exec(q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	if _, err := d.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	var money, real, hat, hats, samurai, others int64
	d.QueryRow(`select money, real_money from wormix_user_profiles where user_id = 2`).Scan(&money, &real)
	d.QueryRow(`select hat from wormix_worms_data where owner_id = 2`).Scan(&hat)
	d.QueryRow(`select count(*) from wormix_users_weapons where owner_id = 2 and weapon_id >= 1500 and weapon_id < 2000`).Scan(&hats)
	d.QueryRow(`select count(*) from wormix_users_weapons where owner_id = 2 and weapon_id between 1500 and 1509`).Scan(&samurai)
	d.QueryRow(`select count(*) from wormix_users_weapons where owner_id <> 2 and weapon_id >= 1500 and weapon_id < 2000`).Scan(&others)
	if money != 450+100000 || real != 3+20000 {
		t.Errorf("money %d, rubies %d", money, real)
	}
	if hats != 15 || samurai != 1 || hat != 20205 || others != 0 {
		t.Errorf("hats %d (samurai %d), worn %d, others' %d", hats, samurai, hat, others)
	}
	var best int64
	d.QueryRow(`select count(*) from wormix_users_weapons where owner_id = 2 and weapon_id in (1505, 1514, 1525, 1536, 1545, 1555, 1566, 1577, 1586, 1596, 1606, 1615, 1626, 1636, 1646)`).Scan(&best)
	if best != 15 {
		t.Errorf("best hats %d, want 15", best)
	}
}
