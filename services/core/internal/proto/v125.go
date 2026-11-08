package proto

import (
	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

// The 1.25.0 client speaks the 1.05.0 protocol with changes: its messages are signed with another salt,
// and some messages (Login, EnterAccount, ...) have a new layout. A session is a 1.25.0 one when its
// Login is signed with the 1.25.0 salt (the 1.05.0 Login isn't signed at all).

// IsLogin125 reports whether a Login body comes from the 1.25.0 client.
func IsLogin125(body []byte) bool { return wire.IsSecureWith(body, wire.SecretKey125) }

// ReadLogin125 reads the 1.25.0 Login: id, referrer, auth key, client version, friend ids, social code,
// social params, MD5. The version and the params aren't used.
func ReadLogin125(r *wire.Reader) Login {
	var l Login
	l.ID = r.U32()
	l.ReferrerID = r.U32()
	l.AuthKey = r.UTF()
	r.U32() // version
	n := int(r.U16())
	l.IDs = []uint32{}
	for i := 0; i < n; i++ {
		l.IDs = append(l.IDs, r.U32()) // big-endian (1.05.0 sends them little-endian)
	}
	l.SocialCode = uint32(r.U8())
	return l
}

// Resign125 gives a complete message signed with the 1.05.0 salt the 1.25.0 salt instead. Anything else
// (unsigned messages, pieces of a message) is returned as it is.
func Resign125(frame []byte) []byte {
	if len(frame) < wire.HeaderSize+16 || wire.FrameLength(frame) != len(frame)-wire.HeaderSize {
		return frame
	}
	body := frame[wire.HeaderSize:]
	payload := body[:len(body)-16]
	if string(wire.Secure(payload)) != string(body[len(body)-16:]) {
		return frame
	}
	out := append([]byte(nil), frame[:len(frame)-16]...)
	return append(out, wire.SecureWith(payload, wire.SecretKey125)...)
}

// splitHat unpacks WormStructure.Hat (race*500 + hat id, or just the race when there is no hat), as the
// API's raceAndHat.
func splitHat(hat int16) (race, hatID int) {
	h := int(hat)
	if h < 50 {
		return h, 0
	}
	return (h - 1000) / 500, 1000 + h%500
}

// writeTeamMember125 is the 1.25.0 TeamMemberStructure (it replaces WormStructure).
func writeTeamMember125(w *wire.Writer, m *Worm) {
	race, hat := splitHat(m.Hat)
	w.U32(m.OwnerID)
	w.U8(byte(m.Armor))
	w.U8(byte(m.Attack))
	w.U8(byte(m.Level))
	w.U32(m.Experience)
	w.I16(hat)
	w.U8(byte(race))
	w.I16(int(m.Artifact))
	if m.SocialOwnerID != nil {
		w.UTF(*m.SocialOwnerID)
	} else {
		w.UTF("")
	}
	w.I16(0)     // teamMemberType: a friend or the player himself
	w.Bool(true) // isActive: every worm of the team fights
}

// writeUserProfile125 is the 1.25.0 UserProfileStructure.
func writeUserProfile125(w *wire.Writer, p *UserProfile) {
	w.U32(p.ID)
	w.U32(p.Money)
	w.U32(p.RealMoney)
	w.I32(p.Rating)
	w.U16(uint16(len(p.WormsGroup)))
	for i := range p.WormsGroup {
		writeTeamMember125(w, &p.WormsGroup[i])
	}
	w.U16(uint16(len(p.WeaponRecordList)))
	for _, x := range p.WeaponRecordList {
		w.U32(x.ID)
		w.I32(x.Count)
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
	w.U16(uint16(len(p.Recipes)))
	for _, x := range p.Recipes {
		w.U16(x)
	}
	// ClanMemberStructure: not in a clan.
	w.I16(0)  // rank
	w.I32(0)  // clanId
	w.UTF("") // clanName
	w.U16(0)  // clanEmblem
	w.I32(0)  // clanRating
	w.I32(0)  // clanSeasonRating
	w.I16(0)  // reviewState
	w.I32(0)  // prevSeasonTopPlace
}

// profile125 is a UserProfileStructure with its size in front, as EnterAccount sends them.
func profile125(w *wire.Writer, p *UserProfile) {
	var b wire.Writer
	writeUserProfile125(&b, p)
	w.U16(uint16(len(b.B)))
	w.Bytes(b.B)
}

// Emit125 writes EnterAccount in the 1.25.0 layout (signed with the 1.25.0 salt).
func (e *EnterAccount) Emit125(out Out, serverTime int32) {
	var w wire.Writer
	profile125(&w, &e.UserProfileStructure)
	w.U16(uint16(len(e.UserProfileStructures)))
	for i := range e.UserProfileStructures {
		profile125(&w, &e.UserProfileStructures[i])
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
	w.U16(0) // backpackConf
	w.U16(0) // hotkeys
	w.U8(0)  // loginSequence
	msg := Header(CmdEnterAccount, uint32(len(w.B)+16))
	msg = append(msg, w.B...)
	out(append(msg, wire.SecureWith(w.B, wire.SecretKey125)...))
}

// Emit125 writes the 1.25.0 ArenaResult (no MD5): battles, solo and co-op mission, boss and super boss
// flags, and the heroic missions (none).
func (r *ArenaResult) Emit125(out Out) {
	emitFixed(out, CmdArenaResult, 4+2+2+1+1+2, false, func(w *wire.Writer) {
		w.U32(uint32(r.BattlesCount))
		w.I16(int(r.CurrentMission))
		w.I16(0) // curCooperativeMissionId
		w.Bool(r.BossAvailable)
		w.Bool(false) // superBossAvaliable
		w.U16(0)      // heroic missions
	})
}

// ReadEndBattle125 reads the 1.25.0 EndBattle: like 1.05.0 (random strings between the first fields),
// but item sizes are big-endian and there is no signature block.
func ReadEndBattle125(r *wire.Reader) EndBattle {
	var e EndBattle
	r.UTF()
	e.Result = int32(r.U32())
	r.UTF()
	e.Type = int32(r.U32())
	r.UTF()
	e.ExpBonus = int32(r.U32())
	r.UTF()
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
	r.BytesLenient(16) // hash
	return e
}
