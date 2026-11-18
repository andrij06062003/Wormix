package db

import (
	"context"
	"database/sql"
)

// backpackColumns are the arsenal's columns of wormix_user_profiles: the client's weapons panel layouts, the
// one in use and its hotkeys (app/backpack.go).
var backpackColumns = []struct{ name, sql string }{
	{"backpack_confs", `alter table "wormix_user_profiles" add column "backpack_confs" text not null default '[]'`},
	{"active_backpack_conf", `alter table "wormix_user_profiles" add column "active_backpack_conf" integer not null default '0'`},
	{"hotkeys", `alter table "wormix_user_profiles" add column "hotkeys" text not null default '[]'`},
}

func migrateBackpack(ctx context.Context, tx *sql.Tx) error {
	for _, c := range backpackColumns {
		var has int
		if err := tx.QueryRowContext(ctx, `select count(*) from pragma_table_xinfo('wormix_user_profiles') where name = ?`, c.name).Scan(&has); err != nil {
			return err
		}
		if has > 0 {
			continue
		}
		if _, err := tx.ExecContext(ctx, c.sql); err != nil {
			return err
		}
	}
	return nil
}
