package pvp

import (
	"github.com/andrij06062003/Wormix/services/core/internal/api"
	"github.com/andrij06062003/Wormix/services/core/internal/csx"
	"github.com/andrij06062003/Wormix/services/core/internal/jsonx"
	"github.com/andrij06062003/Wormix/services/core/internal/proto"
)

// Co-op bosses of the 1.25.0 client (missions 101+): the player picks a friend from GetFriendsForBoss, calls
// it with CreateBattleRequest (PVE_FRIEND) like a friend duel, and both fight the boss on the PvP server
// (Battle with pve set). Everyone here is a friend: the list is the 1.25.0 players online and free.

// coopState is pvp/coop_state: the player's profile and whether it may fight the co-op boss now
// (GetFriendsForBossResult state).
func coopState(token string, mission int16) (p proto.UserProfile, state int16, err error) {
	obj, err := api.PostPvP("pvp/coop_state", token, jsonx.NewObject().Int("MissionId", int64(mission)).Bytes())
	if err != nil {
		return p, 0, err
	}
	err = catchErr(func() {
		data := obj.Get("data")
		if data == nil || data.Kind != jsonx.Object {
			csx.NullRef("data")
		}
		ups := data.Obj.Get("UserProfileStructure")
		if ups == nil {
			csx.NullRef("UserProfileStructure")
		}
		p = proto.DecodeUserProfile(ups)
		if s := data.Obj.Get("State"); s != nil {
			state = int16(s.Int32())
		}
	})
	return p, state, err
}

// BossFriends lists the friends the player can call to the co-op boss (its first mission id) with whether
// each can come: the other 1.25.0 players online who aren't in a battle or setting one up.
func BossFriends(me uint32, missions []int16) []proto.BossFriend125 {
	out := []proto.BossFriend125{}
	if len(missions) == 0 {
		return out
	}
	onlineMu.RLock()
	var cands []Player
	for id, p := range online {
		if id != me && is125(p) {
			cands = append(cands, p)
		}
	}
	onlineMu.RUnlock()
	for _, p := range cands {
		exMu.Lock()
		busy := exBusyLocked(p.UserID())
		exMu.Unlock()
		if busy {
			continue
		}
		prof, state, err := coopState(p.Token(), missions[0])
		if err != nil {
			logf("[PvP] can't check %d for co-op boss %d: %v", p.UserID(), missions[0], err)
			continue
		}
		out = append(out, proto.BossFriend125{Profile: prof, State: state})
	}
	return out
}
