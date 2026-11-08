package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/andrij06062003/Wormix/services/api/internal/php"
)

// grantLogin is the account that 2026_11_08_180000_grant_owner_account gives every achievement,
// rating, rubies and fuzes to (the server owner's request). Nothing happens if it doesn't exist.
const grantLogin = "andrii"

const (
	grantRating = 100000
	grantRubies = 10000 // real_money
	grantFuzes  = 10000 // money
)

// migrateGrantOwner sets every achievement of grantLogin to its top level, and its rating, rubies
// and fuzes to the amounts above.
func migrateGrantOwner(ctx context.Context, tx *sql.Tx) error {
	var uid int64
	err := tx.QueryRowContext(ctx, `select "id" from "users" where "login" = ?`, grantLogin).Scan(&uid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	now := Now()
	if _, err := tx.ExecContext(ctx, `update "wormix_user_profiles" set "rating" = ?, "real_money" = ?, "money" = ?, "updated_at" = ? where "user_id" = ?`,
		int64(grantRating), int64(grantRubies), int64(grantFuzes), now, uid); err != nil {
		return err
	}
	cfg, ok := loadJSON("achievements.json").(*php.Arr)
	if !ok {
		return fmt.Errorf("achievements.json: invalid")
	}
	achs, ok := cfg.Val("achievements").(*php.Arr)
	if !ok {
		return fmt.Errorf("achievements.json: no achievements")
	}
	for _, id := range achs.Keys() {
		ach, _ := achs.Val(id).(*php.Arr)
		if ach == nil {
			continue
		}
		levels, _ := ach.Val("levels").(*php.Arr)
		if levels == nil || levels.Len() == 0 {
			continue
		}
		last, _ := levels.Val(levels.Keys()[levels.Len()-1]).(*php.Arr)
		if last == nil {
			continue
		}
		if _, err := tx.ExecContext(ctx, `insert into "wormix_users_achievements" ("user_id", "achievement_id", "value", "created_at", "updated_at") values (?, ?, ?, ?, ?)
			on conflict ("user_id", "achievement_id") do update set "value" = excluded."value", "updated_at" = excluded."updated_at"`,
			uid, php.ToInt(id), php.ToInt(last.Val("max")), now, now); err != nil {
			return err
		}
	}
	return nil
}
