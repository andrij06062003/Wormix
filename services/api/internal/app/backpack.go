package app

import (
	"encoding/json"
	"math"

	"github.com/andrij06062003/Wormix/services/api/internal/db"
)

// The arsenal: the layouts of the weapons panel the client saves (1.61.7: several, and which one is in use;
// 1.25.0: one) and its hotkeys. They are the client's own numbers (weapon ids and places), kept as they come
// (a JSON list of lists of shorts) and given back at login (enterAccount).

const (
	maxBackpackConfs  = 16  // layouts
	maxBackpackShorts = 512 // numbers in one layout, or hotkeys
)

// shortLists parses a JSON list of at most maxLists lists of 16-bit numbers.
func shortLists(s string, maxLists int) ([][]int64, bool) {
	var out [][]int64
	if json.Unmarshal([]byte(s), &out) != nil || out == nil || len(out) > maxLists {
		return nil, false
	}
	for _, l := range out {
		if !shorts(l) {
			return nil, false
		}
	}
	return out, true
}

func shorts(l []int64) bool {
	if len(l) > maxBackpackShorts {
		return false
	}
	for _, n := range l {
		if n < math.MinInt16 || n > math.MaxInt16 {
			return false
		}
	}
	return true
}

// setBackpack saves the arsenal's layouts and the one in use: Result 0, or 1 for ones it can't keep.
func (c *Ctx) setBackpack() Resp {
	c.Validate([]R{{"internal_user_id", "required|integer|exists:users,id"}, {"Configs", "required|string|max:20000"},
		{"Active", "required|integer|min:0|max:127"}})
	confs, ok := shortLists(php2string(c.JSON("Configs")), maxBackpackConfs)
	if !ok {
		return dataResult("Result", int64(1))
	}
	b, _ := json.Marshal(confs)
	b = []byte(oneCellPerFamily(string(b), c.weaponFamilies()))
	c.exec(`update "wormix_user_profiles" set "backpack_confs" = ?, "active_backpack_conf" = ? where "user_id" = ?`,
		string(b), c.JSON("Active"), c.UserID())
	return dataResult("Result", int64(0))
}

// setHotkeys saves the arsenal's hotkeys (no answer to the client).
func (c *Ctx) setHotkeys() Resp {
	c.Validate([]R{{"internal_user_id", "required|integer|exists:users,id"}, {"Hotkeys", "required|string|max:5000"}})
	var keys []int64
	if json.Unmarshal([]byte(php2string(c.JSON("Hotkeys"))), &keys) != nil || keys == nil || !shorts(keys) {
		return dataResult("Result", int64(1))
	}
	b, _ := json.Marshal(keys)
	c.exec(`update "wormix_user_profiles" set "hotkeys" = ? where "user_id" = ?`, string(b), c.UserID())
	return dataResult("Result", int64(0))
}

func php2string(v any) string {
	s, _ := v.(string)
	return s
}

// weaponFamilies maps each upgrade of wormix_craft to its base weapon: the 1.61.7 client keeps one arsenal
// cell per family (a base weapon and all its upgrades), under the base's id.
func (c *Ctx) weaponFamilies() map[int64]int64 {
	links, err := db.RecipeLinks(c.ctx, c.tx)
	if err != nil {
		panic(err)
	}
	prev := map[int64]int64{}
	for _, l := range links {
		prev[l.Upgrade] = l.Prev
	}
	base := map[int64]int64{}
	for up := range prev {
		w := up
		for i := 0; i < len(prev); i++ { // a loop in the table ends here
			p, ok := prev[w]
			if !ok {
				break
			}
			w = p
		}
		base[up] = w
	}
	return base
}

// oneCellPerFamily empties the arsenal cells whose weapon's family has a cell before them in the same layout.
// A layout saved while the client split a family in two (BuildFamilies.pcode) has a cell for each part; the
// client now reads both as the base weapon and shows it twice.
func oneCellPerFamily(raw string, base map[int64]int64) string {
	var confs [][]int64
	if json.Unmarshal([]byte(raw), &confs) != nil {
		return raw
	}
	changed := false
	for _, l := range confs {
		seen := map[int64]bool{}
		for i, id := range l {
			if id <= 0 {
				continue
			}
			root := id
			if b, ok := base[id]; ok {
				root = b
			}
			if seen[root] {
				l[i] = 0
				changed = true
			}
			seen[root] = true
		}
	}
	if !changed {
		return raw
	}
	b, _ := json.Marshal(confs)
	return string(b)
}
