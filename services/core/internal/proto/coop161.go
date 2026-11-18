package proto

import (
	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

// Co-op (and friend battles) of the 1.61.7 client: the messages that differ from 1.25.0 (from the client's
// GetFriendsForMission*, PvpProfileStructure and pvp/ex/*BinarySerializer classes).

// ReadGetFriendsForMission161 reads the 1.61.7 GetFriendsForMission (93): the boss's mission ids (short count,
// shorts) and the battle's wager id (12 for a co-op boss).
func ReadGetFriendsForMission161(r *wire.Reader) (missions []int16, wager int16) {
	n := int(r.U16())
	missions = make([]int16, 0, n)
	for i := 0; i < n; i++ {
		missions = append(missions, int16(r.U16()))
	}
	if r.Remaining() >= 2 {
		wager = int16(r.U16())
	}
	return missions, wager
}

// ownWorm is the player's own worm in its team (the zero worm if it has none).
func ownWorm(p *UserProfile) Worm {
	for _, x := range p.WormsGroup {
		if x.OwnerID == p.ID {
			return x
		}
	}
	return Worm{}
}

// writeSimpleProfile161 is the 1.61.7 SimpleProfileStructure: a player as the friends lists show it.
func writeSimpleProfile161(w *wire.Writer, p *UserProfile, name string) {
	me := ownWorm(p)
	race, hat := splitHat(me.Hat)
	w.U32(p.ID)
	w.UTF(name)
	w.I32(int32(me.Armor))
	w.I32(int32(me.Attack))
	w.I32(int32(me.Level))
	w.I32(int32(p.Money))
	w.I32(int32(p.RealMoney))
	w.I32(p.ReactionRate)
	w.I32(p.Rating)
	w.I32(int32(len(p.WormsGroup)))
	w.I16(hat)
	w.I16(race)
	w.I16(int(me.Artifact))
	if p.SocialID != nil {
		w.UTF(*p.SocialID)
	} else {
		w.UTF("")
	}
	w.U8(0) // socialNetId
	var clan wire.Writer
	writeNoClan(&clan)
	w.U16(uint16(len(clan.B)))
	w.Bytes(clan.B)
	w.U8(0) // rank
	w.U8(0) // skin
}

// EmitGetFriendsForMissionResult161 writes the 1.61.7 GetFriendsForMissionResult (1093): the friends (size +
// SimpleProfileStructure each), then their states (0 can come, 1 no battles left, 2 the boss isn't open).
func EmitGetFriendsForMissionResult161(out Out, friends []BossFriend125) {
	var w wire.Writer
	w.U16(uint16(len(friends)))
	for i := range friends {
		var b wire.Writer
		writeSimpleProfile161(&b, &friends[i].Profile, friends[i].Name)
		w.U16(uint16(len(b.B)))
		w.Bytes(b.B)
	}
	w.U16(uint16(len(friends)))
	for _, f := range friends {
		w.I16(int(f.State))
	}
	out(Frame(CmdGetFriendsForBossResult125, w.B, false))
}

// WritePvpProfile161 is the 1.61.7 PvpProfileStructure with its size in front: the UserProfileStructure, then
// the player's worm names (JSON), its player and team numbers, daily rating, backpack, the worms it fights
// with (none: the client takes its team) and the best rank of each season.
func WritePvpProfile161(w *wire.Writer, p *UserProfile, names string, playerNum, team byte) {
	var b wire.Writer
	writeUserProfile161(&b, p)
	b.UTF(names)
	b.U8(playerNum)
	b.U8(team)
	b.I32(0) // dailyRating
	b.U16(0) // backpackConf
	b.U16(0) // activeTeamMembers
	b.U16(0) // seasonsBestRank
	w.U16(uint16(len(b.B)))
	w.Bytes(b.B)
}
