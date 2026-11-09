package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/andrij06062003/Wormix/services/api/internal/db"
)

// testdata/ref-phaseA.sqlite was created and used by the PHP API (wormix-api:refdev, with artifacts):
// 5 accounts, game logins, purchases, a battle, an achievement session.
// testdata/ref-phaseA.json holds the passwords/tokens/keys the PHP API handed out.
// testdata/ref-preartifacts.sqlite was built by the PHP API before the artifacts migration (wormix-api:ref).

type refVars struct {
	Users map[string]struct {
		ID         int64  `json:"id"`
		Password   string `json:"password"`
		Token      string `json:"token"`
		AuthKey    string `json:"auth_key"`
		SessionKey string `json:"session_key"`
		Tcp        string `json:"tcp"`
		AchSession string `json:"ach_session"`
	} `json:"users"`
}

func copyFixture(t *testing.T) string { return copyNamedFixture(t, "ref-phaseA.sqlite") }

func copyNamedFixture(t *testing.T, name string) string {
	t.Helper()
	src, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "database.sqlite")
	if err := os.WriteFile(p, src, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func openDB(t *testing.T, path string) *db.DB {
	t.Helper()
	d, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func schemaOf(t *testing.T, d *sql.DB) []string {
	rows, err := d.Query(`select type || ' ' || name || ' ' || coalesce(sql, '') from sqlite_master where name != 'sqlite_sequence' order by rowid`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		rows.Scan(&s)
		out = append(out, s)
	}
	return out
}

func tableDump(t *testing.T, d *sql.DB, table string) [][]any {
	rows, err := d.Query(`select * from "` + table + `" order by rowid`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	var out [][]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		rows.Scan(ptrs...)
		for i, c := range cols {
			if c == "created_at" || c == "updated_at" {
				vals[i] = vals[i] != nil
			}
			if b, ok := vals[i].([]byte); ok {
				vals[i] = string(b)
			}
		}
		out = append(out, vals)
	}
	return out
}

var seededTables = []string{"migrations", "wormix_weapons", "daily_bonuses", "wormix_races", "wormix_levels", "wormix_reagents", "wormix_missions", "wormix_craft"}

func TestMigrateEmptyMatchesRef(t *testing.T) {
	ref := openDB(t, copyFixture(t))
	d := openDB(t, filepath.Join(t.TempDir(), "new.sqlite"))
	created, err := d.Migrate(context.Background())
	if err != nil || !created {
		t.Fatalf("migrate: created=%v err=%v", created, err)
	}
	if err := d.Seed(context.Background()); err != nil {
		t.Fatal(err)
	}
	if a, b := schemaOf(t, ref.DB), schemaOf(t, d.DB); !reflect.DeepEqual(a, b) {
		t.Fatalf("schema differs:\nref %q\ngo  %q", a, b)
	}
	// The crafted equipment (1.25.0) came after the PHP API: check it, then leave it out of the comparison.
	var crafted int
	d.QueryRow(`select count(*) from wormix_weapons where ((id > 1500 and id < 2000) or id > 2500) and hide_in_shop = 1 and infinity = 1`).Scan(&crafted)
	if crafted != 128 {
		t.Errorf("crafted equipment rows %d, want 128", crafted)
	}
	d.Exec(`delete from wormix_weapons where (id > 1500 and id < 2000) or id > 2500`)
	d.Exec(`delete from migrations where migration = '2026_11_08_120000_add_wormix_crafted_equip'`)
	d.Exec(`delete from migrations where migration = '2026_11_08_180000_grant_owner_account'`)
	// So did the hats of the 1.25.0 client (1062+).
	var hats125 int
	d.QueryRow(`select count(*) from wormix_weapons where id > 1061 and id < 1500 and infinity = 1`).Scan(&hats125)
	if hats125 != 79 {
		t.Errorf("1.25.0 hat rows %d, want 79", hats125)
	}
	d.Exec(`delete from wormix_weapons where id > 1061 and id < 1500`)
	d.Exec(`delete from migrations where migration = '2026_11_08_200000_add_wormix_hats_125'`)
	// And the Ice Shard sold in the shop.
	var iceHidden int
	d.QueryRow(`select hide_in_shop from wormix_weapons where id = 2019`).Scan(&iceHidden)
	if iceHidden != 0 {
		t.Errorf("Ice Shard hidden in the shop")
	}
	d.Exec(`update wormix_weapons set hide_in_shop = 1 where id = 2019`)
	d.Exec(`delete from migrations where migration = '2026_11_08_210000_sell_wormix_ice_shard'`)
	// wormix_users_teams.inactive (1.25.0) is in the reference DB already: only its migration row is new.
	d.Exec(`delete from migrations where migration = '2026_11_08_220000_add_wormix_teams_inactive'`)
	// And the weapons of the 1.25.0 shop.
	var weapons125 int
	d.QueryRow(`select count(*) from wormix_weapons where id in (81, 82, 83, 94, 95, 96, 105, 107, 109) and hide_in_shop = 0 and infinity = 1 and price > 0 and real_price > 0`).Scan(&weapons125)
	if weapons125 != 9 {
		t.Errorf("1.25.0 weapon rows %d, want 9", weapons125)
	}
	d.Exec(`delete from wormix_weapons where id in (81, 82, 83, 94, 95, 96, 105, 107, 109)`)
	d.Exec(`delete from migrations where migration = '2026_11_09_120000_add_wormix_weapons_125'`)
	// And the weapon upgrades of the 1.25.0 client: 52 new recipes, and its costs for the old ones.
	var recipes, sheep int
	d.QueryRow(`select count(*) from wormix_craft`).Scan(&recipes)
	d.QueryRow(`select count(*) from wormix_craft where id in (47, 48) and prev_upgrade_id = 40 and required_level = 12`).Scan(&sheep)
	if recipes != 89 || sheep != 2 {
		t.Errorf("1.25.0 recipes %d (sheep %d), want 89 (2)", recipes, sheep)
	}
	var reagents string
	d.QueryRow(`select reagents from wormix_craft where id = 3`).Scan(&reagents)
	if reagents != "[[0,40],[21,15],[41,12]]" {
		t.Errorf("recipe 3 reagents %s", reagents)
	}
	d.Exec(`delete from wormix_craft where id > 37`)
	rr, err := ref.Query(`select id, reagents, required_level from wormix_craft`)
	if err != nil {
		t.Fatal(err)
	}
	for rr.Next() {
		var id, lvl int64
		var rg string
		rr.Scan(&id, &rg, &lvl)
		d.Exec(`update wormix_craft set reagents = ?, required_level = ? where id = ?`, rg, lvl, id)
	}
	rr.Close()
	d.Exec(`delete from migrations where migration = '2026_11_09_130000_add_wormix_recipes_125'`)
	// And the races of the 1.25.0 client: dragon, robot, rhino and boar, and its prices for the old ones.
	var races, dragon int
	d.QueryRow(`select count(*) from wormix_races`).Scan(&races)
	d.QueryRow(`select count(*) from wormix_races where race_id = 7 and race_name = 'dragon' and price = 2600 and real_price = 26 and required_level = 14`).Scan(&dragon)
	if races != 10 || dragon != 1 {
		t.Errorf("1.25.0 races %d (dragon %d), want 10 (1)", races, dragon)
	}
	d.Exec(`delete from wormix_races where race_id > 6`)
	rq, err := ref.Query(`select race_id, price, real_price, required_level from wormix_races`)
	if err != nil {
		t.Fatal(err)
	}
	for rq.Next() {
		var id, price, real, lvl int64
		rq.Scan(&id, &price, &real, &lvl)
		d.Exec(`update wormix_races set price = ?, real_price = ?, required_level = ? where race_id = ?`, price, real, lvl, id)
	}
	rq.Close()
	d.Exec(`delete from migrations where migration = '2026_11_09_140000_add_wormix_races_125'`)
	// wormix_users_battle_info.last_coop_mission_id (1.25.0 co-op bosses) is in the reference DB already.
	d.Exec(`delete from migrations where migration = '2026_11_09_150000_add_wormix_coop_progress'`)
	// And the solo bosses of the 1.25.0 client (21 Dark Knight .. 27).
	var missions, knight int
	d.QueryRow(`select count(*) from wormix_missions where mission_id > 20`).Scan(&missions)
	d.QueryRow(`select count(*) from wormix_missions where mission_id = 21 and required_level = 20 and awards like '%[1067,1]%'`).Scan(&knight)
	if missions != 7 || knight != 1 {
		t.Errorf("1.25.0 missions %d (dark knight %d), want 7 (1)", missions, knight)
	}
	d.Exec(`delete from wormix_missions where mission_id > 20`)
	d.Exec(`delete from migrations where migration = '2026_11_09_160000_add_wormix_missions_125'`)
	// users_vk_data.name_changed_at (the profile page's name change) is in the reference DB already.
	d.Exec(`delete from migrations where migration = '2026_11_09_170000_add_users_name_changed_at'`)
	for _, tb := range seededTables {
		if a, b := tableDump(t, ref.DB, tb), tableDump(t, d.DB, tb); !reflect.DeepEqual(a, b) {
			t.Errorf("table %s differs (%d vs %d rows)", tb, len(a), len(b))
		}
	}
}

func TestMigrateRefDBIsNoop(t *testing.T) {
	p := copyFixture(t)
	d := openDB(t, p)
	before := schemaOf(t, d.DB)
	created, err := d.Migrate(context.Background())
	if err != nil || created {
		t.Fatalf("migrate on ref DB: created=%v err=%v", created, err)
	}
	if after := schemaOf(t, d.DB); !reflect.DeepEqual(before, after) {
		t.Fatal("schema changed")
	}
	var n int
	d.QueryRow(`select count(*) from migrations`).Scan(&n)
	if n != len(db.Migrations) {
		t.Fatalf("migrations rows %d", n)
	}
}

func TestMigrateUpgradesPreArtifactsDB(t *testing.T) {
	d := openDB(t, copyNamedFixture(t, "ref-preartifacts.sqlite"))
	created, err := d.Migrate(context.Background())
	if err != nil || created {
		t.Fatalf("migrate: created=%v err=%v", created, err)
	}
	var batch, artifacts, col int
	d.QueryRow(`select batch from migrations where migration = '2026_11_06_120000_add_wormix_artifacts'`).Scan(&batch)
	d.QueryRow(`select count(*) from wormix_weapons where id > 2000 and id < 2500 and infinity = 1 and created_at is not null`).Scan(&artifacts)
	d.QueryRow(`select count(*) from pragma_table_info('wormix_worms_data') where name = 'artifact'`).Scan(&col)
	if batch != 2 || artifacts != 35 || col != 1 {
		t.Fatalf("batch=%d artifacts=%d column=%d", batch, artifacts, col)
	}
	// Same schema as a DB created by the current PHP API.
	ref := openDB(t, copyFixture(t))
	if a, b := schemaOf(t, ref.DB), schemaOf(t, d.DB); !reflect.DeepEqual(a, b) {
		t.Fatalf("schema differs:\nref %q\ngo  %q", a, b)
	}
}

func newServer(t *testing.T) (*httptest.Server, *db.DB, refVars) {
	t.Helper()
	p := copyFixture(t)
	d := openDB(t, p)
	if _, err := d.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	// keep the 6h achievement session alive whatever the fixture's age
	d.Exec(`update cache set expiration = ?`, time.Now().Unix()+3600)
	srv := httptest.NewServer(NewApp(d, t.TempDir()))
	t.Cleanup(srv.Close)
	var v refVars
	b, _ := os.ReadFile("testdata/ref-phaseA.json")
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	return srv, d, v
}

func call(t *testing.T, srv *httptest.Server, method, path string, body any, headers map[string]string) (int, map[string]any) {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, srv.URL+path, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]any
	json.Unmarshal(raw, &out)
	return resp.StatusCode, out
}

func TestRefDBCredentialsWork(t *testing.T) {
	srv, _, v := newServer(t)
	alice, bob := v.Users["alice"], v.Users["bob"]

	// bcrypt password set by PHP
	st, j := call(t, srv, "POST", "/api/account/sign_in", map[string]any{"login": "bob", "password": bob.Password}, nil)
	if st != 200 || j["id"] != float64(2) {
		t.Fatalf("sign_in: %d %v", st, j)
	}
	// Sanctum token issued by PHP
	st, j = call(t, srv, "GET", "/api/account", nil, map[string]string{"Authorization": "Bearer " + alice.Token})
	if st != 200 || j["data"].(map[string]any)["login"] != "alice" {
		t.Fatalf("account with PHP token: %d %v", st, j)
	}
	// game session key written by PHP into the cache table
	st, j = call(t, srv, "POST", "/api/internal/pvp/profile", map[string]any{}, map[string]string{"X-TCP-SESSION": alice.Tcp, "X-SESSION-KEY": alice.SessionKey})
	if st != 200 || j["data"].(map[string]any)["Id"] != float64(1) {
		t.Fatalf("internal auth with PHP session: %d %v", st, j)
	}
	// achievement session (serialized PHP array in the cache table)
	st, j = call(t, srv, "POST", "/api/internal/achievements/get", map[string]any{"session_id": alice.AchSession, "profile_id": "1"}, map[string]string{"X-TCP-SESSION": "achievements"})
	if st != 200 || len(j["indexes"].([]any)) == 0 {
		t.Fatalf("achievement session: %d %v", st, j)
	}
	// wrong password still rejected
	if st, _ = call(t, srv, "POST", "/api/account/sign_in", map[string]any{"login": "bob", "password": "nope"}, nil); st != 403 {
		t.Fatalf("wrong password: %d", st)
	}
}

// A negative balance in one currency (left by an old bug, shown as 0) must not block buying with the other.
func TestBuyWithNegativeOtherCurrency(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	h := map[string]string{"X-TCP-SESSION": alice.Tcp, "X-SESSION-KEY": alice.SessionKey}
	buy := func(moneyType int) string {
		st, j := call(t, srv, "POST", "/api/internal/account/buy/items",
			map[string]any{"ShopItems": []any{map[string]any{"Id": 16, "Count": -1, "MoneyType": moneyType}}}, h)
		if st != 200 {
			t.Fatalf("buy: %d %v", st, j)
		}
		return fmt.Sprint(j["data"].(map[string]any)["Result"])
	}
	d.Exec(`update wormix_user_profiles set money = 1000, real_money = -5 where user_id = 1`)
	if r := buy(1); r != "0" {
		t.Fatalf("fuses with negative rubies: result %s, want 0", r)
	}
	var real int64
	d.QueryRow(`select real_money from wormix_user_profiles where user_id = 1`).Scan(&real)
	if real != -5 {
		t.Fatalf("rubies changed to %d", real)
	}
	d.Exec(`update wormix_user_profiles set money = -5, real_money = 1000 where user_id = 1`)
	if r := buy(0); r != "0" {
		t.Fatalf("rubies with negative fuses: result %s, want 0", r)
	}
	d.Exec(`update wormix_user_profiles set money = 0, real_money = 1000 where user_id = 1`)
	if r := buy(1); r != "3" {
		t.Fatalf("fuses with no fuses: result %s, want 3", r)
	}
}

// The Missions tab's bots get money up to the player's: a negative balance must not give them negative money
// (the game server reads it as unsigned and drops the connection).
func TestArenaBotsWithNegativeBalance(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	h := map[string]string{"X-TCP-SESSION": alice.Tcp, "X-SESSION-KEY": alice.SessionKey}
	d.Exec(`update wormix_user_profiles set money = -500, real_money = -5 where user_id = 1`)
	d.Exec(`update wormix_users_battle_info set battles_count = 5 where user_id = 1`)
	for i := 0; i < 20; i++ {
		st, j := call(t, srv, "POST", "/api/internal/game/get_arena", map[string]any{"ReturnUsersProfiles": true}, h)
		if st != 200 || j["type"] != "ArenaResult" {
			t.Fatalf("get_arena: %d %v", st, j)
		}
		bots := j["data"].(map[string]any)["UserProfileStructures"].([]any)
		if len(bots) == 0 {
			t.Fatal("no bots")
		}
		for _, b := range bots {
			bot := b.(map[string]any)
			if bot["Money"].(float64) < 0 || bot["RealMoney"].(float64) < 0 {
				t.Fatalf("bot with negative money: %v %v", bot["Money"], bot["RealMoney"])
			}
		}
	}
}

// Concurrent purchases must stay atomic: every successful purchase is paid exactly once.
func TestConcurrentPurchasesAreAtomic(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	d.Exec(`update wormix_user_profiles set money = 1000 where user_id = 1`)
	var before int64
	d.QueryRow(`select coalesce(sum(count), 0) from wormix_users_weapons where owner_id = 1 and weapon_id = 6`).Scan(&before)

	h := map[string]string{"X-TCP-SESSION": alice.Tcp, "X-SESSION-KEY": alice.SessionKey}
	const n = 40
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			st, j := call(t, srv, "POST", "/api/internal/account/buy/items",
				map[string]any{"ShopItems": []any{map[string]any{"Id": 6, "Count": 1, "MoneyType": 1}}}, h)
			if st == 200 && fmt.Sprint(j["data"].(map[string]any)["Result"]) == "0" {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	var money, price, after int64
	d.QueryRow(`select money from wormix_user_profiles where user_id = 1`).Scan(&money)
	d.QueryRow(`select price from wormix_weapons where id = 6`).Scan(&price)
	d.QueryRow(`select coalesce(sum(count), 0) from wormix_users_weapons where owner_id = 1 and weapon_id = 6`).Scan(&after)
	if money != 1000-int64(ok)*price || after-before != int64(ok) || money < 0 {
		t.Fatalf("ok=%d money=%d price=%d items %d->%d", ok, money, price, before, after)
	}
	if ok == 0 {
		t.Fatal("no purchase succeeded")
	}
}

// Players can change their name, login and password, but not money, rubies, the worm or battles.
func TestAccountUpdateRefusesGameData(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	h := map[string]string{"Authorization": "Bearer " + alice.Token}
	var money, real, battles int64
	read := func() {
		d.QueryRow(`select money, real_money from wormix_user_profiles where user_id = 1`).Scan(&money, &real)
		d.QueryRow(`select battles_count from wormix_users_battle_info where user_id = 1`).Scan(&battles)
	}
	read()
	m0, r0, b0 := money, real, battles
	for _, body := range []map[string]any{
		{"user_profile": map[string]any{"money": 100000, "real_money": 100000}},
		{"worm_data": map[string]any{"level": 30, "armor": 0, "attack": 60, "race": 1}},
		{"battle_info": map[string]any{"battles_count": 999}},
		{"social_data": map[string]any{"first_name": "x"}, "user_profile": map[string]any{"real_money": 1}},
	} {
		if st, j := call(t, srv, "POST", "/api/account", body, h); st != 403 {
			t.Fatalf("%v: %d %v, want 403", body, st, j)
		}
	}
	read()
	if money != m0 || real != r0 || battles != b0 {
		t.Fatalf("changed: money %d→%d, rubies %d→%d, battles %d→%d", m0, money, r0, real, b0, battles)
	}
	if st, j := call(t, srv, "POST", "/api/account", map[string]any{"social_data": map[string]any{"first_name": "Alice"}}, h); st != 200 {
		t.Fatalf("name: %d %v", st, j)
	}
}

// The 1.25.0 client asks VK's getProfiles for "1,2" (VK's format), 1.05.0 for "[1,2]".
func TestGetProfilesCommaSeparated(t *testing.T) {
	srv, _, _ := newServer(t)
	for _, uids := range []string{"[1,2]", "1,2", "1"} {
		st, j := call(t, srv, "POST", "/api", map[string]any{"method": "getProfiles", "uids": uids}, nil)
		if st != 200 || j["response"] == nil {
			t.Fatalf("uids %q: %d %v", uids, st, j)
		}
	}
}
