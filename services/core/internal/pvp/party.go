package pvp

import (
	"math/rand/v2"
	"slices"

	"github.com/andrij06062003/Wormix/services/core/internal/proto"
)

// Teams (parties) for 2v2 battles.
//
// A player invites another into their team with BattleRequest plus a mode byte 1; the other gets a
// BattleOffer of kind offerTeamInvite and answers with BattleOfferResult as for a challenge. A team holds
// two players and lasts until one of them logs out or either joins another team; the other is then told
// with an offerTeamLeft BattleOffer.
//
// Teammates are matched together. When one of them asks for a 2v2 wager battle and the other isn't looking
// yet, the other gets an offerTeamCall BattleOffer (the wager in seed); agreeing makes their client send its
// own WagerBattleRequest. The pair is only matched once both are in the queue for the same wager, and
// always plays on the same side.

// teams maps each player in a team to their teammate (both ways). Guarded by lobbyMu.
var teams = map[uint32]uint32{}

// teamPartner returns the logged-in teammate of a player, or nil (call with lobbyMu held).
func teamPartner(userID uint32) Player {
	id, ok := teams[userID]
	if !ok {
		return nil
	}
	return Find(id)
}

// leaveTeam takes the player out of their team and returns the former teammate if online (lobbyMu held).
func leaveTeam(userID uint32) Player {
	id, ok := teams[userID]
	if !ok {
		return nil
	}
	delete(teams, userID)
	delete(teams, id)
	return Find(id)
}

// sendTeamLeft tells a player their teammate left the team. Only the ids in the profiles matter.
func sendTeamLeft(to Player, leaverID uint32) {
	c := &challenge{kind: offerTeamLeft}
	from, me := idOnlyProfile(leaverID), idOnlyProfile(to.UserID())
	to.SendMessage(offerMessage(c, &from, &me))
}

// idOnlyProfile is an empty but complete profile structure with just the id.
func idOnlyProfile(id uint32) proto.UserProfile {
	social := ""
	return proto.UserProfile{ID: id, SocialID: &social, WormsGroup: []proto.Worm{}, WeaponRecordList: []proto.Weapon{},
		Stuff: []uint16{}, Recipes: []uint16{}}
}

func onTeamInviteAnswer(c *challenge, res int) {
	if res != reqAgree {
		lobbyLog("%d declined the team invite of %d (result %d)", c.to.UserID(), c.from.UserID(), res)
		answer := reqDisagree
		if res == reqBusy {
			answer = reqBusy
		}
		sendRequestResult(c.from, answer, nil, nil, 0)
		return
	}
	type notice struct {
		to     Player
		leaver uint32
	}
	var notices []notice
	lobbyMu.Lock()
	for _, p := range []Player{c.from, c.to} {
		if old := leaveTeam(p.UserID()); old != nil && old != c.from && old != c.to {
			teamBroken(old)
			notices = append(notices, notice{old, p.UserID()})
		}
	}
	teams[c.from.UserID()] = c.to.UserID()
	teams[c.to.UserID()] = c.from.UserID()
	lobbyMu.Unlock()
	for _, n := range notices {
		sendTeamLeft(n.to, n.leaver)
	}
	lobbyLog("%d and %d are a team", c.from.UserID(), c.to.UserID())
	sendRequestResult(c.from, reqAgree, &c.toProfile, &c.fromProfile, 0)
}

// onTeamCallAnswer: the teammate answered a call to a 2v2 battle. On agree their client asks for the battle
// itself; otherwise the caller stops waiting, unless the teammate is already looking for the same battle.
func onTeamCallAnswer(c *challenge, res int) {
	if res == reqAgree {
		lobbyLog("%d joins teammate %d", c.to.UserID(), c.from.UserID())
		return
	}
	lobbyMu.Lock()
	defer lobbyMu.Unlock()
	if t := queuedTeamTicket(c.to); t != nil && t.wager == c.seed {
		return // they asked for it themselves meanwhile (their client was busy, hence the answer)
	}
	lobbyLog("%d didn't join teammate %d (result %d)", c.to.UserID(), c.from.UserID(), res)
	failTeamCallLocked(c)
}

// failTeamCall: the teammate can't join; the caller leaves the queue and is told (takes lobbyMu).
func failTeamCall(c *challenge) {
	lobbyMu.Lock()
	defer lobbyMu.Unlock()
	failTeamCallLocked(c)
}

func failTeamCallLocked(c *challenge) {
	if t := queuedTeamTicket(c.to); t != nil && t.wager == c.seed {
		return // the teammate is in the queue for this battle after all
	}
	before := len(queue)
	queue = slices.DeleteFunc(queue, func(t *wagerTicket) bool {
		return t.session == c.from && t.mode == modeTeam && t.wager == c.seed
	})
	if len(queue) < before {
		sendWagerError(c.from, wagerErrTeammate)
	}
}

// teamBroken: the player's teammate left, so they stop waiting for a 2v2 battle together (lobbyMu held).
func teamBroken(p Player) {
	before := len(queue)
	queue = slices.DeleteFunc(queue, func(t *wagerTicket) bool { return t.session == p && t.mode == modeTeam })
	if len(queue) < before {
		sendWagerError(p, wagerErrTeammate)
	}
}

// queuedTeamTicket returns the player's ticket in the 2v2 queue, or nil (lobbyMu held).
func queuedTeamTicket(p Player) *wagerTicket {
	for _, t := range queue {
		if t.session == p && t.mode == modeTeam && !t.cancelled {
			return t
		}
	}
	return nil
}

// matchTeams picks four players from the 2v2 queue for this wager (lobbyMu held): teammates count as one
// unit and only once both are queued. Returns them in turn order (sides alternate, teammates on one side,
// a random side and order), or nil if there aren't enough yet.
func matchTeams(wager int32) []*wagerTicket {
	var units [][]*wagerTicket
	used := map[*wagerTicket]bool{}
	for _, t := range queue {
		if used[t] || t.mode != modeTeam || t.wager != wager || t.cancelled || Find(t.session.UserID()) != t.session {
			continue
		}
		if partner := teamPartner(t.session.UserID()); partner != nil {
			pt := queuedTeamTicket(partner)
			if pt == nil || pt.wager != wager || used[pt] {
				continue // waiting for the teammate
			}
			units = append(units, []*wagerTicket{t, pt})
			used[t], used[pt] = true, true
			continue
		}
		units = append(units, []*wagerTicket{t})
		used[t] = true
	}

	// Oldest first: take units while they fit.
	var picked [][]*wagerTicket
	n := 0
	for _, u := range units {
		if n+len(u) <= teamBattleSize {
			picked = append(picked, u)
			n += len(u)
		}
		if n == teamBattleSize {
			break
		}
	}
	if n < teamBattleSize {
		return nil
	}

	// Sides: each pair is a side; single players fill the rest in the order they came.
	var sides [2][]*wagerTicket
	var singles []*wagerTicket
	s := 0
	for _, u := range picked {
		if len(u) == 2 {
			sides[s] = slices.Clone(u)
			s++
		} else {
			singles = append(singles, u[0])
		}
	}
	for _, t := range singles {
		if len(sides[0]) < 2 {
			sides[0] = append(sides[0], t)
		} else {
			sides[1] = append(sides[1], t)
		}
	}
	if len(singles) == 4 {
		// Four strangers: random sides.
		rand.Shuffle(len(singles), func(i, j int) { singles[i], singles[j] = singles[j], singles[i] })
		sides = [2][]*wagerTicket{singles[:2], singles[2:]}
	}
	if rand.IntN(2) == 0 {
		sides[0], sides[1] = sides[1], sides[0]
	}
	for _, side := range sides {
		if rand.IntN(2) == 0 {
			side[0], side[1] = side[1], side[0]
		}
	}
	return []*wagerTicket{sides[0][0], sides[1][0], sides[0][1], sides[1][1]}
}
