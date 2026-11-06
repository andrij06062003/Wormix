package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/andrij06062003/Wormix/services/api/internal/php"
)

// migrateArtifacts is 2026_11_06_120000_add_wormix_artifacts: adds wormix_worms_data.artifact
// (if missing) and inserts the artifacts of game/artifacts.json into wormix_weapons (ids 2001+),
// skipping ids that already exist.
func migrateArtifacts(ctx context.Context, tx *sql.Tx) error {
	var has int
	if err := tx.QueryRowContext(ctx, `select count(*) from pragma_table_xinfo('wormix_worms_data') where name = 'artifact'`).Scan(&has); err != nil {
		return err
	}
	if has == 0 {
		if _, err := tx.ExecContext(ctx, `alter table "wormix_worms_data" add column "artifact" integer not null default '0'`); err != nil {
			return err
		}
	}
	list, ok := loadJSON("artifacts.json").(*php.Arr)
	if !ok {
		return fmt.Errorf("artifacts.json: invalid")
	}
	existing := php.NewArr()
	rows, err := tx.QueryContext(ctx, `select "id" from "wormix_weapons"`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id any
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		existing.Append(id)
	}
	rows.Close()
	now := Now()
	for _, x := range list.Values() {
		a, ok := x.(*php.Arr)
		if !ok {
			continue
		}
		if php.InArray(a.Val("id"), existing, false) {
			continue
		}
		if _, err := tx.ExecContext(ctx, `insert into "wormix_weapons" ("id", "name", "hide_in_shop", "is_starter", "price", "real_price", "required_level", "required_rating", "infinity", "one_day", "created_at", "updated_at") values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			Bind(a.Val("id")), Bind(a.Val("name")), Bind(php.ToBool(a.Val("hideInShop"))), int64(0),
			Bind(orDefault(a, "price", int64(0))), Bind(orDefault(a, "realprice", int64(0))),
			Bind(orDefault(a, "requiredLevel", int64(0))), Bind(orDefault(a, "requiredRating", int64(0))),
			int64(1), int64(0), now, now); err != nil {
			return err
		}
	}
	return nil
}
