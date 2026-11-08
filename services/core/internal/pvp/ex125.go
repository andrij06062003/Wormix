package pvp

import (
	"math/rand/v2"
	"slices"
	"sync"
	"time"

	"github.com/andrij06062003/Wormix/services/core/internal/proto"
	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

// 1.25.0 PvP (the client's "ex" protocol). The 1.25.0 client finds its battle on the PvP connection itself,
// not on the main server:
//   - Friend duel: the caller sends CreateBattleRequest (FRIEND_PVP) and gets CallToBattle; the friend gets
//     CallToBattle on the main connection, connects to the PvP server and answers JoinToBattle or
//     RejectBattleOffer. When everyone has joined, all get BattleCreated.
//   - Wager battle: CreateBattleRequest (WAGER_*) queues the player per wager; when enough players wait, all
//     get BattleCreated.
//
// After BattleCreated every client loads the map and sends ReadyForBattle; when all are ready each gets
// StartBattle (with its player number) and the battle runs on the same connection as a 1.05.0 one (Battle,
// with v125 set). Until then either side can CancelBattle; a failure is BattleCreationFailure.
//
// Player numbers are places in the battle's player list (from 0), which is also the turn order.

const (
	// client -> server
	cmdExCreateBattle = 1001 // 1.05.0's BattleLogin has the same id: this one is signed
	cmdExJoin         = 1002
	cmdExReject       = 1003
	cmdExCancel       = 1004
	cmdExWidenSearch  = 1008
	cmdExReady        = 1023

	// server -> client
	cmdExStartBattle     = 2001
	cmdExCallToBattle    = 2002 // also on the main connection, to the friend
	cmdExCreationFailure = 2003
	cmdExBattleCreated   = 2006
)

// PvpBattleType.
const (
	exPveFriend      = 1
	exPvePartner     = 2
	exFriendPvp      = 3
	exWagerDuel      = 4
	exWager4ForAll   = 5
	exWager2x2       = 6
	exWager2x2Friend = 7
	exWager3ForAll   = 8
)

// CallToBattleResultType: the reason of a BattleCreationFailure.
const (
	exServerError  = 0
	exRejected     = 2
	exOffline      = 3
	exBusy         = 4
	exCanceled     = 6
	exTimeout      = 7
	exClientError  = 8
	exDisconnected = 9
	exNoMoney      = 10
)

// sysPlayerJoined: the PvpSystemMessage the caller gets when the friend has joined.
const sysPlayerJoined = 11

// exReadyTimeout: how long the players get to load the map (the client gives up after 30 s).
var exReadyTimeout = 25 * time.Second

// exWager is a wager of the 1.25.0 client (by its id): the money each player puts in, the battle type and
// how many players it takes.
type exWager struct {
	value   int32
	typ     int
	players int
}

var exWagers = map[int]exWager{
	1: {15, exWagerDuel, 2},
	2: {50, exWagerDuel, 2},
	3: {300, exWagerDuel, 2},
	4: {50, exWager2x2, 4},
	5: {50, exWager3ForAll, 3},
	6: {50, exWager4ForAll, 4},
	7: {50, exWager2x2, 4},
}

type exMember struct {
	userID uint32
	team   int
	names  string // JSON {worm id: name}, as the client sent it
	social byte
	main   Player
	conn   Conn // its PvP connection, once it is in the room
	ready  bool
}

// Stages of a room.
const (
	exInviting = iota // waiting for the friend to join
	exStarting        // taking the wagers in the API
	exLoading         // BattleCreated sent, waiting for ReadyForBattle
)

// exRoom is a battle being set up.
type exRoom struct {
	id                       uint32
	typ                      int
	wagerID                  int
	wager                    int32
	mapID                    uint32
	missions                 []int16
	randomWeapons, randomMap bool
	members                  []*exMember // in player-number order
	stage                    int
	since                    time.Time // start of the stage
}

type exTicket struct {
	m       *exMember
	wagerID int
	mapID   uint32
}

var (
	exMu    sync.Mutex
	exRooms = map[uint32]*exRoom{}
	exQueue []*exTicket // in arrival order
)

func (r *exRoom) member(userID uint32) *exMember {
	for _, m := range r.members {
		if m.userID == userID {
			return m
		}
	}
	return nil
}

func (r *exRoom) memberOf(c Conn) *exMember {
	for _, m := range r.members {
		if m.conn != nil && m.conn == c {
			return m
		}
	}
	return nil
}

// is125 reports whether a main-server player uses the 1.25.0 client.
func is125(p Player) bool {
	v, ok := p.(interface{ V125() bool })
	return ok && v.V125()
}

func message125(cmd uint32, secure bool, body func(w *wire.Writer)) []byte {
	var w wire.Writer
	body(&w)
	return proto.Frame125(cmd, w.B, secure)
}

func exFailure(battleID, refuserID uint32, social byte, reason int) []byte {
	return message125(cmdExCreationFailure, false, func(w *wire.Writer) {
		w.U32(battleID)
		w.U32(refuserID)
		w.U8(social)
		w.I16(reason)
	})
}

func sendLoginError125(c Conn, code int) {
	c.SendMessage(message125(CmdLoginError, false, func(w *wire.Writer) { w.I16(code) }))
}

func readI16s(r *wire.Reader) []int16 {
	n := int(r.U16())
	out := make([]int16, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, int16(r.U16()))
	}
	return out
}

func writeI16s(w *wire.Writer, l []int16) {
	w.U16(uint16(len(l)))
	for _, x := range l {
		w.I16(int(x))
	}
}

// exLogin is the player part that starts CreateBattleRequest, JoinToBattle and RejectBattleOffer.
type exLogin struct {
	id      uint32
	authKey string
	names   string
	social  byte
}

func readExLogin(r *wire.Reader) exLogin {
	var l exLogin
	l.id = r.U32()
	r.UTF() // profileSocialId
	l.social = r.U8()
	l.authKey = r.UTF()
	l.names = r.UTF()
	r.U16() // mainServerId
	return l
}

// exAuth: the player must be logged in to the main server with the same key, from the 1.25.0 client.
func exAuth(c Conn, l exLogin) Player {
	main := Find(l.id)
	if main == nil || l.authKey == "" || main.AuthKey() != l.authKey || !is125(main) {
		logf("[PvP] 1.25.0 login of %d refused: not logged in to the main server with this key", l.id)
		sendLoginError125(c, loginErrIncorrectKey)
		return nil
	}
	return main
}

// handleEx handles the 1.25.0 lobby commands of a PvP connection. Returns false for anything else.
func handleEx(c Conn, cmd uint32, body []byte) bool {
	switch cmd {
	case cmdExCreateBattle:
		if !wire.IsSecureWith(body, wire.SecretKey125) {
			return false // 1.05.0 BattleLogin
		}
		exCreate(c, body)
	case cmdExJoin:
		exJoin(c, body)
	case cmdExReject:
		exReject(c, body)
	case cmdExCancel:
		exCancel(c, exCanceled)
	case cmdExWidenSearch:
		// Any player of the same wager is taken anyway.
	case cmdExReady:
		exReady(c, wire.NewReader(body).U32())
	default:
		return false
	}
	return true
}

// ---- CreateBattleRequest -----------------------------------------------------------------------------------

func exCreate(c Conn, body []byte) {
	r := wire.NewReader(body)
	l := readExLogin(r)
	typ := int(int16(r.U16()))
	mapID := r.U32()
	wagerIDs := readI16s(r)
	missions := readI16s(r)
	ids := make([]uint32, r.U16())
	for i := range ids {
		ids[i] = r.U32()
	}
	teams := make([]int, r.U16())
	for i := range teams {
		teams[i] = int(r.U8())
	}
	randomWeapons := r.Bool()
	randomMap := r.Bool()

	main := exAuth(c, l)
	if main == nil {
		return
	}
	me := &exMember{userID: l.id, names: l.names, social: l.social, main: main, conn: c}
	switch typ {
	case exFriendPvp:
		room := &exRoom{typ: typ, mapID: mapID, missions: missions, randomWeapons: randomWeapons, randomMap: randomMap}
		exInvite(me, room, ids, teams)
	case exWagerDuel, exWager2x2, exWager3ForAll, exWager4ForAll:
		exEnqueue(me, wagerIDs, mapID)
	default:
		logf("[PvP] %d asked for a 1.25.0 battle of type %d: not supported yet", l.id, typ)
		c.SendMessage(exFailure(0, l.id, l.social, exServerError))
	}
}

// exLeaveLocked takes the player out of the queue and out of the rooms it is in (a stale request of its own).
func exLeaveLocked(userID uint32) {
	exQueue = slices.DeleteFunc(exQueue, func(t *exTicket) bool { return t.m.userID == userID })
	for _, room := range exRooms {
		if m := room.member(userID); m != nil {
			exCancelRoomLocked(room, m, exCanceled)
		}
	}
}

// exBusyLocked: in a room, in the queue or in a battle.
func exBusyLocked(userID uint32) bool {
	for _, room := range exRooms {
		if room.member(userID) != nil {
			return true
		}
	}
	for _, t := range exQueue {
		if t.m.userID == userID {
			return true
		}
	}
	return isInBattle(userID)
}

// exCancelRoomLocked ends a room that won't become a battle: the others in it get BattleCreationFailure with
// the reason, and wagers already taken are given back.
func exCancelRoomLocked(room *exRoom, by *exMember, reason int) {
	if exRooms[room.id] != room {
		return
	}
	delete(exRooms, room.id)
	logf("[PvP %d] cancelled by %d (reason %d)", room.id, by.userID, reason)
	msg := exFailure(room.id, by.userID, by.social, reason)
	for _, m := range room.members {
		if m != by && m.conn != nil {
			m.conn.SendMessage(msg)
		}
	}
	if room.stage == exLoading {
		exRefund(room)
	}
	// exStarting: exStart sees the room gone and gives the wagers back itself.
}

func exRefund(room *exRoom) {
	type refund struct {
		token  string
		userID uint32
	}
	var todo []refund
	for _, m := range room.members {
		todo = append(todo, refund{m.main.Token(), m.userID})
	}
	go func() {
		for _, r := range todo {
			tryEndBattle(r.token, r.userID, room.id, 0)
		}
	}()
}

// ---- Friend duel -------------------------------------------------------------------------------------------

func exInvite(me *exMember, room *exRoom, ids []uint32, teams []int) {
	friend := &exMember{names: "{}", team: 1}
	for i, id := range ids {
		team := 0
		if i < len(teams) {
			team = teams[i]
		}
		if id == me.userID {
			me.team = team
		} else if friend.userID == 0 {
			friend.userID, friend.team = id, team
		}
	}
	if friend.userID == 0 || friend.team == me.team {
		me.conn.SendMessage(exFailure(0, friend.userID, 0, exClientError))
		return
	}
	friend.main = Find(friend.userID)
	if friend.main == nil || !is125(friend.main) {
		me.conn.SendMessage(exFailure(0, friend.userID, 0, exOffline))
		return
	}

	exMu.Lock()
	exLeaveLocked(me.userID)
	if exBusyLocked(friend.userID) {
		exMu.Unlock()
		me.conn.SendMessage(exFailure(0, friend.userID, 0, exBusy))
		return
	}
	room.id = newBattleID()
	room.members = []*exMember{me, friend}
	room.stage, room.since = exInviting, time.Now()
	exRooms[room.id] = room
	exMu.Unlock()
	logf("[PvP %d] %d calls %d to a battle", room.id, me.userID, friend.userID)

	profiles, err := exProfiles(room)
	exMu.Lock()
	defer exMu.Unlock()
	if exRooms[room.id] != room {
		return // cancelled meanwhile
	}
	if err != nil {
		logf("[PvP %d] can't load the profiles: %v", room.id, err)
		exCancelRoomLocked(room, friend, exServerError) // tells the caller
		return
	}
	call := message125(cmdExCallToBattle, false, func(w *wire.Writer) {
		w.U32(me.userID)
		w.I16(room.typ)
		w.U32(room.id)
		w.U32(room.mapID)
		writeI16s(w, room.missions)
		exWriteProfiles(w, room, profiles)
		w.Bool(room.randomWeapons)
		w.Bool(room.randomMap)
	})
	me.conn.SendMessage(call)
	friend.main.SendMessage(call)
}

func exProfiles(room *exRoom) ([]proto.UserProfile, error) {
	out := make([]proto.UserProfile, len(room.members))
	for i, m := range room.members {
		p, err := getProfile(m.main.Token())
		if err != nil {
			return nil, err
		}
		out[i] = p
	}
	return out, nil
}

func exWriteProfiles(w *wire.Writer, room *exRoom, profiles []proto.UserProfile) {
	w.U16(uint16(len(room.members)))
	for i, m := range room.members {
		proto.WritePvpProfile125(w, &profiles[i], m.names, m.social, byte(i), byte(m.team))
	}
}

func exJoin(c Conn, body []byte) {
	r := wire.NewReader(body)
	l := readExLogin(r)
	battleID := r.U32()
	main := exAuth(c, l)
	if main == nil {
		return
	}
	exMu.Lock()
	room := exRooms[battleID]
	var m *exMember
	if room != nil {
		m = room.member(l.id)
	}
	if m == nil || room.stage != exInviting || m.conn != nil {
		exMu.Unlock()
		c.SendMessage(exFailure(battleID, l.id, l.social, exCanceled))
		return
	}
	m.conn, m.main, m.names, m.social = c, main, l.names, l.social
	logf("[PvP %d] %d joined", room.id, l.id)
	joined := message125(CmdSystemMessage, false, func(w *wire.Writer) {
		w.I16(sysPlayerJoined)
		w.U8(byte(slices.Index(room.members, m)))
	})
	all := true
	for _, x := range room.members {
		if x.conn == nil {
			all = false
		} else if x != m {
			x.conn.SendMessage(joined)
		}
	}
	if all {
		room.stage, room.since = exStarting, time.Now()
	}
	exMu.Unlock()
	if all {
		exStart(room)
	}
}

func exReject(c Conn, body []byte) {
	r := wire.NewReader(body)
	l := readExLogin(r)
	battleID := r.U32()
	manual := r.Bool()
	if exAuth(c, l) == nil {
		return
	}
	exMu.Lock()
	defer exMu.Unlock()
	room := exRooms[battleID]
	if room == nil || room.stage != exInviting {
		return
	}
	m := room.member(l.id)
	if m == nil {
		return
	}
	m.social = l.social
	reason := exRejected
	if !manual {
		reason = exBusy // the client said no by itself: its player is busy
	}
	exCancelRoomLocked(room, m, reason)
}

// exCancel: the player gave up (CancelBattle) or its connection closed.
func exCancel(c Conn, reason int) {
	exMu.Lock()
	defer exMu.Unlock()
	exQueue = slices.DeleteFunc(exQueue, func(t *exTicket) bool { return t.m.conn == c })
	for _, room := range exRooms {
		if m := room.memberOf(c); m != nil {
			exCancelRoomLocked(room, m, reason)
		}
	}
}

// ---- Wager battles -----------------------------------------------------------------------------------------

func exEnqueue(me *exMember, wagerIDs []int16, mapID uint32) {
	fail := func(reason int) { me.conn.SendMessage(exFailure(0, me.userID, me.social, reason)) }
	if len(wagerIDs) == 0 {
		fail(exClientError)
		return
	}
	wagerID := int(wagerIDs[0])
	wg, ok := exWagers[wagerID]
	if !ok {
		fail(exClientError)
		return
	}
	profile, err := getProfile(me.main.Token())
	if err != nil {
		logf("[PvP] can't load profile of %d: %v", me.userID, err)
		fail(exServerError)
		return
	}
	if int64(profile.Money) < int64(wg.value) {
		fail(exNoMoney)
		return
	}

	exMu.Lock()
	exLeaveLocked(me.userID)
	if isInBattle(me.userID) {
		exMu.Unlock()
		fail(exBusy)
		return
	}
	exQueue = append(exQueue, &exTicket{m: me, wagerID: wagerID, mapID: mapID})
	var group []*exTicket
	for _, t := range exQueue {
		if t.wagerID == wagerID {
			group = append(group, t)
			if len(group) == wg.players {
				break
			}
		}
	}
	if len(group) < wg.players {
		exMu.Unlock()
		logf("[PvP] %d waits for a battle of wager %d (%d of %d)", me.userID, wagerID, len(group), wg.players)
		return
	}
	exQueue = slices.DeleteFunc(exQueue, func(t *exTicket) bool { return slices.Contains(group, t) })
	// The map is the one of who waited longest; the turn order is random.
	room := &exRoom{id: newBattleID(), typ: wg.typ, wagerID: wagerID, wager: wg.value, mapID: group[0].mapID,
		missions: []int16{}, randomMap: true, stage: exStarting, since: time.Now()}
	for _, t := range group {
		room.members = append(room.members, t.m)
	}
	rand.Shuffle(len(room.members), func(i, j int) { room.members[i], room.members[j] = room.members[j], room.members[i] })
	for i, m := range room.members {
		m.team = i // each for himself
		if wg.typ == exWager2x2 {
			m.team = i % 2 // turns alternate between the teams
		}
	}
	exRooms[room.id] = room
	exMu.Unlock()
	exStart(room)
}

// ---- Start -------------------------------------------------------------------------------------------------

// exStart records the battle in the API for everyone (taking the wagers) and sends BattleCreated.
func exStart(room *exRoom) {
	profiles := make([]proto.UserProfile, len(room.members))
	var reagents []int32 // the first player's, for everyone
	for i, m := range room.members {
		give := reagents
		if room.wager == 0 {
			give = []int32{}
		}
		p, got, err := startBattle(m.main.Token(), room.id, room.wager, give)
		if err != nil {
			logf("[PvP %d] can't start the battle for %d: %v", room.id, m.userID, err)
			for _, x := range room.members[:i] {
				tryEndBattle(x.main.Token(), x.userID, room.id, 0)
			}
			reason := exServerError
			if isNotEnoughMoney(err) {
				reason = exNoMoney
			}
			exMu.Lock()
			if exRooms[room.id] == room {
				delete(exRooms, room.id)
				msg := exFailure(room.id, m.userID, m.social, reason)
				for _, x := range room.members {
					if x.conn != nil {
						x.conn.SendMessage(msg)
					}
				}
			}
			exMu.Unlock()
			return
		}
		if i == 0 {
			reagents = got
		}
		profiles[i] = p
	}

	exMu.Lock()
	defer exMu.Unlock()
	if exRooms[room.id] != room {
		logf("[PvP %d] cancelled before it started", room.id)
		exRefund(room)
		return
	}
	room.stage, room.since = exLoading, time.Now()
	msg := message125(cmdExBattleCreated, true, func(w *wire.Writer) {
		w.I16(room.typ)
		w.U32(room.id)
		w.I16(room.wagerID)
		w.U32(room.mapID)
		writeI16s(w, room.missions)
		exWriteProfiles(w, room, profiles)
		w.I32(int32(rand.IntN(1000)))
		w.U16(uint16(len(reagents)))
		for _, x := range reagents {
			w.U8(byte(x))
		}
	})
	ids := make([]uint32, len(room.members))
	for i, m := range room.members {
		ids[i] = m.userID
		m.conn.SendMessage(msg)
	}
	logf("[PvP %d] created: type %d, wager %d, players %s", room.id, room.typ, room.wager, joinIDs(ids))
}

func exReady(c Conn, battleID uint32) {
	exMu.Lock()
	defer exMu.Unlock()
	room := exRooms[battleID]
	if room == nil || room.stage != exLoading {
		return
	}
	m := room.memberOf(c)
	if m == nil {
		return
	}
	m.ready = true
	for _, x := range room.members {
		if !x.ready {
			return
		}
	}
	exLaunchLocked(room)
}

// exLaunchLocked turns the room into a battle and starts it for the players who are ready (one who isn't
// loses when it doesn't show up).
func exLaunchLocked(room *exRoom) {
	delete(exRooms, room.id)
	players := make([]*player, len(room.members))
	for i, m := range room.members {
		players[i] = &player{userID: m.userID, team: m.team, token: m.main.Token()}
	}
	b := newBattle(room.id, room.wager, players, players)
	b.v125 = true
	addBattle(b)
	for _, m := range room.members {
		if m.ready {
			b.Login(m.conn, m.userID)
		}
	}
}

// exTick expires invitations nobody answered and rooms whose players don't finish loading.
func exTick() {
	exMu.Lock()
	defer exMu.Unlock()
	now := time.Now()
	for _, room := range exRooms {
		switch room.stage {
		case exInviting:
			if now.Sub(room.since) > ChallengeTimeout {
				for _, m := range room.members {
					if m.conn == nil {
						exCancelRoomLocked(room, m, exTimeout)
						break
					}
				}
			}
		case exLoading:
			if now.Sub(room.since) > exReadyTimeout {
				if slices.ContainsFunc(room.members, func(m *exMember) bool { return m.ready }) {
					exLaunchLocked(room)
				} else {
					exCancelRoomLocked(room, room.members[0], exTimeout)
				}
			}
		}
	}
}
