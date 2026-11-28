package proto

import (
	"bytes"
	"testing"

	"github.com/andrij06062003/Wormix/services/core/internal/jsonx"
	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

func TestLogin161(t *testing.T) {
	var w wire.Writer
	w.U32(7)
	w.U32(0)
	w.UTF("KEY")
	w.U32(1617)
	w.U16(0) // friend ids
	w.U8(0)  // no social network
	body := append(w.B, wire.SecureWith(w.B, wire.SecretKey161)...)
	if !IsLogin161(body) || IsLogin125(body) {
		t.Fatal("1.61.7 login not recognized")
	}
	if l := ReadLogin125(wire.NewReader(body)); l.ID != 7 || l.AuthKey != "KEY" {
		t.Fatalf("login %+v", l)
	}
	if IsLogin161(login125Body(7, "KEY")) {
		t.Fatal("1.25.0 login taken for a 1.61.7 one")
	}
}

func TestResign161(t *testing.T) {
	payload := []byte{1, 2, 3}
	for _, signed := range [][]byte{Frame(CmdShopResult, payload, true), Frame125(CmdShopResult, payload, true)} {
		got := Resign161(signed)
		if !wire.IsSecureWith(got[wire.HeaderSize:], wire.SecretKey161) || !bytes.Equal(got[:len(got)-16], signed[:len(signed)-16]) {
			t.Fatalf("not re-signed: %x", got)
		}
	}
	plain := Frame(CmdShopResult, payload, false)
	if !bytes.Equal(Resign161(plain), plain) {
		t.Fatal("unsigned message changed")
	}
}

// readProfile161 reads a UserProfileStructure as the 1.61.7 client does; worms: owner, hat, race, artifact,
// active.
func readProfile161(t *testing.T, r *wire.Reader) (id uint32, money uint32, worms [][]int, weapons [][2]int) {
	t.Helper()
	id = r.U32()
	money = r.U32()
	r.U32() // realMoney
	r.U32() // rating
	n := int(r.U16())
	for i := 0; i < n; i++ {
		size := int(r.U16())
		before := r.Remaining()
		owner := r.U32()
		r.U8()
		r.U8()
		r.U8()
		r.U32()
		hat := int(int16(r.U16()))
		race := int(r.U8())
		r.U8() // skin
		artifact := int(int16(r.U16()))
		r.UTF() // socialOwnerId
		r.UTF() // name
		r.U16() // teamMemberType
		active := int(r.U8())
		if before-r.Remaining() != size {
			t.Fatalf("worm size %d, read %d", size, before-r.Remaining())
		}
		worms = append(worms, []int{int(owner), hat, race, artifact, active})
	}
	n = int(r.U16())
	for i := 0; i < n; i++ {
		weapons = append(weapons, [2]int{int(int16(r.U16())), int(int16(r.U16()))})
	}
	n = int(r.U16())
	for i := 0; i < n; i++ {
		r.U16()
	}
	n = int(r.U16()) / 6
	for i := 0; i < n; i++ {
		r.U16()
		r.U32()
	}
	r.U32() // reactionRate
	r.UTF() // socialId
	r.U8()  // socialNetId
	n = int(r.U16())
	for i := 0; i < n; i++ {
		r.U16()
	}
	size := int(r.U16())
	before := r.Remaining()
	r.U16() // rank
	r.U32() // clanId
	r.UTF() // clanName
	for e := int(r.U16()); e > 0; e-- {
		r.U8()
	}
	r.U32()
	r.U32()
	r.U16()
	r.U32()
	if before-r.Remaining() != size {
		t.Fatalf("clan size %d, read %d", size, before-r.Remaining())
	}
	r.U8()  // extraGroupSlotsCount
	r.U32() // rankPoints
	r.U8()  // bestRank
	return
}

func TestEnterAccount161(t *testing.T) {
	social, key := "s", "1.k"
	worm := Worm{OwnerID: 1, SocialOwnerID: &social, Armor: 3, Attack: 5, Level: 4, Hat: 2*500 + 1005, Artifact: 2001}
	bare := Worm{OwnerID: 2, SocialOwnerID: &social, Level: 1, Hat: 3}
	me := UserProfile{ID: 1, SocialID: &social, Money: 450, RealMoney: 7, WormsGroup: []Worm{worm, bare},
		WeaponRecordList: []Weapon{{ID: 5, Count: -1}, {ID: 12, Count: 3}}, Stuff: []uint16{1005}, Recipes: []uint16{4}}
	friend := UserProfile{ID: 2, SocialID: &social, WormsGroup: []Worm{bare}, WeaponRecordList: []Weapon{}, Stuff: []uint16{}, Recipes: []uint16{}}
	e := EnterAccount{UserProfileStructure: me, UserProfileStructures: []UserProfile{friend}, OnlineFriends: 1, Friends: 1,
		SessionKey: &key, AvailableSearchKeys: 10, Reagents: []uint32{3}}
	var got []byte
	e.Emit161(func(b []byte) { got = append(got, b...) }, 1234)

	cmd, n, err := wire.ReadHeader(bytes.NewReader(got))
	if err != nil || cmd != CmdEnterAccount || int(n) != len(got)-wire.HeaderSize {
		t.Fatalf("header %d %d %v", cmd, n, err)
	}
	body := got[wire.HeaderSize:]
	if !wire.IsSecureWith(body, wire.SecretKey161) {
		t.Fatal("not signed with the 1.61.7 salt")
	}
	r := wire.NewReader(body[:len(body)-16])
	size := int(r.U16())
	before := r.Remaining()
	id, money, worms, weapons := readProfile161(t, r)
	if before-r.Remaining() != size {
		t.Fatalf("profile size %d, read %d", size, before-r.Remaining())
	}
	if id != 1 || money != 450 || len(weapons) != 2 || weapons[0] != [2]int{5, -1} || weapons[1] != [2]int{12, 3} {
		t.Fatalf("profile %d %d %v", id, money, weapons)
	}
	if w := worms[0]; w[1] != 1005 || w[2] != 2 || w[3] != 2001 || w[4] != 1 {
		t.Fatalf("worm with a hat %v", w)
	}
	if w := worms[1]; w[1] != 0 || w[2] != 3 {
		t.Fatalf("worm without a hat %v", w)
	}
	if n := r.U16(); n != 1 {
		t.Fatalf("friends %d", n)
	}
	r.U16()
	if id, _, _, _ := readProfile161(t, r); id != 2 {
		t.Fatalf("friend %d", id)
	}
	if r.U16() != 0 || r.U16() != 1 || r.U16() != 1 || r.UTF() != "1.k" || r.U8() != 10 {
		t.Fatal("login awards / friends / session key / search keys")
	}
	if r.U16() != 1 || r.U32() != 3 {
		t.Fatal("reagents")
	}
	r.U16() // curSoloMissionId
	r.U16() // curCooperativeMissionId
	if r.U16() != 0 || r.U32() != 1234 {
		t.Fatal("invites / server time")
	}
	if r.U16() != 0 || r.U8() != 0 || r.U16() != 0 {
		t.Fatal("backpacks / active backpack / hotkeys")
	}
	r.U8() // loginSequence
	if races := r.U16(); races != 1<<2 {
		t.Fatalf("races %b", races)
	}
	r.U32() // selectRaceTimeLeft
	if r.U16() != 0 {
		t.Fatal("skins")
	}
	r.U32() // lastPaymentTime
	if r.U16() != 0 || r.U16() != 0 {
		t.Fatal("restrictions / cookies")
	}
	r.U32() // vipSubscriptionId
	if r.U16() != 0 {
		t.Fatal("battle to reconnect to")
	}
	r.U8() // downgradesLeft
	if r.Remaining() != 0 {
		t.Fatalf("%d bytes left", r.Remaining())
	}
}

func TestArenaResult161(t *testing.T) {
	var got []byte
	(&ArenaResult{BattlesCount: 7, CurrentMission: 3, BossAvailable: true, CurrentCoopMission: 101}).Emit161(func(b []byte) { got = append(got, b...) })
	r := wire.NewReader(got[wire.HeaderSize:])
	if r.U32() != 7 || r.U16() != 3 || r.U16() != 101 || !r.Bool() || r.Bool() || r.U16() != 0 {
		t.Fatal("1.25.0 fields")
	}
	r.U32()
	r.U32()
	if r.U16() != 0 || r.U16() != 0 || r.U16() != 0 {
		t.Fatal("wagers / heroic progress")
	}
	r.U32()
	if s := r.UTF(); s != "{}" {
		t.Fatalf("timetable %q", s)
	}
	r.U32() // wagerWinAwardToken
	if n := r.U32(); n == 0 {
		t.Fatal("bossWinAwardToken 0: the client keeps the next boss closed until tomorrow")
	}
	size := int(r.U16())
	before := r.Remaining()
	r.U16()
	r.U16()
	if before-r.Remaining() != size {
		t.Fatal("boss awards size")
	}
	r.U32()
	if r.Remaining() != 0 {
		t.Fatalf("%d bytes left", r.Remaining())
	}
}

func TestStartBattleResult161(t *testing.T) {
	var got []byte
	(&StartBattleResult{BattleID: 42, ReagentForBattle: []byte{3, 4}}).Emit161(func(b []byte) { got = append(got, b...) })
	body := got[wire.HeaderSize:]
	if !wire.IsSecure(body) {
		t.Fatal("not signed (the session re-signs it)")
	}
	r := wire.NewReader(body[:len(body)-16])
	if r.U32() != 42 || r.U16() != 2 || r.U8() != 3 || r.U8() != 4 || r.Remaining() != 0 {
		t.Fatal("battle id / reagents")
	}
}

func TestReadEndBattle161(t *testing.T) {
	var w wire.Writer
	w.U32(1)  // result
	w.U16(2)  // type
	w.U32(5)  // expBonus
	w.U32(42) // battleId
	w.U16(7)  // missionId
	w.U16(1)  // items
	w.U16(8)
	w.U32(12)
	w.U32(1)
	w.U16(0)
	w.UTF("")
	w.U16(2) // reagents
	w.U8(3)
	w.U8(4)
	w.U16(2) // TurnStructureV2.VERSION_MARKER and the rest: not read
	w.U16(9)
	e := ReadEndBattle161(wire.NewReader(w.B))
	if e.Result != 1 || e.Type != 2 || e.ExpBonus != 5 || e.BattleID != 42 || e.MissionID != 7 ||
		len(e.Items) != 1 || e.Items[0].ID != 12 || len(e.CollectedReagents) != 2 || e.CollectedReagents[1] != 4 {
		t.Fatalf("end battle %+v", e)
	}
}

// The whole 1.61.7 EndBattle (statistics, journal, map, session key and signature), and its EndBattleResult.
func TestEndBattleResult161(t *testing.T) {
	var w wire.Writer
	w.U32(1043) // result + battle id
	w.U16(1)
	w.U32(0)
	w.U32(1042)
	w.U16(25)
	w.U16(0) // items
	w.U16(0)
	w.UTF("")
	w.U16(0) // reagents
	w.U16(2) // VERSION_MARKER
	w.U16(14)
	w.U32(300)
	w.U32(3508)
	w.U16(1) // items used
	w.U16(8)
	w.U32(54)
	w.U32(2)
	w.U16(2) // journal
	w.U16(3)
	w.Bytes([]byte{1, 2, 3})
	w.U16(0)
	w.U16(77) // map
	w.UTF("key")
	w.Bytes(make([]byte, 16)) // signature
	e := ReadEndBattle161(wire.NewReader(w.B))
	if e.Result != 1043 || e.BattleID != 1042 || e.MissionID != 25 || e.SessionKey != "key" {
		t.Fatalf("end battle %+v", e)
	}
	var got []byte
	EmitEndBattleResult161(func(b []byte) { got = append(got, b...) }, e, nil)
	if wire.FrameLength(got) != len(got)-wire.HeaderSize {
		t.Fatal("frame length")
	}
	body := got[wire.HeaderSize:]
	if !wire.IsSecure(body) {
		t.Fatal("not signed (the session re-signs it)")
	}
	r := wire.NewReader(body[:len(body)-16])
	if r.U16() != 0 || r.U32() != 1043 || r.U32() != 1042 || r.U16() != 25 || r.UTF() != "key" || r.U16() != 0 || r.Remaining() != 0 {
		t.Fatal("end battle result")
	}
}

// The awards game/end_battle gave go into EndBattleResult, each a sized GenericAwardStructure.
func TestEndBattleResult161Awards(t *testing.T) {
	v, err := jsonx.Parse([]byte(`{"Status":"OK","Awards":[{"Kind":1,"Count":75,"ItemId":0},{"Kind":5,"Count":4,"ItemId":72},"x"]}`))
	if err != nil {
		t.Fatal(err)
	}
	awards := DecodeBattleAwards(v)
	if len(awards) != 2 || awards[0] != (BattleAward{1, 75, 0}) || awards[1] != (BattleAward{5, 4, 72}) {
		t.Fatalf("awards %+v", awards)
	}
	if DecodeBattleAwards(nil) != nil {
		t.Fatal("no answer: awards")
	}
	var got []byte
	EmitEndBattleResult161(func(b []byte) { got = append(got, b...) }, EndBattle{Result: 1, BattleID: 2, MissionID: 30}, awards)
	body := got[wire.HeaderSize:]
	r := wire.NewReader(body[:len(body)-16])
	r.U16()
	r.U32()
	r.U32()
	r.U16()
	r.UTF()
	if r.U16() != 2 {
		t.Fatal("award count")
	}
	for _, want := range awards {
		if r.U16() != 10 || int16(r.U16()) != want.Kind || int32(r.U32()) != want.Count || int32(r.U32()) != want.ItemID {
			t.Fatalf("award %+v", want)
		}
	}
	if r.Remaining() != 0 {
		t.Fatalf("%d bytes left", r.Remaining())
	}
}
