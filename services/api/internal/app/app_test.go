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
	d.QueryRow(`select count(*) from wormix_weapons where id > 2000 and infinity = 1 and created_at is not null`).Scan(&artifacts)
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
