package db

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"log"
	"time"

	"github.com/andrij06062003/Wormix/services/api/internal/assets"
	"github.com/andrij06062003/Wormix/services/api/internal/php"
)

// StarterWeapons is config('wormix.starter.weapons').
var StarterWeapons = []int64{1, 2, 4}

// Now returns the current time in Laravel's storage format (UTC).
func Now() string { return time.Now().UTC().Format("2006-01-02 15:04:05") }

// Bind converts a PHP value into a database/sql argument the way Laravel+PDO bind it:
// bools become ints, floats are bound as their string form, arrays are an error.
func Bind(v any) any {
	switch t := v.(type) {
	case bool:
		if t {
			return int64(1)
		}
		return int64(0)
	case int:
		return int64(t)
	case float64:
		return php.FloatString(t)
	case *php.Arr:
		php.Fail("Array to string conversion")
	}
	return v
}

func loadJSON(name string) any {
	b, err := assets.Game(name)
	if err != nil {
		return nil
	}
	v, ok := php.JSONDecode(b)
	if !ok {
		return nil
	}
	return v
}

func loadMessages() map[string]string {
	b, err := assets.Game("items.messages_ru.xml")
	out := map[string]string{}
	if err != nil {
		return out
	}
	var doc struct {
		Items []struct {
			Name  string `xml:"name,attr"`
			Value string `xml:"value,attr"`
		} `xml:",any"`
	}
	if err := xml.Unmarshal(b, &doc); err != nil {
		log.Printf("seed: parse messages: %v", err)
		return out
	}
	for _, m := range doc.Items {
		out[m.Name] = m.Value
	}
	return out
}

type seeder struct {
	ctx context.Context
	db  *DB
}

// insert runs one insert in its own transaction (like the per-row DB::beginTransaction
// in InitGameData); errors are logged and the row skipped.
func (s *seeder) insert(table string, cols []string, vals []any) error {
	q := `insert into "` + table + `" (`
	ph := ""
	for i, c := range cols {
		if i > 0 {
			q += ", "
			ph += ", "
		}
		q += `"` + c + `"`
		ph += "?"
	}
	q += ") values (" + ph + ")"
	args := make([]any, len(vals))
	for i, v := range vals {
		args[i] = Bind(v)
	}
	_, err := s.db.ExecContext(s.ctx, q, args...)
	return err
}

func get(a *php.Arr, k string) any { return a.Val(k) }

func orDefault(a *php.Arr, k string, d any) any {
	if v, ok := a.Get(k); ok && v != nil {
		return v
	}
	return d
}

// Seed is `php artisan game:init`.
func (d *DB) Seed(ctx context.Context) error {
	d.wmu.Lock()
	defer d.wmu.Unlock()
	s := &seeder{ctx: ctx, db: d}
	msgs := loadMessages()
	name := func(raw any) any {
		if k, ok := raw.(string); ok {
			if m, ok := msgs[k]; ok {
				return m
			}
		}
		return raw
	}
	isStarter := func(id any) bool {
		n := php.ToInt(id)
		for _, w := range StarterWeapons {
			if w == n {
				return true
			}
		}
		return false
	}

	// weapons.json
	if w, ok := loadJSON("weapons.json").(*php.Arr); ok {
		for _, x := range w.Values() {
			a, ok := x.(*php.Arr)
			if !ok {
				continue
			}
			if a.Has("name") {
				err := s.insert("wormix_weapons",
					[]string{"id", "name", "hide_in_shop", "is_starter", "price", "real_price", "required_friends", "required_level", "infinity"},
					[]any{get(a, "id"), name(get(a, "name")), a.Has("hideInShop"), isStarter(get(a, "id")),
						orDefault(a, "price", int64(0)), orDefault(a, "realprice", int64(0)),
						orDefault(a, "requiredFriends", int64(0)), orDefault(a, "requiredLevel", int64(0)), a.Has("infinite")})
				if err != nil {
					log.Printf("seed: weapon %v: %v", get(a, "id"), err)
				}
			} else if a.Has("refId") {
				err := s.insert("wormix_weapons",
					[]string{"id", "ref_id", "hide_in_shop", "price", "required_friends", "required_level"},
					[]any{get(a, "id"), get(a, "refId"), orDefault(a, "hideInShop", true), get(a, "price"),
						orDefault(a, "requiredFriends", int64(0)), orDefault(a, "requiredLevel", int64(0))})
				if err != nil {
					// InitGameData only catches Mockery\Exception here: a query error aborts the command.
					return fmt.Errorf("seed: ref weapon %v: %w", get(a, "id"), err)
				}
			}
		}
	}

	// hats.json
	if h, ok := loadJSON("hats.json").(*php.Arr); ok {
		for _, x := range h.Values() {
			a, ok := x.(*php.Arr)
			if !ok {
				continue
			}
			inf, infIsBool := get(a, "infinite").(bool)
			_, oneDayIsBool := get(a, "oneDay").(bool)
			err := s.insert("wormix_weapons",
				[]string{"id", "name", "hide_in_shop", "is_starter", "price", "real_price", "required_rating", "required_level", "infinity", "one_day"},
				[]any{get(a, "id"), name(get(a, "name")), a.Has("hideInShop"), false,
					orDefault(a, "price", int64(0)), orDefault(a, "realprice", int64(0)),
					orDefault(a, "requiredRating", int64(0)), orDefault(a, "requiredLevel", int64(0)),
					infIsBool && inf, oneDayIsBool && php.ToBool(get(a, "infinite"))})
			if err != nil {
				log.Printf("seed: hat %v: %v", get(a, "id"), err)
			}
		}
	}

	// weapons_start.json: Eloquent builder update also touches updated_at.
	if st, ok := loadJSON("weapons_start.json").(*php.Arr); ok && st.Len() > 0 {
		q := `update "wormix_weapons" set "is_starter" = ?, "updated_at" = ? where "id" in (`
		args := []any{int64(1), Now()}
		for i, v := range st.Values() {
			if i > 0 {
				q += ", "
			}
			q += "?"
			args = append(args, Bind(v))
		}
		if _, err := d.ExecContext(ctx, q+")", args...); err != nil {
			log.Printf("seed: start items: %v", err)
		}
	}

	// gifts.json
	if g, ok := loadJSON("gifts.json").(*php.Arr); ok {
		for _, x := range g.Values() {
			a, _ := x.(*php.Arr)
			if a == nil {
				continue
			}
			if err := s.insert("daily_bonuses", []string{"login_sequence", "bonus_type", "bonus_value", "random_gift"},
				[]any{get(a, "sequence"), get(a, "type"), get(a, "value"), get(a, "random")}); err != nil {
				log.Printf("seed: gift: %v", err)
			}
		}
	}

	// races.json
	if r, ok := loadJSON("races.json").(*php.Arr); ok {
		for _, x := range r.Values() {
			a, _ := x.(*php.Arr)
			if a == nil {
				continue
			}
			if err := s.insert("wormix_races", []string{"race_id", "race_name", "price", "real_price", "required_level"},
				[]any{get(a, "raceId"), get(a, "configName"), get(a, "price"), get(a, "realPrice"), get(a, "requiredLevel")}); err != nil {
				return fmt.Errorf("seed: race: %w", err)
			}
		}
	}
	// The 1.25.0 client's races (migrateRaces125 leaves a new database to this).
	if err := applyRaces125(ctx, d); err != nil {
		return fmt.Errorf("seed: races 1.25.0: %w", err)
	}

	// level_awards.json
	if l, ok := loadJSON("level_awards.json").(*php.Arr); ok {
		for _, x := range l.Values() {
			a, _ := x.(*php.Arr)
			if a == nil {
				continue
			}
			if err := s.insert("wormix_levels", []string{"required_experience", "max_worms_count", "awards"},
				[]any{get(a, "required_experience"), get(a, "max_worms_count"), php.MustJSON(get(a, "reward_weapons"))}); err != nil {
				log.Printf("seed: level: %v", err)
			}
		}
	}

	// reagents.json
	if r, ok := loadJSON("reagents.json").(*php.Arr); ok {
		for _, x := range r.Values() {
			a, _ := x.(*php.Arr)
			if a == nil {
				continue
			}
			if err := s.insert("wormix_reagents", []string{"reagent_id", "name", "reagent_price"},
				[]any{get(a, "id"), get(a, "name"), get(a, "price")}); err != nil {
				log.Printf("seed: reagent: %v", err)
			}
		}
	}

	// missions_awards.json: one transaction, Eloquent save() (timestamps).
	if m, ok := loadJSON("missions_awards.json").(*php.Arr); ok {
		err := func() error {
			tx, err := d.BeginTx(ctx, nil)
			if err != nil {
				return err
			}
			for _, x := range m.Values() {
				a, _ := x.(*php.Arr)
				if a == nil {
					continue
				}
				now := Now()
				if _, err := tx.ExecContext(ctx, `insert into "wormix_missions" ("mission_id", "awards", "required_level", "updated_at", "created_at") values (?, ?, ?, ?, ?)`,
					Bind(get(a, "id")), php.MustJSON(get(a, "awards")), Bind(get(a, "required_level")), now, now); err != nil {
					tx.Rollback()
					return err
				}
			}
			return tx.Commit()
		}()
		if err != nil {
			return fmt.Errorf("seed: missions: %w", err)
		}
	}
	// The 1.25.0 client's solo bosses 21-27 (migrateMissions125 leaves a new database to this).
	if err := applyMissions125(ctx, d); err != nil {
		return fmt.Errorf("seed: missions 1.25.0: %w", err)
	}

	// recipes.json
	if r, ok := loadJSON("recipes.json").(*php.Arr); ok {
		for _, x := range r.Values() {
			a, _ := x.(*php.Arr)
			if a == nil {
				continue
			}
			now := Now()
			if err := s.insert("wormix_craft",
				[]string{"id", "description", "upgrade_id", "prev_upgrade_id", "reagents", "level", "required_level", "updated_at", "created_at"},
				[]any{get(a, "id"), get(a, "description"), get(a, "upgradeId"), get(a, "prevUpgradeId"),
					php.MustJSON(get(a, "reagents")), get(a, "level"), get(a, "requiredLevel"), now, now}); err != nil {
				return fmt.Errorf("seed: recipe: %w", err)
			}
		}
	}
	// The 1.25.0 client's recipes (migrateRecipes125 leaves a new database to this).
	if err := applyRecipes125(ctx, d); err != nil {
		return fmt.Errorf("seed: recipes 1.25.0: %w", err)
	}
	log.Printf("game:init: SETUP IS COMPLETED")
	return nil
}

var _ = sql.ErrNoRows
