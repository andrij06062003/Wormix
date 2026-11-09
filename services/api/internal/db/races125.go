package db

import (
	"context"
	"database/sql"
)

// races125 are the races of the 1.25.0 client's race config that a player can buy: 7-10 are new
// (dragon, robot, rhino, boar), and the old ones have new prices and levels there.
var races125 = []struct {
	id        int64
	name      string
	price     int64
	realPrice int64
	minLevel  int64
}{
	{0, "worm", 300, 3, 1},
	{2, "boxer", 300, 3, 1},
	{5, "zombie", 1200, 12, 8},
	{4, "rabbit", 1600, 16, 8},
	{3, "daemon", 2000, 20, 10},
	{6, "cat", 2300, 23, 12},
	{7, "dragon", 2600, 26, 14},
	{8, "robot", 3000, 30, 16},
	{9, "rhino", 3400, 34, 18},
	{10, "boar", 300, 3, 1},
}

// migrateRaces125 is 2026_11_09_140000_add_wormix_races_125. Without a row for a new race the API refused
// buying it (exists:wormix_races,race_id) and the core dropped the session. A new database gets the races
// from game:init, after races.json.
func migrateRaces125(ctx context.Context, tx *sql.Tx) error {
	var n int
	if err := tx.QueryRowContext(ctx, `select count(*) from "wormix_races"`).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return nil
	}
	return applyRaces125(ctx, tx)
}

// applyRaces125 gives the existing races the 1.25.0 prices and levels, and inserts the missing ones.
func applyRaces125(ctx context.Context, q execQuerier) error {
	now := Now()
	for _, r := range races125 {
		var n int
		if err := q.QueryRowContext(ctx, `select count(*) from "wormix_races" where "race_id" = ?`, r.id).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			if _, err := q.ExecContext(ctx, `update "wormix_races" set "price" = ?, "real_price" = ?, "required_level" = ? where "race_id" = ?`,
				r.price, r.realPrice, r.minLevel, r.id); err != nil {
				return err
			}
			continue
		}
		if _, err := q.ExecContext(ctx, `insert into "wormix_races" ("race_id", "race_name", "price", "real_price", "required_level", "created_at", "updated_at") values (?, ?, ?, ?, ?, ?, ?)`,
			r.id, r.name, r.price, r.realPrice, r.minLevel, now, now); err != nil {
			return err
		}
	}
	return nil
}
