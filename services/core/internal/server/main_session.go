package server

import (
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/andrij06062003/Wormix/services/core/internal/api"
	"github.com/andrij06062003/Wormix/services/core/internal/csx"
	"github.com/andrij06062003/Wormix/services/core/internal/jsonx"
	"github.com/andrij06062003/Wormix/services/core/internal/proto"
	"github.com/andrij06062003/Wormix/services/core/internal/pvp"
	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

// MaxMessageLength is the largest message a client may send (the biggest real ones, PvP move batches, are a
// few KB).
const MaxMessageLength = 1024 * 1024

// LoginErrorDelay is how long a failed login waits before the connection is closed.
var LoginErrorDelay = 5 * time.Second

var sleep1s = func() { time.Sleep(time.Second) }

// readLoop reads messages until the connection ends or a command fails, which closes the connection (like
// the C# ProcessMessage). dispatch runs each command; it may panic with a csx exception.
func readLoop(c *conn, kind string, logCommands bool, dispatch func(cmd uint32, body []byte)) {
	for {
		var h [wire.HeaderSize]byte
		n, err := io.ReadFull(c.c, h[:])
		if err != nil {
			if n > 0 && !errors.Is(err, net.ErrClosed) {
				log.Printf("[%s %s] read error: %v; closing", kind, c.short(), err)
			}
			return
		}
		cmd, length, err := wire.ParseHeader(h[:])
		if err != nil {
			log.Printf("[%s %s] %v; closing", kind, c.short(), err)
			return
		}
		if length > MaxMessageLength {
			log.Printf("[%s %s] message too long (%d bytes); closing", kind, c.short(), length)
			return
		}
		body := make([]byte, length)
		if _, err := io.ReadFull(c.c, body); err != nil {
			if !errors.Is(err, net.ErrClosed) {
				log.Printf("[%s %s] read error: %v; closing", kind, c.short(), err)
			}
			return
		}
		if logCommands {
			log.Printf("[%s %s] cmd %d, %d bytes", kind, c.short(), cmd, length)
		}
		if exc := csx.Catch(func() { dispatch(cmd, body) }); exc != nil {
			log.Printf("[%s %s] cmd %d failed: %s; closing", kind, c.short(), cmd, exc.Msg)
			return
		}
	}
}

// mainSession is a connection to the main server.
type mainSession struct {
	*conn
	userID        atomic.Uint32
	mu            sync.Mutex
	authKey       string
	token         *string // the API session key ("{userId}.{key}"), nil before login
	wipeRequested bool
	// v125: the player uses the 1.25.0 client (its Login was signed with the 1.25.0 salt).
	v125 atomic.Bool
}

func newMainSession(c *conn) *mainSession { return &mainSession{conn: c} }

// UserID implements pvp.Player.
func (s *mainSession) UserID() uint32 { return s.userID.Load() }

// AuthKey implements pvp.Player.
func (s *mainSession) AuthKey() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.authKey
}

// Token implements pvp.Player.
func (s *mainSession) Token() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token == nil {
		return ""
	}
	return *s.token
}

func (s *mainSession) tokenPtr() *string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.token
}

func (s *mainSession) run() {
	log.Printf("[main %s] %s connected", s.short(), s.c.RemoteAddr())
	readLoop(s.conn, "main", true, s.dispatch)
}

func (s *mainSession) stop() {
	if s.stopped.Swap(true) {
		return
	}
	s.c.Close()
	Cache.Clear(s.sid)
	pvp.OnMainDisconnected(s)
	pvp.Unregister(s)
	if id := s.UserID(); id != 0 {
		log.Printf("[main %s] user %d disconnected", s.short(), id)
	} else {
		log.Printf("[main %s] disconnected", s.short())
	}
}

func (s *mainSession) out(b []byte) {
	if s.v125.Load() {
		b = proto.Resign125(b)
	}
	s.write(b)
}

func (s *mainSession) post(route string, body []byte) *jsonx.Value {
	return api.Post(route, body, s.sid, s.tokenPtr())
}

// refuseBattle answers a refused StartBattle with ArenaLocked: the arena's delay when the battles are used up,
// 0 otherwise. The client sets its battle timer from it and leaves the battle loading screen.
func (s *mainSession) refuseBattle() {
	locked := proto.ArenaLocked{ErrorCode: 1}
	o := rootObject(s.post("game/get_arena", proto.GetArena{}.JSON()))
	if o != nil {
		if t := o.Get("type"); t != nil && t.ToString() == "ArenaLocked" {
			locked = proto.DecodeArenaLocked(dataBangObj(o))
		}
	}
	locked.Emit(s.out)
}

// rootObject is the answer's ToObject<JObject>(): nil for a JSON null, an error for a non-object.
func rootObject(v *jsonx.Value) *jsonx.Obj {
	switch v.Kind {
	case jsonx.Null:
		return nil
	case jsonx.Object:
		return v.Obj
	}
	csx.Throw(csx.Generic, "API answer is not a JSON object")
	return nil
}

// dataOpt is `answer.ToObject<JObject>()?["data"]`: nil (no reply) if missing.
func dataOpt(v *jsonx.Value) *jsonx.Value {
	o := rootObject(v)
	if o == nil {
		return nil
	}
	return o.Get("data")
}

// dataBang is `answer.ToObject<JObject>()!["data"]!`: missing is an error.
func dataBang(v *jsonx.Value) *jsonx.Value {
	o := rootObject(v)
	if o == nil {
		csx.NullRef("API answer")
	}
	d := o.Get("data")
	if d == nil {
		csx.NullRef("data")
	}
	return d
}

// shortReply posts a command whose answer is {data: {Result}} (the `?.` controllers: no data, no reply).
func (s *mainSession) shortReply(route string, body []byte, replyCmd uint32) {
	if d := dataOpt(s.post(route, body)); d != nil {
		proto.DecodeShortResult(replyCmd, d).Emit(s.out)
	}
}

func (s *mainSession) dispatch(cmd uint32, body []byte) {
	r := wire.NewReader(body)
	switch cmd {
	case proto.CmdLogin:
		if proto.IsLogin125(body) {
			s.v125.Store(true)
			s.login(proto.ReadLogin125(r))
			break
		}
		s.login(proto.ReadLogin(r))
	case proto.CmdNeedMoney:
		proto.ReadNeedMoney(r)
		proto.EmitNeedMoneyResult(s.out, 0, 3)
	case proto.CmdBuyShopItems:
		req := proto.ReadBuyShopItems(r)
		if len(req.ShopItems) != 0 {
			res := proto.DecodeShopResult(dataBang(s.post("account/buy/items", req.JSON())))
			res.Emit(s.out)
			log.Printf("[main %s] bought, %d items added", s.short(), len(res.Weapons))
		}
	case proto.CmdGetArena:
		req := proto.ReadGetArena(r)
		o := rootObject(s.post("game/get_arena", req.JSON()))
		if o == nil {
			csx.NullRef("API answer")
		}
		typ := ""
		if t := o.Get("type"); t != nil {
			typ = t.ToString()
		}
		switch typ {
		case "ArenaResult":
			res := proto.DecodeArenaResult(dataBangObj(o))
			res.Emit(s.out)
		case "ArenaLocked":
			proto.DecodeArenaLocked(dataBangObj(o)).Emit(s.out)
		default:
			proto.ArenaLocked{Delay: 2147483647, ErrorCode: 1}.Emit(s.out)
		}
	case proto.CmdStartBattle:
		req := proto.ReadStartBattle(r)
		v, refused := api.PostRefusable("game/start_battle", req.JSON(), s.sid, s.tokenPtr())
		if refused {
			// No battles left (the client counts them itself and can be wrong after a reload), or a mission
			// it may not start: answer with the arena state instead of dropping the connection.
			s.refuseBattle()
			break
		}
		res := proto.DecodeStartBattleResult(dataBang(v))
		res.Emit(s.out)
	case proto.CmdBuyBattle:
		s.shortReply("account/buy/battles", proto.ReadBuyBattle(r).JSON(), proto.CmdBuyBattleResult)
	case proto.CmdAddToGroup:
		s.shortReply("team/add", proto.ReadAddToGroup(r).JSON(), proto.CmdAddToGroupResult)
	case proto.CmdRemoveFromGroup:
		s.shortReply("team/remove", proto.ReadRemoveFromGroup(r).JSON(), proto.CmdRemoveFromGroupResult)
	case proto.CmdDistributePoints:
		s.shortReply("account/distribute_points", proto.ReadDistributePoints(r).JSON(), proto.CmdDistributePointsResult)
	case proto.CmdResetParameters:
		s.shortReply("account/reset/parameters", proto.ReadResetParameters(r).JSON(), proto.CmdResetParametersResult)
	case proto.CmdPing:
		proto.EmitPong(s.out)
	case proto.CmdReorderGroup:
		s.shortReply("team/reorder", proto.ReadReorderGroup(r).JSON(), proto.CmdReorderGroupResult)
	case proto.CmdSelectStuff:
		req := proto.ReadSelectStuff(r)
		proto.DecodeSelectStuffResult(dataBang(s.post("account/select_stuff", req.JSON()))).Emit(s.out)
	case proto.CmdChangeRace:
		req := proto.ReadChangeRace(r)
		proto.DecodeShortResult(proto.CmdChangeRaceResult, dataBang(s.post("account/buy/race", req.JSON()))).Emit(s.out)
	case proto.CmdGetWhoPumpedReaction:
		req := proto.ReadGetWhoPumpedReaction(r)
		proto.DecodeWhoPumpedReactionResult(dataBang(s.post("info/pumped_reaction", req.JSON()))).Emit(s.out)
	case proto.CmdWipeProfile:
		req := proto.ReadWipeProfile(r)
		if s.wipeRequested {
			s.shortReply("account/reset/account", req.JSON(), proto.CmdWipeProfileResult)
		}
	case proto.CmdBuyReactionRate:
		req := proto.ReadBuyReactionRate(r)
		if d := dataOpt(s.post("account/buy/reaction", req.JSON())); d != nil {
			proto.DecodeBuyReactionRateResult(d).Emit(s.out)
		}
	case proto.CmdSendWipeConfirmCode:
		proto.ReadSendWipeConfirmCode(r)
		proto.EmitSendWipeConfirmCodeResponse(s.out, 0)
		s.wipeRequested = true
	case proto.CmdBuyUnlockMission:
		s.shortReply("account/buy/mission", proto.ReadBuyUnlockMission(r).JSON(), proto.CmdBuyUnlockMissionResult)
	case proto.CmdSearchTheHouse:
		req := proto.ReadSearchTheHouse(r)
		proto.DecodeSearchTheHouseResult(dataBang(s.post("house/search", req.JSON()))).Emit(s.out)
	case proto.CmdPumpReactionRates:
		req := proto.ReadPumpReactionRates(r)
		if d := dataOpt(s.post("house/pump_reactions", req.JSON())); d != nil {
			proto.DecodePumpReactionRatesResult(d).Emit(s.out)
		}
	case proto.CmdPumpReactionRate:
		req := proto.ReadPumpReactionRate(r)
		proto.DecodePumpReactionTheHouseResult(dataBang(s.post("house/pump_reaction", req.JSON()))).Emit(s.out)
	case proto.CmdEndBattle:
		req := proto.ReadEndBattle(r)
		s.post("game/end_battle", req.JSON())
	case proto.CmdUpgradeWeapon:
		req := proto.ReadRecipe(r)
		if d := dataOpt(s.post("craft/upgrade_weapon", req.JSON())); d != nil {
			proto.DecodeWeaponRecipeResult(proto.CmdUpgradeWeaponResult, d).Emit(s.out)
		}
	case proto.CmdDowngradeWeapon:
		req := proto.ReadRecipe(r)
		if d := dataOpt(s.post("craft/downgrade_weapon", req.JSON())); d != nil {
			proto.DecodeWeaponRecipeResult(proto.CmdDowngradeWeaponResult, d).Emit(s.out)
		}
	default:
		if !pvp.HandleLobby(s, cmd, body) {
			log.Printf("[main %s] unknown command %d", s.short(), cmd)
		}
	}
}

func dataBangObj(o *jsonx.Obj) *jsonx.Value {
	d := o.Get("data")
	if d == nil {
		csx.NullRef("data")
	}
	return d
}

func (s *mainSession) login(req proto.Login) {
	if req.ID == 0 {
		csx.Throw(csx.Argument, "Invalid login struct")
	}
	o := rootObject(s.post("login", req.JSON()))
	if o == nil {
		csx.NullRef("API answer")
	}
	t := o.Get("type")
	if t == nil {
		csx.NullRef("type")
	}
	switch t.ToString() {
	case "EnterAccount":
		if old := o.Get("old_session"); old != nil {
			if os := old.ToString(); !isBlank(os) {
				id, err := parseGUID(os)
				if err != nil {
					csx.Throw(csx.Generic, "bad old_session: %v", err)
				}
				if prev := s.srv.find(id); prev != nil {
					log.Printf("[main %s] user %d logged in again: closing %s", s.short(), req.ID, id[:8])
					prev.stop()
				}
			}
		}
		account := proto.DecodeEnterAccount(dataBangObj(o))
		Cache.Init(s.sid)
		s.mu.Lock()
		s.token = account.SessionKey
		s.authKey = req.AuthKey
		s.mu.Unlock()
		s.userID.Store(req.ID)
		pvp.Register(s)
		account.PutOnlineFirst(func(id uint32) bool { return id != req.ID && pvp.Find(id) != nil })
		if s.v125.Load() {
			log.Printf("[main %s] user %d logged in (1.25.0 client)", s.short(), req.ID)
			account.Emit125(s.write, int32(time.Now().Unix()))
			break
		}
		log.Printf("[main %s] user %d logged in", s.short(), req.ID)
		account.Emit(s.out)
	case "LoginError":
		e := proto.DecodeLoginError(dataBangObj(o))
		s.loginFailed(req.ID, &e)
	default:
		s.loginFailed(req.ID, &proto.LoginError{Result: proto.LoginErrorInternal})
	}
}

// loginFailed sends the error, waits and closes the connection.
func (s *mainSession) loginFailed(id uint32, e *proto.LoginError) {
	log.Printf("[main %s] login of %d refused (error %d)", s.short(), id, e.Result)
	e.Emit(s.out)
	time.Sleep(LoginErrorDelay)
	s.Close()
}

func isBlank(s string) bool {
	for _, r := range s {
		switch r {
		case ' ', '\t', '\n', '\v', '\f', '\r', 0x85, 0xA0, 0x1680, 0x2028, 0x2029, 0x202F, 0x205F, 0x3000:
			continue
		}
		if r >= 0x2000 && r <= 0x200A {
			continue
		}
		return false
	}
	return true
}

// ---- PvP server --------------------------------------------------------------------------------------------

type pvpSession struct {
	*conn
	battle atomic.Pointer[pvp.Battle]
}

func newPvpSession(c *conn) *pvpSession { return &pvpSession{conn: c} }

func (s *pvpSession) Battle() *pvp.Battle     { return s.battle.Load() }
func (s *pvpSession) SetBattle(b *pvp.Battle) { s.battle.Store(b) }

func (s *pvpSession) run() {
	log.Printf("[pvp %s] %s connected", s.short(), s.c.RemoteAddr())
	// Moves arrive many times a second: don't log every command.
	readLoop(s.conn, "pvp", false, func(cmd uint32, body []byte) { pvp.HandlePvp(s, cmd, body) })
}

func (s *pvpSession) stop() {
	if s.stopped.Swap(true) {
		return
	}
	s.c.Close()
	log.Printf("[pvp %s] disconnected", s.short())
	pvp.OnPvpDisconnected(s)
}
