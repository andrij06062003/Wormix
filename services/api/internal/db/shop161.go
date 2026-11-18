package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/andrij06062003/Wormix/services/api/internal/php"
)

// migrateShop161 is 2026_11_18_140000_shop_161: the shop of the 1.61.7 client (game/shop161.json).
// Items wormix_weapons lacks are inserted (hats and artifacts infinite, weapons as the client has them).
// The items the client sells get its prices and levels, so the server charges what the client shows, and are
// no longer hidden (22, a pack of teleports hidden in 1.05.0, is the ion cannon in 1.61.7). The ones it hides
// are left as they are: the 1.25.0 client may sell them at its prices. A new database (no weapons yet) gets
// the shop from game:init, after weapons.json and hats.json, whose ids it would otherwise take.
func migrateShop161(ctx context.Context, tx *sql.Tx) error {
	var n int
	if err := tx.QueryRowContext(ctx, `select count(*) from "wormix_weapons"`).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return nil
	}
	return applyShop161(ctx, tx)
}

// applyShop161 brings wormix_weapons to the 1.61.7 shop.
func applyShop161(ctx context.Context, tx execQuerier) error {
	cfg, ok := loadJSON("shop161.json").(*php.Arr)
	if !ok {
		return fmt.Errorf("shop161.json: invalid")
	}
	items, ok := cfg.Val("items").(*php.Arr)
	if !ok {
		return fmt.Errorf("shop161.json: no items")
	}
	now := Now()
	for _, x := range items.Values() {
		it, ok := x.(*php.Arr)
		if !ok {
			continue
		}
		id := php.ToInt(it.Val("id"))
		price, realPrice := php.ToInt(orDefault(it, "price", int64(0))), php.ToInt(orDefault(it, "realprice", int64(0)))
		level, friends := php.ToInt(orDefault(it, "requiredLevel", int64(0))), php.ToInt(orDefault(it, "requiredFriends", int64(0)))
		rating := php.ToInt(orDefault(it, "requiredRating", int64(0)))
		hidden := php.ToBool(it.Val("hideInShop"))
		infinite := id >= 1000 || php.ToBool(it.Val("infinite"))

		var n int
		if err := tx.QueryRowContext(ctx, `select count(*) from "wormix_weapons" where "id" = ?`, id).Scan(&n); err != nil {
			return err
		}
		if n == 0 {
			if _, err := tx.ExecContext(ctx, `insert into "wormix_weapons" ("id", "name", "hide_in_shop", "is_starter", "price", "real_price", "infinity", "one_day", "required_friends", "required_level", "required_rating", "created_at", "updated_at") values (?, ?, ?, 0, ?, ?, ?, 0, ?, ?, ?, ?, ?)`,
				id, php.ToString(it.Val("name")), Bind(hidden), price, realPrice, Bind(infinite), friends, level, rating, now, now); err != nil {
				return fmt.Errorf("shop161 %d: %w", id, err)
			}
			continue
		}
		if hidden {
			continue
		}
		q := `update "wormix_weapons" set "price" = ?, "real_price" = ?, "required_friends" = ?, "required_level" = ?, "required_rating" = ?, "hide_in_shop" = 0, "updated_at" = ?`
		args := []any{price, realPrice, friends, level, rating, now}
		if id < 1000 {
			q += `, "infinity" = ?`
			args = append(args, Bind(infinite))
		}
		if _, err := tx.ExecContext(ctx, q+` where "id" = ?`, append(args, id)...); err != nil {
			return fmt.Errorf("shop161 %d: %w", id, err)
		}
	}
	return nil
}
