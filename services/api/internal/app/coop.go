package app

import (
	"sync"
	"time"

	"github.com/andrij06062003/Wormix/services/api/internal/assets"
	"github.com/andrij06062003/Wormix/services/api/internal/php"
)

// Co-op bosses of the 1.25.0 client: missions 101+ fought by two players on the PvP server. They have their
// own progress (last_coop_mission_id; the client counts from 100) and their own config
// (game/coop_missions125.json: required level, awards for the first win and for a repeat). The battle costs
// each player one battle; the PvP server records it with pvp/start_battle (MissionId) and pays the result
// out with pvp/end_battle: online battles don't send the main server's EndBattle.

const (
	battleTypeCoop = 3

	coopFirstMission = 101
	coopBase         = 100 // what the client counts the co-op progress from
)

// GetFriendsForBossResult states.
const (
	coopOK         = 0
	coopNoBattles  = 1
	coopLocked     = 2
	coopNoSuchBoss = coopLocked
)

type coopMission struct {
	requiredLevel int64
	awards        *php.Arr // as wormix_missions.awards
}

var (
	coopOnce     sync.Once
	coopMissions map[int64]coopMission
)

func coopMissionCfg(id int64) (coopMission, bool) {
	coopOnce.Do(func() {
		b, err := assets.Game("coop_missions125.json")
		if err != nil {
			panic(err)
		}
		v, ok := php.JSONDecode(b)
		if !ok {
			panic("coop_missions125.json: invalid")
		}
		coopMissions = map[int64]coopMission{}
		list, _ := idx(v, "missions").(*php.Arr)
		if list == nil {
			panic("coop_missions125.json: no missions")
		}
		for _, m := range list.Values() {
			aw, _ := idx(m, "awards").(*php.Arr)
			coopMissions[php.ToInt(idx(m, "mission_id"))] = coopMission{php.ToInt(idx(m, "required_level")), aw}
		}
	})
	m, ok := coopMissions[id]
	return m, ok
}

// lastCoopMission: the last co-op boss beaten, as the client counts it (100 = none yet).
func lastCoopMission(bi *Model) int64 { return max(coopBase, php.ToInt(bi.Get("last_coop_mission_id"))) }

// battlesNow is battles_count with the battles regained since the last one, as getArena counts them.
func battlesNow(bi *Model) int64 {
	n := php.ToInt(bi.Get("battles_count"))
	if n < missionsMax {
		add := php.ToInt(php.Div(php.Sub(time.Now().Unix(), bi.Get("last_battle_time")), int64(missionsDelay)))
		n = min(n+min(add, missionsMax), missionsMax)
	}
	return n
}

// coopState: whether the player may fight co-op boss mid now (a GetFriendsForBossResult state).
func (c *Ctx) coopState(bi *Model, mid int64) int {
	m, ok := coopMissionCfg(mid)
	if !ok {
		return coopNoSuchBoss
	}
	if mid > lastCoopMission(bi)+1 {
		return coopLocked
	}
	w := c.First(defWorm, `where "owner_id" = ?`, bi.Get("user_id"))
	if w == nil || php.ToInt(w.Get("level")) < m.requiredLevel {
		return coopLocked
	}
	if battlesNow(bi) <= 0 {
		return coopNoBattles
	}
	return coopOK
}

// pvpCoopState is pvp/coop_state: the player's profile and whether it may fight the co-op boss MissionId
// (for the friends list of the boss, GetFriendsForBoss).
func (c *Ctx) pvpCoopState() Resp {
	c.Validate([]R{{"internal_user_id", "required|exists:users,id"}, {"MissionId", "required|integer"}})
	uid := c.UserID()
	p := c.findOrFail(defProfile, uid)
	bi := c.findOrFail(defBattleInfo, uid)
	return dataResult("UserProfileStructure", c.userProfileStructure(p), "State", int64(c.coopState(bi, php.ToInt(c.JSON("MissionId")))))
}

// coopStartBattle is pvp/start_battle with a MissionId: takes a battle and records the co-op boss fight.
func (c *Ctx) coopStartBattle(mid int64) Resp {
	uid := c.UserID()
	p := c.findOrFail(defProfile, uid)
	bi := c.findOrFail(defBattleInfo, uid)
	switch c.coopState(bi, mid) {
	case coopNoBattles:
		return jsonResp(422, php.Assoc("message", "Not enough battles"))
	case coopLocked:
		return jsonResp(422, php.Assoc("message", "Try to start invalid mission"))
	}
	reagents := c.JSON("Reagents")
	if reagents == nil {
		ids := c.Values(`select "reagent_id" from "wormix_reagents"`)
		reagents = php.List(randomSubset(ids, int(phpRand(0, 3)))...)
	}
	bi.Set("battles_count", battlesNow(bi)-1)
	bi.Set("last_battle_time", time.Now().Unix())
	bi.Set("current_battle_id", c.JSON("BattleId"))
	bi.Set("battle_type", int64(battleTypeCoop))
	bi.Set("mission_id", mid)
	bi.Set("awards", php.Assoc("reagents", reagents, "awards", php.NewArr(), "wager", int64(0), "settled", false, "items_done", true))
	c.Save(bi)
	return dataResult("UserProfileStructure", c.userProfileStructure(p), "ReagentsForBattle", reagents)
}

// coopEndBattle is pvp/end_battle of a co-op boss fight: a win moves the co-op progress on and gives the
// boss's awards (the repeat ones for a boss beaten before) and the battle's reagents; a cancelled battle
// (0) gives the battle back.
func (c *Ctx) coopEndBattle(bi *Model, result int64) Resp {
	awards, _ := bi.Arr("awards").(*php.Arr)
	if awards == nil {
		awards = php.NewArr()
	}
	settled, ok := awards.Get("settled")
	if !ok || settled == nil {
		settled = true
	}
	if php.ToInt(bi.Get("current_battle_id")) != php.ToInt(c.JSON("BattleId")) || php.ToBool(settled) {
		return jsonResp(200, php.Assoc("Status", "IGNORED"))
	}
	switch result {
	case 1:
		mid := php.ToInt(bi.Get("mission_id"))
		if m, ok := coopMissionCfg(mid); ok {
			twice := mid <= lastCoopMission(bi)
			if !twice {
				bi.Set("last_coop_mission_id", mid)
			}
			bi.Set("last_boss_fight_time", time.Now().Unix())
			p := must(c.First(defProfile, `where "user_id" = ?`, bi.Get("user_id")))
			w := must(c.First(defWorm, `where "owner_id" = ?`, bi.Get("user_id")))
			if r, ok := awards.Val("reagents").(*php.Arr); ok {
				c.addReagents(p, r)
			}
			c.giveAwards(m.awards, twice, p, w)
		}
	case 0:
		bi.Set("battles_count", php.ToInt(bi.Get("battles_count"))+1)
	}
	awards.Set("settled", true)
	bi.Set("awards", awards)
	c.Save(bi)
	return jsonResp(200, php.Assoc("Status", "OK"))
}
