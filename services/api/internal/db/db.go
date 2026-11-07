// Package db opens the SQLite database, applies the Laravel-compatible migrations
// and runs the game:init seeding on a new database.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"sync"

	_ "modernc.org/sqlite" // pure-Go SQLite driver
)

// DB wraps the connection pool. Writes are serialized in-process (SQLite allows one
// writer anyway) so concurrent requests never fail with SQLITE_BUSY.
type DB struct {
	*sql.DB
	wmu sync.Mutex
}

// Open opens (creating if needed) the database file with WAL, busy timeout and
// foreign keys on (Laravel enables foreign_key_constraints for SQLite).
func Open(path string) (*DB, error) {
	q := url.Values{}
	q.Add("_pragma", "busy_timeout(10000)")
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "synchronous(FULL)")
	q.Set("_txlock", "immediate")
	dsn := "file:" + path + "?" + q.Encode()
	sdb, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	sdb.SetMaxOpenConns(8)
	sdb.SetMaxIdleConns(8)
	if err := sdb.Ping(); err != nil {
		sdb.Close()
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	return &DB{DB: sdb}, nil
}

// Tx runs fn in an immediate (write) transaction; it commits when fn returns nil
// and rolls back on error or panic (the panic is re-raised).
func (d *DB) Tx(ctx context.Context, fn func(tx *sql.Tx) error) (err error) {
	d.wmu.Lock()
	defer d.wmu.Unlock()
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
		if err != nil {
			tx.Rollback()
		}
	}()
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
