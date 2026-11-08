package app

import (
	"encoding/json"
	"math/rand/v2"
	"strconv"
	"sync"

	"github.com/andrij06062003/Wormix/services/api/internal/assets"
	"github.com/andrij06062003/Wormix/services/api/internal/php"
)

// Crafted equipment (1.25.0): a hat or artifact made from other items and badges. Its id is
// familyId + prefix; the prefix is the rarity, rolled when it is made. Making it again ("remake")
// rolls a new prefix and replaces the old item.

type craftCost struct {
	Price     int64      `json:"price"`
	RealPrice int64      `json:"realprice"`
	Items     [][2]int64 `json:"items"` // [id, count]: ids from stuffStartIndex are items, lower ids reagents
}

type craftFamily struct {
	FamilyID int64      `json:"familyId"`
	Name     string     `json:"name"`
	Craft    *craftCost `json:"craft"`
	Remake   *craftCost `json:"remake"`
	Prefixes []int64    `json:"prefixes"`
}

type craftConfig struct {
	ReagentPrices map[string]int64 `json:"reagentPrices"`
	Families      []craftFamily    `json:"families"`
}

var (
	craftOnce sync.Once
	craftCfg  craftConfig
)

func craftedEquip() *craftConfig {
	craftOnce.Do(func() {
		b, err := assets.Game("crafted_equip.json")
		if err != nil {
			panic(err)
		}
		if err := json.Unmarshal(b, &craftCfg); err != nil {
			panic(err)
		}
	})
	return &craftCfg
}

func (cfg *craftConfig) family(id int64) *craftFamily {
	for i := range cfg.Families {
		if cfg.Families[i].FamilyID == id {
			return &cfg.Families[i]
		}
	}
	return nil
}

// prefixWeights: the chance of each prefix, best last (the rarer the better). A family has 4 to 7.
var prefixWeights = []int64{40, 25, 15, 10, 6, 3, 1}

// rollPrefix picks a prefix of the family, weighted towards the common ones.
func rollPrefix(f *craftFamily, roll func(n int64) int64) int64 {
	var total int64
	for i := range f.Prefixes {
		total += prefixWeights[min(i, len(prefixWeights)-1)]
	}
	r := roll(total)
	for i, p := range f.Prefixes {
		r -= prefixWeights[min(i, len(prefixWeights)-1)]
		if r < 0 {
			return p
		}
	}
	return f.Prefixes[len(f.Prefixes)-1]
}

// craftRoll is the random source of rollPrefix (tests replace it).
var craftRoll = func(n int64) int64 { return rand.Int64N(n) }

// Results of assembleEquip (AssembleEquipResult.result: 0 is success).
const (
	assembleOK = 0

	// assembleError: unknown family, missing items, or the best one is made already.
	assembleError = 1

	assembleNoMoney = 3

	// assembleNoReagents: not enough reagents when paying with fuzy.
	assembleNoReagents = 4
)

// assembleEquip is craft/assemble_equip: make (or remake) the crafted equipment of a family.
// MoneyType 0 = rubies (missing reagents are bought with rubies), 1 = fuzy (all reagents needed).
func (c *Ctx) assembleEquip() Resp {
	c.Validate([]R{{"internal_user_id", "required|integer|exists:users,id"}, {"FamilyId", "required|integer"}, {"MoneyType", "required|integer|between:0,1"}})
	familyID := php.ToInt(c.JSON("FamilyId"))
	result := func(code, equipID int64) Resp {
		return dataResult("Result", code, "EquipId", equipID, "FamilyId", familyID)
	}
	cfg := craftedEquip()
	f := cfg.family(familyID)
	if f == nil || len(f.Prefixes) == 0 || f.Craft == nil {
		return result(assembleError, 0)
	}
	uid := c.UserID()

	// The family's item the player has now, if any: making it again is a remake.
	var current *Model
	for _, p := range f.Prefixes {
		if m := c.First(defUserWeapon, `where "owner_id" = ? and "weapon_id" = ? limit 1`, uid, f.FamilyID+p); m != nil {
			current = m
			break
		}
	}
	cost := f.Craft
	if current != nil {
		if php.ToInt(current.Get("weapon_id")) == f.FamilyID+f.Prefixes[len(f.Prefixes)-1] {
			return result(assembleError, 0) // the best one already
		}
		if f.Remake != nil {
			cost = f.Remake
		}
	}

	p := must(c.First(defProfile, `where "user_id" = ?`, uid))
	reagents, _ := p.Arr("reagents").(*php.Arr)
	if reagents == nil {
		reagents = php.NewArr()
	}
	have := func(id int64) int64 {
		v, ok := reagents.Get(id)
		if !ok {
			return 0
		}
		return php.ToInt(v)
	}
	realMoney := php.ToInt(p.Get("real_money"))
	money := php.ToInt(p.Get("money"))
	rubies := !php.LooseEq(c.JSON("MoneyType"), int64(1))

	var used []*Model // items used up
	var missingPrice int64
	for _, it := range cost.Items {
		id, n := it[0], it[1]
		if id >= stuffStartIndex {
			m := c.First(defUserWeapon, `where "owner_id" = ? and "weapon_id" = ? limit 1`, uid, id)
			if m == nil || !c.ownsValidItem(uid, id) {
				return result(assembleError, 0)
			}
			used = append(used, m)
			continue
		}
		if short := n - have(id); short > 0 {
			if !rubies {
				return result(assembleNoReagents, 0)
			}
			missingPrice += short * cfg.ReagentPrices[strconv.FormatInt(id, 10)]
		}
	}
	price := cost.Price
	if rubies {
		price = cost.RealPrice + int64(phpRound(float64(missingPrice)/100))
		if realMoney < price {
			return result(assembleNoMoney, 0)
		}
	} else if money < price {
		return result(assembleNoMoney, 0)
	}

	// Pay: money, reagents (what the player has of them), items.
	if rubies {
		p.Set("real_money", realMoney-price)
	} else {
		p.Set("money", money-price)
	}
	for _, it := range cost.Items {
		if id, n := it[0], it[1]; id < stuffStartIndex {
			for int64(reagents.Len()) <= id {
				reagents.Append(int64(0))
			}
			reagents.Set(id, max(have(id)-n, 0))
		}
	}
	p.Set("reagents", reagents)
	c.Save(p)

	worm := must(c.First(defWorm, `where "owner_id" = ?`, uid))
	race, hat := raceAndHat(php.ToInt(worm.Get("hat")))
	artifact := php.ToInt(worm.Get("artifact"))
	for _, m := range used {
		id := php.ToInt(m.Get("weapon_id"))
		if id == hat {
			hat = 0
		}
		if id == artifact {
			artifact = 0
		}
		c.Delete(m)
	}

	// The new item, worn at once (as the client shows it).
	equipID := f.FamilyID + rollPrefix(f, craftRoll)
	if current != nil {
		c.Delete(current)
	}
	nw := New(defUserWeapon)
	nw.Set("owner_id", uid)
	nw.Set("weapon_id", equipID)
	nw.Set("count", int64(-1))
	c.Save(nw)
	if equipID >= artifactStart {
		artifact = equipID
	} else {
		hat = equipID
	}
	worm.Set("hat", hatByRaceAndHat(hat, race))
	worm.Set("artifact", artifact)
	c.saveWorm(worm)
	return result(assembleOK, equipID)
}
