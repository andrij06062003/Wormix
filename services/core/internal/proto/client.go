package proto

import (
	"github.com/andrij06062003/Wormix/services/core/internal/jsonx"
	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

// Command ids of the main server's client commands.
const (
	CmdLogin                = 1
	CmdNeedMoney            = 2
	CmdBuyShopItems         = 3
	CmdGetArena             = 4
	CmdGetArena125          = 708 // 1.25.0: empty body
	CmdAssembleEquip125     = 99  // 1.25.0: crafted equipment
	CmdToggleTeamMember125  = 106 // 1.25.0: a friend of the team in battles or not
	CmdStartBattle          = 6
	CmdBuyBattle            = 11
	CmdAddToGroup           = 12
	CmdRemoveFromGroup      = 13
	CmdDistributePoints     = 14
	CmdResetParameters      = 15
	CmdPing                 = 16
	CmdReorderGroup         = 22
	CmdSelectStuff          = 25
	CmdChangeRace           = 36
	CmdGetWhoPumpedReaction = 46
	CmdWipeProfile          = 48
	CmdBuyReactionRate      = 49
	CmdSendWipeConfirmCode  = 52
	CmdBuyUnlockMission     = 58
	CmdSearchTheHouse       = 81
	CmdPumpReactionRates    = 82
	CmdPumpReactionRate     = 83
	CmdEndBattle            = 84
	CmdUpgradeWeapon        = 86
	CmdDowngradeWeapon      = 88
)

// Each client message has a Read function (the C# *BinarySerializer.DeserializeCommand) and a JSON method
// giving the body the C# server posted to the API (JsonConvert.SerializeObject: public fields in declaration
// order, original casing).

func ints[T int8 | int16 | int32 | uint8 | uint16 | uint32](v []T) []int64 {
	out := make([]int64, len(v))
	for i, x := range v {
		out[i] = int64(x)
	}
	return out
}

// Login (1).
type Login struct {
	ID         uint32
	ReferrerID uint32
	AuthKey    string
	IDs        []uint32
	SocialCode uint32
}

// ReadLogin reads Login. The friend ids are read little-endian, as the C# server did.
func ReadLogin(r *wire.Reader) Login {
	var l Login
	l.ID = r.U32()
	l.ReferrerID = r.U32()
	l.AuthKey = r.UTF()
	n := int(r.U16())
	l.IDs = []uint32{}
	for i := 0; i < n; i++ {
		l.IDs = append(l.IDs, r.U32LE())
	}
	l.SocialCode = uint32(r.U8())
	return l
}

// JSON is the API request body.
func (l Login) JSON() []byte {
	return jsonx.NewObject().Int("Id", int64(l.ID)).Int("ReferrerId", int64(l.ReferrerID)).Str("AuthKey", l.AuthKey).
		Ints("Ids", ints(l.IDs)).Int("SocialCode", int64(l.SocialCode)).Bytes()
}

// ShopItem is ShopItemStructure.
type ShopItem struct {
	ID        uint32
	Count     int32
	MoneyType int32
}

// BuyShopItems (3).
type BuyShopItems struct{ ShopItems []ShopItem }

// ReadBuyShopItems reads BuyShopItems.
func ReadBuyShopItems(r *wire.Reader) BuyShopItems {
	b := BuyShopItems{ShopItems: []ShopItem{}}
	n := int(r.U16())
	for i := 0; i < n; i++ {
		r.U16() // size
		var it ShopItem
		it.ID = r.U32()
		it.Count = int32(r.U32())
		it.MoneyType = int32(r.U32())
		b.ShopItems = append(b.ShopItems, it)
	}
	return b
}

// JSON is the API request body.
func (b BuyShopItems) JSON() []byte {
	items := make([]*jsonx.ObjectWriter, len(b.ShopItems))
	for i, it := range b.ShopItems {
		items[i] = jsonx.NewObject().Int("Id", int64(it.ID)).Int("Count", int64(it.Count)).Int("MoneyType", int64(it.MoneyType))
	}
	return jsonx.NewObject().Objects("ShopItems", items).Bytes()
}

// GetArena (4).
type GetArena struct{ ReturnUsersProfiles bool }

// ReadGetArena reads GetArena.
func ReadGetArena(r *wire.Reader) GetArena { return GetArena{ReturnUsersProfiles: r.Bool()} }

// JSON is the API request body.
func (g GetArena) JSON() []byte {
	return jsonx.NewObject().Bool("ReturnUsersProfiles", g.ReturnUsersProfiles).Bytes()
}

// IntField is a client message with a single integer field (StartBattle, BuyBattle, ...).
type IntField struct {
	Name  string
	Value int64
}

// JSON is the API request body.
func (f IntField) JSON() []byte { return jsonx.NewObject().Int(f.Name, f.Value).Bytes() }

// ReadStartBattle reads StartBattle (6): MissionId.
func ReadStartBattle(r *wire.Reader) IntField {
	return IntField{"MissionId", int64(int16(r.U16()))}
}

// ReadBuyBattle reads BuyBattle (11): MoneyType.
func ReadBuyBattle(r *wire.Reader) IntField { return IntField{"MoneyType", int64(int32(r.U32()))} }

// ReadRemoveFromGroup reads RemoveFromGroup (13): ProfileId.
func ReadRemoveFromGroup(r *wire.Reader) IntField { return IntField{"ProfileId", int64(r.U32())} }

// ReadResetParameters reads ResetParameters (15): MoneyType.
func ReadResetParameters(r *wire.Reader) IntField {
	return IntField{"MoneyType", int64(int32(r.U32()))}
}

// ReadSelectStuff reads SelectStuff (25): StuffId.
func ReadSelectStuff(r *wire.Reader) IntField { return IntField{"StuffId", int64(int16(r.U16()))} }

// ReadBuyReactionRate reads BuyReactionRate (49): ReactionRateCount.
func ReadBuyReactionRate(r *wire.Reader) IntField {
	return IntField{"ReactionRateCount", int64(r.U8())}
}

// ReadBuyUnlockMission reads BuyUnlockMission (58): MissionId.
func ReadBuyUnlockMission(r *wire.Reader) IntField {
	return IntField{"MissionId", int64(int16(r.U16()))}
}

// ReadPumpReactionRate reads PumpReactionRate (83): FriendId.
func ReadPumpReactionRate(r *wire.Reader) IntField { return IntField{"FriendId", int64(r.U32())} }

// ReadRecipe reads UpgradeWeapon (86) / DowngradeWeapon (88): RecipeId.
func ReadRecipe(r *wire.Reader) IntField { return IntField{"RecipeId", int64(int16(r.U16()))} }

// AddToGroup (12).
type AddToGroup struct {
	ProfileID uint32
	MoneyType int32
}

// ReadAddToGroup reads AddToGroup.
func ReadAddToGroup(r *wire.Reader) AddToGroup {
	return AddToGroup{ProfileID: r.U32(), MoneyType: int32(int16(r.U16()))}
}

// JSON is the API request body.
func (a AddToGroup) JSON() []byte {
	return jsonx.NewObject().Int("ProfileId", int64(a.ProfileID)).Int("MoneyType", int64(a.MoneyType)).Bytes()
}

// DistributePoints (14).
type DistributePoints struct{ Armor, Attack int32 }

// ReadDistributePoints reads DistributePoints.
func ReadDistributePoints(r *wire.Reader) DistributePoints {
	return DistributePoints{Armor: int32(r.U32()), Attack: int32(r.U32())}
}

// JSON is the API request body.
func (d DistributePoints) JSON() []byte {
	return jsonx.NewObject().Int("Armor", int64(d.Armor)).Int("Attack", int64(d.Attack)).Bytes()
}

// ReorderGroup (22).
type ReorderGroup struct{ ReorderedWormGroup []uint32 }

// ReadReorderGroup reads ReorderGroup. The count is a signed short: a negative count reads nothing.
func ReadReorderGroup(r *wire.Reader) ReorderGroup {
	g := ReorderGroup{ReorderedWormGroup: []uint32{}}
	n := int16(r.U16())
	for i := int16(0); i < n; i++ {
		g.ReorderedWormGroup = append(g.ReorderedWormGroup, r.U32())
	}
	return g
}

// JSON is the API request body.
func (g ReorderGroup) JSON() []byte {
	return jsonx.NewObject().Ints("ReorderedWormGroup", ints(g.ReorderedWormGroup)).Bytes()
}

// ChangeRace (36).
type ChangeRace struct{ RaceID, MoneyType byte }

// ReadChangeRace reads ChangeRace.
func ReadChangeRace(r *wire.Reader) ChangeRace { return ChangeRace{RaceID: r.U8(), MoneyType: r.U8()} }

// JSON is the API request body.
func (c ChangeRace) JSON() []byte {
	return jsonx.NewObject().Int("RaceId", int64(c.RaceID)).Int("MoneyType", int64(c.MoneyType)).Bytes()
}

// GetWhoPumpedReaction (46).
type GetWhoPumpedReaction struct{ TodayOnly bool }

// ReadGetWhoPumpedReaction reads GetWhoPumpedReaction.
func ReadGetWhoPumpedReaction(r *wire.Reader) GetWhoPumpedReaction {
	return GetWhoPumpedReaction{TodayOnly: r.Bool()}
}

// JSON is the API request body.
func (g GetWhoPumpedReaction) JSON() []byte {
	return jsonx.NewObject().Bool("TodayOnly", g.TodayOnly).Bytes()
}

// WipeProfile (48).
type WipeProfile struct{ ConfirmCode string }

// ReadWipeProfile reads WipeProfile.
func ReadWipeProfile(r *wire.Reader) WipeProfile { return WipeProfile{ConfirmCode: r.UTF()} }

// JSON is the API request body.
func (w WipeProfile) JSON() []byte {
	return jsonx.NewObject().Str("ConfirmCode", w.ConfirmCode).Bytes()
}

// ReadSendWipeConfirmCode reads SendWipeConfirmCode (52): level, experience, rating (unused).
func ReadSendWipeConfirmCode(r *wire.Reader) {
	r.U32()
	r.U32()
	r.U32()
}

// ReadNeedMoney reads NeedMoney (2): value, money type (unused).
func ReadNeedMoney(r *wire.Reader) {
	r.U32()
	r.U32()
}

// SearchTheHouse (81).
type SearchTheHouse struct {
	SessionKey string
	FriendID   uint32
	KeyNum     byte
}

// ReadSearchTheHouse reads SearchTheHouse.
func ReadSearchTheHouse(r *wire.Reader) SearchTheHouse {
	var s SearchTheHouse
	s.SessionKey = r.UTF()
	s.FriendID = r.U32()
	s.KeyNum = r.U8()
	return s
}

// JSON is the API request body.
func (s SearchTheHouse) JSON() []byte {
	return jsonx.NewObject().Str("SessionKey", s.SessionKey).Int("FriendId", int64(s.FriendID)).Int("KeyNum", int64(s.KeyNum)).Bytes()
}

// PumpReactionRates (82).
type PumpReactionRates struct{ FriendsIDs []uint32 }

// ReadPumpReactionRates reads PumpReactionRates (ids, then an MD5 that isn't checked).
func ReadPumpReactionRates(r *wire.Reader) PumpReactionRates {
	p := PumpReactionRates{FriendsIDs: []uint32{}}
	n := int(r.U16())
	for i := 0; i < n; i++ {
		p.FriendsIDs = append(p.FriendsIDs, r.U32())
	}
	r.BytesLenient(16)
	return p
}

// JSON is the API request body.
func (p PumpReactionRates) JSON() []byte {
	return jsonx.NewObject().Ints("FriendsIds", ints(p.FriendsIDs)).Bytes()
}

// EndBattle (84).
type EndBattle struct {
	Result            int32
	Type              int32
	ExpBonus          int32
	BattleID          int32
	MissionID         int32
	Items             []Weapon
	Signature         []byte
	BanType           int16
	BanNote           string
	CollectedReagents []int32
}

// ReadEndBattle reads EndBattle. Each of the first four values is preceded by a string the server skips.
func ReadEndBattle(r *wire.Reader) EndBattle {
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
	n := int16(r.U16())
	e.Items = []Weapon{}
	for i := 0; i < int(n); i++ {
		r.U16LE() // size
		var w Weapon
		w.ID = r.U32()
		w.Count = int32(r.U32())
		e.Items = append(e.Items, w)
	}
	sl := int16(r.U16())
	e.Signature = []byte{}
	for i := 0; i < int(sl); i++ {
		e.Signature = append(e.Signature, r.U8())
	}
	e.BanType = int16(r.U16())
	e.BanNote = r.UTF()
	rs := int16(r.U16())
	e.CollectedReagents = []int32{}
	for i := 0; i < int(rs); i++ {
		e.CollectedReagents = append(e.CollectedReagents, int32(r.U8()))
	}
	r.BytesLenient(16) // hash
	return e
}

// JSON is the API request body.
func (e EndBattle) JSON() []byte {
	items := make([]*jsonx.ObjectWriter, len(e.Items))
	for i, it := range e.Items {
		items[i] = jsonx.NewObject().Int("Id", int64(it.ID)).Int("Count", int64(it.Count))
	}
	return jsonx.NewObject().Int("Result", int64(e.Result)).Int("Type", int64(e.Type)).Int("ExpBonus", int64(e.ExpBonus)).
		Int("BattleId", int64(e.BattleID)).Int("MissionId", int64(e.MissionID)).Objects("Items", items).
		Ints("Signature", ints(e.Signature)).Int("BanType", int64(e.BanType)).Str("BanNote", e.BanNote).
		Ints("CollectedReagents", ints(e.CollectedReagents)).Bytes()
}
