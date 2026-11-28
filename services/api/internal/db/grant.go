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

// migrateGrantOwnerHats is 2026_11_10_130000_grant_owner_crafted_hats: another one-off change of one
// account (fuzes, rubies and crafted hats), done already. A no-op for the same reason.
func migrateGrantOwnerHats(ctx context.Context, tx *sql.Tx) error {
	return nil
}

// grantUserID is the account (by id) that 2026_11_18_130000_grant_owner_161 gives the top level of every
// crafted hat and artifact of the 1.61.7 client and every achievement at its top level (the server
// owner's request). Nothing happens if it doesn't exist.
const grantUserID = 4

// The hat a worm wears (wormix_worms_data.hat) for a crafted hat, as in the app package: craftedHatBase or,
// from grantCraftedHat2Start, craftedHat2Base, + race*craftedHatRace + the hat's place in its range.
const (
	grantCraftedHatStart  = 1500
	grantCraftedHatBase   = 20000
	grantCraftedHat2Start = 1700
	grantCraftedHat2Base  = 26000
	grantCraftedHatRace   = 200
)

// grantWornHat packs a crafted hat for a race (0 if it isn't one).
func grantWornHat(race, hat int64) int64 {
	switch {
	case hat >= grantCraftedHat2Start && hat < 2000:
		return grantCraftedHat2Base + race*grantCraftedHatRace + hat - grantCraftedHat2Start
	case hat >= grantCraftedHatStart && hat < grantCraftedHat2Start:
		return grantCraftedHatBase + race*grantCraftedHatRace + hat - grantCraftedHatStart
	}
	return 0
}

// grantSplitHat unpacks a worn crafted hat: its race and hat (0, 0 if it isn't one).
func grantSplitHat(worn int64) (race, hat int64) {
	switch {
	case worn >= grantCraftedHat2Base:
		return (worn - grantCraftedHat2Base) / grantCraftedHatRace, grantCraftedHat2Start + (worn-grantCraftedHat2Base)%grantCraftedHatRace
	case worn >= grantCraftedHatBase:
		return (worn - grantCraftedHatBase) / grantCraftedHatRace, grantCraftedHatStart + (worn-grantCraftedHatBase)%grantCraftedHatRace
	}
	return 0, 0
}

// migrateGrantOwner161 gives grantUserID the top level (best of crafted_equip161.json) of every crafted hat
// and artifact family of the 1.61.7 client: a family's item it has already becomes the top one, worn or not.
// Every achievement of achievements.json is set to its top level.
func migrateGrantOwner161(ctx context.Context, tx *sql.Tx) error {
	uid := int64(grantUserID)
	var n int
	if err := tx.QueryRowContext(ctx, `select count(*) from "users" where "id" = ?`, uid).Scan(&n); err != nil || n == 0 {
		return err
	}
	now := Now()
	cfg, ok := loadJSON("crafted_equip161.json").(*php.Arr)
	if !ok {
		return fmt.Errorf("crafted_equip161.json: invalid")
	}
	best, ok := cfg.Val("best").(*php.Arr)
	if !ok {
		return fmt.Errorf("crafted_equip161.json: no best")
	}
	var wornHat, wornArtifact int64
	if err := tx.QueryRowContext(ctx, `select "hat", "artifact" from "wormix_worms_data" where "owner_id" = ?`, uid).Scan(&wornHat, &wornArtifact); err != nil &&
		!errors.Is(err, sql.ErrNoRows) {
		return err
	}
	for _, x := range best.Values() {
		f, ok := x.(*php.Arr)
		if !ok {
			continue
		}
		family := php.ToInt(f.Val("familyId"))
		top := family + php.ToInt(f.Val("best"))
		var rowID, have int64
		err := tx.QueryRowContext(ctx, `select "id", "weapon_id" from "wormix_users_weapons" where "owner_id" = ? and "weapon_id" between ? and ? order by "weapon_id" desc limit 1`,
			uid, family, family+9).Scan(&rowID, &have)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			if _, err := tx.ExecContext(ctx, `insert into "wormix_users_weapons" ("owner_id", "weapon_id", "count", "expire_at", "created_at", "updated_at") values (?, ?, -1, -1, ?, ?)`,
				uid, top, now, now); err != nil {
				return err
			}
			continue
		case err != nil:
			return err
		case have == top:
			continue
		}
		if _, err := tx.ExecContext(ctx, `update "wormix_users_weapons" set "weapon_id" = ?, "count" = -1, "expire_at" = -1, "updated_at" = ? where "id" = ?`, top, now, rowID); err != nil {
			return err
		}
		if race, hat := grantSplitHat(wornHat); hat == have {
			wornHat = grantWornHat(race, top)
		}
		if wornArtifact == have {
			wornArtifact = top
		}
	}
	if _, err := tx.ExecContext(ctx, `update "wormix_worms_data" set "hat" = ?, "artifact" = ?, "updated_at" = ? where "owner_id" = ?`,
		wornHat, wornArtifact, now, uid); err != nil {
		return err
	}
	ach, ok := loadJSON("achievements.json").(*php.Arr)
	if !ok {
		return fmt.Errorf("achievements.json: invalid")
	}
	achs, ok := ach.Val("achievements").(*php.Arr)
	if !ok {
		return fmt.Errorf("achievements.json: no achievements")
	}
	for _, id := range achs.Keys() {
		a, _ := achs.Val(id).(*php.Arr)
		if a == nil {
			continue
		}
		levels, _ := a.Val("levels").(*php.Arr)
		if levels == nil || levels.Len() == 0 {
			continue
		}
		last, _ := levels.Val(levels.Keys()[levels.Len()-1]).(*php.Arr)
		if last == nil {
			continue
		}
		if _, err := tx.ExecContext(ctx, `insert into "wormix_users_achievements" ("user_id", "achievement_id", "value", "created_at", "updated_at") values (?, ?, ?, ?, ?)
			on conflict ("user_id", "achievement_id") do update set "value" = excluded."value", "updated_at" = excluded."updated_at"`,
			uid, php.ToInt(id), php.ToInt(last.Val("max")), now, now); err != nil {
			return err
		}
	}
	return nil
}
