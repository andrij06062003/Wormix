package db

import (
	"context"
	"database/sql"
)

// migrateGrantOwner is 2026_11_08_180000_grant_owner_account: a one-off change of one account, done
// already. It stays registered (as a no-op) so the databases that ran it and those that didn't agree.
func migrateGrantOwner(ctx context.Context, tx *sql.Tx) error {
	return nil
}
