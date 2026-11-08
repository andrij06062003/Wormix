package proto

import (
	"math"

	"github.com/andrij06062003/Wormix/services/core/internal/jsonx"
	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

// Command ids of the main server's replies.
const (
	CmdEnterAccount                = 10001
	CmdLoginError                  = 10002
	CmdShopResult                  = 10003
	CmdArenaResult                 = 10004
	CmdStartBattleResult           = 10006
	CmdArenaLocked                 = 10007
	CmdNeedMoneyResult             = 10008
	CmdBuyBattleResult             = 10012
	CmdAddToGroupResult            = 10013
	CmdRemoveFromGroupResult       = 10014
	CmdDistributePointsResult      = 10015
	CmdResetParametersResult       = 10016
	CmdPong                        = 10017
	CmdSearchTheHouseResult        = 10021
	CmdSelectStuffResult           = 10022
	CmdChangeRaceResult            = 10028
	CmdPumpReactionRateResult      = 10032
	CmdPumpReactionRatesResult     = 10033
	CmdWhoPumpedReaction           = 10034
	CmdWipeProfileResult           = 10036
	CmdBuyReactionRateResult       = 10037
	CmdSendWipeConfirmCodeResponse = 10040
	CmdBuyUnlockMissionResult      = 10042
	CmdUpgradeWeaponResult         = 10086
	CmdDowngradeWeaponResult       = 10088
	CmdReorderGroupResult          = 1022 // sic: the C# serializer uses 1022, not 10022
)

// ---- EnterAccount ------------------------------------------------------------------------------------------

// EnterAccount is the successful login reply.
type EnterAccount struct {
	UserProfileStructure  UserProfile
	UserProfileStructures []UserProfile
	DailyBonusStructure   DailyBonus
	OnlineFriends         int16
	Friends               int16
	SessionKey            *string
	IsBonusDay            bool
	BonusDaysStructure    BonusDays
	AvailableSearchKeys   byte
	Reagents              []uint32
	IsSecure              bool
}

// DecodeEnterAccount reads EnterAccount from API JSON.
func DecodeEnterAccount(v *jsonx.Value) EnterAccount {
	e := EnterAccount{
		UserProfileStructure:  defaultUserProfile(),
		UserProfileStructures: emptyIfCtor[UserProfile](),
		SessionKey:            emptyStrIfCtor(),
		BonusDaysStructure:    BonusDays{BonusMessage: emptyStrIfCtor()},
		Reagents:              emptyIfCtor[uint32](),
	}
	v.Struct("EnterAccount").Members([]string{"UserProfileStructure", "UserProfileStructures", "DailyBonusStructure",
		"OnlineFriends", "Friends", "SessionKey", "IsBonusDay", "BonusDaysStructure", "AvailableSearchKeys", "Reagents", "IsSecure"},
		func(f int, x *jsonx.Value) {
			switch f {
			case 0:
				e.UserProfileStructure = DecodeUserProfile(x)
			case 1:
				e.UserProfileStructures = decodeUserProfiles(x)
			case 2:
				e.DailyBonusStructure = decodeDailyBonus(x)
			case 3:
				e.OnlineFriends = x.Int16()
			case 4:
				e.Friends = x.Int16()
			case 5:
				e.SessionKey = x.StringPtr()
			case 6:
				e.IsBonusDay = x.Bool()
			case 7:
				e.BonusDaysStructure = decodeBonusDays(x)
			case 8:
				e.AvailableSearchKeys = x.Byte()
			case 9:
				e.Reagents = decodeU32s(x)
			case 10:
				e.IsSecure = x.Bool()
			}
		})
	return e
}

func (e *EnterAccount) size() uint32 {
	n := int64(2) + int64(e.UserProfileStructure.Size())
	n += 2
	for i := range notNil(e.UserProfileStructures, "UserProfileStructures") {
		n += int64(e.UserProfileStructures[i].Size()) + 2
	}
	n += 2 + dailyBonusSize
	n += 2 + 2
	n += 2 + strLen(e.SessionKey, "SessionKey")
	n++
	if e.IsBonusDay {
		n += 2 + e.BonusDaysStructure.size()
	}
	n++
	n += 2 + 4*int64(len(notNil(e.Reagents, "Reagents")))
	return uint32(n)
}

func (e *EnterAccount) write(w *wire.Writer) {
	w.U16(uint16(e.UserProfileStructure.Size()))
	e.UserProfileStructure.Write(w)
	w.U16(uint16(len(e.UserProfileStructures)))
	for i := range e.UserProfileStructures {
		w.U16(uint16(e.UserProfileStructures[i].Size()))
		e.UserProfileStructures[i].Write(w)
	}
	w.U16(dailyBonusSize)
	e.DailyBonusStructure.write(w)
	w.U16(uint16(e.OnlineFriends))
	w.U16(uint16(e.Friends))
	w.UTF(str(e.SessionKey, "SessionKey"))
	w.Bool(e.IsBonusDay)
	if e.IsBonusDay {
		w.U16(uint16(e.BonusDaysStructure.size()))
		e.BonusDaysStructure.write(w)
	}
	w.U8(e.AvailableSearchKeys)
	w.U16(uint16(len(e.Reagents)))
	for _, r := range e.Reagents {
		w.U32(r)
	}
}

// Emit writes the EnterAccount message (fixed buffer + MD5).
func (e *EnterAccount) Emit(out Out) { emitFixed(out, CmdEnterAccount, e.size(), true, e.write) }

// PutOnlineFirst moves the players that are online to the front of the friends list and sets OnlineFriends to
// their number. The client lights the green "online" lamp on the first OnlineFriends friends of the list (in
// this order), so this is how it learns who is online. It is a snapshot taken at login, as in the original game.
func (e *EnterAccount) PutOnlineFirst(isOnline func(userID uint32) bool) {
	online := make([]UserProfile, 0, len(e.UserProfileStructures))
	offline := make([]UserProfile, 0, len(e.UserProfileStructures))
	for _, p := range e.UserProfileStructures {
		if p.ID != 0 && isOnline(p.ID) {
			online = append(online, p)
		} else {
			offline = append(offline, p)
		}
	}
	if len(online) > math.MaxInt16 {
		online, offline = online[:math.MaxInt16], append(online[math.MaxInt16:], offline...)
	}
	e.OnlineFriends = int16(len(online))
	if e.UserProfileStructures != nil {
		e.UserProfileStructures = append(online, offline...)
	}
}

// ---- LoginError --------------------------------------------------------------------------------------------

// LoginError is the failed login reply.
type LoginError struct{ Result int32 }

// LoginErrorInternal is LoginError.InternalServerError.
const LoginErrorInternal = 3

// DecodeLoginError reads LoginError from API JSON.
func DecodeLoginError(v *jsonx.Value) LoginError {
	var e LoginError
	v.Struct("LoginError").Members([]string{"Result"}, func(_ int, x *jsonx.Value) { e.Result = x.Int32() })
	return e
}

// Emit writes the LoginError message.
func (e LoginError) Emit(out Out) {
	emitStreaming(out, CmdLoginError, 4, func(w *wire.Writer) { w.U32(uint32(e.Result)) })
}

// ---- Results with just a short Result ----------------------------------------------------------------------

// ShortResult is any reply that is just a 2-byte Result (BuyBattleResult, AddToGroupResult, ...).
type ShortResult struct {
	Cmd    uint32
	Result int16
}

// DecodeShortResult reads {Result} from API JSON for the reply with the given command id.
func DecodeShortResult(cmd uint32, v *jsonx.Value) ShortResult {
	r := ShortResult{Cmd: cmd}
	v.Struct("Result").Members([]string{"Result"}, func(_ int, x *jsonx.Value) { r.Result = x.Int16() })
	return r
}

// Emit writes the reply.
func (r ShortResult) Emit(out Out) {
	emitStreaming(out, r.Cmd, 2, func(w *wire.Writer) { w.U16(uint16(r.Result)) })
}

// ---- ShopResult --------------------------------------------------------------------------------------------

// ShopResult is the reply to a purchase.
type ShopResult struct {
	Result  int16
	Weapons []Weapon
	Stuff   []int16
}

// DecodeShopResult reads ShopResult from API JSON.
func DecodeShopResult(v *jsonx.Value) ShopResult {
	r := ShopResult{Weapons: emptyIfCtor[Weapon](), Stuff: emptyIfCtor[int16]()}
	if CtorDefaults {
		r.Result = 4
	}
	v.Struct("ShopResult").Members([]string{"Result", "Weapons", "Stuff"}, func(f int, x *jsonx.Value) {
		switch f {
		case 0:
			r.Result = x.Int16()
		case 1:
			r.Weapons = decodeWeapons(x)
		case 2:
			r.Stuff = decodeI16s(x)
		}
	})
	return r
}

func (r *ShopResult) size() uint32 {
	return uint32(2 + 2 + (weaponSize+2)*int64(len(notNil(r.Weapons, "Weapons"))) + 2 + 2*int64(len(notNil(r.Stuff, "Stuff"))))
}

// Emit writes the reply (fixed buffer + MD5).
func (r *ShopResult) Emit(out Out) {
	emitFixed(out, CmdShopResult, r.size(), true, func(w *wire.Writer) {
		w.U16(uint16(r.Result))
		w.U16(uint16(len(r.Weapons)))
		for _, x := range r.Weapons {
			w.U16(weaponSize)
			x.write(w)
		}
		w.U16(uint16(len(r.Stuff)))
		for _, s := range r.Stuff {
			w.U16(uint16(s))
		}
	})
}

// ---- Arena -------------------------------------------------------------------------------------------------

// ArenaResult is the arena (opponents list) reply.
type ArenaResult struct {
	UserProfileStructures []UserProfile
	BattlesCount          int32
	CurrentMission        int32
	BossAvailable         bool
}

// DecodeArenaResult reads ArenaResult from API JSON.
func DecodeArenaResult(v *jsonx.Value) ArenaResult {
	r := ArenaResult{UserProfileStructures: emptyIfCtor[UserProfile]()}
	v.Struct("ArenaResult").Members([]string{"UserProfileStructures", "BattlesCount", "CurrentMission", "BossAvailable"}, func(f int, x *jsonx.Value) {
		switch f {
		case 0:
			r.UserProfileStructures = decodeUserProfiles(x)
		case 1:
			r.BattlesCount = x.Int32()
		case 2:
			r.CurrentMission = x.Int32()
		case 3:
			r.BossAvailable = x.Bool()
		}
	})
	return r
}

// Emit writes the reply (fixed buffer, no MD5).
func (r *ArenaResult) Emit(out Out) {
	n := int64(2)
	for i := range notNil(r.UserProfileStructures, "UserProfileStructures") {
		n += int64(r.UserProfileStructures[i].Size()) + 2
	}
	n += 4 + 2 + 1
	emitFixed(out, CmdArenaResult, uint32(n), false, func(w *wire.Writer) {
		w.U16(uint16(len(r.UserProfileStructures)))
		for i := range r.UserProfileStructures {
			w.U16(uint16(r.UserProfileStructures[i].Size()))
			r.UserProfileStructures[i].Write(w)
		}
		w.U32(uint32(r.BattlesCount))
		w.U16(uint16(r.CurrentMission))
		w.Bool(r.BossAvailable)
	})
}

// ArenaLocked is the reply when the arena isn't available.
type ArenaLocked struct {
	Delay     uint32
	MissionID int16
	ErrorCode byte
}

// DecodeArenaLocked reads ArenaLocked from API JSON.
func DecodeArenaLocked(v *jsonx.Value) ArenaLocked {
	var r ArenaLocked
	// The API names the mission "CurrentMission" (as in ArenaResult); "MissionId" is the C# field name.
	v.Struct("ArenaLocked").Members([]string{"Delay", "MissionId", "ErrorCode", "CurrentMission"}, func(f int, x *jsonx.Value) {
		switch f {
		case 0:
			r.Delay = x.Uint32()
		case 1, 3:
			r.MissionID = x.Int16()
		case 2:
			r.ErrorCode = x.Byte()
		}
	})
	return r
}

// Emit writes the reply.
func (r ArenaLocked) Emit(out Out) {
	emitStreaming(out, CmdArenaLocked, 7, func(w *wire.Writer) {
		w.U32(r.Delay)
		w.U16(uint16(r.MissionID))
		w.U8(r.ErrorCode)
	})
}

// StartBattleResult is the reply to starting a mission battle.
type StartBattleResult struct {
	Awards           []Weapon
	BattleID         uint32
	ReagentForBattle []byte
}

// DecodeStartBattleResult reads StartBattleResult from API JSON.
func DecodeStartBattleResult(v *jsonx.Value) StartBattleResult {
	r := StartBattleResult{Awards: emptyIfCtor[Weapon](), ReagentForBattle: emptyIfCtor[byte]()}
	v.Struct("StartBattleResult").Members([]string{"Awards", "BattleId", "ReagentForBattle"}, func(f int, x *jsonx.Value) {
		switch f {
		case 0:
			r.Awards = decodeWeapons(x)
		case 1:
			r.BattleID = x.Uint32()
		case 2:
			r.ReagentForBattle = decodeBytes(x)
		}
	})
	return r
}

// Emit writes the reply (fixed buffer + MD5).
func (r *StartBattleResult) Emit(out Out) {
	n := 2 + (weaponSize+2)*int64(len(notNil(r.Awards, "Awards"))) + 4 + 2 + int64(len(notNil(r.ReagentForBattle, "ReagentForBattle")))
	emitFixed(out, CmdStartBattleResult, uint32(n), true, func(w *wire.Writer) {
		w.U16(uint16(len(r.Awards)))
		for _, a := range r.Awards {
			w.U16(weaponSize)
			a.write(w)
		}
		w.U32(r.BattleID)
		w.U16(uint16(len(r.ReagentForBattle)))
		w.Bytes(r.ReagentForBattle)
	})
}

// ---- Static replies ----------------------------------------------------------------------------------------

// EmitPong writes Pong (empty body).
func EmitPong(out Out) { emitStreaming(out, CmdPong, 0, func(*wire.Writer) {}) }

// EmitNeedMoneyResult writes NeedMoneyResult (fixed buffer).
func EmitNeedMoneyResult(out Out, value int32, result int16) {
	emitFixed(out, CmdNeedMoneyResult, 6, false, func(w *wire.Writer) {
		w.U32(uint32(value))
		w.U16(uint16(result))
	})
}

// EmitSendWipeConfirmCodeResponse writes SendWipeConfirmCodeResponse.
func EmitSendWipeConfirmCodeResponse(out Out, result byte) {
	emitStreaming(out, CmdSendWipeConfirmCodeResponse, 1, func(w *wire.Writer) { w.U8(result) })
}

// ---- Account replies ---------------------------------------------------------------------------------------

// BuyReactionRateResult is the reply to buying reaction rate.
type BuyReactionRateResult struct {
	Result            int16
	ReactionRateCount byte
}

// DecodeBuyReactionRateResult reads it from API JSON.
func DecodeBuyReactionRateResult(v *jsonx.Value) BuyReactionRateResult {
	var r BuyReactionRateResult
	v.Struct("BuyReactionRateResult").Members([]string{"Result", "ReactionRateCount"}, func(f int, x *jsonx.Value) {
		switch f {
		case 0:
			r.Result = x.Int16()
		case 1:
			r.ReactionRateCount = x.Byte()
		}
	})
	return r
}

// Emit writes the reply.
func (r BuyReactionRateResult) Emit(out Out) {
	emitStreaming(out, CmdBuyReactionRateResult, 3, func(w *wire.Writer) {
		w.U16(uint16(r.Result))
		w.U8(r.ReactionRateCount)
	})
}

// SelectStuffResult is the reply to selecting a hat/stuff.
type SelectStuffResult struct {
	Result   int16
	StuffID  int16
	IsSecure bool
}

// DecodeSelectStuffResult reads it from API JSON.
func DecodeSelectStuffResult(v *jsonx.Value) SelectStuffResult {
	var r SelectStuffResult
	v.Struct("SelectStuffResult").Members([]string{"Result", "StuffId", "IsSecure"}, func(f int, x *jsonx.Value) {
		switch f {
		case 0:
			r.Result = x.Int16()
		case 1:
			r.StuffID = x.Int16()
		case 2:
			r.IsSecure = x.Bool()
		}
	})
	return r
}

// Emit writes the reply (fixed buffer + MD5).
func (r SelectStuffResult) Emit(out Out) {
	emitFixed(out, CmdSelectStuffResult, 4, true, func(w *wire.Writer) {
		w.U16(uint16(r.Result))
		w.U16(uint16(r.StuffID))
	})
}

// WeaponRecipeResult is UpgradeWeaponResult / DowngradeWeaponResult.
type WeaponRecipeResult struct {
	Cmd      uint32
	RecipeID int16
	Result   int16
}

// DecodeWeaponRecipeResult reads it from API JSON.
func DecodeWeaponRecipeResult(cmd uint32, v *jsonx.Value) WeaponRecipeResult {
	r := WeaponRecipeResult{Cmd: cmd}
	v.Struct("WeaponResult").Members([]string{"RecipeId", "Result"}, func(f int, x *jsonx.Value) {
		switch f {
		case 0:
			r.RecipeID = x.Int16()
		case 1:
			r.Result = x.Int16()
		}
	})
	return r
}

// Emit writes the reply (fixed buffer + MD5).
func (r WeaponRecipeResult) Emit(out Out) {
	emitFixed(out, r.Cmd, 4, true, func(w *wire.Writer) {
		w.U16(uint16(r.RecipeID))
		w.U16(uint16(r.Result))
	})
}

// ---- House -------------------------------------------------------------------------------------------------

// SearchTheHouseResult is the reply to searching a friend's house.
type SearchTheHouseResult struct {
	Result              byte
	Value               int32
	AvailableSearchKeys byte
	FriendID            uint32
	IsSecure            bool
}

// DecodeSearchTheHouseResult reads it from API JSON.
func DecodeSearchTheHouseResult(v *jsonx.Value) SearchTheHouseResult {
	var r SearchTheHouseResult
	v.Struct("SearchTheHouseResult").Members([]string{"Result", "Value", "AvailableSearchKeys", "FriendId", "IsSecure"}, func(f int, x *jsonx.Value) {
		switch f {
		case 0:
			r.Result = x.Byte()
		case 1:
			r.Value = x.Int32()
		case 2:
			r.AvailableSearchKeys = x.Byte()
		case 3:
			r.FriendID = x.Uint32()
		case 4:
			r.IsSecure = x.Bool()
		}
	})
	return r
}

// Emit writes the reply (fixed buffer + MD5).
func (r SearchTheHouseResult) Emit(out Out) {
	emitFixed(out, CmdSearchTheHouseResult, 10, true, func(w *wire.Writer) {
		w.U8(r.Result)
		w.U32(uint32(r.Value))
		w.U8(r.AvailableSearchKeys)
		w.U32(r.FriendID)
	})
}

// PumpReactionTheHouseResult is the reply to pumping one friend's reaction.
type PumpReactionTheHouseResult struct{ Result byte }

// DecodePumpReactionTheHouseResult reads it from API JSON.
func DecodePumpReactionTheHouseResult(v *jsonx.Value) PumpReactionTheHouseResult {
	var r PumpReactionTheHouseResult
	v.Struct("PumpReactionTheHouseResult").Members([]string{"Result"}, func(_ int, x *jsonx.Value) { r.Result = x.Byte() })
	return r
}

// Emit writes the reply.
func (r PumpReactionTheHouseResult) Emit(out Out) {
	emitStreaming(out, CmdPumpReactionRateResult, 1, func(w *wire.Writer) { w.U8(r.Result) })
}

// PumpReactionRatesResult is the reply to pumping several friends' reaction.
type PumpReactionRatesResult struct{ PumpedFriends []PumpReactionRate }

// DecodePumpReactionRatesResult reads it from API JSON.
func DecodePumpReactionRatesResult(v *jsonx.Value) PumpReactionRatesResult {
	r := PumpReactionRatesResult{PumpedFriends: emptyIfCtor[PumpReactionRate]()}
	v.Struct("PumpReactionRatesResult").Members([]string{"PumpedFriends"}, func(_ int, x *jsonx.Value) {
		r.PumpedFriends = []PumpReactionRate{}
		ok := x.List("List<PumpReactionRateStructure>", func(e *jsonx.Value) {
			var p PumpReactionRate
			e.Struct("PumpReactionRateStructure").Members([]string{"FriendId", "Result"}, func(f int, y *jsonx.Value) {
				switch f {
				case 0:
					p.FriendID = y.Uint32()
				case 1:
					p.Result = y.Byte()
				}
			})
			r.PumpedFriends = append(r.PumpedFriends, p)
		})
		if !ok {
			r.PumpedFriends = nil
		}
	})
	return r
}

// Emit writes the reply (fixed buffer). The C# GetSize multiplies by the count twice
// (2 + n * n * 7), so for more than one friend the body is padded with zeros; kept as is.
func (r PumpReactionRatesResult) Emit(out Out) {
	n := int64(len(notNil(r.PumpedFriends, "PumpedFriends")))
	size := uint32(2 + n*(n*(pumpReactionRateSize+2)))
	emitFixed(out, CmdPumpReactionRatesResult, size, false, func(w *wire.Writer) {
		w.U16(uint16(n))
		for _, p := range r.PumpedFriends {
			w.U16(pumpReactionRateSize)
			w.U32(p.FriendID)
			w.U8(p.Result)
		}
	})
}

// ---- Info --------------------------------------------------------------------------------------------------

// WhoPumpedReactionResult lists who pumped the player's reaction.
type WhoPumpedReactionResult struct {
	TodayPumped      []ProfileDoubleKey
	YesterdayPumped  []ProfileDoubleKey
	TwoDaysAgoPumped []ProfileDoubleKey
}

// DecodeWhoPumpedReactionResult reads it from API JSON.
func DecodeWhoPumpedReactionResult(v *jsonx.Value) WhoPumpedReactionResult {
	r := WhoPumpedReactionResult{
		TodayPumped:      emptyIfCtor[ProfileDoubleKey](),
		YesterdayPumped:  emptyIfCtor[ProfileDoubleKey](),
		TwoDaysAgoPumped: emptyIfCtor[ProfileDoubleKey](),
	}
	v.Struct("WhoPumpedReactionResult").Members([]string{"TodayPumped", "YesterdayPumped", "TwoDaysAgoPumped"}, func(f int, x *jsonx.Value) {
		switch f {
		case 0:
			r.TodayPumped = decodeProfileDoubleKeys(x)
		case 1:
			r.YesterdayPumped = decodeProfileDoubleKeys(x)
		case 2:
			r.TwoDaysAgoPumped = decodeProfileDoubleKeys(x)
		}
	})
	return r
}

func pdkListSize(l []ProfileDoubleKey, what string) int64 {
	n := int64(2)
	for i := range notNil(l, what) {
		n += l[i].size() + 2
	}
	return n
}

// Emit writes the reply (header with the C# size, then the body as written).
func (r WhoPumpedReactionResult) Emit(out Out) {
	size := pdkListSize(r.TodayPumped, "TodayPumped") + pdkListSize(r.YesterdayPumped, "YesterdayPumped") +
		pdkListSize(r.TwoDaysAgoPumped, "TwoDaysAgoPumped")
	emitStreaming(out, CmdWhoPumpedReaction, uint32(size), func(w *wire.Writer) {
		for _, l := range [][]ProfileDoubleKey{r.TodayPumped, r.YesterdayPumped, r.TwoDaysAgoPumped} {
			w.U16(uint16(len(l)))
			for i := range l {
				w.U16(uint16(l[i].size()))
				l[i].write(w)
			}
		}
	})
}
