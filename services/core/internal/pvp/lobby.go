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
//   - 2v2 wager battle: the same WagerBattleRequest with a mode byte (2v2Mode) before the MD5 puts the player in
//     a 2v2 queue per wager. When four are waiting they are split into two teams and all four get
//     WagerBattleRequestResult with the four players appended (see sendWagerResult).

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

// WagerBattleRequest modes (a byte after mapId; clients before the 2v2 patch send none).
const (
	modeDuel = 0
	modeTeam = 1 // 2v2
)

// teamBattleSize is how many players a 2v2 battle takes.
const teamBattleSize = 4

type wagerTicket struct {
	session   Player
	wager     int32
	mode      int
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
	// What's left is the MD5 (16 bytes; tests may leave it out), with the mode byte before it if there is one.
	mode := modeDuel
	if r.Remaining()%16 == 1 {
		mode = int(r.U8())
	}
	if mode != modeDuel && mode != modeTeam {
		sendWagerError(s, wagerErrError)
		return
	}

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

	me := &wagerTicket{session: s, wager: wager, mode: mode, host: host, port: port, mapID: mapID}
	if mode == modeTeam {
		onTeamWagerRequest(me)
		return
	}
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
			if t.mode == modeDuel && t.wager == wager && t.session.UserID() != s.UserID() && Find(t.session.UserID()) == t.session {
				w = t
				break
			}
		}
		if w == nil {
			queue = append(queue, me)
			n := 0
			for _, t := range queue {
				if t.mode == modeDuel && t.wager == wager {
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

// teamSeat is one player of a 2v2 battle in WagerBattleRequestResult.
type teamSeat struct {
	team    int
	profile *proto.UserProfile
}

// sendWagerResult: for 2v2, seats lists all four players in turn order, appended after the reagents as
// u8 count, then per player u8 team and the profile. A 1v1 result has nothing appended, as before.
func sendWagerResult(to *wagerTicket, me, enemy *proto.UserProfile, firstPlayerID uint32, seed int32, battleID uint32, reagents []int32, mapOwner *wagerTicket, seats ...teamSeat) {
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
		if len(seats) > 0 {
			w.U8(byte(len(seats)))
			for _, st := range seats {
				w.U8(byte(st.team))
				writeProfile(w, st.profile)
			}
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

// ---- 2v2 wager battles -------------------------------------------------------------------------------------

func onTeamWagerRequest(me *wagerTicket) {
	s := me.session
	group := func() []*wagerTicket {
		lobbyMu.Lock()
		defer lobbyMu.Unlock()
		dropEverything(s)
		if isInBattle(s.UserID()) {
			sendWagerError(s, wagerErrError)
			return nil
		}
		g := []*wagerTicket{}
		for _, t := range queue {
			if t.mode == modeTeam && t.wager == me.wager && t.session.UserID() != s.UserID() &&
				Find(t.session.UserID()) == t.session {
				g = append(g, t)
				if len(g) == teamBattleSize-1 {
					break
				}
			}
		}
		if len(g) < teamBattleSize-1 {
			queue = append(queue, me)
			lobbyLog("%d waits for a 2v2 %d wager battle (%d of %d)", s.UserID(), me.wager, len(g)+1, teamBattleSize)
			return nil
		}
		queue = slices.DeleteFunc(queue, func(t *wagerTicket) bool { return slices.Contains(g, t) })
		g = append(g, me)
		matching = append(matching, g...)
		return g
	}()
	if group == nil {
		return
	}
	defer func() {
		lobbyMu.Lock()
		matching = slices.DeleteFunc(matching, func(t *wagerTicket) bool { return slices.Contains(group, t) })
		lobbyMu.Unlock()
	}()
	startTeamWagerBattle(group)
}

// startTeamWagerBattle starts a 2v2 battle for four matched tickets (the first waited longest).
func startTeamWagerBattle(group []*wagerTicket) {
	battleID := newBattleID()
	wager := group[0].wager
	mapOwner := group[0]

	// Take the wager from all four, with the same reagents for everyone. Whoever can't pay any more is told so;
	// the ones who already paid get their wager back and go back to the queue.
	profiles := make([]proto.UserProfile, len(group))
	var reagents []int32
	for i, t := range group {
		p, got, err := startBattle(t.session.Token(), battleID, wager, reagents)
		if err != nil {
			lobbyLog("can't start battle %d for %d: %v", battleID, t.session.UserID(), err)
			for _, paid := range group[:i] {
				tryEndBattle(paid.session.Token(), paid.session.UserID(), battleID, 0)
			}
			sendWagerError(t.session, wagerErrorCode(err))
			for j, other := range group {
				if j != i {
					requeue(other)
				}
			}
			return
		}
		profiles[i] = p
		if i == 0 {
			reagents = got
		}
	}

	// Someone pressed Cancel while we were talking to the API: give everyone their wager back.
	lobbyMu.Lock()
	cancelled := slices.ContainsFunc(group, func(t *wagerTicket) bool { return t.cancelled })
	if cancelled {
		lobbyLog("battle %d cancelled before it started", battleID)
		for _, t := range group {
			tryEndBattle(t.session.Token(), t.session.UserID(), battleID, 0)
			if !t.cancelled {
				queue = append(queue, t)
			}
		}
		lobbyMu.Unlock()
		return
	}
	lobbyMu.Unlock()

	// Random teams; teams alternate in the turn order (A1, B1, A2, B2) and a random team starts.
	idx := rand.Perm(len(group))
	order := []int{idx[0], idx[2], idx[1], idx[3]}
	seats := make([]teamSeat, len(order))
	players := make([]*player, len(order))
	for i, gi := range order {
		team := i%2 + 1
		t := group[gi]
		seats[i] = teamSeat{team, &profiles[gi]}
		players[i] = &player{userID: t.session.UserID(), firstTurn: i == 0, team: team, token: t.session.Token()}
	}
	seed := int32(rand.IntN(1000))
	createTeamBattle(battleID, wager, players)
	lobbyLog("2v2 wager %d: %d+%d vs %d+%d, battle %d", wager, players[0].userID, players[2].userID,
		players[1].userID, players[3].userID, battleID)

	for i, gi := range order {
		// The first enemy in the turn order stands in as "the" enemy for code that knows only one.
		enemy := seats[(i+1)%len(seats)].profile
		sendWagerResult(group[gi], &profiles[gi], enemy, players[0].userID, seed, battleID, reagents, mapOwner, seats...)
	}
}
