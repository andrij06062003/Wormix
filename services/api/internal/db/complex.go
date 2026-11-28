package db

import (
	"context"
	"database/sql"
)

// ComplexMaxLevel lists the 1.61.7 client's "complex" weapons (isComplex in its weapon config) and their
// top level (their maxShots: a level is one more shot a battle). The client owns such a weapon for good at
// a level and buys or is given it by the level (a Count is a number of levels, not shots). It reads the count
// the original server sent: -1 the top level, -10-L level L below it, a positive count plain shots of level 1
// that a battle uses up. wormix_users_weapons keeps that same count for them.
var ComplexMaxLevel = map[int64]int64{
	6: 2, 7: 4, 10: 4, 12: 4, 14: 4, 16: 4, 19: 4, 23: 4, 26: 3, 29: 4, 30: 4, 31: 4, 33: 4, 34: 4,
	35: 4, 37: 4, 38: 4, 40: 3, 41: 2, 42: 3, 44: 4, 45: 4, 46: 3, 47: 3, 48: 4, 50: 4, 52: 4, 53: 2,
	54: 4, 55: 4, 57: 2, 60: 2, 69: 3, 73: 2, 74: 2, 81: 3, 82: 2, 83: 3, 94: 2, 95: 3, 96: 2, 109: 2,
	117: 2, 136: 2,
}

// MigrateComplexWeapons is 2026_11_18_160000_complex_weapons_161. The levels of complex weapons the 1.61.7
// client bought were kept as shots (a positive count): the client took them for shots of level 1, and a battle
// used them up, to 0, which it leaves out of the arsenal (the sniper rifle, the ice bazooka). Each is made the
// level it was bought at (the top one at most), and one used up level 1.
func MigrateComplexWeapons(ctx context.Context, tx *sql.Tx) error {
	for id, top := range ComplexMaxLevel {
		for _, q := range []string{
			`update "wormix_users_weapons" set "count" = -1 where "weapon_id" = ? and "count" >= ?`,
			`update "wormix_users_weapons" set "count" = -10 - "count" where "weapon_id" = ? and "count" > 0 and "count" < ?`,
			`update "wormix_users_weapons" set "count" = -11 where "weapon_id" = ? and "count" = 0 and ? > 0`,
		} {
			if _, err := tx.ExecContext(ctx, q, id, top); err != nil {
				return err
			}
		}
	}
	return nil
}
