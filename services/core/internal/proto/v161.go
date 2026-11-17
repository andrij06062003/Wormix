package proto

import (
	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

// The 1.61.7 client speaks the 1.25.0 protocol with changes (from its *BinarySerializer classes): another
// salt, and new fields in EnterAccount and in the profile structures. Its Login has the 1.25.0 layout. A
// session is a 1.61.7 one when its Login is signed with the 1.61.7 salt; everything that isn't different
// in 1.61.7 is answered as for 1.25.0.

// IsLogin161 reports whether a Login body comes from the 1.61.7 client.
func IsLogin161(body []byte) bool { return wire.IsSecureWith(body, wire.SecretKey161) }

// Resign gives a complete message signed with the 1.05.0 salt another salt instead. Anything else
// (unsigned messages, pieces of a message) is returned as it is.
func Resign(frame []byte, key string) []byte {
	if len(frame) < wire.HeaderSize+16 || wire.FrameLength(frame) != len(frame)-wire.HeaderSize {
		return frame
	}
	body := frame[wire.HeaderSize:]
	payload := body[:len(body)-16]
	if string(wire.Secure(payload)) != string(body[len(body)-16:]) {
		return frame
	}
	out := append([]byte(nil), frame[:len(frame)-16]...)
	return append(out, wire.SecureWith(payload, key)...)
}

// Resign161 gives a complete message signed with the 1.05.0 or the 1.25.0 salt (what is built for the 1.25.0
// client) the 1.61.7 salt instead.
func Resign161(frame []byte) []byte {
	if len(frame) < wire.HeaderSize+16 || wire.FrameLength(frame) != len(frame)-wire.HeaderSize {
		return frame
	}
	if wire.IsSecureWith(frame[wire.HeaderSize:], wire.SecretKey125) {
		body := frame[wire.HeaderSize:]
		out := append([]byte(nil), frame[:len(frame)-16]...)
		return append(out, wire.SecureWith(body[:len(body)-16], wire.SecretKey161)...)
	}
	return Resign(frame, wire.SecretKey161)
}

// writeTeamMember161 is the 1.61.7 TeamMemberStructure: the 1.25.0 one with the skin after the race and the
// worm's name after the social id ("" lets the client name it).
func writeTeamMember161(w *wire.Writer, m *Worm) {
	race, hat := splitHat(m.Hat)
	w.U32(m.OwnerID)
	w.U8(byte(m.Armor))
	w.U8(byte(m.Attack))
	w.U8(byte(m.Level))
	w.U32(m.Experience)
	w.I16(hat)
	w.U8(byte(race))
	w.U8(0) // skin
	w.I16(int(m.Artifact))
	if m.SocialOwnerID != nil {
		w.UTF(*m.SocialOwnerID)
	} else {
		w.UTF("")
	}
	w.UTF("")           // name
	w.I16(0)            // teamMemberType: a friend or the player himself
	w.Bool(!m.Inactive) // isActive: fights in battles
}

// writeUserProfile161 is the 1.61.7 UserProfileStructure (what the client's deserializer reads).
func writeUserProfile161(w *wire.Writer, p *UserProfile) {
	w.U32(p.ID)
	w.U32(p.Money)
	w.U32(p.RealMoney)
	w.I32(p.Rating)
	w.U16(uint16(len(p.WormsGroup)))
	for i := range p.WormsGroup {
		var b wire.Writer
		writeTeamMember161(&b, &p.WormsGroup[i])
		w.U16(uint16(len(b.B)))
		w.Bytes(b.B)
	}
	// Weapons: a count, then short id and short count each.
	w.U16(uint16(len(p.WeaponRecordList)))
	for _, x := range p.WeaponRecordList {
		w.U16(uint16(x.ID))
		w.U16(uint16(int16(x.Count)))
	}
	w.U16(uint16(len(p.Stuff)))
	for _, x := range p.Stuff {
		w.U16(x)
	}
	w.U16(0) // temporalStuff (items that expire), in bytes: none yet
	w.I32(p.ReactionRate)
	if p.SocialID != nil {
		w.UTF(*p.SocialID)
	} else {
		w.UTF("")
	}
	w.U8(0) // socialNetId
	w.U16(uint16(len(p.Recipes)))
	for _, x := range p.Recipes {
		w.U16(x)
	}
	var clan wire.Writer
	writeNoClan(&clan)
	w.U16(uint16(len(clan.B)))
	w.Bytes(clan.B)
	w.U8(0)  // extraGroupSlotsCount
	w.I32(0) // rankPoints
	w.U8(0)  // bestRank
}

// writeNoClan is a ClanMemberStructure of a player who isn't in a clan.
func writeNoClan(w *wire.Writer) {
	w.I16(0)  // rank
	w.I32(0)  // clanId
	w.UTF("") // clanName
	w.U16(0)  // clanEmblem
	w.I32(0)  // clanRating
	w.I32(0)  // clanSeasonRating
	w.I16(0)  // reviewState
	w.I32(0)  // prevSeasonTopPlace
}

// profile161 is a UserProfileStructure with its size in front, as EnterAccount sends them.
func profile161(w *wire.Writer, p *UserProfile) {
	var b wire.Writer
	writeUserProfile161(&b, p)
	w.U16(uint16(len(b.B)))
	w.Bytes(b.B)
}

// Emit161 writes EnterAccount in the 1.61.7 layout (signed with the 1.61.7 salt): the 1.25.0 one, then the
// backpacks, races, skins, restrictions, cookies, VIP subscription, battle to reconnect to and downgrades.
func (e *EnterAccount) Emit161(out Out, serverTime int32) {
	var w wire.Writer
	profile161(&w, &e.UserProfileStructure)
	w.U16(uint16(len(e.UserProfileStructures)))
	for i := range e.UserProfileStructures {
		profile161(&w, &e.UserProfileStructures[i])
	}
	w.U16(0) // loginAwards
	w.U16(uint16(e.OnlineFriends))
	w.U16(uint16(e.Friends))
	w.UTF(str(e.SessionKey, "SessionKey"))
	w.U8(e.AvailableSearchKeys)
	w.U16(uint16(len(e.Reagents)))
	for _, r := range e.Reagents {
		w.U32(r)
	}
	w.I16(0) // curSoloMissionId
	w.I16(0) // curCooperativeMissionId
	w.U16(0) // clan invites
	w.I32(serverTime)
	w.U16(0)            // backpackConfs
	w.U8(0)             // activeBackpackConf
	w.U16(0)            // hotkeys
	w.U8(0)             // loginSequence
	w.I16(e.races161()) // races: a bit per race the player has (the client adds the worm's race itself)
	w.I32(0)            // selectRaceTimeLeft
	w.U16(0)            // skins
	w.I32(0)            // lastPaymentTime
	w.U16(0)            // restrictions
	w.U16(0)            // cookies
	w.I32(0)            // vipSubscriptionId
	w.I16(0)            // no battle to reconnect to
	w.U8(127)           // downgradesLeft: as many as the player wants
	msg := Header(CmdEnterAccount, uint32(len(w.B)+16))
	msg = append(msg, w.B...)
	out(append(msg, wire.SecureWith(w.B, wire.SecretKey161)...))
}

// races161 is the races field of EnterAccount: the race of the player's own worm.
func (e *EnterAccount) races161() int {
	p := &e.UserProfileStructure
	for i := range p.WormsGroup {
		if p.WormsGroup[i].OwnerID == p.ID {
			if race, _ := splitHat(p.WormsGroup[i].Hat); race >= 0 && race < 15 {
				return 1 << race
			}
		}
	}
	return 0
}

// Emit161 writes the 1.61.7 ArenaResult (no MD5): the 1.25.0 fields, then the arena's timers, the wagers
// that are closed, the heroic missions' progress, the awards' tokens, the boss awards (none: the server
// gives them) and the zombie's cooldown.
func (r *ArenaResult) Emit161(out Out) {
	var w wire.Writer
	w.U32(uint32(r.BattlesCount))
	w.I16(int(r.CurrentMission))
	w.I16(int(r.CurrentCoopMission))
	w.Bool(r.BossAvailable)
	w.Bool(false) // superBossAvailable
	w.U16(0)      // heroic missions
	w.U32(0)      // delay
	w.U32(0)      // restoreBattlesDelay
	w.U16(0)      // restrictedWagers
	w.U16(0)      // restrictedWagersLeftTime
	w.U16(0)      // heroicMissionDailyProgress
	w.I32(0)      // defeatContributionMoney
	w.UTF("{}")   // extraBattlesTimetable (JSON)
	w.I32(0)      // wagerWinAwardToken
	w.I32(0)      // bossWinAwardToken
	w.U16(4)      // BossBattleAwardsStructure, its size:
	w.U16(0)      //   boss awards
	w.U16(0)      //   heroic awards
	w.I32(0)      // zombieRiseCoolDownTime
	out(Frame(CmdArenaResult, w.B, false))
}

// Emit161 writes the 1.61.7 StartBattleResult: the battle id and the reagents of the battle (no awards
// list), MD5.
func (r *StartBattleResult) Emit161(out Out) {
	var w wire.Writer
	w.U32(r.BattleID)
	w.U16(uint16(len(r.ReagentForBattle)))
	w.Bytes(r.ReagentForBattle)
	out(Frame(CmdStartBattleResult, w.B, true))
}

// ReadEndBattle161 reads the 1.61.7 EndBattle: result, type (short), experience bonus, battle id, mission,
// items, ban, reagents, then the battle's statistics, journal, map and session key, which the API doesn't
// keep.
func ReadEndBattle161(r *wire.Reader) EndBattle {
	var e EndBattle
	e.Result = int32(r.U32())
	e.Type = int32(int16(r.U16()))
	e.ExpBonus = int32(r.U32())
	e.BattleID = int32(r.U32())
	e.MissionID = int32(int16(r.U16()))
	n := int(r.U16())
	e.Items = []Weapon{}
	for i := 0; i < n; i++ {
		r.U16() // size
		var w Weapon
		w.ID = r.U32()
		w.Count = int32(r.U32())
		e.Items = append(e.Items, w)
	}
	e.Signature = []byte{}
	e.BanType = int16(r.U16())
	e.BanNote = r.UTF()
	rs := int(r.U16())
	e.CollectedReagents = []int32{}
	for i := 0; i < rs; i++ {
		e.CollectedReagents = append(e.CollectedReagents, int32(r.U8()))
	}
	if r.Remaining() <= 16 {
		return e // no statistics (only the signature is left)
	}
	r.U16() // TurnStructureV2.VERSION_MARKER
	r.U16() // turns
	r.U32() // damage to the player
	r.U32() // damage to the boss
	for range 2 { // items used, the journal's turns: each with its size in front
		n := int(r.U16())
		for i := 0; i < n; i++ {
			r.BytesLenient(int(r.U16()))
		}
	}
	r.U16() // map
	e.SessionKey = r.UTF()
	return e
}

// CmdEndBattleResult161 answers EndBattle (1.61.7): the client waits for it ("waiting for data") after a battle.
const CmdEndBattleResult161 = 10121

// EmitEndBattleResult161 writes the 1.61.7 EndBattleResult: validation result (0), the battle's result, id,
// mission and session key as sent, and no awards (the client shows the battle's own), MD5.
func EmitEndBattleResult161(out Out, e EndBattle) {
	var w wire.Writer
	w.I16(0)
	w.U32(uint32(e.Result))
	w.U32(uint32(e.BattleID))
	w.I16(int(e.MissionID))
	w.UTF(e.SessionKey)
	w.U16(0)
	out(Frame(CmdEndBattleResult161, w.B, true))
}

// Emit161 writes the 1.61.7 ArenaLocked: its error code is a short.
func (r ArenaLocked) Emit161(out Out) {
	var w wire.Writer
	w.U32(r.Delay)
	w.U16(uint16(r.MissionID))
	w.I16(int(r.ErrorCode))
	out(Frame(CmdArenaLocked, w.B, false))
}

// Emit161 writes the 1.61.7 LoginError: its code is a short.
func (e LoginError) Emit161(out Out) {
	var w wire.Writer
	w.I16(int(e.Result))
	out(Frame(CmdLoginError, w.B, false))
}
