package app

import (
	"math"
	"math/rand/v2"
	"regexp"
	"sync"
	"time"

	"github.com/andrij06062003/Wormix/services/api/internal/php"
)

const battleTypePvp = 2

// ---- ArenaController ----

func (c *Ctx) getArena() Resp {
	c.Validate([]R{{"internal_user_id", "required|exists:users,id"}, {"ReturnUsersProfiles", "required|boolean"}})
	bi := must(c.First(defBattleInfo, `where "user_id" = ?`, c.UserID()))
	if php.Lt(bi.Get("battles_count"), int64(missionsMax)) {
		add := php.ToInt(php.Div(php.Sub(time.Now().Unix(), bi.Get("last_battle_time")), int64(missionsDelay)))
		bi.Set("battles_count", php.Add(bi.Get("battles_count"), min(add, missionsMax)))
		if php.Gt(bi.Get("battles_count"), int64(missionsMax)) {
			bi.Set("battles_count", int64(missionsMax))
		}
		c.Save(bi)
	}
	if php.StrictEq(bi.Get("battles_count"), int64(0)) {
		return jsonResp(200, php.Assoc("type", "ArenaLocked", "data", php.Assoc(
			"Delay", php.Mul(php.Sub(int64(missionsDelay), php.Sub(time.Now().Unix(), bi.Get("last_battle_time"))), int64(1000)),
			// The last completed mission, as in ArenaResult (the client keeps it as lastCompletedMissionId).
			"CurrentMission", bi.Get("last_mission_id"),
			"ErrorCode", int64(0),
		)))
	}
	var profiles any = php.NewArr()
	if php.ToBool(c.JSON("ReturnUsersProfiles")) {
		profiles = c.generateBots(must(c.First(defWorm, `where "owner_id" = ?`, bi.Get("user_id"))))
	}
	return jsonResp(200, php.Assoc("type", "ArenaResult", "data", php.Assoc(
		"UserProfileStructures", profiles,
		"BattlesCount", bi.Get("battles_count"),
		"CurrentMission", bi.Get("last_mission_id"),
		"BossAvailable", true,
		// 1.25.0: the last co-op boss beaten (curCooperativeMissionId).
		"CurrentCoopMission", bi.Get("last_coop_mission_id"),
	)))
}

// randomSubset is Collection::random($n) (values keep their original order).
func randomSubset(vals []any, n int) []any {
	if n > len(vals) {
		php.Fail("You requested %d items, but there are only %d items available.", n, len(vals))
	}
	perm := rand.Perm(len(vals))[:n]
	pick := make([]bool, len(vals))
	for _, i := range perm {
		pick[i] = true
	}
	var out []any
	for i, v := range vals {
		if pick[i] {
			out = append(out, v)
		}
	}
	return out
}

func (c *Ctx) startBattle() Resp {
	c.Validate([]R{{"MissionId", "required|integer"}, {"internal_user_id", "required|exists:users,id"}})
	bi := must(c.First(defBattleInfo, `where "user_id" = ?`, c.UserID()))
	if php.Le(bi.Get("battles_count"), int64(0)) {
		return jsonResp(422, php.Assoc("message", "Not enough battles"))
	}
	mid := c.JSON("MissionId")
	if !php.StrictEq(mid, int64(0)) {
		last := bi.Get("last_mission_id")
		if php.Gt(mid, php.Add(last, int64(1))) || (!php.StrictEq(mid, php.Sub(last, int64(1))) && php.Lt(last, int64(-1))) {
			return jsonResp(422, php.Assoc("message", "Try to start invalid mission"))
		}
		m := must(c.First(defMission, `where "mission_id" = ?`, mid))
		w := must(c.First(defWorm, `where "owner_id" = ?`, c.UserID()))
		if php.Lt(w.Get("level"), m.Get("required_level")) {
			return jsonResp(422, php.Assoc("message", "Mission required level mismatch"))
		}
	}
	bi.Set("last_battle_time", time.Now().Unix())
	if php.Ge(mid, int64(0)) {
		bi.Set("battles_count", php.Sub(bi.Get("battles_count"), int64(1)))
	}
	bi.Set("current_battle_id", php.Add(php.Add(int64(1000), bi.Get("user_id")), mid))
	reagents := php.NewArr()
	if php.Ge(mid, int64(0)) {
		ids := c.Values(`select "reagent_id" from "wormix_reagents"`)
		reagents = php.List(randomSubset(ids, int(phpRand(0, 3)))...)
	}
	var awards any
	if php.StrictEq(mid, int64(0)) {
		bi.Set("battle_type", int64(0))
		awards = php.NewArr()
	} else {
		bi.Set("battle_type", int64(1))
		awards = must(c.First(defMission, `where "mission_id" = ?`, mid)).Arr("awards")
	}
	bi.Set("mission_id", mid)
	bi.Set("awards", php.Assoc("reagents", reagents, "awards", awards))
	c.Save(bi)
	return dataResult("Awards", awards, "BattleId", bi.Get("current_battle_id"), "ReagentForBattle", reagents)
}

func (c *Ctx) endBattle() Resp {
	c.Validate([]R{
		{"internal_user_id", "required|exists:users,id"},
		{"Result", "required|integer"},
		{"Type", "required|integer|min:0|max:3"},
		{"ExpBonus", "required|integer|min:0|max:10"},
		{"BattleId", "required|integer"},
		{"Items", "array"},
		{"CollectedReagents", "array"},
	})
	bi := must(c.First(defBattleInfo, `where "user_id" = ?`, c.UserID()))
	if php.ToInt(bi.Get("battle_type")) == battleTypePvp {
		return c.pvpItemsEndBattle(bi)
	}
	result := php.Sub(c.JSON("Result"), bi.Get("current_battle_id"))
	abs := result
	if php.Lt(result, int64(0)) {
		abs = php.Sub(int64(0), result)
	}
	if php.Ge(abs, int64(2)) {
		return jsonResp(200, php.NewArr())
	}
	if !php.StrictEq(c.JSON("MissionId"), bi.Get("mission_id")) {
		return jsonResp(200, php.NewArr())
	}
	c.processBattleResult(php.IntArg(result), bi)
	return jsonResp(200, php.Assoc("Status", "OK"))
}

func (c *Ctx) decreaseUsedWeapons(userID any) {
	items, _ := c.JSON("Items").(*php.Arr)
	if items == nil {
		if c.JSON("Items") != nil {
			php.Fail("foreach() argument must be of type array|object")
		}
		return
	}
	for _, it := range items.Values() {
		if php.Le(idx(it, "Count"), int64(0)) {
			continue
		}
		uw := c.First(defUserWeapon, `where "weapon_id" = ? and "count" != ? and "owner_id" = ?`, idx(it, "Id"), int64(-1), userID)
		if uw == nil {
			continue
		}
		uw.Set("count", php.Sub(uw.Get("count"), idx(it, "Count")))
		if php.Lt(uw.Get("count"), int64(0)) {
			uw.Set("count", int64(0))
		}
		c.Save(uw)
	}
}

func (c *Ctx) processBattleResult(result int64, bi *Model) {
	w := must(c.First(defWorm, `where "owner_id" = ?`, bi.Get("user_id")))
	p := must(c.First(defProfile, `where "user_id" = ?`, bi.Get("user_id")))
	bt := bi.Get("battle_type")
	if php.StrictEq(bt, int64(0)) {
		w.Set("experience", php.Add(w.Get("experience"), c.JSON("ExpBonus")))
	}
	c.decreaseUsedWeapons(bi.Get("user_id"))
	award := func(money, exp int64) {
		p.Set("money", php.Add(p.Get("money"), money))
		w.Set("experience", php.Add(w.Get("experience"), exp))
	}
	typ := c.JSON("Type")
	var mission *Model
	twice := false
	switch result {
	case 0:
		if php.StrictEq(bt, int64(0)) {
			switch {
			case php.LooseEq(typ, int64(0)):
				award(25, 6)
			case php.LooseEq(typ, int64(1)):
				award(30, 8)
			case php.LooseEq(typ, int64(2)):
				award(20, 4)
			}
		}
	case 1:
		if php.StrictEq(bt, int64(1)) {
			mission = c.First(defMission, `where "mission_id" = ?`, bi.Get("mission_id"))
			if php.Lt(bi.Get("mission_id"), int64(0)) {
				bi.Set("last_mission_id", bi.Get("mission_id"))
			} else {
				twice = php.Le(php.Sub(bi.Get("mission_id"), bi.Get("last_mission_id")), int64(0))
				if !twice {
					bi.Set("last_mission_id", bi.Get("mission_id"))
				}
				bi.Set("last_boss_fight_time", time.Now().Unix())
			}
		} else {
			switch {
			case php.LooseEq(typ, int64(0)):
				award(35, 10)
			case php.LooseEq(typ, int64(1)):
				award(40, 12)
			case php.LooseEq(typ, int64(2)):
				award(30, 8)
			}
		}
	case -1:
		if php.LooseEq(bt, int64(0)) {
			award(5, 3)
		}
	}
	c.Save(p)
	c.saveWorm(w)

	collected, ok := c.JSON("CollectedReagents").(*php.Arr)
	if !ok {
		php.Fail("foreach() argument must be of type array|object, %s given", php.TypeName(c.JSON("CollectedReagents")))
	}
	valid := true
	for _, r := range collected.Values() {
		aw, _ := bi.Arr("awards").(*php.Arr)
		if !php.InArray(r, toArr(idx(aw, "reagents")), false) {
			valid = false
			break
		}
	}
	if valid {
		c.addReagents(p, collected)
	}
	if mission != nil {
		c.processAwards(mission, twice, p, w)
	}
	bi.Set("current_battle_id", int64(0))
	bi.Set("mission_id", int64(0))
	bi.Set("awards", php.NewArr())
	c.Save(bi)
}

func toArr(v any) *php.Arr {
	a, ok := v.(*php.Arr)
	if !ok {
		php.Fail("in_array(): Argument #2 ($haystack) must be of type array, %s given", php.TypeName(v))
	}
	return a
}

func (c *Ctx) processAwards(m *Model, double bool, p, w *Model) {
	awards, _ := m.Arr("awards").(*php.Arr)
	c.giveAwards(awards, double, p, w)
}

// giveAwards gives a mission's awards (wormix_missions.awards): the first set, or the second for a mission
// beaten before (double) when there is one.
func (c *Ctx) giveAwards(awards *php.Arr, double bool, p, w *Model) {
	if awards == nil {
		php.Fail("count(): Argument #1 ($value) must be of type Countable|array")
	}
	if awards.Len() == 0 {
		return
	}
	var a any
	if awards.Len() == 1 || !double {
		a = idx(awards, 0)
	} else {
		a = idx(awards, 1)
	}
	p.Set("money", php.Add(p.Get("money"), idx(a, "money")))
	p.Set("real_money", php.Add(p.Get("real_money"), idx(a, "real_money")))
	c.Save(p)
	wa, ok := idx(a, "weapons").(*php.Arr)
	if !ok {
		php.Fail("addWeaponsAwards(): Argument #1 ($awards) must be of type array")
	}
	c.addWeaponsAwards(wa, w)
	w.Set("experience", php.Add(w.Get("experience"), idx(a, "experience")))
	c.saveWorm(w)
}

func (c *Ctx) pvpItemsEndBattle(bi *Model) Resp {
	awards, _ := bi.Arr("awards").(*php.Arr)
	if awards == nil {
		awards = php.NewArr()
	}
	done, ok := awards.Get("items_done")
	if !ok || done == nil {
		done = true
	}
	if php.ToInt(c.JSON("BattleId")) != php.ToInt(bi.Get("current_battle_id")) || php.ToBool(done) {
		return jsonResp(200, php.NewArr())
	}
	c.decreaseUsedWeapons(bi.Get("user_id"))
	collected, _ := c.JSON("CollectedReagents").(*php.Arr)
	if collected == nil {
		collected = php.NewArr()
	}
	allowed, _ := awards.Val("reagents").(*php.Arr)
	if allowed == nil {
		allowed = php.NewArr()
	}
	reagents := php.NewArr()
	for _, r := range collected.Values() {
		if php.InArray(r, allowed, false) {
			reagents.Append(r)
		}
	}
	if reagents.Len() > 0 {
		c.addReagents(must(c.First(defProfile, `where "user_id" = ?`, bi.Get("user_id"))), reagents)
	}
	awards.Set("items_done", true)
	bi.Set("awards", awards)
	c.Save(bi)
	return jsonResp(200, php.Assoc("Status", "OK"))
}

// ---- PvpController ----

func (c *Ctx) findOrFail(def *Def, id any) *Model {
	m := c.Find(def, id)
	if m == nil {
		abort(404, "Not Found")
	}
	return m
}

func (c *Ctx) pvpProfile() Resp {
	p := c.findOrFail(defProfile, c.UserID())
	return jsonResp(200, php.Assoc("data", c.userProfileStructure(p)))
}

func (c *Ctx) pvpStartBattle() Resp {
	c.Validate([]R{
		{"internal_user_id", "required|exists:users,id"},
		{"BattleId", "required|integer|min:1"},
		{"Wager", "required|integer|min:0"},
		{"Reagents", "nullable|array"},
		{"Reagents.*", "integer"},
		{"MissionId", "nullable|integer"},
	})
	if mid := c.JSON("MissionId"); mid != nil {
		return c.coopStartBattle(php.ToInt(mid))
	}
	uid := c.UserID()
	wager := php.ToInt(c.JSON("Wager"))
	p := c.findOrFail(defProfile, uid)
	bi := c.findOrFail(defBattleInfo, uid)
	if php.Lt(p.Get("money"), wager) {
		return jsonResp(422, php.Assoc("message", "Not enough money"))
	}
	reagents := c.JSON("Reagents")
	if reagents == nil {
		if wager > 0 {
			ids := c.Values(`select "reagent_id" from "wormix_reagents"`)
			rand.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
			n := int(phpRand(0, 3))
			if n > len(ids) {
				n = len(ids)
			}
			reagents = php.List(ids[:n]...)
		} else {
			reagents = php.NewArr()
		}
	}
	p.Set("money", php.Sub(p.Get("money"), wager))
	c.Save(p)
	bi.Set("current_battle_id", c.JSON("BattleId"))
	bi.Set("battle_type", int64(battleTypePvp))
	bi.Set("mission_id", int64(0))
	bi.Set("awards", php.Assoc("reagents", reagents, "awards", php.NewArr(), "wager", wager, "settled", false, "items_done", false))
	c.Save(bi)
	fresh := c.findOrFail(defProfile, uid)
	return dataResult("UserProfileStructure", c.userProfileStructure(fresh), "ReagentsForBattle", reagents)
}

func (c *Ctx) pvpEndBattle() Resp {
	c.Validate([]R{
		{"internal_user_id", "required|exists:users,id"},
		{"BattleId", "required|integer|min:1"},
		{"Result", "required|integer|min:-1|max:1"},
	})
	uid := c.UserID()
	result := php.ToInt(c.JSON("Result"))
	bi := c.findOrFail(defBattleInfo, uid)
	awards, _ := bi.Arr("awards").(*php.Arr)
	if awards == nil {
		awards = php.NewArr()
	}
	if php.ToInt(bi.Get("battle_type")) == battleTypeCoop {
		return c.coopEndBattle(bi, result)
	}
	settled, ok := awards.Get("settled")
	if !ok || settled == nil {
		settled = true
	}
	if php.ToInt(bi.Get("battle_type")) != battleTypePvp || php.ToInt(bi.Get("current_battle_id")) != php.ToInt(c.JSON("BattleId")) || php.ToBool(settled) {
		return jsonResp(200, php.Assoc("Status", "IGNORED"))
	}
	wager := php.ToInt(awards.Val("wager"))
	if wager > 0 {
		p := c.findOrFail(defProfile, uid)
		switch result {
		case 1:
			p.Set("money", php.Add(p.Get("money"), wager+int64(math.Floor(float64(wager)*0.8))))
		case 0:
			p.Set("money", php.Add(p.Get("money"), wager))
		}
		rating := php.IntArg(p.Get("rating"))
		p.Set("rating", max(0, rating+c.ratingDelta(uid, rating, result, wager)))
		c.Save(p)
	}
	awards.Set("settled", true)
	bi.Set("awards", awards)
	c.Save(bi)
	return jsonResp(200, php.Assoc("Status", "OK"))
}

func (c *Ctx) ratingDelta(uid any, rating, result, wager int64) int64 {
	if result == 0 {
		return 0
	}
	level := int64(1)
	if v := c.Values(`select "level" from "wormix_worms_data" where "owner_id" = ? limit 1`, uid); len(v) > 0 && v[0] != nil {
		level = php.ToInt(v[0])
	}
	team := c.Count(`select count(*) as aggregate from "wormix_users_teams" where "user_id" = ?`, uid)
	teamLevel := level * max(1, team)
	basket := max(1, level/3)
	if result < 0 {
		delta := -min(rating, int64(math.Ceil(float64(basket*basket)/4))+int64(math.Floor(float64(teamLevel-level)/6)))
		if rating > 20000 {
			delta -= (rating - 20000) / 200
		}
		return delta
	}
	delta := 1 + int64(math.Ceil(float64(basket*basket)/3)) + int64(math.Floor(float64(teamLevel-level)/3))
	if wager == 50 {
		delta = int64(math.Round(1.1 * float64(delta)))
	} else if wager == 300 {
		delta = int64(math.Round(1.25 * float64(delta)))
	}
	return delta
}

// ---- AchievementsController ----

type achConfig struct {
	raw            *php.Arr
	maxValue       int64
	pointsPerAward int64
	statsFirst     int64
	statsLast      int64
	achievements   *php.Arr
	awardItems     *php.Arr
}

var (
	achOnce sync.Once
	achCfg  *achConfig
)

func achievementsConfig() *achConfig {
	achOnce.Do(func() {
		a, _ := loadGameJSON("achievements.json").(*php.Arr)
		if a == nil {
			a = php.NewArr()
		}
		achCfg = &achConfig{raw: a,
			maxValue: php.ToInt(a.Val("max_value")), pointsPerAward: php.ToInt(a.Val("points_per_award")),
			statsFirst: php.ToInt(a.Val("stats_first")), statsLast: php.ToInt(a.Val("stats_last"))}
		achCfg.achievements, _ = a.Val("achievements").(*php.Arr)
		achCfg.awardItems, _ = a.Val("award_items").(*php.Arr)
	})
	return achCfg
}

func (cfg *achConfig) levels(index any) []any {
	ach, ok := cfg.achievements.Get(php.ToString(index))
	if !ok {
		return nil
	}
	l, _ := ach.(*php.Arr).Val("levels").(*php.Arr)
	return l.Values()
}

func (cfg *achConfig) isCounter(index int64) bool {
	return cfg.achievements.Has(php.ToString(index)) || (index >= cfg.statsFirst && index <= cfg.statsLast)
}

const achSessionTTL = 6 * 60 * 60

var achSessionRe = regexp.MustCompile(`^[0-9a-f]{32}\n?$`)

func (c *Ctx) achGetSession() *php.Arr {
	sid := php.ToString(c.JSON("session_id"))
	if !achSessionRe.MatchString(sid) {
		return nil
	}
	v, err := c.cacheGet("achieve_session_" + sid)
	if err != nil {
		panic(err)
	}
	s, ok := v.(*php.Arr)
	if !ok {
		return nil
	}
	uid := s.Val("user_id")
	if uid == nil {
		uid = int64(0)
	}
	if c.Count(`select exists(select * from "users" where "id" = ?)`, uid) == 0 {
		return nil
	}
	return s
}

func unauthorized() Resp { return jsonResp(401, php.Assoc("message", "Unauthorized")) }

func (c *Ctx) achLogin() Resp {
	uid := php.ToInt(c.JSON("id"))
	authKey := php.ToString(c.JSON("auth_key"))
	s := c.loadSession(uid)
	if s == nil {
		return jsonResp(200, php.Assoc("ok", false, "code", int64(1)))
	}
	exp := s.authKey()
	if exp == "" || exp != authKey {
		return jsonResp(200, php.Assoc("ok", false, "code", int64(1)))
	}
	sid := hexString(php.RandomBytes(16))
	now := time.Now().Unix()
	c.cachePut("achieve_session_"+sid, php.Assoc("user_id", uid, "last_seq", now, "last_result", nil), achSessionTTL)
	return jsonResp(200, php.Assoc("ok", true, "session_id", sid, "login_time", now))
}

func hexString(b []byte) string {
	const h = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, x := range b {
		out[i*2], out[i*2+1] = h[x>>4], h[x&15]
	}
	return string(out)
}

func (c *Ctx) achGet() Resp {
	if c.achGetSession() == nil {
		return unauthorized()
	}
	cfg := achievementsConfig()
	pid := php.ToString(c.JSON("profile_id"))
	uid := php.ToInt(pid)
	idxs, vals := php.NewArr(), php.NewArr()
	rows := c.query(`select "achievement_id", "value" from "wormix_users_achievements" where "user_id" = ? and "value" > ? order by "achievement_id" asc`, uid, int64(0))
	for rows.Next() {
		var a, v any
		if err := rows.Scan(&a, &v); err != nil {
			rows.Close()
			panic(err)
		}
		idxs.Append(a)
		if php.Lt(cfg.maxValue, v) {
			v = cfg.maxValue
		}
		vals.Append(v)
	}
	rows.Close()
	var invested any = int64(0)
	if p := c.Find(defProfile, uid); p != nil && p.Get("invested_award_points") != nil {
		invested = p.Get("invested_award_points")
	}
	return jsonResp(200, php.Assoc("profile_id", pid, "indexes", idxs, "values", vals, "invested", invested))
}

func increaseResult(idx, vals, ti, to *php.Arr, seq, invested int64) *php.Arr {
	return php.Assoc("indexes", idx, "values", vals, "threshold_indexes", ti, "threshold_old_values", to, "time_sequence", seq, "invested", invested)
}

func castArray(v any, present bool) *php.Arr {
	if !present {
		return php.NewArr()
	}
	switch t := v.(type) {
	case nil:
		return php.NewArr()
	case *php.Arr:
		return t
	default:
		return php.List(t)
	}
}

func (c *Ctx) achIncrease() Resp {
	sid := php.ToString(c.JSON("session_id"))
	s := c.achGetSession()
	if s == nil {
		return unauthorized()
	}
	ts := php.ToInt(c.JSON("time_scale"))
	if php.StrictEq(ts, s.Val("last_seq")) && s.Val("last_result") != nil {
		return jsonResp(200, s.Val("last_result"))
	}
	uid := s.Val("user_id")
	p := c.Find(defProfile, uid)
	if p == nil {
		return unauthorized()
	}
	if php.Le(ts, s.Val("last_seq")) {
		return jsonResp(200, increaseResult(php.NewArr(), php.NewArr(), php.NewArr(), php.NewArr(), ts, php.IntArg(p.Get("invested_award_points"))))
	}
	cfg := achievementsConfig()
	iv, ip := php.GetPath(c.req.json, "indexes")
	rv, rp := php.GetPath(c.req.json, "rises")
	indexes, rises := castArray(iv, ip), castArray(rv, rp)
	rise := php.NewArr()
	for _, i := range indexes.Keys() {
		index := php.ToInt(indexes.Val(i))
		value := php.ToInt(rises.Val(i))
		if !cfg.isCounter(index) || value <= 0 {
			continue
		}
		rise.Set(index, php.Add(rise.Val(index), value))
	}
	p = must(c.Find(defProfile, uid))
	oi, ov, ti, to := php.NewArr(), php.NewArr(), php.NewArr(), php.NewArr()
	for _, index := range rise.Keys() {
		value := rise.Val(index)
		row := c.First(defAchievement, `where ("user_id" = ? and "achievement_id" = ?) limit 1`, uid, index)
		if row == nil {
			row = New(defAchievement)
			row.Set("user_id", uid)
			row.Set("achievement_id", index)
		}
		old := php.ToInt(row.Get("value"))
		nv := min(cfg.maxValue, php.ToInt(php.Add(old, value)))
		if nv == old {
			oi.Append(index)
			ov.Append(old)
			continue
		}
		row.Set("value", nv)
		c.Save(row)
		crossed := false
		for _, l := range cfg.levels(index) {
			la := l.(*php.Arr)
			mx := php.ToInt(la.Val("max"))
			if old < mx && nv >= mx {
				crossed = true
				cnt := php.ToInt(la.Val("count"))
				switch php.ToString(la.Val("award")) {
				case "fuzy":
					p.Set("money", php.Add(p.Get("money"), cnt))
				case "ruby":
					p.Set("real_money", php.Add(p.Get("real_money"), cnt))
				case "reaction":
					p.Set("reaction_rate", php.Add(p.Get("reaction_rate"), cnt))
				}
			}
		}
		if crossed {
			ti.Append(index)
			to.Append(old)
		}
		oi.Append(index)
		ov.Append(nv)
	}
	c.Save(p)
	res := increaseResult(oi, ov, ti, to, ts, php.IntArg(p.Get("invested_award_points")))
	s.Set("last_seq", ts)
	s.Set("last_result", res)
	c.cachePut("achieve_session_"+sid, s, achSessionTTL)
	return jsonResp(200, res)
}

func (c *Ctx) achChooseBonus() Resp {
	s := c.achGetSession()
	if s == nil {
		return unauthorized()
	}
	uid := s.Val("user_id")
	item := php.ToInt(c.JSON("item_id"))
	cfg := achievementsConfig()
	result := func() int64 {
		p := c.Find(defProfile, uid)
		if p == nil {
			return 1
		}
		var level any
		for _, l := range cfg.awardItems.Keys() {
			items, _ := cfg.awardItems.Val(l).(*php.Arr)
			if php.InArray(item, items, true) {
				level = l
				break
			}
		}
		if level == nil {
			return 1
		}
		invested := php.ToInt(p.Get("invested_award_points"))
		var points int64
		for _, row := range c.All(defAchievement, `where "user_id" = ?`, uid) {
			for _, l := range cfg.levels(row.Get("achievement_id")) {
				la := l.(*php.Arr)
				if php.Ge(row.Get("value"), la.Val("max")) {
					points += php.ToInt(la.Val("points"))
				}
			}
		}
		available := points/cfg.pointsPerAward - invested
		if available <= 0 || php.Gt(level, invested) {
			return 2
		}
		owned := c.First(defUserWeapon, `where "owner_id" = ? and "weapon_id" = ? limit 1`, uid, item)
		if owned != nil && (item > stuffStartIndex || php.StrictEq(owned.Get("count"), int64(-1))) {
			return 1
		}
		if owned == nil {
			owned = New(defUserWeapon)
			owned.Set("owner_id", uid)
			owned.Set("weapon_id", item)
		}
		owned.Set("count", int64(-1))
		c.Save(owned)
		p.Set("invested_award_points", invested+1)
		c.Save(p)
		return 0
	}()
	return jsonResp(200, php.Assoc("result", result, "item_id", item))
}
