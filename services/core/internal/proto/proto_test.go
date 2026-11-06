package proto

import (
	"encoding/hex"
	"reflect"
	"strings"
	"testing"

	"github.com/andrij06062003/Wormix/services/core/internal/csx"
	"github.com/andrij06062003/Wormix/services/core/internal/jsonx"
	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

// frames splits what a serializer wrote into (cmd, body) pairs, checking the headers.
func frames(t *testing.T, data []byte) (cmds []uint32, bodies [][]byte) {
	t.Helper()
	for len(data) > 0 {
		cmd, n, err := wire.ParseHeader(data[:wire.HeaderSize])
		if err != nil {
			t.Fatalf("bad header %x: %v", data[:7], err)
		}
		end := wire.HeaderSize + int(n)
		if end > len(data) {
			end = len(data)
		}
		cmds = append(cmds, cmd)
		bodies = append(bodies, data[wire.HeaderSize:end])
		data = data[end:]
	}
	return
}

func emit(t *testing.T, f func(Out)) (uint32, []byte) {
	t.Helper()
	var out []byte
	f(func(b []byte) { out = append(out, b...) })
	cmds, bodies := frames(t, out)
	if len(cmds) != 1 {
		t.Fatalf("%d messages", len(cmds))
	}
	return cmds[0], bodies[0]
}

func parse(t *testing.T, s string) *jsonx.Value {
	t.Helper()
	v, err := jsonx.Parse([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func profileJSON(id, social string) string {
	return `{"Id":` + id + `,"SocialId":"` + social + `","Money":450,"RealMoney":7,"Rating":0,
	"WormsGroup":[{"OwnerId":` + id + `,"SocialOwnerId":"` + social + `","Armor":3,"Attack":4,"Level":2,"Experience":25,"Hat":5},
	              {"OwnerId":` + id + `,"SocialOwnerId":"` + social + `","Armor":3,"Attack":4,"Level":1,"Experience":25,"Hat":0}],
	"WeaponRecordList":[{"Id":1,"Count":-1},{"Id":7,"Count":3}],"Stuff":[1,2,300],"ReactionRate":12,"Recipes":[4,5]}`
}

// Goldens: bodies the reference (C#) server sent for the same API answers (services/core/test/diff_fake.py).
const goldenEnterAccount = "006f00000001000001c200000007000000000002000000010000000300000004000000020000001900050001310000000000010000000300000004000000010000001900000001310000000200000001ffffffff0000000700000003000300010002012c0000000c0001310002000400050002006f00000002000001c200000007000000000002000000020000000300000004000000020000001900050001320000000000020000000300000004000000010000001900000001320000000200000001ffffffff0000000700000003000300010002012c0000000c000132000200040005006f000000030000000a00000007000000000002000000030000000300000004000000020000001900050001330000000000030000000300000004000000010000001900000001330000000200000001ffffffff0000000700000003000300010002012c0000000c000133000200040005000c000000030000000100000014000000020005312e61626300050003000000010000000000000002e864a7f5b6fbb239a60e625c36752364"

func TestEnterAccountGolden(t *testing.T) {
	p3 := strings.Replace(profileJSON("3", "3"), `"Money":450`, `"Money":10`, 1)
	j := `{"UserProfileStructure":` + profileJSON("1", "1") + `,"UserProfileStructures":[` + profileJSON("2", "2") + `,` + p3 + `],
		"AvailableSearchKeys":5,"Friends":2,"OnlineFriends":0,"IsBonusDay":false,
		"DailyBonusStructure":{"LoginSequence":3,"DailyBonusType":1,"DailyBonusCount":20},"Reagents":[1,0,2],"SessionKey":"1.abc"}`
	e := DecodeEnterAccount(parse(t, j))
	cmd, body := emit(t, e.Emit)
	if cmd != CmdEnterAccount || hex.EncodeToString(body) != goldenEnterAccount {
		t.Fatalf("cmd %d body\n%x\nwant\n%s", cmd, body, goldenEnterAccount)
	}
}

func TestRepliesGolden(t *testing.T) {
	cases := []struct {
		name, json string
		emit       func(*jsonx.Value, Out)
		cmd        uint32
		body       string
	}{
		{"shop", `{"Result":0,"Weapons":[{"Id":5,"Count":2}],"Stuff":[9]}`,
			func(v *jsonx.Value, o Out) { r := DecodeShopResult(v); r.Emit(o) }, CmdShopResult,
			"0000000100080000000500000002000100096319fea52baa2fc1f6abf3c16d3f5a6d"},
		{"start battle", `{"Awards":[{"Id":1,"Count":2}],"BattleId":1077,"ReagentForBattle":[1,2,255]}`,
			func(v *jsonx.Value, o Out) { r := DecodeStartBattleResult(v); r.Emit(o) }, CmdStartBattleResult,
			"0001000800000001000000020000043500030102ff8defb9cac230618c46d2655090cf1226"},
		{"pump reactions: C# size quirk pads with zeros",
			`{"PumpedFriends":[{"FriendId":2,"Result":0},{"FriendId":3,"Result":1}]}`,
			func(v *jsonx.Value, o Out) { DecodePumpReactionRatesResult(v).Emit(o) }, CmdPumpReactionRatesResult,
			"000200050000000200000500000003010000000000000000000000000000"},
		{"arena", `{"UserProfileStructures":[` + profileJSON("2", "2") + `,` + profileJSON("3", "3") + `],"BattlesCount":5,"CurrentMission":70000,"BossAvailable":true}`,
			func(v *jsonx.Value, o Out) { r := DecodeArenaResult(v); r.Emit(o) }, CmdArenaResult,
			"0002006f00000002000001c200000007000000000002000000020000000300000004000000020000001900050001320000000000020000000300000004000000010000001900000001320000000200000001ffffffff0000000700000003000300010002012c0000000c000132000200040005006f00000003000001c200000007000000000002000000030000000300000004000000020000001900050001330000000000030000000300000004000000010000001900000001330000000200000001ffffffff0000000700000003000300010002012c0000000c00013300020004000500000005117001"},
	}
	for _, c := range cases {
		cmd, body := emit(t, func(o Out) { c.emit(parse(t, c.json), o) })
		if cmd != c.cmd || hex.EncodeToString(body) != c.body {
			t.Errorf("%s: cmd %d body %x\nwant %d %s", c.name, cmd, body, c.cmd, c.body)
		}
	}
}

func TestWhoPumpedNonASCIIHeaderQuirk(t *testing.T) {
	// The C# GetSize counts "Пётр" as 4 (UTF-16) while 8 bytes are written: the header announces 36 bytes
	// and 4 more follow. The reference server sent exactly this.
	r := DecodeWhoPumpedReactionResult(parse(t, `{"TodayPumped":[{"LongId":2,"StringId":"Пётр"}],"YesterdayPumped":[],
		"TwoDaysAgoPumped":[{"LongId":3,"StringId":"3"},{"LongId":4,"StringId":"4"}]}`))
	var out []byte
	r.Emit(func(b []byte) { out = append(out, b...) })
	cmd, n, _ := wire.ParseHeader(out[:7])
	if cmd != CmdWhoPumpedReaction || n != 36 || len(out)-7 != 40 {
		t.Fatalf("cmd %d announced %d written %d", cmd, n, len(out)-7)
	}
	if got := hex.EncodeToString(out[7 : 7+36]); got != "0001000a000000020008d09fd191d182d180000000020007000000030001330007000000" {
		t.Fatalf("body %s", got)
	}
}

func TestFixedBufferOverflow(t *testing.T) {
	// Non-ASCII text in a fixed-size (GetSize) buffer overflows it: an exception, nothing written.
	p := DecodeUserProfile(parse(t, profileJSON("9", "Вася")))
	a := ArenaResult{UserProfileStructures: []UserProfile{p}}
	var out []byte
	exc := csx.Catch(func() { a.Emit(func(b []byte) { out = append(out, b...) }) })
	if exc == nil || len(out) != 0 {
		t.Fatalf("exc %v, wrote %d", exc, len(out))
	}
}

func TestNullFieldsThrow(t *testing.T) {
	// Newtonsoft leaves missing struct fields null: serializing them fails like the C# server did.
	e := DecodeEnterAccount(parse(t, `{"UserProfileStructure":{"Id":8,"SocialId":"8"},"SessionKey":"8.x"}`))
	if csx.Catch(func() { e.Emit(func([]byte) {}) }) == nil {
		t.Fatal("null lists serialized")
	}
	s := DecodeShopResult(parse(t, `{"Weapons":[],"Stuff":[]}`))
	if s.Result != 0 {
		t.Fatalf("missing Result = %d, want 0 (default(T), as Newtonsoft)", s.Result)
	}
}

// ---- round trips of every message type --------------------------------------------------------------------

func readProfile(r *wire.Reader) UserProfile {
	p := UserProfile{ID: r.U32(), Money: r.U32(), RealMoney: r.U32(), Rating: int32(r.U32())}
	p.WormsGroup = []Worm{}
	for n := r.U16(); n > 0; n-- {
		w := Worm{OwnerID: r.U32(), Armor: r.U32(), Attack: r.U32(), Level: r.U32(), Experience: r.U32(), Hat: int16(r.U16())}
		s := r.UTF()
		w.SocialOwnerID = &s
		w.Artifact = int16(r.U16())
		p.WormsGroup = append(p.WormsGroup, w)
	}
	p.WeaponRecordList = []Weapon{}
	for n := r.U16(); n > 0; n-- {
		p.WeaponRecordList = append(p.WeaponRecordList, Weapon{r.U32(), int32(r.U32())})
	}
	p.Stuff = []uint16{}
	for n := r.U16(); n > 0; n-- {
		p.Stuff = append(p.Stuff, r.U16())
	}
	p.ReactionRate = int32(r.U32())
	s := r.UTF()
	p.SocialID = &s
	p.Recipes = []uint16{}
	for n := r.U16(); n > 0; n-- {
		p.Recipes = append(p.Recipes, r.U16())
	}
	return p
}

func sp(s string) *string { return &s }

func sampleProfile(id uint32) UserProfile {
	return UserProfile{ID: id, SocialID: sp("s" + string(rune('0'+id))), Money: 100 * id, RealMoney: 3, Rating: -7,
		WormsGroup:       []Worm{{OwnerID: id, SocialOwnerID: sp("w"), Armor: 1, Attack: 2, Level: 3, Experience: 4, Hat: -1}},
		WeaponRecordList: []Weapon{{1, -1}, {2, 5}}, Stuff: []uint16{1, 65535}, ReactionRate: 9, Recipes: []uint16{}}
}

func TestRoundTripEnterAccount(t *testing.T) {
	in := EnterAccount{UserProfileStructure: sampleProfile(1), UserProfileStructures: []UserProfile{sampleProfile(2), sampleProfile(3)},
		DailyBonusStructure: DailyBonus{1, 2, 3}, OnlineFriends: -1, Friends: 2, SessionKey: sp("1.key"), IsBonusDay: true,
		BonusDaysStructure: BonusDays{1, 2, 3, sp("bonus")}, AvailableSearchKeys: 5, Reagents: []uint32{1, 4294967295}}
	cmd, body := emit(t, in.Emit)
	if cmd != CmdEnterAccount || !wire.IsSecure(body) {
		t.Fatal("cmd/md5")
	}
	r := wire.NewReader(body[:len(body)-16])
	var out EnterAccount
	if int(r.U16()) != int(in.UserProfileStructure.Size()) {
		t.Fatal("profile size prefix")
	}
	out.UserProfileStructure = readProfile(r)
	out.UserProfileStructures = []UserProfile{}
	for n := r.U16(); n > 0; n-- {
		r.U16()
		out.UserProfileStructures = append(out.UserProfileStructures, readProfile(r))
	}
	r.U16()
	out.DailyBonusStructure = DailyBonus{int32(r.U32()), int32(r.U32()), int32(r.U32())}
	out.OnlineFriends, out.Friends = int16(r.U16()), int16(r.U16())
	out.SessionKey = sp(r.UTF())
	out.IsBonusDay = r.Bool()
	r.U16()
	out.BonusDaysStructure = BonusDays{int32(r.U32()), int32(r.U32()), int32(r.U32()), sp(r.UTF())}
	out.AvailableSearchKeys = r.U8()
	out.Reagents = []uint32{}
	for n := r.U16(); n > 0; n-- {
		out.Reagents = append(out.Reagents, r.U32())
	}
	if r.Remaining() != 0 || !reflect.DeepEqual(in, out) {
		t.Fatalf("round trip\n%+v\n%+v", in, out)
	}
}

func TestRoundTripReplies(t *testing.T) {
	check := func(name string, wantCmd uint32, f func(Out), secure bool, read func(*wire.Reader) any, want any) {
		t.Helper()
		cmd, body := emit(t, f)
		if cmd != wantCmd {
			t.Errorf("%s: cmd %d", name, cmd)
		}
		if secure {
			if !wire.IsSecure(body) {
				t.Errorf("%s: md5", name)
			}
			body = body[:len(body)-16]
		}
		r := wire.NewReader(body)
		got := read(r)
		if r.Remaining() != 0 || !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %+v (left %d), want %+v", name, got, r.Remaining(), want)
		}
	}
	le := LoginError{Result: 4}
	check("LoginError", CmdLoginError, le.Emit, false, func(r *wire.Reader) any { return LoginError{int32(r.U32())} }, le)
	for _, c := range []uint32{CmdBuyBattleResult, CmdAddToGroupResult, CmdRemoveFromGroupResult, CmdDistributePointsResult,
		CmdResetParametersResult, CmdChangeRaceResult, CmdBuyUnlockMissionResult, CmdWipeProfileResult, CmdReorderGroupResult} {
		sr := ShortResult{Cmd: c, Result: -3}
		check("ShortResult", c, sr.Emit, false, func(r *wire.Reader) any { return ShortResult{c, int16(r.U16())} }, sr)
	}
	shop := ShopResult{Result: 2, Weapons: []Weapon{{1, 2}, {3, -4}}, Stuff: []int16{-1, 7}}
	check("ShopResult", CmdShopResult, shop.Emit, true, func(r *wire.Reader) any {
		s := ShopResult{Result: int16(r.U16()), Weapons: []Weapon{}, Stuff: []int16{}}
		for n := r.U16(); n > 0; n-- {
			if r.U16() != 8 {
				t.Error("weapon size")
			}
			s.Weapons = append(s.Weapons, Weapon{r.U32(), int32(r.U32())})
		}
		for n := r.U16(); n > 0; n-- {
			s.Stuff = append(s.Stuff, int16(r.U16()))
		}
		return s
	}, shop)
	arena := ArenaResult{UserProfileStructures: []UserProfile{sampleProfile(4)}, BattlesCount: 3, CurrentMission: -2, BossAvailable: true}
	check("ArenaResult", CmdArenaResult, arena.Emit, false, func(r *wire.Reader) any {
		a := ArenaResult{UserProfileStructures: []UserProfile{}}
		for n := r.U16(); n > 0; n-- {
			r.U16()
			a.UserProfileStructures = append(a.UserProfileStructures, readProfile(r))
		}
		a.BattlesCount, a.CurrentMission, a.BossAvailable = int32(r.U32()), int32(int16(r.U16())), r.Bool()
		return a
	}, arena)
	locked := ArenaLocked{Delay: 2147483647, MissionID: -1, ErrorCode: 1}
	check("ArenaLocked", CmdArenaLocked, locked.Emit, false, func(r *wire.Reader) any {
		return ArenaLocked{r.U32(), int16(r.U16()), r.U8()}
	}, locked)
	sb := StartBattleResult{Awards: []Weapon{{9, 1}}, BattleID: 1077, ReagentForBattle: []byte{1, 255}}
	check("StartBattleResult", CmdStartBattleResult, sb.Emit, true, func(r *wire.Reader) any {
		s := StartBattleResult{Awards: []Weapon{}, ReagentForBattle: []byte{}}
		for n := r.U16(); n > 0; n-- {
			r.U16()
			s.Awards = append(s.Awards, Weapon{r.U32(), int32(r.U32())})
		}
		s.BattleID = r.U32()
		for n := r.U16(); n > 0; n-- {
			s.ReagentForBattle = append(s.ReagentForBattle, r.U8())
		}
		return s
	}, sb)
	check("Pong", CmdPong, EmitPong, false, func(r *wire.Reader) any { return nil }, nil)
	check("NeedMoneyResult", CmdNeedMoneyResult, func(o Out) { EmitNeedMoneyResult(o, -5, 3) }, false,
		func(r *wire.Reader) any { return [2]int{int(int32(r.U32())), int(int16(r.U16()))} }, [2]int{-5, 3})
	check("SendWipeConfirmCodeResponse", CmdSendWipeConfirmCodeResponse, func(o Out) { EmitSendWipeConfirmCodeResponse(o, 2) }, false,
		func(r *wire.Reader) any { return r.U8() }, byte(2))
	brr := BuyReactionRateResult{Result: 1, ReactionRateCount: 200}
	check("BuyReactionRateResult", CmdBuyReactionRateResult, brr.Emit, false, func(r *wire.Reader) any {
		return BuyReactionRateResult{int16(r.U16()), r.U8()}
	}, brr)
	ss := SelectStuffResult{Result: 0, StuffID: 300}
	check("SelectStuffResult", CmdSelectStuffResult, ss.Emit, true, func(r *wire.Reader) any {
		return SelectStuffResult{Result: int16(r.U16()), StuffID: int16(r.U16())}
	}, ss)
	for _, c := range []uint32{CmdUpgradeWeaponResult, CmdDowngradeWeaponResult} {
		wr := WeaponRecipeResult{Cmd: c, RecipeID: 4, Result: -1}
		check("WeaponRecipeResult", c, wr.Emit, true, func(r *wire.Reader) any {
			return WeaponRecipeResult{c, int16(r.U16()), int16(r.U16())}
		}, wr)
	}
	sh := SearchTheHouseResult{Result: 2, Value: -30, AvailableSearchKeys: 4, FriendID: 99}
	check("SearchTheHouseResult", CmdSearchTheHouseResult, sh.Emit, true, func(r *wire.Reader) any {
		return SearchTheHouseResult{Result: r.U8(), Value: int32(r.U32()), AvailableSearchKeys: r.U8(), FriendID: r.U32()}
	}, sh)
	pr := PumpReactionTheHouseResult{Result: 3}
	check("PumpReactionTheHouseResult", CmdPumpReactionRateResult, pr.Emit, false, func(r *wire.Reader) any {
		return PumpReactionTheHouseResult{r.U8()}
	}, pr)
	prs := PumpReactionRatesResult{PumpedFriends: []PumpReactionRate{{FriendID: 5, Result: 1}}}
	check("PumpReactionRatesResult", CmdPumpReactionRatesResult, prs.Emit, false, func(r *wire.Reader) any {
		p := PumpReactionRatesResult{PumpedFriends: []PumpReactionRate{}}
		for n := r.U16(); n > 0; n-- {
			r.U16()
			p.PumpedFriends = append(p.PumpedFriends, PumpReactionRate{r.U32(), r.U8()})
		}
		return p
	}, prs)
	who := WhoPumpedReactionResult{TodayPumped: []ProfileDoubleKey{{1, sp("a")}}, YesterdayPumped: []ProfileDoubleKey{},
		TwoDaysAgoPumped: []ProfileDoubleKey{{2, sp("bb")}, {3, sp("")}}}
	check("WhoPumpedReactionResult", CmdWhoPumpedReaction, who.Emit, false, func(r *wire.Reader) any {
		var w WhoPumpedReactionResult
		read := func() []ProfileDoubleKey {
			l := []ProfileDoubleKey{}
			for n := r.U16(); n > 0; n-- {
				r.U16()
				l = append(l, ProfileDoubleKey{r.U32(), sp(r.UTF())})
			}
			return l
		}
		w.TodayPumped, w.YesterdayPumped, w.TwoDaysAgoPumped = read(), read(), read()
		return w
	}, who)
}

// ---- client messages: binary -> struct -> API JSON ---------------------------------------------------------

func TestClientMessages(t *testing.T) {
	w := func(f func(*wire.Writer)) *wire.Reader {
		var x wire.Writer
		f(&x)
		return wire.NewReader(x.B)
	}
	cases := []struct {
		name string
		json func() []byte
		want string
	}{
		{"Login", func() []byte {
			return ReadLogin(w(func(x *wire.Writer) {
				x.U32(5)
				x.U32(0)
				x.UTF("k\"é")
				x.U16(1)
				x.Bytes([]byte{4, 3, 2, 1}) // little-endian, as the C# server read it
				x.U8(1)
			})).JSON()
		}, `{"Id":5,"ReferrerId":0,"AuthKey":"k\"é","Ids":[16909060],"SocialCode":1}`},
		{"BuyShopItems", func() []byte {
			return ReadBuyShopItems(w(func(x *wire.Writer) { x.U16(1); x.U16(12); x.U32(5); x.U32(2); x.U32(1) })).JSON()
		}, `{"ShopItems":[{"Id":5,"Count":2,"MoneyType":1}]}`},
		{"GetArena", func() []byte { return ReadGetArena(w(func(x *wire.Writer) { x.U8(1) })).JSON() }, `{"ReturnUsersProfiles":true}`},
		{"StartBattle", func() []byte { return ReadStartBattle(w(func(x *wire.Writer) { x.U16(65535) })).JSON() }, `{"MissionId":-1}`},
		{"BuyBattle", func() []byte { return ReadBuyBattle(w(func(x *wire.Writer) { x.U32(1) })).JSON() }, `{"MoneyType":1}`},
		{"AddToGroup", func() []byte { return ReadAddToGroup(w(func(x *wire.Writer) { x.U32(7); x.U16(65535) })).JSON() }, `{"ProfileId":7,"MoneyType":-1}`},
		{"RemoveFromGroup", func() []byte { return ReadRemoveFromGroup(w(func(x *wire.Writer) { x.U32(7) })).JSON() }, `{"ProfileId":7}`},
		{"DistributePoints", func() []byte {
			return ReadDistributePoints(w(func(x *wire.Writer) { x.U32(2); x.U32(0xFFFFFFFF) })).JSON()
		}, `{"Armor":2,"Attack":-1}`},
		{"ResetParameters", func() []byte { return ReadResetParameters(w(func(x *wire.Writer) { x.U32(0) })).JSON() }, `{"MoneyType":0}`},
		{"ReorderGroup", func() []byte {
			return ReadReorderGroup(w(func(x *wire.Writer) { x.U16(2); x.U32(1); x.U32(2) })).JSON()
		}, `{"ReorderedWormGroup":[1,2]}`},
		{"SelectStuff", func() []byte { return ReadSelectStuff(w(func(x *wire.Writer) { x.U16(300) })).JSON() }, `{"StuffId":300}`},
		{"ChangeRace", func() []byte { return ReadChangeRace(w(func(x *wire.Writer) { x.U8(2); x.U8(1) })).JSON() }, `{"RaceId":2,"MoneyType":1}`},
		{"GetWhoPumpedReaction", func() []byte { return ReadGetWhoPumpedReaction(w(func(x *wire.Writer) { x.U8(0) })).JSON() }, `{"TodayOnly":false}`},
		{"WipeProfile", func() []byte { return ReadWipeProfile(w(func(x *wire.Writer) { x.UTF("12") })).JSON() }, `{"ConfirmCode":"12"}`},
		{"BuyReactionRate", func() []byte { return ReadBuyReactionRate(w(func(x *wire.Writer) { x.U8(5) })).JSON() }, `{"ReactionRateCount":5}`},
		{"BuyUnlockMission", func() []byte { return ReadBuyUnlockMission(w(func(x *wire.Writer) { x.U16(12) })).JSON() }, `{"MissionId":12}`},
		{"SearchTheHouse", func() []byte {
			return ReadSearchTheHouse(w(func(x *wire.Writer) { x.UTF("1.a"); x.U32(2); x.U8(1) })).JSON()
		}, `{"SessionKey":"1.a","FriendId":2,"KeyNum":1}`},
		{"PumpReactionRates", func() []byte {
			return ReadPumpReactionRates(w(func(x *wire.Writer) { x.U16(2); x.U32(2); x.U32(3) })).JSON()
		}, `{"FriendsIds":[2,3]}`},
		{"PumpReactionRate", func() []byte { return ReadPumpReactionRate(w(func(x *wire.Writer) { x.U32(2) })).JSON() }, `{"FriendId":2}`},
		{"UpgradeWeapon", func() []byte { return ReadRecipe(w(func(x *wire.Writer) { x.U16(4) })).JSON() }, `{"RecipeId":4}`},
		{"EndBattle", func() []byte {
			return ReadEndBattle(w(func(x *wire.Writer) {
				x.UTF("r")
				x.U32(1)
				x.UTF("t")
				x.U32(0)
				x.UTF("e")
				x.U32(5)
				x.UTF("b")
				x.U32(1077)
				x.U16(3)
				x.U16(1)
				x.U16(8)
				x.U32(5)
				x.U32(0xFFFFFFFF)
				x.U16(2)
				x.Bytes([]byte{1, 2})
				x.U16(0)
				x.UTF("n")
				x.U16(1)
				x.U8(0x80)
			})).JSON()
		}, `{"Result":1,"Type":0,"ExpBonus":5,"BattleId":1077,"MissionId":3,"Items":[{"Id":5,"Count":-1}],"Signature":[1,2],"BanType":0,"BanNote":"n","CollectedReagents":[128]}`},
	}
	for _, c := range cases {
		var got []byte
		if exc := csx.Catch(func() { got = c.json() }); exc != nil {
			t.Errorf("%s: %v", c.name, exc)
			continue
		}
		if string(got) != c.want {
			t.Errorf("%s:\n got %s\nwant %s", c.name, got, c.want)
		}
	}
	// Truncated commands throw EndOfStream (the C# server closed the connection).
	if exc := csx.Catch(func() { ReadDistributePoints(wire.NewReader([]byte{0, 0, 0, 1})) }); exc == nil || exc.Kind != csx.EndOfStream {
		t.Errorf("truncated: %v", exc)
	}
}
