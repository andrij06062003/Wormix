package db

import (
	"context"
	"database/sql"
)

// migrateTeamInactive is 2026_11_08_220000_add_wormix_teams_inactive: adds wormix_users_teams.inactive
// (if missing). The 1.25.0 client can leave a friend of the team out of battles; 1 = left out.
func migrateTeamInactive(ctx context.Context, tx *sql.Tx) error {
	var has int
	if err := tx.QueryRowContext(ctx, `select count(*) from pragma_table_xinfo('wormix_users_teams') where name = 'inactive'`).Scan(&has); err != nil {
		return err
	}
	if has > 0 {
		return nil
	}
	_, err := tx.ExecContext(ctx, `alter table "wormix_users_teams" add column "inactive" integer not null default '0'`)
	return err
}
