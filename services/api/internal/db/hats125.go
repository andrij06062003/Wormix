package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/andrij06062003/Wormix/services/api/internal/php"
)

// migrateHats125 is 2026_11_08_200000_add_wormix_hats_125: inserts the hats of game/hats125.json
// (the 1.25.0 client's hats that 1.05.0 doesn't have) into wormix_weapons, skipping ids that already
// exist. Like every hat they are infinite; boss and event hats stay hidden in the shop.
func migrateHats125(ctx context.Context, tx *sql.Tx) error {
	cfg, ok := loadJSON("hats125.json").(*php.Arr)
	if !ok {
		return fmt.Errorf("hats125.json: invalid")
	}
	hats, ok := cfg.Val("hats").(*php.Arr)
	if !ok {
		return fmt.Errorf("hats125.json: no hats")
	}
	now := Now()
	for _, x := range hats.Values() {
		h, ok := x.(*php.Arr)
		if !ok {
			continue
		}
		id := php.ToInt(h.Val("id"))
		var n int
		if err := tx.QueryRowContext(ctx, `select count(*) from "wormix_weapons" where "id" = ?`, id).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		if _, err := tx.ExecContext(ctx, `insert into "wormix_weapons" ("id", "name", "hide_in_shop", "is_starter", "price", "real_price", "required_level", "required_rating", "infinity", "one_day", "created_at", "updated_at") values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, php.ToString(h.Val("name")), Bind(php.ToBool(h.Val("hideInShop"))), int64(0),
			Bind(orDefault(h, "price", int64(0))), Bind(orDefault(h, "realprice", int64(0))),
			Bind(orDefault(h, "requiredLevel", int64(0))), Bind(orDefault(h, "requiredRating", int64(0))),
			int64(1), int64(0), now, now); err != nil {
			return err
		}
	}
	return nil
}
