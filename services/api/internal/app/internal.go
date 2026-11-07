package app

import (
	"math"

	"github.com/andrij06062003/Wormix/services/api/internal/db"
	"math/rand/v2"
	"time"

	"github.com/andrij06062003/Wormix/services/api/internal/php"
)

func dataResult(kv ...any) Resp { return jsonResp(200, php.Assoc("data", php.Assoc(kv...))) }

// catchPHP runs f and converts a PHP runtime error into ok=false (a try/catch(\Exception)).
func catchPHP(f func()) (ok bool) {
	defer func() {
		if p := recover(); p != nil {
			if _, isPHP := p.(*php.Error); isPHP {
				ok = false
				return
			}
			panic(p)
		}
	}()
	f()
	return true
}

// ---- InternalLoginController ----

func (c *Ctx) internalLogin() Resp {
	c.Validate([]R{{"tcp_session", "required|uuid"}, {"Id", "required|integer"}, {"AuthKey", "required|string"}},
		M{"tcp_session.required", "TCP session is required."},
		M{"tcp_session.uuid", "TCP session must be is GUID."},
		M{"Id.required", "id is required"},
		M{"Id.integer", "id must be an integer"},
		M{"AuthKey.required", "auth_key is required"},
		M{"AuthKey.string", "auth_key must be a string"},
	)
	loginError := func(code int64) Resp {
		return jsonResp(200, php.Assoc("type", "LoginError", "data", php.Assoc("Result", code)))
	}
	s := c.loadSession(php.IntArg(c.JSON("Id")))
	if s == nil {
		return loginError(0)
	}
	if !php.LooseEq(s.authKey(), c.JSON("AuthKey")) || s.isLoggedIn() {
		return loginError(1)
	}
	var old, key string
	if !catchPHP(func() {
		old = s.tcpSession()
		s.setTcpSession(php.ToString(c.JSON("tcp_session")))
		c.loginEvent(s.user)
		s.loggedIn()
		key = php.ToString(c.JSON("Id")) + "." + s.setSessionKey()
	}) {
		return loginError(0)
	}
	return jsonResp(200, php.Assoc("type", "EnterAccount", "old_session", old, "data", c.enterAccount(s.user, key)))
}

// loginEvent is LoginEventListener::handleInternalLoginEvent.
func (c *Ctx) loginEvent(u *Model) {
	ls := must(c.userLoginSeq(u))
	newAccount := ls.Get("last_login") == nil || ls.Get("last_login") == ""
	now := time.Now().UTC()
	old := now
	if !newAccount {
		t, err := asDateTime(ls.Get("last_login"))
		if err != nil {
			php.Fail("%v", err)
		}
		old = t
	}
	days := int64(math.Max(0, math.Trunc(now.Sub(old).Hours()/24)))
	if days == 1 {
		ls.Set("login_sequence", php.Add(ls.Get("login_sequence"), int64(1)))
	} else if days > 1 {
		ls.Set("login_sequence", int64(1))
	}
	if php.Gt(ls.Get("login_sequence"), int64(5)) {
		ls.Set("login_sequence", int64(1))
	}
	ls.Set("last_login", now.Format("2006-01-02"))
	ls.Set("gift_accepted", false)
	if days == 0 && !newAccount {
		ls.Set("gift_accepted", true)
		c.SaveAndSync(ls)
	} else {
		c.saveLoginSeq(ls)
	}

	worm := must(c.userWorm(u))
	race, hatID := raceAndHat(php.IntArg(worm.Get("hat")))
	cur := c.First(defUserWeapon, `where "owner_id" = ? and "weapon_id" = ?`, u.Get("id"), hatID)
	expired := cur == nil || (php.ToInt(cur.Get("expire_at")) != -1 && php.Lt(cur.Get("expire_at"), time.Now().Unix()))
	if hatID != 0 && expired {
		worm.Set("hat", hatByRaceAndHat(0, race))
		c.saveWorm(worm)
	}
	// Take the artifact off when it's gone or expired, like the hat.
	if art := php.ToInt(worm.Get("artifact")); art != 0 && !c.ownsValidItem(u.Get("id"), art) {
		worm.Set("artifact", int64(0))
		c.saveWorm(worm)
	}
	bi := c.First(defBattleInfo, `where "user_id" = ?`, u.Get("id"))
	if php.Gt(worm.Get("level"), int64(5)) && php.Lt(must(bi).Get("last_mission_id"), int64(0)) {
		bi.Set("last_mission_id", int64(0))
		c.Save(bi)
	}
	for _, id := range c.Values(`select "id" from "wormix_users_weapons" where "expire_at" != ? and "expire_at" < ?`, int64(-1), time.Now().Unix()) {
		c.exec(`delete from "wormix_users_weapons" where "id" = ?`, id)
	}
}

// saveLoginSeq saves LoginSequence and runs LoginSequenceObserver::updated.
func (c *Ctx) saveLoginSeq(ls *Model) {
	if !c.Save(ls) {
		ls.syncOriginal()
		return
	}
	uid := ls.Get("user_id")
	p := c.First(defProfile, `where "user_id" = ?`, uid)
	bonus := c.First(defDailyBonus, `where "login_sequence" = ?`, ls.Get("login_sequence"))
	bi := c.First(defBattleInfo, `where "user_id" = ?`, uid)
	if p == nil || bonus == nil || bi == nil {
		ls.syncOriginal()
		return
	}
	amount, typ := bonus.Get("bonus_value"), bonus.Get("bonus_type")
	if bonus.Bool("random_gift") {
		amount = phpRand(php.IntArg(bonus.Get("rand_min")), php.IntArg(bonus.Get("rand_max")))
		typ = phpRand(1, 4)
	}
	switch {
	case php.LooseEq(typ, int64(1)):
		p.Set("money", php.Add(p.Get("money"), amount))
		c.Save(p)
	case php.LooseEq(typ, int64(2)):
		p.Set("real_money", php.Add(p.Get("real_money"), amount))
		c.Save(p)
	case php.LooseEq(typ, int64(3)):
		bi.Set("battles_count", php.Add(bi.Get("battles_count"), amount))
		c.Save(bi)
	case php.LooseEq(typ, int64(4)):
		p.Set("reaction_rate", php.Add(p.Get("reaction_rate"), amount))
		c.Save(p)
	}
	ls.Set("bonus_count", amount)
	ls.Set("bonus_type", typ)
	c.SaveAndSync(ls)
}

// enterAccount is the EnterAccount resource.
func (c *Ctx) enterAccount(u *Model, sessionKey string) *php.Arr {
	id := u.Get("id")
	profile := must(c.userProfile(u))
	others := php.NewArr()
	for _, p := range c.All(defProfile, `where "user_id" != ?`, id) {
		others.Append(c.userProfileStructure(p))
	}
	ls := must(c.userLoginSeq(u))
	daily := php.Assoc("LoginSequence", int64(0), "DailyBonusType", int64(0), "DailyBonusCount", int64(0))
	if !ls.Bool("gift_accepted") {
		daily = php.Assoc("LoginSequence", ls.Get("login_sequence"), "DailyBonusType", ls.Get("bonus_type"), "DailyBonusCount", ls.Get("bonus_count"))
	}
	return php.Assoc(
		"UserProfileStructure", c.userProfileStructure(profile),
		"UserProfileStructures", others,
		"AvailableSearchKeys", c.searchKeys(id),
		"Friends", c.Count(`select count(*) as aggregate from "wormix_user_profiles" where "user_id" != ?`, id),
		"OnlineFriends", int64(0),
		"IsBonusDay", false,
		"DailyBonusStructure", daily,
		"Reagents", profile.Arr("reagents"),
		"SessionKey", sessionKey,
	)
}

// ---- InternalAccountController ----

func (c *Ctx) selectStuff() Resp {
	c.ValidateWith([]R{{"internal_user_id", "required|integer|exists:users,id"}, {"StuffId", "required|integer|min:1000"}},
		map[string]Closure{"StuffId": func(v any) string {
			// an item id, or artifactStart (no item) to take the artifact off
			if php.ToInt(v) != artifactStart && c.Count(`select exists(select * from "wormix_weapons" where "wormix_weapons"."id" = ?)`, v) == 0 {
				return "Unknown item."
			}
			return ""
		}})
	if id := php.ToInt(c.JSON("StuffId")); id >= artifactStart && id < artifactEnd {
		return c.selectArtifact(id)
	}
	uw := c.First(defUserWeapon, `where "owner_id" = ? and "weapon_id" = ?`, c.UserID(), c.JSON("StuffId"))
	if uw == nil {
		return dataResult("Result", int64(1), "StuffId", int64(0))
	}
	worm := must(c.First(defWorm, `where "owner_id" = ?`, c.UserID()))
	race, _ := raceAndHat(php.IntArg(worm.Get("hat")))
	worm.Set("hat", hatByRaceAndHat(php.IntArg(uw.Get("weapon_id")), race))
	c.saveWorm(worm)
	return dataResult("Result", int64(0), "StuffId", php.IntArg(c.JSON("StuffId")))
}

// selectArtifact equips an owned (not expired) artifact, or takes it off with artifactStart.
func (c *Ctx) selectArtifact(id int64) Resp {
	uid := c.UserID()
	if id != artifactStart && !c.ownsValidItem(uid, id) {
		return dataResult("Result", int64(1), "StuffId", int64(0))
	}
	w := c.First(defWorm, `where "owner_id" = ? limit 1`, uid)
	if w == nil {
		abort(404, "Not Found")
	}
	if id == artifactStart {
		w.Set("artifact", int64(0))
	} else {
		w.Set("artifact", id)
	}
	c.saveWorm(w)
	return dataResult("Result", int64(0), "StuffId", id)
}

// ownsValidItem: the player has the item and it never expires or hasn't expired yet.
func (c *Ctx) ownsValidItem(uid, weaponID any) bool {
	return c.Count(`select exists(select * from "wormix_users_weapons" where "owner_id" = ? and "weapon_id" = ? and ("expire_at" = ? or "expire_at" > ?))`,
		uid, weaponID, int64(-1), time.Now().Unix()) > 0
}

func (c *Ctx) distributePoints() Resp {
	c.Validate([]R{{"internal_user_id", "required|integer|exists:users,id"}, {"Armor", "required|integer|min:0|max:60"}, {"Attack", "required|integer|min:0|max:60"}})
	w := must(c.First(defWorm, `where "owner_id" = ?`, c.UserID()))
	avail := php.Sub(php.Mul(w.Get("level"), int64(2)), php.Add(w.Get("armor"), w.Get("attack")))
	if php.Lt(avail, php.Add(c.JSON("Armor"), c.JSON("Attack"))) {
		return dataResult("Result", int64(2))
	}
	w.Set("armor", php.Add(w.Get("armor"), c.JSON("Armor")))
	w.Set("attack", php.Add(w.Get("attack"), c.JSON("Attack")))
	c.saveWorm(w)
	return dataResult("Result", int64(0))
}

// ---- InfoController ----

func (c *Ctx) getRating() Resp {
	return Resp{Status: 200, Raw: []byte{}, ContentType: "text/html; charset=UTF-8"}
}

func (c *Ctx) getPumpedReaction() Resp {
	c.Validate([]R{{"internal_user_id", "required|integer|exists:users,id"}})
	u := must(c.First(defUser, `where "id" = ?`, c.UserID()))
	list := func(where string, args ...any) *php.Arr {
		out := php.NewArr()
		for _, a := range c.All(defHouseAction, `where "wormix_house_actions"."to_user_id" = ? and "wormix_house_actions"."to_user_id" is not null and "action_type" = ? `+where,
			append([]any{u.Get("id"), int64(0)}, args...)...) {
			p := must(c.First(defProfile, `where "wormix_user_profiles"."user_id" = ? limit 1`, a.Get("user_id")))
			out.Append(php.Assoc("LongId", p.Get("user_id"), "StringId", php.ToString(p.Get("user_id"))))
		}
		return out
	}
	return jsonResp(200, php.Assoc("data", php.Assoc(
		"TodayPumped", list(`and "created_at" > ?`, today(0)),
		"YesterdayPumped", list(`and "created_at" > ? and "created_at" < ?`, today(1), today(0)),
		"TwoDaysAgoPumped", list(`and "created_at" > ? and "created_at" < ?`, today(2), today(1)),
	)))
}

// ---- TeamController ----

func (c *Ctx) addTeammate() Resp {
	c.Validate([]R{{"internal_user_id", "required|integer|exists:users,id"}, {"ProfileId", "required|integer|exists:users,id"}, {"MoneyType", "required|integer|between:0,1"}})
	u := must(c.First(defUser, `where "id" = ?`, c.UserID()))
	worm := must(c.userWorm(u))
	lm := must(c.levelModel(worm))
	profile := must(c.userProfile(u))
	mates := int64(len(c.All(defTeam, `where "wormix_users_teams"."user_id" in (?)`, profile.Get("user_id"))))
	mt := c.JSON("MoneyType")
	if c.Count(`select exists(select * from "wormix_users_teams" where "user_id" = ? and "teammate_id" = ?)`, c.UserID(), c.JSON("ProfileId")) > 0 ||
		php.Le(lm.Get("max_worms_count"), mates) ||
		(php.Lt(profile.Get("money"), int64(950)) && php.StrictEq(mt, int64(1))) ||
		(php.Lt(profile.Get("real_money"), int64(10)) && php.StrictEq(mt, int64(0))) {
		return dataResult("Result", int64(1))
	}
	t := New(defTeam)
	t.Set("user_id", c.UserID())
	t.Set("teammate_id", c.JSON("ProfileId"))
	t.Set("order", mates+1)
	c.Save(t)
	if php.StrictEq(mt, int64(1)) { // MoneyType 1 = fuzy, 0 = rubies
		profile.Set("money", php.Sub(profile.Get("money"), int64(950)))
	} else {
		profile.Set("real_money", php.Sub(profile.Get("real_money"), int64(10)))
	}
	c.Save(profile)
	return dataResult("Result", int64(0))
}

func (c *Ctx) deleteTeammate() Resp {
	c.Validate([]R{{"internal_user_id", "required|integer|exists:users,id"}, {"ProfileId", "required|integer|exists:users,id|different:internal_user_id"}})
	for _, id := range c.Values(`select "id" from "wormix_users_teams" where "user_id" = ? and "teammate_id" = ?`, c.UserID(), c.JSON("ProfileId")) {
		c.exec(`delete from "wormix_users_teams" where "id" = ?`, id)
	}
	return dataResult("Result", int64(0))
}

func (c *Ctx) reorderTeam() Resp {
	c.Validate([]R{{"internal_user_id", "required|integer|exists:users,id"}, {"ReorderedWormGroup", "required|array|max:4"}, {"ReorderedWormGroup.*", "integer|exists:users,id"}})
	group := c.JSON("ReorderedWormGroup").(*php.Arr)
	for _, k := range group.Keys() {
		t := c.First(defTeam, `where "user_id" = ? and "teammate_id" = ?`, c.UserID(), group.Val(k))
		if t == nil {
			return dataResult("Result", int64(1))
		}
		t.Set("order", k)
		c.Save(t)
	}
	return dataResult("Result", int64(0))
}

// ---- HouseController ----

func (c *Ctx) pumpReactions() Resp {
	c.Validate([]R{{"internal_user_id", "required|integer|exists:users,id"}, {"FriendsIds", "required|array"}, {"FriendsIds.*", "integer|exists:users,id"}})
	result := php.NewArr()
	me := c.UserID()
	for _, f := range c.JSON("FriendsIds").(*php.Arr).Values() {
		if c.Count(`select exists(select * from "wormix_house_actions" where "user_id" = ? and "to_user_id" = ? and "action_type" = ? and "created_at" > ?)`, me, f, int64(0), subDay(1)) > 0 ||
			c.Count(`select exists(select * from "wormix_house_actions" where "user_id" = ? and "to_user_id" = ? and "action_type" = ? and "created_at" > ?)`, f, me, int64(0), subDay(3)) == 0 {
			continue
		}
		result.Append(php.Assoc("FriendId", f, "Result", int64(0)))
		p := must(c.First(defProfile, `where "user_id" = ?`, f))
		p.Set("reaction_rate", php.Add(p.Get("reaction_rate"), int64(1)))
		c.Save(p)
		a := New(defHouseAction)
		a.Set("action_type", int64(0))
		a.Set("user_id", me)
		a.Set("to_user_id", f)
		c.Save(a)
	}
	return dataResult("PumpedFriends", result)
}

func (c *Ctx) pumpReaction() Resp {
	c.Validate([]R{{"FriendId", "required|integer|exists:users,id"}, {"internal_user_id", "required|integer|exists:users,id"}})
	me, f := c.UserID(), c.JSON("FriendId")
	if php.StrictEq(me, f) {
		return dataResult("Result", int64(2))
	}
	if c.First(defHouseAction, `where "to_user_id" = ? and "user_id" = ? and "action_type" = ? and "created_at" > ?`, f, me, int64(0), subDay(1)) != nil {
		return dataResult("Result", int64(1))
	}
	p := must(c.First(defProfile, `where "user_id" = ?`, f))
	p.Set("reaction_rate", php.Add(p.Get("reaction_rate"), int64(1)))
	c.Save(p)
	a := New(defHouseAction)
	a.Set("user_id", me)
	a.Set("to_user_id", f)
	a.Set("action_type", int64(0))
	c.Save(a)
	return dataResult("Result", int64(0))
}

func (c *Ctx) searchTheHouse() Resp {
	c.Validate([]R{{"internal_user_id", "required|exists:users,id"}, {"FriendId", "required|exists:users,id"}})
	me, f := c.UserID(), c.JSON("FriendId")
	a := New(defHouseAction)
	a.Set("user_id", me)
	a.Set("to_user_id", f)
	a.Set("action_type", int64(1))
	res := func(result, value int64) Resp {
		return dataResult("Result", result, "Value", value, "AvailableSearchKeys", c.searchKeys(a.Get("user_id")), "FriendId", a.Get("to_user_id"))
	}
	if php.StrictEq(me, f) {
		return res(3, 0)
	}
	if c.searchKeys(me) <= 0 {
		return res(5, 0)
	}
	if c.isSearchedToday(me, f) {
		return res(0, 0)
	}
	p := must(c.First(defProfile, `where "user_id" = ?`, me))
	c.Save(a)
	switch phpRand(0, 3) {
	case 0:
		return res(0, 0)
	case 1:
		m := phpRand(1, 20)
		p.Set("money", php.Add(p.Get("money"), m))
		c.Save(p)
		return res(2, m)
	case 2:
		m := phpRand(1, 5)
		p.Set("real_money", php.Add(p.Get("real_money"), m))
		c.Save(p)
		return res(1, m)
	default:
		ids := c.Values(`select "reagent_id" from "wormix_reagents"`)
		if len(ids) == 0 {
			php.Fail("array_rand(): Argument #1 ($array) cannot be empty")
		}
		r := ids[rand.IntN(len(ids))]
		c.addReagents(p, php.List(r))
		return res(7, php.IntArg(r))
	}
}

// ---- ShopController ----

func shopError() Resp {
	return jsonResp(200, php.Assoc("data", php.Assoc("Result", int64(1), "Weapons", php.NewArr(), "Stuff", php.NewArr())))
}

func (c *Ctx) buyItems() Resp {
	c.Validate([]R{
		{"internal_user_id", "required|integer|exists:users,id"},
		{"ShopItems", "required|array"},
		{"ShopItems.*.Id", "required|exists:wormix_weapons,id"},
		{"ShopItems.*.Count", "required|integer|min:-1"},
		{"ShopItems.*.MoneyType", "required|integer|max:1|min:0"},
	},
		M{"ShopItems.required", "The ShopItems is required."},
		M{"ShopItems.array", "The ShopItems must be an array."},
		M{"ShopItems.*.Id.required", "The ShopItems field Id is required."},
		M{"ShopItems.*.Id.exists", "The ShopItems not exists required."},
		M{"ShopItems.*.Count.required", "The ShopItems Count field is required."},
		M{"ShopItems.*.MoneyType.required", "The ShopItems MoneyType field is required."},
	)
	var out Resp
	if !catchPHP(func() { out = c.buyItemsInner() }) {
		return shopError()
	}
	return out
}

func (c *Ctx) buyItemsInner() Resp {
	items := c.JSON("ShopItems").(*php.Arr)
	shop := php.NewArr()
	for _, it := range items.Values() {
		shop.Set(php.ToString(idx(it, "Id")), php.Assoc("Count", idx(it, "Count"), "MoneyType", idx(it, "MoneyType")))
	}
	sum, realSum := any(int64(0)), any(int64(0))
	keys := shop.Keys()
	if len(keys) > 0 {
		for _, w := range c.All(defWeapon, `where "id" in (`+placeholders(len(keys))+`)`, keys...) {
			e := idx(shop, w.Get("id")).(*php.Arr)
			if php.ToBool(w.Get("hide_in_shop")) && w.Get("ref_id") == nil {
				php.Fail("Attempt to buy hidden item!")
			}
			if !php.ToBool(w.Get("infinity")) && php.StrictEq(e.Val("Count"), int64(-1)) {
				return shopError()
			}
			if php.StrictEq(e.Val("MoneyType"), int64(0)) {
				if php.StrictEq(w.Get("real_price"), int64(0)) {
					return shopError()
				}
				if php.ToBool(w.Get("infinity")) {
					realSum = php.Add(realSum, w.Get("real_price"))
				} else {
					realSum = php.Add(realSum, php.Mul(w.Get("real_price"), e.Val("Count")))
				}
			}
			if php.StrictEq(e.Val("MoneyType"), int64(1)) {
				if php.StrictEq(w.Get("price"), int64(0)) {
					return shopError()
				}
				if php.ToBool(w.Get("infinity")) {
					sum = php.Add(sum, w.Get("price"))
				} else {
					sum = php.Add(sum, php.Mul(w.Get("price"), e.Val("Count")))
				}
			}
		}
	}
	p := must(c.First(defProfile, `where "user_id" = ?`, c.UserID()))
	if php.Lt(p.Get("money"), sum) || php.Lt(p.Get("real_money"), realSum) {
		return jsonResp(200, php.Assoc("data", php.Assoc("Result", int64(3), "Weapons", php.NewArr(), "Stuff", php.NewArr())))
	}
	p.Set("money", php.Sub(p.Get("money"), sum))
	p.Set("real_money", php.Sub(p.Get("real_money"), realSum))
	c.Save(p)
	weapons := php.NewArr()
	for _, it := range items.Values() {
		old := c.First(defUserWeapon, `where "owner_id" = ? and "weapon_id" = ?`, c.UserID(), idx(it, "Id"))
		if php.LooseEq(idx(it, "Count"), int64(-1)) || old == nil {
			uw := New(defUserWeapon)
			uw.Set("owner_id", c.UserID())
			uw.Set("weapon_id", idx(it, "Id"))
			uw.Set("count", idx(it, "Count"))
			c.Save(uw)
			weapons.Append(php.Assoc("Id", uw.Get("weapon_id"), "Count", uw.Get("count")))
			continue
		}
		w := must(c.First(defWeapon, `where "wormix_weapons"."id" = ? limit 1`, old.Get("weapon_id")))
		if php.ToBool(w.Get("infinity")) {
			old.Set("count", idx(it, "Count"))
		} else {
			old.Set("count", php.Add(old.Get("count"), idx(it, "Count")))
		}
		c.Save(old)
		weapons.Append(php.Assoc("Id", old.Get("weapon_id"), "Count", idx(it, "Count")))
	}
	// The client puts a bought artifact on at once: do the same here.
	var last int64 = -1
	for _, it := range items.Values() {
		if id := php.ToInt(idx(it, "Id")); id > artifactStart && id < artifactEnd {
			last = id
		}
	}
	if last >= 0 {
		c.exec(`update "wormix_worms_data" set "artifact" = ?, "updated_at" = ? where "owner_id" = ?`, last, db.Now(), c.UserID())
	}
	return jsonResp(200, php.Assoc("data", php.Assoc("Result", int64(0), "Weapons", weapons, "Stuff", php.NewArr())))
}

func (c *Ctx) changeRace() Resp {
	c.Validate([]R{{"internal_user_id", "required|integer|exists:users,id"}, {"RaceId", "required|integer|exists:wormix_races,race_id"}, {"MoneyType", "required|integer|min:0|max:1"}})
	worm := must(c.First(defWorm, `where "owner_id" = ?`, c.UserID()))
	p := must(c.First(defProfile, `where "user_id" = ?`, c.UserID()))
	race := c.First(defRace, `where "race_id" = ?`, c.JSON("RaceId"))
	cur, hatID := raceAndHat(php.IntArg(worm.Get("hat")))
	rid, mt := c.JSON("RaceId"), c.JSON("MoneyType")
	if php.StrictEq(cur, rid) {
		return dataResult("Result", int64(1))
	}
	race = must(race)
	if (php.StrictEq(mt, int64(1)) && php.Lt(worm.Get("level"), race.Get("required_level"))) ||
		(php.StrictEq(mt, int64(0)) && php.Lt(p.Get("real_money"), race.Get("real_price"))) ||
		(php.StrictEq(mt, int64(1)) && php.Lt(p.Get("money"), race.Get("price"))) {
		return dataResult("Result", int64(2))
	}
	worm.Set("hat", hatByRaceAndHat(hatID, php.IntArg(rid)))
	c.saveWorm(worm)
	if php.StrictEq(mt, int64(0)) {
		p.Set("real_money", php.Sub(p.Get("real_money"), race.Get("real_price")))
	} else {
		p.Set("money", php.Sub(p.Get("money"), race.Get("price")))
	}
	c.Save(p)
	return dataResult("Result", int64(0))
}

func (c *Ctx) buyReaction() Resp {
	c.Validate([]R{{"internal_user_id", "required|integer|exists:users,id"}, {"ReactionRateCount", "required|integer|between:3,120"}})
	p := must(c.First(defProfile, `where "user_id" = ?`, c.UserID()))
	n := c.JSON("ReactionRateCount")
	if !php.StrictEq(php.Mod(n, int64(3)), int64(0)) || php.Lt(p.Get("real_money"), php.Div(n, int64(3))) {
		return dataResult("Result", int64(1), "ReactionRateCount", int64(0))
	}
	p.Set("real_money", php.Sub(p.Get("real_money"), php.Div(n, int64(3))))
	p.Set("reaction_rate", php.Add(p.Get("reaction_rate"), n))
	c.Save(p)
	return dataResult("Result", int64(0), "ReactionRateCount", php.IntArg(n))
}

func (c *Ctx) buyBattle() Resp {
	c.Validate([]R{{"internal_user_id", "required|integer|exists:users,id"}, {"MoneyType", "required|integer|between:0,1"}})
	bi := must(c.First(defBattleInfo, `where "user_id" = ?`, c.UserID()))
	p := must(c.First(defProfile, `where "user_id" = ?`, c.UserID()))
	mt := c.JSON("MoneyType")
	if php.Ge(bi.Get("battles_count"), int64(missionsMax)) {
		return dataResult("Result", int64(1))
	}
	if (php.StrictEq(mt, int64(0)) && php.Lt(p.Get("real_money"), int64(1))) || (php.StrictEq(mt, int64(1)) && php.Lt(p.Get("money"), int64(100))) {
		return dataResult("Result", int64(3))
	}
	bi.Set("battles_count", php.Add(bi.Get("battles_count"), int64(1)))
	c.Save(bi)
	if php.StrictEq(mt, int64(0)) {
		p.Set("real_money", php.Sub(p.Get("real_money"), int64(1)))
	} else {
		p.Set("money", php.Sub(p.Get("money"), int64(100)))
	}
	c.Save(p)
	return dataResult("Result", int64(0))
}

func (c *Ctx) unlockMission() Resp {
	c.Validate([]R{{"internal_user_id", "required|integer|exists:users,id"}, {"MissionId", "required|min:1|exists:wormix_missions,mission_id"}})
	u := must(c.First(defUser, `where "id" = ?`, c.UserID()))
	bi := must(c.First(defBattleInfo, `where "wormix_users_battle_info"."user_id" in (?)`, u.Get("id")))
	worm := must(c.First(defWorm, `where "wormix_worms_data"."owner_id" in (?)`, u.Get("id")))
	p := must(c.First(defProfile, `where "wormix_user_profiles"."user_id" in (?)`, u.Get("id")))
	m := must(c.First(defMission, `where "mission_id" = ?`, c.JSON("MissionId")))
	price := php.Mul(php.Sub(php.Sub(c.JSON("MissionId"), int64(1)), bi.Get("last_mission_id")), int64(10))
	if php.Lt(worm.Get("level"), m.Get("required_level")) || php.Ge(bi.Get("last_mission_id"), m.Get("mission_id")) || php.Lt(p.Get("real_money"), price) {
		return dataResult("Result", int64(1))
	}
	bi.Set("last_mission_id", php.Sub(c.JSON("MissionId"), int64(1)))
	p.Set("real_money", php.Sub(p.Get("real_money"), price))
	c.Save(p)
	c.Save(bi)
	return dataResult("Result", int64(0))
}

// ---- ResetController ----

func (c *Ctx) resetAccount() Resp {
	c.Validate([]R{{"internal_user_id", "required|integer|exists:users,id"}})
	uid := c.UserID()
	p := must(c.First(defProfile, `where "user_id" = ?`, uid))
	w := must(c.First(defWorm, `where "owner_id" = ?`, uid))
	bi := must(c.First(defBattleInfo, `where "user_id" = ?`, uid))
	ls := must(c.First(defLoginSeq, `where "user_id" = ?`, uid))
	level, realMoney := c.app.Config.starter()

	bi.Set("battles_count", int64(10))
	bi.Set("last_mission_id", int64(-1))
	bi.Set("awards", php.NewArr())
	bi.Set("battle_type", int64(0))
	bi.Set("mission_id", int64(0))
	bi.Set("current_battle_id", int64(0))
	bi.Set("last_boss_fight_time", int64(0))
	bi.Set("last_battle_time", int64(0))
	c.Save(bi)

	w.Set("armor", int64(0))
	w.Set("attack", int64(0))
	w.Set("level", level)
	w.Set("experience", int64(0))
	w.Set("hat", int64(0))
	c.SaveAndSync(w)

	p.Set("money", c.app.Config.StarterMoney)
	p.Set("real_money", realMoney)
	p.Set("rating", int64(0))
	p.Set("reaction_rate", int64(0))
	p.Set("reagents", php.NewArr())
	p.Set("recipes", php.NewArr())
	c.Save(p)

	ls.Set("last_login", time.Now().UTC().Format("2006-01-02"))
	ls.Set("bonus_type", int64(0))
	ls.Set("bonus_count", int64(0))
	ls.Set("login_sequence", int64(1))
	ls.Set("gift_accepted", int64(1))
	c.saveLoginSeq(ls)

	for _, id := range c.Values(`select "id" from "wormix_house_actions" where "user_id" = ? or "to_user_id" = ?`, uid, uid) {
		c.exec(`delete from "wormix_house_actions" where "id" = ?`, id)
	}
	for _, id := range c.Values(`select "id" from "wormix_users_weapons" where "owner_id" = ?`, uid) {
		c.exec(`delete from "wormix_users_weapons" where "id" = ?`, id)
	}
	for _, wid := range c.Values(`select "id" from "wormix_weapons" where "is_starter" = ?`, int64(1)) {
		nw := New(defUserWeapon)
		nw.Set("weapon_id", wid)
		nw.Set("count", int64(-1))
		nw.Set("owner_id", uid)
		c.Save(nw)
	}
	c.devStarter(w, level, uid)
	c.cacheForget("user_session_" + php.ToString(uid))
	return dataResult("Result", int64(0))
}

func (c *Ctx) resetParameters() Resp {
	c.Validate([]R{{"internal_user_id", "required|integer|exists:users,id"}, {"MoneyType", "required|integer|min:0|max:1"}})
	p := must(c.First(defProfile, `where "user_id" = ?`, c.UserID()))
	w := must(c.First(defWorm, `where "owner_id" = ?`, c.UserID()))
	mt := c.JSON("MoneyType")
	if (php.StrictEq(mt, int64(1)) && php.Lt(p.Get("money"), int64(300))) || (php.StrictEq(mt, int64(0)) && php.Lt(p.Get("real_money"), int64(3))) {
		return dataResult("Result", int64(3))
	}
	if php.StrictEq(mt, int64(1)) {
		p.Set("money", php.Sub(p.Get("money"), int64(300)))
	} else {
		p.Set("real_money", php.Sub(p.Get("real_money"), int64(3)))
	}
	w.Set("armor", int64(0))
	w.Set("attack", int64(0))
	c.saveWorm(w)
	c.Save(p)
	return dataResult("Result", int64(0))
}

// ---- UpgradeController ----

func (c *Ctx) upgradeWeapon() Resp {
	c.Validate([]R{{"RecipeId", "required|integer|exists:wormix_craft,id"}, {"internal_user_id", "required|integer|exists:users,id"}})
	rid := c.JSON("RecipeId")
	craft := must(c.First(defCraft, `where "id" = ?`, rid))
	p := must(c.First(defProfile, `where "user_id" = ?`, c.UserID()))
	w := must(c.First(defWorm, `where "owner_id" = ?`, c.UserID()))
	recipes := func() *php.Arr {
		r, ok := p.Arr("recipes").(*php.Arr)
		if !ok {
			php.Fail("in_array(): Argument #2 ($haystack) must be of type array")
		}
		return r
	}
	prevUpgrade := func() *Model {
		if craft.Get("prev_upgrade_id") == nil {
			return nil
		}
		return c.First(defCraft, `where "wormix_craft"."upgrade_id" = ? limit 1`, craft.Get("prev_upgrade_id"))
	}
	optID := func(m *Model) any {
		if m == nil {
			return nil
		}
		return m.Get("id")
	}
	errResp := func(code int64) Resp { return dataResult("Result", code, "RecipeId", php.IntArg(rid)) }
	prev := craft.Get("prev_upgrade_id")
	if (php.Lt(prev, int64(300)) && c.Count(`select count(*) as aggregate from "wormix_users_weapons" where "owner_id" = ? and "weapon_id" = ? and "count" = ?`, c.UserID(), prev, int64(-1)) == 0) ||
		(php.Gt(prev, int64(300)) && !php.InArray(optID(prevUpgrade()), recipes(), false)) ||
		php.InArray(craft.Get("id"), recipes(), false) ||
		php.Lt(w.Get("level"), craft.Get("required_level")) ||
		php.InArray(func() any {
			pu := optID(prevUpgrade())
			var other *Model
			if pu == nil {
				other = c.First(defCraft, `where "upgrade_id" is null and "id" != ?`, rid)
			} else {
				other = c.First(defCraft, `where "upgrade_id" = ? and "id" != ?`, pu, rid)
			}
			return optID(other)
		}(), recipes(), false) {
		return errResp(1)
	}
	creagents, ok := craft.Arr("reagents").(*php.Arr)
	if !ok || creagents.Len() == 0 {
		php.Fail("max(): Argument #1 ($value) must contain at least one element")
	}
	var ids []any
	var maxID any
	for i, r := range creagents.Values() {
		ids = append(ids, idx(r, 0))
		if i == 0 || php.Gt(idx(r, 0), maxID) {
			maxID = idx(r, 0)
		}
	}
	prices := php.NewArr()
	rows := c.query(`select "reagent_id", "reagent_price" from "wormix_reagents" where "reagent_id" in (`+placeholders(len(ids))+`)`, ids...)
	for rows.Next() {
		var k, v any
		if err := rows.Scan(&k, &v); err != nil {
			rows.Close()
			panic(err)
		}
		prices.Set(k, v)
	}
	rows.Close()

	changed, _ := p.Arr("reagents").(*php.Arr)
	if changed == nil {
		php.Fail("count(): Argument #1 must be of type Countable|array")
	}
	if php.Lt(int64(changed.Len()), php.Add(maxID, int64(1))) {
		old := changed
		changed = php.NewArr()
		n := php.ToInt(php.Add(maxID, int64(1)))
		for i := int64(0); i < n; i++ {
			changed.Set(i, int64(0))
		}
		old.Each(func(k, v any) { changed.Set(k, v) })
	}
	sum := any(int64(0))
	for _, r := range creagents.Values() {
		have := idx(changed, idx(r, 0))
		x := php.Mul(php.Sub(idx(r, 1), have), idx(prices, php.ToString(idx(r, 0))))
		if php.Gt(x, int64(0)) {
			sum = php.Add(sum, x)
		} else {
			sum = php.Add(sum, int64(0))
		}
		left := php.Sub(have, idx(r, 1))
		if php.Gt(left, int64(0)) {
			changed.Set(idx(r, 0), left)
		} else {
			changed.Set(idx(r, 0), int64(0))
		}
	}
	sumInt := php.ToInt(phpRound(php.Div(sum, int64(100))))
	if php.Lt(p.Get("real_money"), sumInt) {
		return errResp(3)
	}
	p.Set("real_money", php.Sub(p.Get("real_money"), sumInt))
	merged := php.NewArr()
	for _, v := range recipes().Values() {
		merged.Append(v)
	}
	merged.Append(rid)
	p.Set("recipes", merged)
	p.Set("reagents", changed)
	c.Save(p)
	return errResp(0)
}

func phpRound(v any) float64 {
	f, _ := php.ToNumber(v).(float64)
	if i, ok := php.ToNumber(v).(int64); ok {
		f = float64(i)
	}
	return math.Round(f)
}

func (c *Ctx) downgradeWeapon() Resp {
	c.Validate([]R{{"RecipeId", "required|integer|exists:wormix_craft,id"}, {"internal_user_id", "required|integer|exists:users,id"}})
	rid := c.JSON("RecipeId")
	craft := must(c.First(defCraft, `where "id" = ?`, rid))
	p := must(c.First(defProfile, `where "user_id" = ?`, c.UserID()))
	recipes, ok := p.Arr("recipes").(*php.Arr)
	if !ok {
		php.Fail("in_array(): Argument #2 ($haystack) must be of type array")
	}
	reagents, _ := p.Arr("reagents").(*php.Arr)
	if !php.InArray(craft.Get("id"), recipes, false) || php.Lt(p.Get("real_money"), int64(5)) {
		return dataResult("Result", int64(1), "RecipeId", php.IntArg(rid))
	}
	if reagents == nil {
		reagents = php.NewArr()
	}
	if cr, ok := craft.Arr("reagents").(*php.Arr); ok {
		for _, r := range cr.Values() {
			cur, ok := reagents.Get(idx(r, 0))
			if !ok {
				php.Fail("Undefined array key %v", idx(r, 0))
			}
			reagents.Set(idx(r, 0), php.Add(cur, php.ToInt(php.Mul(idx(r, 1), 0.8))))
		}
	}
	var found any = false
	for _, k := range recipes.Keys() {
		if php.LooseEq(recipes.Val(k), rid) {
			found = k
			break
		}
	}
	recipes.Delete(found)
	p.Set("recipes", php.ArrayValues(recipes))
	p.Set("reagents", reagents)
	p.Set("real_money", php.Sub(p.Get("real_money"), int64(5)))
	c.Save(p)
	return dataResult("Result", int64(0), "RecipeId", php.IntArg(rid))
}

// devStarter gives a dev-environment account the level awards up to its level and all
// achievement points (UserObserver::created / ResetController::resetAccount).
func (c *Ctx) devStarter(worm *Model, level int64, uid any) {
	if !c.app.Config.IsDev {
		return
	}
	if level > 1 {
		for _, l := range c.All(defLevel, `where "id" between ? and ?`, int64(2), level) {
			if aw, ok := l.Arr("awards").(*php.Arr); ok && aw.Len() > 0 {
				c.addWeaponsAwards(aw, worm)
			}
		}
	}
	c.grantDevAchievements(uid)
}

// grantDevAchievements is AchievementsController::grantDevAchievements.
func (c *Ctx) grantDevAchievements(uid any) {
	cfg := achievementsConfig()
	for _, id := range cfg.achievements.Keys() {
		ach, _ := cfg.achievements.Val(id).(*php.Arr)
		levels, _ := ach.Val("levels").(*php.Arr)
		if levels.Len() == 0 {
			continue
		}
		last := levels.Val(levels.Keys()[levels.Len()-1]).(*php.Arr)
		aid := php.ToInt(id)
		row := c.First(defAchievement, `where ("user_id" = ? and "achievement_id" = ?) limit 1`, uid, aid)
		if row == nil {
			row = New(defAchievement)
			row.Set("user_id", uid)
			row.Set("achievement_id", aid)
		}
		row.Set("value", last.Val("max"))
		c.Save(row)
	}
}
