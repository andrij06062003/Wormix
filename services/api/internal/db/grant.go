package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/andrij06062003/Wormix/services/api/internal/php"
)

// migrateGrantOwner is 2026_11_08_180000_grant_owner_account: a one-off change of one account, done
// already. It stays registered (as a no-op) so the databases that ran it and those that didn't agree.
func migrateGrantOwner(ctx context.Context, tx *sql.Tx) error {
	return nil
}

// grantLogin is the account that 2026_11_10_130000_grant_owner_crafted_hats gives fuzes, rubies and the
// best of every crafted hat to (the server owner's request). Nothing happens if it doesn't exist.
const grantLogin = "andrii"

const (
	grantFuzes  = 100000 // money, added
	grantRubies = 20000  // real_money, added

	// The hat a worm wears (wormix_worms_data.hat) for a crafted hat: craftedHatBase + race*craftedHatRace +
	// (hat - craftedHatStart), as in the app package.
	grantCraftedHatStart = 1500
	grantCraftedHatBase  = 20000
	grantCraftedHatRace  = 200
	grantArtifactStart   = 2000
)

// migrateGrantOwnerHats adds the fuzes and rubies above to grantLogin and gives it the best item (last
// prefix) of every crafted hat family of crafted_equip.json. A family's hat it has already is made the best
// one, the worn hat too.
func migrateGrantOwnerHats(ctx context.Context, tx *sql.Tx) error {
	var uid int64
	err := tx.QueryRowContext(ctx, `select "id" from "users" where "login" = ?`, grantLogin).Scan(&uid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	now := Now()
	if _, err := tx.ExecContext(ctx, `update "wormix_user_profiles" set "money" = "money" + ?, "real_money" = "real_money" + ?, "updated_at" = ? where "user_id" = ?`,
		int64(grantFuzes), int64(grantRubies), now, uid); err != nil {
		return err
	}
	cfg, ok := loadJSON("crafted_equip.json").(*php.Arr)
	if !ok {
		return fmt.Errorf("crafted_equip.json: invalid")
	}
	fams, ok := cfg.Val("families").(*php.Arr)
	if !ok {
		return fmt.Errorf("crafted_equip.json: no families")
	}
	var wornHat int64
	if err := tx.QueryRowContext(ctx, `select "hat" from "wormix_worms_data" where "owner_id" = ?`, uid).Scan(&wornHat); err != nil &&
		!errors.Is(err, sql.ErrNoRows) {
		return err
	}
	for _, x := range fams.Values() {
		f, ok := x.(*php.Arr)
		if !ok {
			continue
		}
		family := php.ToInt(f.Val("familyId"))
		prefixes, _ := f.Val("prefixes").(*php.Arr)
		if family >= grantArtifactStart || prefixes == nil || prefixes.Len() == 0 {
			continue
		}
		vals := prefixes.Values()
		best := family + php.ToInt(vals[len(vals)-1])
		var rowID, have int64
		err := tx.QueryRowContext(ctx, `select "id", "weapon_id" from "wormix_users_weapons" where "owner_id" = ? and "weapon_id" between ? and ? order by "weapon_id" desc limit 1`,
			uid, family, family+9).Scan(&rowID, &have)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			if _, err := tx.ExecContext(ctx, `insert into "wormix_users_weapons" ("owner_id", "weapon_id", "count", "expire_at", "created_at", "updated_at") values (?, ?, -1, -1, ?, ?)`,
				uid, best, now, now); err != nil {
				return err
			}
		case err != nil:
			return err
		case have != best:
			if _, err := tx.ExecContext(ctx, `update "wormix_users_weapons" set "weapon_id" = ?, "updated_at" = ? where "id" = ?`, best, now, rowID); err != nil {
				return err
			}
			if wornHat >= grantCraftedHatBase && grantCraftedHatStart+(wornHat-grantCraftedHatBase)%grantCraftedHatRace == have {
				wornHat += best - have
				if _, err := tx.ExecContext(ctx, `update "wormix_worms_data" set "hat" = ?, "updated_at" = ? where "owner_id" = ?`, wornHat, now, uid); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
