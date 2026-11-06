package pvp

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/andrij06062003/Wormix/services/core/internal/jsonx"
	"github.com/andrij06062003/Wormix/services/core/internal/proto"
	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

// Lobby: main-server side of PvP, finding an opponent before a battle starts on the PvP server.
//
// Two ways, both from the 1.05.0 client:
//   - Challenge a player by game id: BattleRequest -> the other player gets BattleOffer -> answers
//     BattleOfferResult -> the challenger gets BattleRequestResult. The challenger can CancelBattle.
//   - Wager battle (Arena > PvP): WagerBattleRequest puts the player in a queue per wager amount; the next
//     player asking for the same wager is matched with them and both get WagerBattleRequestResult.
//     WagerCancelBattle leaves the queue.

const (
	cmdBattleRequest       = 1002 // client -> server
	cmdBattleOfferResult   = 1003 // client -> server
	cmdCancelBattle        = 1004 // both ways
	cmdBattleOffer         = 2002 // server -> client
	cmdBattleRequestResult = 2003 // server -> client
	cmdWagerRequest        = 2005 // client -> server
	cmdWagerResult         = 2006 // server -> client
	cmdWagerCancel         = 2007 // client -> server
	cmdWagerError          = 2008 // server -> client
)

const (
	reqAgree           = 0
	reqDisagree        = 1
	reqBusy            = 2
	reqEnemyIDNotFound = 3
	reqNotOnline       = 4
)

const (
	wagerErrError          = 1
	wagerErrNotEnoughMoney = 3
)

// wagers offered in the client's Arena > PvP tab.
var wagers = []int32{15, 50, 300}

// ChallengeTimeout is how long a challenge waits for an answer.
var ChallengeTimeout = 60 * time.Second

type challenge struct {
	battleID    uint32
	from, to    Player
	host        string
	port        int32
	mapID       uint32
	seed        int32
	fromProfile proto.UserProfile
	toProfile   proto.UserProfile
	created     time.Time
}

type wagerTicket struct {
	session   Player
	wager     int32
	host      string
	port      int32
	mapID     uint32
	cancelled bool
}

var (
	lobbyMu    sync.Mutex
	challenges []*challenge // in creation order
	queue      []*wagerTicket
	matching   []*wagerTicket // matched, battle being set up
)

func lobbyLog(format string, args ...any) { logf("[Lobby] %s", fmt.Sprintf(format, args...)) }

func findChallenge(id uint32) *challenge {
	for _, c := range challenges {
		if c.battleID == id {
			return c
		}
	}
	return nil
}

func removeChallenge(c *challenge) {
	challenges = slices.DeleteFunc(challenges, func(x *challenge) bool { return x == c })
}

// HandleLobby handles a PvP command on the main server. Returns false for other commands.
func HandleLobby(s Player, cmd uint32, body []byte) bool {
	switch cmd {
	case cmdBattleRequest, cmdBattleOfferResult, cmdCancelBattle, cmdWagerRequest, cmdWagerCancel:
	default:
		return false
	}
	if s.UserID() == 0 {
		return true // not logged in
	}
	defer func() {
		if r := recover(); r != nil {
			if isEndOfStream(r) {
				lobbyLog("truncated command %d from %d", cmd, s.UserID())
				return
			}
			panic(r)
		}
	}()
	switch cmd {
	case cmdBattleRequest:
		onBattleRequest(s, wire.NewReader(body))
	case cmdBattleOfferResult:
		onBattleOfferResult(s, wire.NewReader(body))
	case cmdCancelBattle:
		onCancelBattle(s)
	case cmdWagerRequest:
		onWagerRequest(s, body)
	case cmdWagerCancel:
		onWagerCancel(s)
	}
	return true
}

// isBusy: call with lobbyMu held.
func isBusy(s Player) bool {
	for _, c := range challenges {
		if c.from == s || c.to == s {
			return true
		}
	}
	for _, t := range queue {
		if t.session == s {
			return true
		}
	}
	for _, t := range matching {
		if t.session == s {
			return true
		}
	}
	return isInBattle(s.UserID())
}

// ---- Challenges --------------------------------------------------------------------------------------------

func isNetWhiteSpace(r rune) bool { return unicode.IsSpace(r) || r == 0x85 || r == 0xA0 }

func onBattleRequest(from Player, r *wire.Reader) {
	host := r.UTFStrict()
	port := int32(r.U32())
	mapID := r.U32()
	seed := int32(r.U32())
	enemyID := r.UTFStrict()

	toID, ok := jsonx.ParseNetUint32(strings.TrimFunc(enemyID, isNetWhiteSpace))
	if !ok || toID == 0 || toID == from.UserID() {
		sendRequestResult(from, reqEnemyIDNotFound, nil, nil, 0)
		return
	}
	to := Find(toID)
	if to == nil {
		sendRequestResult(from, reqNotOnline, nil, nil, 0)
		return
	}

	c := func() *challenge {
		lobbyMu.Lock()
		defer lobbyMu.Unlock()
		dropEverything(from) // a stale request or queue place of the challenger's own
		if isBusy(to) {
			sendRequestResult(from, reqBusy, nil, nil, 0)
			return nil
		}
		c := &challenge{battleID: newBattleID(), from: from, to: to, host: host, port: port, mapID: mapID, seed: seed, created: time.Now()}
		challenges = append(challenges, c)
		return c
	}()
	if c == nil {
		return
	}

	fromProfile, err := getProfile(from.Token())
	var toProfile proto.UserProfile
	if err == nil {
		toProfile, err = getProfile(to.Token())
	}
	if err != nil {
		lobbyLog("can't load profiles for %d -> %d: %v", from.UserID(), to.UserID(), err)
		lobbyMu.Lock()
		removeChallenge(c)
		lobbyMu.Unlock()
		sendRequestResult(from, reqEnemyIDNotFound, nil, nil, 0)
		return
	}

	lobbyMu.Lock()
	if findChallenge(c.battleID) == nil {
		lobbyMu.Unlock()
		return // cancelled meanwhile
	}
	c.fromProfile, c.toProfile = fromProfile, toProfile
	lobbyMu.Unlock()

	lobbyLog("%d challenges %d (battle %d)", from.UserID(), to.UserID(), c.battleID)
	to.SendMessage(message(cmdBattleOffer, false, func(w *wire.Writer) {
		w.UTF(c.host)
		w.I32(c.port)
		w.U32(c.mapID)
		w.I32(c.seed)
		writeProfile(w, &fromProfile) // enemy (for the one receiving the offer)
		writeProfile(w, &toProfile)   // their own
		w.U32(c.battleID)
	}))
}

func onBattleOfferResult(to Player, r *wire.Reader) {
	res := int(int16(r.U16()))
	r.U32() // enemy id
	battleID := r.U32()

	lobbyMu.Lock()
	c := findChallenge(battleID)
	if c == nil || c.to != to {
		lobbyMu.Unlock()
		return
	}
	removeChallenge(c)
	fromProfile, toProfile := c.fromProfile, c.toProfile
	lobbyMu.Unlock()

	if res != reqAgree {
		lobbyLog("%d declined %d (result %d)", to.UserID(), c.from.UserID(), res)
		answer := reqDisagree
		if res == reqBusy {
			answer = reqBusy
		}
		sendRequestResult(c.from, answer, nil, nil, 0)
		return
	}

	// The player who accepted is already loading the battle. The challenger moves first.
	lobbyLog("%d accepted %d: battle %d", to.UserID(), c.from.UserID(), c.battleID)
	createBattle(c.battleID, 0, &player{userID: c.from.UserID(), firstTurn: true, token: c.from.Token()},
		&player{userID: to.UserID(), firstTurn: false, token: to.Token()})
	// Register the battle with the API so the players' EndBattle (used weapons) is accepted. No wager in friend battles.
	for _, s := range []Player{c.from, to} {
		if _, _, err := startBattle(s.Token(), c.battleID, 0, []int32{}); err != nil {
			lobbyLog("can't register battle %d for %d: %v", c.battleID, s.UserID(), err)
		}
	}
	sendRequestResult(c.from, reqAgree, &toProfile, &fromProfile, c.battleID)
}

func onCancelBattle(from Player) {
	lobbyMu.Lock()
	var cancelled []*challenge
	for _, c := range challenges {
		if c.from == from {
			cancelled = append(cancelled, c)
		}
	}
	for _, c := range cancelled {
		removeChallenge(c)
	}
	lobbyMu.Unlock()
	for _, c := range cancelled {
		lobbyLog("%d cancelled the challenge to %d", from.UserID(), c.to.UserID())
		sendCancel(c)
	}
}

func sendCancel(c *challenge) {
	c.to.SendMessage(message(cmdCancelBattle, false, func(w *wire.Writer) {
		w.U32(c.from.UserID())
		w.U32(c.to.UserID())
	}))
}

func sendRequestResult(s Player, res int, enemy, me *proto.UserProfile, battleID uint32) {
	s.SendMessage(message(cmdBattleRequestResult, false, func(w *wire.Writer) {
		w.I16(res)
		if res != reqAgree {
			return
		}
		writeProfile(w, enemy)
		writeProfile(w, me)
		w.U32(battleID)
	}))
}

// ---- Wager battles -----------------------------------------------------------------------------------------

func onWagerRequest(s Player, body []byte) {
	r := wire.NewReader(body)
	host := r.UTFStrict()
	port := int32(r.U32())
	wager := int32(r.U32())
	mapID := r.U32()

	if !slices.Contains(wagers, wager) {
		sendWagerError(s, wagerErrError)
		return
	}
	profile, err := getProfile(s.Token())
	if err != nil {
		lobbyLog("can't load profile of %d: %v", s.UserID(), err)
		sendWagerError(s, wagerErrError)
		return
	}
	if int64(profile.Money) < int64(wager) {
		sendWagerError(s, wagerErrNotEnoughMoney)
		return
	}

	me := &wagerTicket{session: s, wager: wager, host: host, port: port, mapID: mapID}
	waiting := func() *wagerTicket {
		lobbyMu.Lock()
		defer lobbyMu.Unlock()
		dropEverything(s)
		if isInBattle(s.UserID()) {
			sendWagerError(s, wagerErrError)
			return nil
		}
		var w *wagerTicket
		for _, t := range queue {
			if t.wager == wager && t.session.UserID() != s.UserID() && Find(t.session.UserID()) == t.session {
				w = t
				break
			}
		}
		if w == nil {
			queue = append(queue, me)
			n := 0
			for _, t := range queue {
				if t.wager == wager {
					n++
				}
			}
			lobbyLog("%d waits for a %d wager battle (%d waiting)", s.UserID(), wager, n)
			return nil
		}
		queue = slices.DeleteFunc(queue, func(t *wagerTicket) bool { return t == w })
		matching = append(matching, w, me)
		return w
	}()
	if waiting == nil {
		return
	}
	defer func() {
		lobbyMu.Lock()
		matching = slices.DeleteFunc(matching, func(t *wagerTicket) bool { return t == waiting || t == me })
		lobbyMu.Unlock()
	}()
	startWagerBattle(waiting, me)
}

func startWagerBattle(first, second *wagerTicket) {
	battleID := newBattleID()
	wager := first.wager

	// Take the wager from both. If the second can't pay any more, give the first theirs back.
	aProfile, aReagents, err := startBattle(first.session.Token(), battleID, wager, nil)
	if err != nil {
		lobbyLog("can't start battle %d for %d: %v", battleID, first.session.UserID(), err)
		sendWagerError(first.session, wagerErrorCode(err))
		requeue(second)
		return
	}
	bProfile, _, err := startBattle(second.session.Token(), battleID, wager, aReagents)
	if err != nil {
		lobbyLog("can't start battle %d for %d: %v", battleID, second.session.UserID(), err)
		tryEndBattle(first.session.Token(), first.session.UserID(), battleID, 0)
		sendWagerError(second.session, wagerErrorCode(err))
		requeue(first)
		return
	}

	// Someone pressed Cancel while we were talking to the API: give both their wager back.
	lobbyMu.Lock()
	if first.cancelled || second.cancelled {
		lobbyLog("battle %d cancelled before it started", battleID)
		tryEndBattle(first.session.Token(), first.session.UserID(), battleID, 0)
		tryEndBattle(second.session.Token(), second.session.UserID(), battleID, 0)
		if !first.cancelled {
			queue = append(queue, first)
		}
		if !second.cancelled {
			queue = append(queue, second)
		}
		lobbyMu.Unlock()
		return
	}
	lobbyMu.Unlock()

	firstMoves := second
	if rand.IntN(2) == 0 {
		firstMoves = first
	}
	seed := int32(rand.IntN(1000))
	createBattle(battleID, wager,
		&player{userID: first.session.UserID(), firstTurn: firstMoves == first, token: first.session.Token()},
		&player{userID: second.session.UserID(), firstTurn: firstMoves == second, token: second.session.Token()})
	lobbyLog("wager %d: %d vs %d, battle %d", wager, first.session.UserID(), second.session.UserID(), battleID)

	// The map is the one the player who waited longer picked.
	sendWagerResult(first, &aProfile, &bProfile, firstMoves.session.UserID(), seed, battleID, aReagents, first)
	sendWagerResult(second, &bProfile, &aProfile, firstMoves.session.UserID(), seed, battleID, aReagents, first)
}

func wagerErrorCode(err error) int {
	if isNotEnoughMoney(err) {
		return wagerErrNotEnoughMoney
	}
	return wagerErrError
}

func requeue(t *wagerTicket) {
	lobbyMu.Lock()
	defer lobbyMu.Unlock()
	if !t.cancelled && Find(t.session.UserID()) == t.session {
		queue = append(queue, t)
	}
}

func sendWagerResult(to *wagerTicket, me, enemy *proto.UserProfile, firstPlayerID uint32, seed int32, battleID uint32, reagents []int32, mapOwner *wagerTicket) {
	to.session.SendMessage(message(cmdWagerResult, true, func(w *wire.Writer) {
		w.UTF(mapOwner.host)
		w.I32(mapOwner.port)
		w.U32(mapOwner.mapID)
		writeProfile(w, enemy)
		writeProfile(w, me)
		w.U32(firstPlayerID)
		w.I32(seed)
		w.U32(battleID)
		w.U16(uint16(len(reagents)))
		for _, r := range reagents {
			w.U8(byte(r))
		}
	}))
}

func sendWagerError(s Player, code int) {
	s.SendMessage(message(cmdWagerError, false, func(w *wire.Writer) { w.I16(code) }))
}

func onWagerCancel(s Player) {
	lobbyMu.Lock()
	defer lobbyMu.Unlock()
	for _, t := range append(slices.Clone(queue), matching...) {
		if t.session == s {
			t.cancelled = true
		}
	}
	before := len(queue)
	queue = slices.DeleteFunc(queue, func(t *wagerTicket) bool { return t.session == s })
	if len(queue) < before {
		lobbyLog("%d left the wager queue", s.UserID())
	}
}

// ---- Cleanup -----------------------------------------------------------------------------------------------

// dropEverything forgets everything the player was waiting for (call with lobbyMu held).
func dropEverything(s Player) {
	for _, t := range append(slices.Clone(queue), matching...) {
		if t.session == s {
			t.cancelled = true
		}
	}
	queue = slices.DeleteFunc(queue, func(t *wagerTicket) bool { return t.session == s })
	for _, c := range slices.Clone(challenges) {
		if c.from == s {
			removeChallenge(c)
			sendCancel(c)
		}
	}
}

// OnMainDisconnected is called when a player's main connection closes.
func OnMainDisconnected(s Player) {
	if s.UserID() == 0 {
		return
	}
	lobbyMu.Lock()
	defer lobbyMu.Unlock()
	dropEverything(s)
	for _, c := range slices.Clone(challenges) {
		if c.to == s {
			removeChallenge(c)
			sendRequestResult(c.from, reqNotOnline, nil, nil, 0)
		}
	}
}

// lobbyTick expires unanswered challenges. Called about once a second.
func lobbyTick() {
	lobbyMu.Lock()
	defer lobbyMu.Unlock()
	now := time.Now()
	for _, c := range slices.Clone(challenges) {
		if now.Sub(c.created) > ChallengeTimeout {
			removeChallenge(c)
			lobbyLog("challenge %d -> %d not answered", c.from.UserID(), c.to.UserID())
			sendCancel(c)
			sendRequestResult(c.from, reqDisagree, nil, nil, 0)
		}
	}
}
