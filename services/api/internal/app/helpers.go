package app

import (
	"math/rand/v2"
	"time"

	"github.com/andrij06062003/Wormix/services/api/internal/php"
)

// Constants from config/wormix.php and the helpers.
const (
	raceBase        = 500
	raceLimit       = 50
	stuffStartIndex = 1000
	// Artifacts (from 1.25.0) are items 2001..2999; selecting artifactStart takes the artifact off.
	artifactStart = 2000
	artifactEnd   = 3000
	botBase       = -1000

	missionsDelay = 120
	missionsMax   = 5
	searchKeysDay = 10
)

// Crafted hats (1.25.0) are items 1500..1699: race*500 + id doesn't fit them, so a worm's hat is
// craftedHatBase + race*craftedHatRace + (id - craftedHatStart) for those (it stays under 32767).
const (
	craftedHatStart = 1500
	craftedHatBase  = 20000
	craftedHatRace  = 200
)

// phpRand is rand($min, $max) (bounds swapped like PHP when max < min).
func phpRand(min, max int64) int64 {
	if max < min {
		min, max = max, min
	}
	return min + rand.Int64N(max-min+1)
}

// raceAndHat is WormixTrashHelper::getRaceAndHatIds.
func raceAndHat(hat int64) (race, hatID int64) {
	if hat < raceLimit {
		return hat, 0
	}
	if hat >= craftedHatBase {
		return (hat - craftedHatBase) / craftedHatRace, craftedHatStart + (hat-craftedHatBase)%craftedHatRace
	}
	return php.ToInt(php.Div(hat-stuffStartIndex, int64(raceBase))), stuffStartIndex + hat%raceBase
}

// hatByRaceAndHat is WormixTrashHelper::getHatByRaceAndHatIds.
func hatByRaceAndHat(hatID, raceID int64) int64 {
	if hatID == 0 {
		return raceID
	}
	if hatID >= craftedHatStart && hatID < artifactStart {
		return craftedHatBase + raceID*craftedHatRace + hatID - craftedHatStart
	}
	return raceID*raceBase + hatID
}

// stripData is WormixBotHelper::stripData.
func stripData(level, toLevel, armor, attack int64) (int64, int64) {
	levelPoints := level * 2
	toLevelPoints := toLevel * 2
	if armor+attack < levelPoints {
		armor += levelPoints - (armor + attack)
	}
	armorPoints := min(toLevelPoints, max(0, php.ToInt(php.Mul(php.Div(armor, levelPoints), toLevelPoints))))
	return armorPoints, toLevelPoints - armorPoints
}

func subDay(days int) string {
	return time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")
}

func today(offsetDays int) string {
	return time.Now().UTC().AddDate(0, 0, -offsetDays).Format("2006-01-02")
}

// searchKeys is WormixTrashHelper::getSearchKeys.
func (c *Ctx) searchKeys(userID any) int64 {
	n := c.Count(`select count(*) as aggregate from "wormix_house_actions" where "action_type" = ? and "user_id" = ? and "created_at" >= ?`,
		int64(1), php.IntArg(userID), subDay(1))
	return max(0, searchKeysDay-n)
}

func (c *Ctx) isSearchedToday(userID, toUserID any) bool {
	return c.Count(`select count(*) as aggregate from "wormix_house_actions" where "user_id" = ? and "to_user_id" = ? and "action_type" = ? and "created_at" >= ?`,
		php.IntArg(userID), php.IntArg(toUserID), int64(1), subDay(1)) > 0
}

// addReagents is WormixTrashHelper::addReagents.
func (c *Ctx) addReagents(profile *Model, reagents *php.Arr) {
	if reagents.Len() == 0 {
		return
	}
	user, _ := profile.Arr("reagents").(*php.Arr)
	if user == nil {
		user = php.NewArr()
		if profile.Arr("reagents") != nil {
			php.Fail("count(): Argument #1 must be of type Countable|array")
		}
	}
	var maxR any
	for i, r := range reagents.Values() {
		if i == 0 || php.Gt(r, maxR) {
			maxR = r
		}
	}
	if php.Gt(php.Add(maxR, int64(1)), int64(user.Len())) {
		old := user
		user = php.NewArr()
		n := php.ToInt(php.Add(maxR, int64(1)))
		for i := int64(0); i < n; i++ {
			user.Set(i, int64(0))
		}
		cnt := int64(old.Len())
		for i := int64(0); i < cnt; i++ {
			v, ok := old.Get(i)
			if !ok {
				php.Fail("Undefined array key %d", i)
			}
			user.Set(i, v)
		}
	}
	for _, r := range reagents.Values() {
		v, ok := user.Get(r)
		if !ok {
			php.Fail("Undefined array key")
		}
		user.Set(r, php.Add(v, int64(1)))
	}
	profile.Set("reagents", user)
	c.Save(profile)
}

// addWeaponsAwards is WormixTrashHelper::addWeaponsAwards.
func (c *Ctx) addWeaponsAwards(awards *php.Arr, worm *Model) {
	if awards == nil || awards.Len() == 0 {
		return
	}
	owner := worm.Get("owner_id")
	var ids []any
	for _, a := range awards.Values() {
		ids = append(ids, idx(a, 0))
	}
	owned := c.Values(`select "weapon_id" from "wormix_users_weapons" where "owner_id" = ? and "weapon_id" in (`+placeholders(len(ids))+`) and "count" != ?`,
		append(append([]any{owner}, ids...), "-1")...)
	q := `select "id" from "wormix_weapons" where "id" in (` + placeholders(len(ids)) + `)`
	args := append([]any{}, ids...)
	if len(owned) > 0 {
		q += ` and "id" not in (` + placeholders(len(owned)) + `)`
		args = append(args, owned...)
	}
	weaponIDs := php.List(c.Values(q, args...)...)
	for _, a := range awards.Values() {
		if !php.InArray(idx(a, 0), weaponIDs, false) {
			continue
		}
		uw := c.First(defUserWeapon, `where "owner_id" = ? and "weapon_id" = ?`, owner, idx(a, 0))
		if uw == nil {
			uw = New(defUserWeapon)
		}
		uw.Set("owner_id", owner)
		uw.Set("weapon_id", idx(a, 0))
		uw.Set("count", idx(a, 1))
		if php.Ge(idx(a, 0), int64(stuffStartIndex)) {
			uw.Set("expire_at", time.Now().Unix()+24*60*60)
		}
		c.Save(uw)
	}
}

// idx is $a[$i] on a decoded array.
func idx(a any, k any) any {
	arr, ok := a.(*php.Arr)
	if !ok {
		if a == nil {
			return nil
		}
		php.Fail("Cannot use a scalar value as an array")
	}
	v, ok := arr.Get(k)
	if !ok {
		php.Fail("Undefined array key %v", k)
	}
	return v
}

// levelModel is WormData::level_model (cached on the model like Eloquent does).
func (c *Ctx) levelModel(w *Model) *Model {
	return w.rel("level_model", func() *Model {
		return c.First(defLevel, `where "wormix_levels"."id" = ? and "wormix_levels"."id" is not null limit 1`, w.Get("level"))
	})
}

// saveWorm saves WormData and runs WormDataObserver::updated.
func (c *Ctx) saveWorm(w *Model) {
	if !c.Save(w) {
		w.syncOriginal()
		return
	}
	lm := must(c.levelModel(w))
	if php.Lt(w.Get("experience"), lm.Get("required_experience")) {
		w.syncOriginal()
		return
	}
	if php.StrictEq(w.Get("level"), int64(30)) {
		w.Set("experience", lm.Get("required_experience"))
		c.SaveAndSync(w)
		return
	}
	w.Set("level", php.Add(w.Get("level"), int64(1)))
	w.Set("experience", php.Sub(w.Get("experience"), lm.Get("required_experience")))
	c.SaveAndSync(w)
	newLevel := must(c.First(defLevel, `where "id" = ?`, w.Get("level")))
	aw, _ := newLevel.Arr("awards").(*php.Arr)
	if newLevel.Arr("awards") == nil {
		php.Fail("addWeaponsAwards(): Argument #1 ($awards) must be of type array, null given")
	}
	c.addWeaponsAwards(aw, w)
	owner := must(w.rel("owner", func() *Model { return c.First(defUser, `where "users"."id" = ? limit 1`, w.Get("owner_id")) }))
	p := must(c.userProfile(owner))
	p.Set("money", php.Add(p.Get("money"), int64(150)))
	c.Save(p)
}

// ---- resources ----

// userProfileStructure is UserProfileStructure.
func (c *Ctx) userProfileStructure(p *Model) *php.Arr {
	uid := p.Get("user_id")
	weapons := c.All(defUserWeapon, `where "wormix_users_weapons"."owner_id" = ? and "wormix_users_weapons"."owner_id" is not null`, uid)
	wl := php.NewArr()
	for _, w := range weapons {
		wl.Append(php.Assoc("Id", w.Get("weapon_id"), "Count", w.Get("count")))
	}
	team := c.All(defTeam, `where "wormix_users_teams"."user_id" = ? and "wormix_users_teams"."user_id" is not null order by "order" asc`, uid)
	wg := php.NewArr()
	for _, t := range team {
		wg.Append(c.wormStructure(t))
	}
	stuff := php.List(c.Values(`select "weapon_id" from "wormix_users_weapons" where "wormix_users_weapons"."owner_id" = ? and "wormix_users_weapons"."owner_id" is not null and "weapon_id" > ?`, uid, int64(1000))...)
	return php.Assoc(
		"Id", uid,
		"SocialId", php.ToString(uid),
		// never negative: the game server reads these as unsigned numbers
		"Money", max(0, php.ToInt(p.Get("money"))),
		"RealMoney", max(0, php.ToInt(p.Get("real_money"))),
		"ReactionRate", p.Get("reaction_rate"),
		"Rating", p.Get("rating"),
		"WeaponRecordList", wl,
		"WormsGroup", wg,
		"Recipes", p.Arr("recipes"),
		"Stuff", stuff,
	)
}

// wormStructure is WormStructure for a team row.
func (c *Ctx) wormStructure(t *Model) *php.Arr {
	mate := must(c.First(defWorm, `where "wormix_worms_data"."owner_id" = ? limit 1`, t.Get("teammate_id")))
	armor, attack, level := mate.Get("armor"), mate.Get("attack"), mate.Get("level")
	if !php.StrictEq(t.Get("user_id"), t.Get("teammate_id")) {
		owner := must(c.First(defWorm, `where "wormix_worms_data"."owner_id" = ? limit 1`, t.Get("user_id")))
		if !php.StrictEq(owner.Get("level"), mate.Get("level")) {
			a, b := stripData(php.IntArg(level), php.IntArg(owner.Get("level")), php.IntArg(armor), php.IntArg(attack))
			level, armor, attack = owner.Get("level"), a, b
		}
	}
	return php.Assoc(
		"OwnerId", t.Get("teammate_id"),
		"SocialOwnerId", php.ToString(t.Get("teammate_id")),
		"Armor", armor,
		"Attack", attack,
		"Experience", mate.Get("experience"),
		"Level", level,
		"Hat", mate.Get("hat"),
		"Artifact", php.ToInt(mate.Get("artifact")),
		"Inactive", php.ToBool(t.Get("inactive")),
	)
}

// generateBots is WormixBotHelper::GenerateBots.
func (c *Ctx) generateBots(worm *Model) *php.Arr {
	base := c.Count(`select count("id") as aggregate from "users"`)
	owner := must(c.First(defUser, `where "users"."id" = ? limit 1`, worm.Get("owner_id")))
	profile := must(c.userProfile(owner))
	level := worm.Get("level")
	stuff := c.Values(`select "id" from "wormix_weapons" where "id" > ? and "id" < ? and "required_level" <= ? and "required_rating" <= ? and "hide_in_shop" = ? and "price" != ?`,
		int64(stuffStartIndex), int64(artifactStart), level, profile.Get("rating"), int64(0), int64(0))
	races := c.Values(`select "race_id" from "wormix_races" where "required_level" <= ?`, level)
	bi := must(c.userBattleInfo(owner))
	lvl := php.IntArg(level)
	count := 4
	if lvl >= 5 || php.Lt(bi.Get("mission_id"), int64(-2)) || php.Ge(bi.Get("mission_id"), int64(0)) {
		count = 6
	}
	bots := php.NewArr()
	for i := int64(0); i < int64(count); i++ {
		group := php.NewArr()
		rl := phpRand(max(1, lvl-1), min(lvl+1, 30))
		lm := must(c.First(defLevel, `where "id" = ?`, rl))
		n := phpRand(1, php.IntArg(lm.Get("max_worms_count")))
		var hat any = int64(0)
		for j := int64(0); j < n; j++ {
			armor := phpRand(0, rl*2)
			hat = int64(0)
			if len(stuff) > 0 {
				hat = stuff[rand.IntN(len(stuff))]
			}
			if phpRand(0, 10)%5 == 0 {
				hat = int64(0)
			}
			var race any = int64(0)
			if len(races) > 0 {
				race = races[rand.IntN(len(races))]
			}
			group.Append(php.Assoc(
				"Armor", armor,
				"Attack", rl*2-armor,
				"Experience", int64(0),
				"Level", rl,
				"Hat", hatByRaceAndHat(php.IntArg(hat), php.IntArg(race)),
				"OwnerId", base+(i+j+1)*10,
				"SocialOwnerId", php.ToString(0-base+botBase-(i+1+j)*10),
			))
		}
		st := php.NewArr()
		if !php.LooseEq(hat, int64(0)) {
			st.Append(hat)
		}
		bots.Append(php.Assoc(
			"Id", base+(i+1)*10,
			"Rating", phpRand(0, php.IntArg(profile.Get("rating"))),
			// never negative (a negative balance would give rand(0, -n)): the game server reads these as unsigned
			"Money", phpRand(0, max(0, php.IntArg(profile.Get("money")))),
			"RealMoney", phpRand(0, max(0, php.IntArg(profile.Get("real_money")))),
			"Recipes", php.NewArr(),
			"SocialId", php.ToString(0-base+botBase-(i+1)*10),
			"Stuff", st,
			"WeaponRecordList", php.List(php.Assoc("Id", int64(1), "Count", int64(-1))),
			"WormsGroup", group,
		))
	}
	return bots
}
