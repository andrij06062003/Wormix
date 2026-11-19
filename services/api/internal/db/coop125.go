package db

import (
	"context"
	"database/sql"
)

// migrateCoopProgress is 2026_11_09_150000_add_wormix_coop_progress: the last co-op boss (missions 101+ of the
// 1.25.0 client, fought with a friend) the player has beaten. 0 is none; the client counts from 100.
func migrateCoopProgress(ctx context.Context, tx *sql.Tx) error {
	var has int
	if err := tx.QueryRowContext(ctx, `select count(*) from pragma_table_xinfo('wormix_users_battle_info') where name = 'last_coop_mission_id'`).Scan(&has); err != nil {
		return err
	}
	if has > 0 {
		return nil
	}
	_, err := tx.ExecContext(ctx, `alter table "wormix_users_battle_info" add column "last_coop_mission_id" integer not null default '0'`)
	return err
}
