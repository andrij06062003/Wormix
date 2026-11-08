package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/andrij06062003/Wormix/services/api/internal/php"
)

// migrateCraftedEquip is 2026_11_08_120000_add_wormix_crafted_equip: inserts the crafted equipment of
// game/crafted_equip.json into wormix_weapons, one row per family and prefix (id = familyId + prefix),
// hidden in the shop, skipping ids that already exist.
func migrateCraftedEquip(ctx context.Context, tx *sql.Tx) error {
	cfg, ok := loadJSON("crafted_equip.json").(*php.Arr)
	if !ok {
		return fmt.Errorf("crafted_equip.json: invalid")
	}
	fams, ok := cfg.Val("families").(*php.Arr)
	if !ok {
		return fmt.Errorf("crafted_equip.json: no families")
	}
	now := Now()
	for _, x := range fams.Values() {
		f, ok := x.(*php.Arr)
		if !ok {
			continue
		}
		family := php.ToInt(f.Val("familyId"))
		prefixes, _ := f.Val("prefixes").(*php.Arr)
		if prefixes == nil {
			continue
		}
		for _, p := range prefixes.Values() {
			id := family + php.ToInt(p)
			var n int
			if err := tx.QueryRowContext(ctx, `select count(*) from "wormix_weapons" where "id" = ?`, id).Scan(&n); err != nil {
				return err
			}
			if n > 0 {
				continue
			}
			if _, err := tx.ExecContext(ctx, `insert into "wormix_weapons" ("id", "name", "hide_in_shop", "is_starter", "price", "real_price", "required_level", "required_rating", "infinity", "one_day", "created_at", "updated_at") values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				id, fmt.Sprintf("%s#%d", php.ToString(f.Val("name")), php.ToInt(p)), int64(1), int64(0), int64(0), int64(0), int64(0), int64(0),
				int64(1), int64(0), now, now); err != nil {
				return err
			}
		}
	}
	return nil
}
