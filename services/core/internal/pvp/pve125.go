package pvp

import (
	"slices"
	"time"

	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

// Co-op boss battles (Battle.pve) of the 1.25.0 client: the players fight the client's AI together.
//
// Every client runs the whole game, AI included, and decides itself whose turn it is (it ignores the player
// number of StartTurn). So the server:
//   - takes the turn's mover from its moves: the first player to send an Action or EndTurn this turn is the
//     actor, the others answer it with EndTurnResponse;
//   - on an AI turn nobody moves: every client plays the AI and answers EndTurnResponse;
//   - ends the turn when every player who still has worms has answered (and the actor has ended it), checks
//     they agree on what was eliminated and sends StartTurn.
//
// The client doesn't say who won a battle online: when the boss is dead it just waits for PvpEndBattle
// ("Ожидание результата боя..."). The players lose as soon as all their squads are eliminated (squads
// 0..n-1; the AI's are n and up). The battle is won when it goes quiet: a turn has started and nobody
// moves or answers for longer than an AI turn takes (pveQuietLimit) while everyone is connected.
//
// A player who leaves before their worms are dead (surrender, no reconnect, stuck) cancels the battle for
// the others (their battle is given back) and loses it.

// aiSquad marks an AI squad number among the eliminated (players are their user ids).
const aiSquad = 0x80000000

// claimTurn makes p the mover of this turn if nobody is yet.
func (b *Battle) claimTurn(p *player) {
	if b.actor == nil && !p.left && !p.out && p.end == nil {
		b.actor = p
	}
}

// pveCancel: p loses, the others get their battle back.
func (b *Battle) pveCancel(p *player) []result {
	res := b.all(0)
	for i, x := range b.players {
		if x == p {
			res[i].r = -1
		}
	}
	return res
}

// pveResults: everyone still in the battle gets r, those who left lose.
func (b *Battle) pveResults(r int) []result {
	res := b.all(r)
	for i, x := range b.players {
		if x.left {
			res[i].r = -1
		}
	}
	return res
}

func (b *Battle) pveQuietLimit() time.Duration {
	if b.maxAITurn == 0 {
		return PveOverTimeout
	}
	return min(max(2*b.maxAITurn+5*time.Second, PveOverMin), PveOverTimeout)
}

func (b *Battle) finishTurnPve() {
	if b.Over() || (b.actor != nil && b.activeEnd == nil) {
		return
	}
	resp := b.responders()
	for _, p := range resp {
		if p.end == nil {
			return
		}
	}
	ref := b.activeEnd
	if ref == nil {
		if len(resp) == 0 {
			return
		}
		ref = resp[0].end
	}
	for _, p := range resp {
		if !sameSet(ref.eliminated, p.end.eliminated) {
			b.logf("desync at turn %d: eliminated [%s] vs [%s] (%d)", b.turn, joinIDs(ref.eliminated),
				joinIDs(p.end.eliminated), p.userID)
			b.finish(b.all(0), true)
			return
		}
		if ref.state != p.end.state {
			b.logf("state hash differs at turn %d (%d vs %d); continuing", b.turn, ref.state, p.end.state)
		}
	}
	now := time.Now()
	if b.actor == nil && b.turn > 1 {
		b.maxAITurn = max(b.maxAITurn, now.Sub(b.turnStartedAt))
	}
	alive := false
	for _, x := range b.players {
		if slices.Contains(ref.eliminated, x.userID) {
			x.out = true
		}
		if !x.out && !x.left {
			alive = true
		}
	}
	if !alive {
		b.logf("all players' squads eliminated at turn %d: lost", b.turn)
		b.finish(b.all(-1), false)
		return
	}

	b.turn++
	b.actor = nil
	b.turnCommands = nil
	b.lastCommandNum = 0
	b.activeEnd = nil
	b.firstRespAt = time.Time{}
	for _, x := range b.players {
		x.end = nil
	}
	b.lastActivity, b.turnStartedAt, b.quietSince = now, now, now
	b.lastStartTurn = b.msg(CmdStartTurn, true, func(w *wire.Writer) {
		w.U32(b.ID)
		w.U8(0) // the client picks the mover itself
		w.U16(uint16(b.turn))
		w.U16(0)
		w.U16(0) // dropped players
	})
	for _, x := range b.players {
		if !x.left {
			send(x, b.lastStartTurn)
		}
	}
}

func (b *Battle) tickPve(now time.Time) {
	// Someone never showed up, or didn't come back after a disconnect.
	for _, p := range b.players {
		if p.left || !((!p.everConnected && now.Sub(b.created) > NoShowTimeout) ||
			(p.disconnectedAt != nil && now.Sub(*p.disconnectedAt) > ReconnectTimeout)) {
			continue
		}
		p.left = true
		if p.out {
			b.logf("%d (eliminated) didn't come back", p.userID)
			b.finishTurnPve()
			if b.Over() {
				return
			}
			continue
		}
		b.logf("%d didn't come back: cancelled", p.userID)
		b.sendOthers(p, b.sysMsg(sysDroppedByReconnectionTimeout, p))
		b.finish(b.pveCancel(p), false)
		return
	}
	if !slices.ContainsFunc(b.players, func(p *player) bool { return !p.left }) {
		b.finish(b.all(0), false)
		return
	}

	switch {
	case b.actor != nil && b.activeEnd == nil:
		// The mover stopped playing (the client sends moves about every second, even idle).
		if act := b.actor; act.connected() && now.Sub(b.lastActivity) > TurnTimeout {
			b.logf("%d inactive: cancelled", act.userID)
			b.sendOthers(act, b.sysMsg(sysDroppedByCommandTimeout, act))
			b.finish(b.pveCancel(act), false)
		}
		return
	case b.activeEnd != nil || !b.firstRespAt.IsZero():
		var waiting []*player
		for _, p := range b.responders() {
			if p.end == nil && p.connected() {
				waiting = append(waiting, p)
			}
		}
		if len(waiting) == 0 {
			return
		}
		since := b.firstRespAt
		if b.activeEnd != nil {
			since = b.activeEndAt
		}
		if now.Sub(since) > ResponseTimeout {
			pas := waiting[0]
			b.logf("%d never confirmed the turn: cancelled", pas.userID)
			b.sendOthers(pas, b.sysMsg(sysDroppedByResponseTimeout, pas))
			b.finish(b.pveCancel(pas), false)
		} else if b.activeEnd != nil && now.Sub(b.lastEndTurn) > EndTurnResendInterval {
			b.lastEndTurn = now
			if n := len(b.turnCommands); n > 0 {
				for _, p := range waiting {
					send(p, b.turnCommands[n-1].msg)
				}
			}
		}
		return
	}

	// Nothing this turn yet: an AI turn being played, or the battle is over on the clients.
	for _, p := range b.players {
		if !p.left && !p.connected() {
			return // wait for it to come back
		}
	}
	quiet := now.Sub(b.quietSince)
	if b.turn == 1 {
		if quiet > TurnTimeout {
			b.logf("nobody started the battle: cancelled")
			b.finish(b.all(0), false)
		}
		return
	}
	if quiet > b.pveQuietLimit() {
		b.logf("quiet for %s at turn %d: won", quiet.Round(time.Second), b.turn)
		b.finish(b.pveResults(1), false)
	}
}
