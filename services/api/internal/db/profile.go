package db

import (
	"context"
	"database/sql"
)

// migrateNameChangedAt is 2026_11_09_170000_add_users_name_changed_at: when the player last changed the name the
// game shows (the profile page allows it once in 60 days). Null: never.
func migrateNameChangedAt(ctx context.Context, tx *sql.Tx) error {
	var has int
	if err := tx.QueryRowContext(ctx, `select count(*) from pragma_table_xinfo('users_vk_data') where name = 'name_changed_at'`).Scan(&has); err != nil {
		return err
	}
	if has > 0 {
		return nil
	}
	_, err := tx.ExecContext(ctx, `alter table "users_vk_data" add column "name_changed_at" datetime`)
	return err
}
