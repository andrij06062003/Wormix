package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/andrij06062003/Wormix/services/api/internal/php"
)

// execQuerier is a *sql.Tx or a *DB.
type execQuerier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// migrateRecipes125 is 2026_11_09_130000_add_wormix_recipes_125: the weapon upgrades of the 1.25.0 client.
// Without their wormix_craft rows the API refused an upgrade (exists:wormix_craft,id) and the core dropped
// the session ("the game lost the connection to the server"). A new database gets them from game:init,
// after recipes.json: inserting them here first would make its recipe inserts fail.
func migrateRecipes125(ctx context.Context, tx *sql.Tx) error {
	var n int
	if err := tx.QueryRowContext(ctx, `select count(*) from "wormix_craft"`).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return nil
	}
	return applyRecipes125(ctx, tx)
}

// applyRecipes125 writes game/recipes125.json into wormix_craft: a recipe that exists gets the 1.25.0
// reagents and levels, a missing one is inserted.
func applyRecipes125(ctx context.Context, q execQuerier) error {
	cfg, ok := loadJSON("recipes125.json").(*php.Arr)
	if !ok {
		return fmt.Errorf("recipes125.json: invalid")
	}
	list, ok := cfg.Val("recipes").(*php.Arr)
	if !ok {
		return fmt.Errorf("recipes125.json: no recipes")
	}
	now := Now()
	for _, x := range list.Values() {
		a, ok := x.(*php.Arr)
		if !ok {
			continue
		}
		vals := []any{php.ToString(get(a, "description")), Bind(get(a, "upgradeId")), Bind(get(a, "prevUpgradeId")),
			php.MustJSON(get(a, "reagents")), Bind(get(a, "level")), Bind(get(a, "requiredLevel"))}
		id := php.ToInt(get(a, "id"))
		var n int
		if err := q.QueryRowContext(ctx, `select count(*) from "wormix_craft" where "id" = ?`, id).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			if _, err := q.ExecContext(ctx, `update "wormix_craft" set "description" = ?, "upgrade_id" = ?, "prev_upgrade_id" = ?, "reagents" = ?, "level" = ?, "required_level" = ?, "updated_at" = ? where "id" = ?`,
				append(vals, now, id)...); err != nil {
				return err
			}
			continue
		}
		if _, err := q.ExecContext(ctx, `insert into "wormix_craft" ("description", "upgrade_id", "prev_upgrade_id", "reagents", "level", "required_level", "updated_at", "created_at", "id") values (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			append(vals, now, now, id)...); err != nil {
			return err
		}
	}
	return nil
}
