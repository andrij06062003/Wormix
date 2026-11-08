// Package proto holds the Wormix game messages: how the server reads client commands, the JSON it sends to
// the API for them, how it reads the API's answers, and how it encodes its replies. Sizes follow the C# GetSize
// methods exactly (strings counted in UTF-16 code units, as the original did), because the client sees them in
// headers and nested size prefixes.
package proto

import (
	"github.com/andrij06062003/Wormix/services/core/internal/csx"
	"github.com/andrij06062003/Wormix/services/core/internal/jsonx"
	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

// CtorDefaults says whether a struct the server reads from API JSON starts from its C# parameterless
// constructor (lists empty, ShopResult.Result = 4, ...) or from default(T) (lists null). Newtonsoft.Json
// creates C# structs without running their parameterless constructors, so fields missing from the JSON are
// zero/null. This only matters when the API leaves fields out.
var CtorDefaults = false

func str(s *string, what string) string {
	if s == nil {
		csx.NullRef(what)
	}
	return *s
}

func strLen(s *string, what string) int64 { return int64(wire.UTF16Len(str(s, what))) }

func notNil[T any](v []T, what string) []T {
	if v == nil {
		csx.NullRef(what)
	}
	return v
}

func emptyIfCtor[T any]() []T {
	if CtorDefaults {
		return []T{}
	}
	return nil
}

func emptyStrIfCtor() *string {
	if CtorDefaults {
		s := ""
		return &s
	}
	return nil
}

// ---- WeaponStructure ---------------------------------------------------------------------------------------

// Weapon is WeaponStructure: an item id and count.
type Weapon struct {
	ID    uint32
	Count int32
}

const weaponSize = 8

func (w Weapon) write(wr *wire.Writer) {
	wr.U32(w.ID)
	wr.U32(uint32(w.Count))
}

func decodeWeapon(v *jsonx.Value) Weapon {
	var w Weapon
	v.Struct("WeaponStructure").Members([]string{"Id", "Count"}, func(f int, x *jsonx.Value) {
		switch f {
		case 0:
			w.ID = x.Uint32()
		case 1:
			w.Count = x.Int32()
		}
	})
	return w
}

func decodeWeapons(v *jsonx.Value) []Weapon {
	out := []Weapon{}
	if !v.List("List<WeaponStructure>", func(e *jsonx.Value) { out = append(out, decodeWeapon(e)) }) {
		return nil
	}
	return out
}

// ---- WormStructure -----------------------------------------------------------------------------------------

// Worm is WormStructure: one worm of a team.
type Worm struct {
	OwnerID       uint32
	SocialOwnerID *string
	Armor         uint32
	Attack        uint32
	Level         uint32
	Experience    uint32
	Hat           int16
	// Artifact is the equipped artifact (item id 2001+), 0 = none. Not in the original 1.05.0 protocol:
	// added for the artifacts port, read by the patched client right after SocialOwnerId.
	Artifact int16
}

func (w *Worm) size() int64 {
	return 4 + 4 + 4 + 4 + 4 + 2 + 2 + strLen(w.SocialOwnerID, "WormStructure.SocialOwnerId") + 2
}

func (w *Worm) write(wr *wire.Writer) {
	wr.U32(w.OwnerID)
	wr.U32(w.Armor)
	wr.U32(w.Attack)
	wr.U32(w.Level)
	wr.U32(w.Experience)
	hat, artifact := w.Hat, w.Artifact
	if hat >= craftedHatBase { // crafted hats (1.25.0) are unknown to the 1.05.0 client: no hat
		race, _ := splitHat(hat)
		hat = int16(race)
	}
	if IsCrafted(int(artifact)) {
		artifact = 0
	}
	wr.U16(uint16(hat))
	wr.UTF(str(w.SocialOwnerID, "WormStructure.SocialOwnerId"))
	wr.U16(uint16(artifact))
}

// stuff105 is the stuff the 1.05.0 client knows: without the 1.25.0 crafted equipment.
func stuff105(stuff []uint16) []uint16 {
	out := stuff[:0:0]
	for _, s := range stuff {
		if !IsCrafted(int(s)) {
			out = append(out, s)
		}
	}
	return out
}

func decodeWorm(v *jsonx.Value) Worm {
	var w Worm
	v.Struct("WormStructure").Members([]string{"OwnerId", "SocialOwnerId", "Armor", "Attack", "Level", "Experience", "Hat", "Artifact"},
		func(f int, x *jsonx.Value) {
			switch f {
			case 0:
				w.OwnerID = x.Uint32()
			case 1:
				w.SocialOwnerID = x.StringPtr()
			case 2:
				w.Armor = x.Uint32()
			case 3:
				w.Attack = x.Uint32()
			case 4:
				w.Level = x.Uint32()
			case 5:
				w.Experience = x.Uint32()
			case 6:
				w.Hat = x.Int16()
			case 7:
				w.Artifact = x.Int16()
			}
		})
	return w
}

// ---- UserProfileStructure ----------------------------------------------------------------------------------

// UserProfile is UserProfileStructure. Nil slices/strings are C# nulls.
type UserProfile struct {
	ID               uint32
	SocialID         *string
	Money            uint32
	RealMoney        uint32
	Rating           int32
	WormsGroup       []Worm
	WeaponRecordList []Weapon
	Stuff            []uint16
	ReactionRate     int32
	Recipes          []uint16
}

// NewUserProfile is `new UserProfileStructure()` (the constructor runs).
func NewUserProfile() UserProfile {
	return UserProfile{WormsGroup: []Worm{}, WeaponRecordList: []Weapon{}, Stuff: []uint16{}, Recipes: []uint16{}}
}

// Size is UserProfileStructure.GetSize().
func (p *UserProfile) Size() uint32 {
	n := int64(4 + 4 + 4 + 4 + 2)
	for i := range notNil(p.WormsGroup, "WormsGroup") {
		n += p.WormsGroup[i].size()
	}
	n += 2 + weaponSize*int64(len(notNil(p.WeaponRecordList, "WeaponRecordList")))
	n += 2 + 2*int64(len(stuff105(notNil(p.Stuff, "Stuff"))))
	n += 4
	n += 2 + strLen(p.SocialID, "SocialId")
	n += 2 + 2*int64(len(notNil(p.Recipes, "Recipes")))
	return uint32(n)
}

// Write is UserProfileStructure.Serialize.
func (p *UserProfile) Write(wr *wire.Writer) {
	wr.U32(p.ID)
	wr.U32(p.Money)
	wr.U32(p.RealMoney)
	wr.U32(uint32(p.Rating))
	wr.U16(uint16(len(notNil(p.WormsGroup, "WormsGroup"))))
	for i := range p.WormsGroup {
		p.WormsGroup[i].write(wr)
	}
	wr.U16(uint16(len(notNil(p.WeaponRecordList, "WeaponRecordList"))))
	for _, w := range p.WeaponRecordList {
		w.write(wr)
	}
	stuff := stuff105(notNil(p.Stuff, "Stuff"))
	wr.U16(uint16(len(stuff)))
	for _, s := range stuff {
		wr.U16(s)
	}
	wr.U32(uint32(p.ReactionRate))
	wr.UTF(str(p.SocialID, "SocialId"))
	wr.U16(uint16(len(notNil(p.Recipes, "Recipes"))))
	for _, r := range p.Recipes {
		wr.U16(r)
	}
}

func defaultUserProfile() UserProfile {
	if CtorDefaults {
		return NewUserProfile()
	}
	return UserProfile{}
}

// DecodeUserProfile reads a UserProfileStructure from API JSON.
func DecodeUserProfile(v *jsonx.Value) UserProfile {
	p := defaultUserProfile()
	v.Struct("UserProfileStructure").Members(
		[]string{"Id", "SocialId", "Money", "RealMoney", "Rating", "WormsGroup", "WeaponRecordList", "Stuff", "ReactionRate", "Recipes"},
		func(f int, x *jsonx.Value) {
			switch f {
			case 0:
				p.ID = x.Uint32()
			case 1:
				p.SocialID = x.StringPtr()
			case 2:
				p.Money = x.Uint32()
			case 3:
				p.RealMoney = x.Uint32()
			case 4:
				p.Rating = x.Int32()
			case 5:
				p.WormsGroup = []Worm{}
				if !x.List("List<WormStructure>", func(e *jsonx.Value) { p.WormsGroup = append(p.WormsGroup, decodeWorm(e)) }) {
					p.WormsGroup = nil
				}
			case 6:
				p.WeaponRecordList = decodeWeapons(x)
			case 7:
				p.Stuff = decodeU16s(x)
			case 8:
				p.ReactionRate = x.Int32()
			case 9:
				p.Recipes = decodeU16s(x)
			}
		})
	return p
}

func decodeUserProfiles(v *jsonx.Value) []UserProfile {
	out := []UserProfile{}
	if !v.List("List<UserProfileStructure>", func(e *jsonx.Value) { out = append(out, DecodeUserProfile(e)) }) {
		return nil
	}
	return out
}

func decodeU16s(v *jsonx.Value) []uint16 {
	out := []uint16{}
	if !v.List("List<UInt16>", func(e *jsonx.Value) { out = append(out, e.Uint16()) }) {
		return nil
	}
	return out
}

func decodeI16s(v *jsonx.Value) []int16 {
	out := []int16{}
	if !v.List("List<Int16>", func(e *jsonx.Value) { out = append(out, e.Int16()) }) {
		return nil
	}
	return out
}

func decodeU32s(v *jsonx.Value) []uint32 {
	out := []uint32{}
	if !v.List("List<UInt32>", func(e *jsonx.Value) { out = append(out, e.Uint32()) }) {
		return nil
	}
	return out
}

func decodeBytes(v *jsonx.Value) []byte {
	out := []byte{}
	if !v.List("List<Byte>", func(e *jsonx.Value) { out = append(out, e.Byte()) }) {
		return nil
	}
	return out
}

// DecodeInts reads a List<int> (nil for JSON null).
func DecodeInts(v *jsonx.Value) []int32 {
	out := []int32{}
	if !v.List("List<Int32>", func(e *jsonx.Value) { out = append(out, e.Int32()) }) {
		return nil
	}
	return out
}

// ---- DailyBonusStructure, BonusDaysStructure ---------------------------------------------------------------

// DailyBonus is DailyBonusStructure.
type DailyBonus struct {
	LoginSequence   int32
	DailyBonusType  int32
	DailyBonusCount int32
}

const dailyBonusSize = 12

func (d DailyBonus) write(wr *wire.Writer) {
	wr.U32(uint32(d.LoginSequence))
	wr.U32(uint32(d.DailyBonusType))
	wr.U32(uint32(d.DailyBonusCount))
}

func decodeDailyBonus(v *jsonx.Value) DailyBonus {
	var d DailyBonus
	v.Struct("DailyBonusStructure").Members([]string{"LoginSequence", "DailyBonusType", "DailyBonusCount"}, func(f int, x *jsonx.Value) {
		switch f {
		case 0:
			d.LoginSequence = x.Int32()
		case 1:
			d.DailyBonusType = x.Int32()
		case 2:
			d.DailyBonusCount = x.Int32()
		}
	})
	return d
}

// BonusDays is BonusDaysStructure.
type BonusDays struct {
	Money        int32
	RealMoney    int32
	BattlesCount int32
	BonusMessage *string
}

func (b *BonusDays) size() int64 {
	return 4 + 4 + 4 + 2 + strLen(b.BonusMessage, "BonusMessage")
}

func (b *BonusDays) write(wr *wire.Writer) {
	wr.U32(uint32(b.Money))
	wr.U32(uint32(b.RealMoney))
	wr.U32(uint32(b.BattlesCount))
	wr.UTF(str(b.BonusMessage, "BonusMessage"))
}

func decodeBonusDays(v *jsonx.Value) BonusDays {
	b := BonusDays{BonusMessage: emptyStrIfCtor()}
	v.Struct("BonusDaysStructure").Members([]string{"Money", "RealMoney", "BattlesCount", "BonusMessage"}, func(f int, x *jsonx.Value) {
		switch f {
		case 0:
			b.Money = x.Int32()
		case 1:
			b.RealMoney = x.Int32()
		case 2:
			b.BattlesCount = x.Int32()
		case 3:
			b.BonusMessage = x.StringPtr()
		}
	})
	return b
}

// ---- ProfileDoubleKeyStructure, PumpReactionRateStructure -------------------------------------------------

// ProfileDoubleKey is ProfileDoubleKeyStructure.
type ProfileDoubleKey struct {
	LongID   uint32
	StringID *string
}

func (p *ProfileDoubleKey) size() int64 { return 4 + 2 + strLen(p.StringID, "StringId") }

func (p *ProfileDoubleKey) write(wr *wire.Writer) {
	wr.U32(p.LongID)
	wr.UTF(str(p.StringID, "StringId"))
}

func decodeProfileDoubleKeys(v *jsonx.Value) []ProfileDoubleKey {
	out := []ProfileDoubleKey{}
	ok := v.List("List<ProfileDoubleKeyStructure>", func(e *jsonx.Value) {
		var p ProfileDoubleKey
		e.Struct("ProfileDoubleKeyStructure").Members([]string{"LongId", "StringId"}, func(f int, x *jsonx.Value) {
			switch f {
			case 0:
				p.LongID = x.Uint32()
			case 1:
				p.StringID = x.StringPtr()
			}
		})
		out = append(out, p)
	})
	if !ok {
		return nil
	}
	return out
}

// PumpReactionRate is PumpReactionRateStructure.
type PumpReactionRate struct {
	FriendID uint32
	Result   byte
}

const pumpReactionRateSize = 5
