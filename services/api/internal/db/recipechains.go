package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// RecipeLink is a wormix_craft row's chain: the weapon an upgrade starts from and the one it makes.
type RecipeLink struct{ Prev, Upgrade int64 }

// upgradeStart is the first weapon id an upgrade makes: an id below it is a base weapon.
const upgradeStart = 300

// ConsistentRecipes keeps the crafted recipes the 1.61.7 client can take: one branch of each weapon (two
// recipes starting from the same weapon are two branches, as 313 -> 360 and 313 -> 361), each with the
// recipes before it down to the base weapon. The client puts a weapon's whole family in one arsenal cell
// and shows only the first member owned, so a second branch vanished when dragged there; and at login it
// walks each branch down to its base, forever when a recipe on the way is missing or taken by the other
// branch. The last crafted branch wins (the one the player wanted most recently); the others are dropped.
// Recipes unknown to wormix_craft are kept (the client leaves them out). Both lists keep the crafted order.
func ConsistentRecipes(crafted []int64, links map[int64]RecipeLink) (keep, drop []int64) {
	makes := map[int64]int64{} // upgrade weapon -> the crafted recipe making it
	for _, id := range crafted {
		if l, ok := links[id]; ok {
			if _, dup := makes[l.Upgrade]; !dup {
				makes[l.Upgrade] = id
			}
		}
	}
	kept := map[int64]bool{}
	startsKept := map[int64]int64{} // weapon -> the kept recipe starting from it
	for i := len(crafted) - 1; i >= 0; i-- {
		id := crafted[i]
		l, known := links[id]
		if !known {
			kept[id] = true
			continue
		}
		if kept[id] {
			continue
		}
		chain := []int64{id}
		ok := true
		for w := l.Prev; w >= upgradeStart; {
			p, found := makes[w]
			if !found || len(chain) > len(crafted) { // missing, or a loop in the table
				ok = false
				break
			}
			chain = append(chain, p)
			w = links[p].Prev
		}
		for _, r := range chain {
			if other, taken := startsKept[links[r].Prev]; ok && taken && other != r {
				ok = false
			}
		}
		if !ok {
			continue
		}
		for _, r := range chain {
			kept[r] = true
			startsKept[links[r].Prev] = r
		}
	}
	seen := map[int64]bool{}
	for _, id := range crafted {
		if kept[id] && !seen[id] {
			keep = append(keep, id)
		} else {
			drop = append(drop, id)
		}
		seen[id] = true
	}
	return keep, drop
}

// RecipeLinks reads wormix_craft's chains.
func RecipeLinks(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}) (map[int64]RecipeLink, error) {
	rows, err := q.QueryContext(ctx, `select "id", "prev_upgrade_id", "upgrade_id" from "wormix_craft" where "prev_upgrade_id" is not null and "upgrade_id" is not null`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	links := map[int64]RecipeLink{}
	for rows.Next() {
		var id int64
		var l RecipeLink
		if err := rows.Scan(&id, &l.Prev, &l.Upgrade); err != nil {
			return nil, err
		}
		links[id] = l
	}
	return links, rows.Err()
}

// MigrateRecipeBranches is 2026_11_18_170000_recipe_branches_161. The API let a player craft both branches
// of an upgrade (its check for the other branch compared the wrong column), and un-craft a recipe a later
// one starts from, which the 1.61.7 client can't take (ConsistentRecipes). Each profile keeps the recipes
// ConsistentRecipes keeps, and gets back the reagents of the ones dropped.
func MigrateRecipeBranches(ctx context.Context, tx *sql.Tx) error {
	links, err := RecipeLinks(ctx, tx)
	if err != nil {
		return err
	}
	costs := map[int64][][2]int64{}
	rows, err := tx.QueryContext(ctx, `select "id", "reagents" from "wormix_craft"`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int64
		var raw sql.NullString
		if err := rows.Scan(&id, &raw); err != nil {
			rows.Close()
			return err
		}
		var list [][]int64
		if json.Unmarshal([]byte(raw.String), &list) == nil {
			for _, r := range list {
				if len(r) >= 2 {
					costs[id] = append(costs[id], [2]int64{r[0], r[1]})
				}
			}
		}
	}
	rows.Close()

	type profile struct {
		user              int64
		recipes, reagents string
	}
	var profiles []profile
	rows, err = tx.QueryContext(ctx, `select "user_id", "recipes", "reagents" from "wormix_user_profiles"`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var p profile
		var rec, rea sql.NullString
		if err := rows.Scan(&p.user, &rec, &rea); err != nil {
			rows.Close()
			return err
		}
		p.recipes, p.reagents = rec.String, rea.String
		profiles = append(profiles, p)
	}
	rows.Close()

	for _, p := range profiles {
		var crafted []int64
		if json.Unmarshal([]byte(p.recipes), &crafted) != nil {
			continue
		}
		keep, drop := ConsistentRecipes(crafted, links)
		if len(drop) == 0 {
			continue
		}
		reagents := refundReagents(p.reagents, drop, costs)
		if keep == nil {
			keep = []int64{}
		}
		kb, _ := json.Marshal(keep)
		if _, err := tx.ExecContext(ctx, `update "wormix_user_profiles" set "recipes" = ?, "reagents" = ? where "user_id" = ?`, string(kb), reagents, p.user); err != nil {
			return err
		}
	}
	return nil
}

// refundReagents adds the reagents of the dropped recipes to a profile's reagents (a JSON list by reagent id,
// or an object by id).
func refundReagents(raw string, drop []int64, costs map[int64][][2]int64) string {
	var list []int64
	if raw == "" {
		raw = "[]"
	}
	if json.Unmarshal([]byte(raw), &list) != nil {
		var byID map[string]int64
		if json.Unmarshal([]byte(raw), &byID) != nil {
			return raw // not a shape it knows: left as it is rather than failing the migration
		}
		for _, id := range drop {
			for _, r := range costs[id] {
				byID[fmt.Sprint(r[0])] += r[1]
			}
		}
		b, _ := json.Marshal(byID)
		return string(b)
	}
	for _, id := range drop {
		for _, r := range costs[id] {
			for int64(len(list)) <= r[0] {
				list = append(list, 0)
			}
			list[r[0]] += r[1]
		}
	}
	b, _ := json.Marshal(list)
	return string(b)
}
