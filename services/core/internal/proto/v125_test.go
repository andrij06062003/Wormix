package proto

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

func login125Body(id uint32, key string) []byte {
	var w wire.Writer
	w.U32(id)
	w.U32(0)
	w.UTF(key)
	w.U32(1250)
	w.U16(0) // friend ids
	w.U8(1)
	w.U16(1) // params
	w.UTF("p")
	return append(w.B, wire.SecureWith(w.B, wire.SecretKey125)...)
}

// A Login sent by the real 1.25.0 client (user 8 with 8 friends), captured from the browser.
const capturedLogin125 = "000000080000000000403632356463356434306262616564363465616432303961616233343534396561616563343266333234353937643631383737303137333935313338306232393101190000000800000005000000070000000400000008000000060000000200000003000000010100020003666c760003333230050e1641751c690112393151f0c696a0"

func TestCapturedLogin125(t *testing.T) {
	body, _ := hex.DecodeString(capturedLogin125)
	if !IsLogin125(body) {
		t.Fatal("the real client's login not recognized")
	}
	r := wire.NewReader(body)
	l := ReadLogin125(r)
	if l.ID != 8 || len(l.AuthKey) != 64 || l.SocialCode != 1 {
		t.Fatalf("login %+v", l)
	}
	if want := []uint32{5, 7, 4, 8, 6, 2, 3, 1}; fmt.Sprint(l.IDs) != fmt.Sprint(want) {
		t.Fatalf("friend ids %v, want %v", l.IDs, want)
	}
}

func TestLogin125(t *testing.T) {
	body := login125Body(7, "KEY")
	if !IsLogin125(body) {
		t.Fatal("1.25.0 login not recognized")
	}
	l := ReadLogin125(wire.NewReader(body))
	if l.ID != 7 || l.AuthKey != "KEY" || l.SocialCode != 1 {
		t.Fatalf("login %+v", l)
	}
	var old wire.Writer // a 1.05.0 login: not signed
	old.U32(7)
	old.U32(0)
	old.UTF("KEY")
	old.U16(0)
	old.U8(1)
	if IsLogin125(old.B) {
		t.Fatal("1.05.0 login taken for a 1.25.0 one")
	}
}

func TestResign125(t *testing.T) {
	payload := []byte{1, 2, 3}
	signed := Frame(CmdShopResult, payload, true)
	got := Resign125(signed)
	if !wire.IsSecureWith(got[wire.HeaderSize:], wire.SecretKey125) || !bytes.Equal(got[:len(got)-16], signed[:len(signed)-16]) {
		t.Fatalf("not re-signed: %x", got)
	}
	plain := Frame(CmdShopResult, payload, false)
	if !bytes.Equal(Resign125(plain), plain) {
		t.Fatal("unsigned message changed")
	}
	if part := signed[:5]; !bytes.Equal(Resign125(part), part) {
		t.Fatal("piece of a message changed")
	}
}

// readProfile125 reads a UserProfileStructure in the order of the 1.25.0 client.
func readProfile125(t *testing.T, r *wire.Reader) (id uint32, money, real int32, worms [][]int, weapons int, social string) {
	id = r.U32()
	money, real = int32(r.U32()), int32(r.U32())
	r.U32() // rating
	for n := int(r.U16()); n > 0; n-- {
		owner := r.U32()
		armor, attack, level := r.U8(), r.U8(), r.U8()
		r.U32() // experience
		hat := int16(r.U16())
		race := r.U8()
		artifact := int16(r.U16())
		r.UTF()
		r.U16()
		active := r.Bool()
		worms = append(worms, []int{int(owner), int(armor), int(attack), int(level), int(hat), int(race), int(artifact), btoi(active)})
	}
	weapons = int(r.U16())
	for i := 0; i < weapons; i++ {
		r.U32()
		r.U32()
	}
	for n := int(r.U16()); n > 0; n-- {
		r.U16()
	}
	for n := int(r.U16()) / 6; n > 0; n-- {
		r.U16()
		r.U32()
	}
	r.U32() // reactionRate
	social = r.UTF()
	for n := int(r.U16()); n > 0; n-- {
		r.U16()
	}
	r.U16()
	r.U32()
	r.UTF()
	for n := int(r.U16()); n > 0; n-- {
		r.U8()
	}
	r.U32()
	r.U32()
	r.U16()
	r.U32()
	return
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func TestEnterAccount125(t *testing.T) {
	social, key := "s", "1.k"
	worm := Worm{OwnerID: 1, SocialOwnerID: &social, Armor: 3, Attack: 5, Level: 4, Hat: 2*500 + 1005, Artifact: 2001}
	bare := Worm{OwnerID: 2, SocialOwnerID: &social, Level: 1, Hat: 3}
	me := UserProfile{ID: 1, SocialID: &social, Money: 450, RealMoney: 7, WormsGroup: []Worm{worm, bare},
		WeaponRecordList: []Weapon{{ID: 5, Count: -1}}, Stuff: []uint16{1005}, Recipes: []uint16{}}
	friend := UserProfile{ID: 2, SocialID: &social, WormsGroup: []Worm{bare}, WeaponRecordList: []Weapon{}, Stuff: []uint16{}, Recipes: []uint16{}}
	e := EnterAccount{UserProfileStructure: me, UserProfileStructures: []UserProfile{friend}, OnlineFriends: 1, Friends: 1,
		SessionKey: &key, AvailableSearchKeys: 10, Reagents: []uint32{3}}
	var got []byte
	e.Emit125(func(b []byte) { got = append(got, b...) }, 1234)

	cmd, n, err := wire.ReadHeader(bytes.NewReader(got))
	if err != nil || cmd != CmdEnterAccount || int(n) != len(got)-wire.HeaderSize {
		t.Fatalf("header %d %d %v", cmd, n, err)
	}
	body := got[wire.HeaderSize:]
	if !wire.IsSecureWith(body, wire.SecretKey125) {
		t.Fatal("not signed with the 1.25.0 salt")
	}
	r := wire.NewReader(body[:len(body)-16])
	size := int(r.U16())
	before := r.Remaining()
	id, money, real, worms, weapons, soc := readProfile125(t, r)
	if before-r.Remaining() != size {
		t.Fatalf("profile size %d, read %d", size, before-r.Remaining())
	}
	if id != 1 || money != 450 || real != 7 || weapons != 1 || soc != "s" {
		t.Fatalf("profile %d %d %d %d %q", id, money, real, weapons, soc)
	}
	// owner, armor, attack, level, hat, race, artifact, active
	if w := worms[0]; w[4] != 1005 || w[5] != 2 || w[6] != 2001 || w[1] != 3 || w[2] != 5 || w[3] != 4 || w[7] != 1 {
		t.Fatalf("worm with a hat %v", w)
	}
	if w := worms[1]; w[4] != 0 || w[5] != 3 {
		t.Fatalf("worm without a hat %v", w)
	}
	if n := r.U16(); n != 1 {
		t.Fatalf("friends %d", n)
	}
	r.U16()
	if id, _, _, _, _, _ := readProfile125(t, r); id != 2 {
		t.Fatalf("friend %d", id)
	}
	if r.U16() != 0 || r.U16() != 1 || r.U16() != 1 || r.UTF() != "1.k" || r.U8() != 10 {
		t.Fatal("login awards / friends / session key / search keys")
	}
	if r.U16() != 1 || r.U32() != 3 {
		t.Fatal("reagents")
	}
	r.U16()
	r.U16()
	if r.U16() != 0 || r.U32() != 1234 || r.U16() != 0 || r.U16() != 0 {
		t.Fatal("invites / server time / backpack / hotkeys")
	}
	r.U8()
	if r.Remaining() != 0 {
		t.Fatalf("%d bytes left", r.Remaining())
	}
}

func TestArenaResult125(t *testing.T) {
	var got []byte
	(&ArenaResult{BattlesCount: 7, CurrentMission: 3, BossAvailable: true}).Emit125(func(b []byte) { got = append(got, b...) })
	cmd, n, err := wire.ReadHeader(bytes.NewReader(got))
	if err != nil || cmd != CmdArenaResult || int(n) != len(got)-wire.HeaderSize || n != 12 {
		t.Fatalf("header %d %d %v", cmd, n, err)
	}
	r := wire.NewReader(got[wire.HeaderSize:])
	if b, solo, coop, boss, super, heroic := r.U32(), r.U16(), r.U16(), r.Bool(), r.Bool(), r.U16(); b != 7 || solo != 3 || coop != 0 || !boss || super || heroic != 0 {
		t.Fatalf("got %d %d %d %v %v %d", b, solo, coop, boss, super, heroic)
	}
}

// EndBattle as the 1.25.0 client writes it (a resign with one item and two reagents).
func TestReadEndBattle125(t *testing.T) {
	var w wire.Writer
	w.UTF("abcdefghijklmnop")
	w.I32(-1) // result
	w.UTF("abcde")
	w.I32(2) // type
	w.UTF("abc")
	w.I32(0) // expBonus
	w.UTF("ab")
	w.I32(77) // battleId
	w.I16(-1) // missionId
	w.U16(1)
	w.U16(8)
	w.U32(5)
	w.I32(-3)
	w.I16(0)  // banType
	w.UTF("") // banNote
	w.U16(2)
	w.U8(3)
	w.U8(4)
	b := append(w.B, wire.SecureWith(w.B, wire.SecretKey125)...)
	e := ReadEndBattle125(wire.NewReader(b))
	if e.Result != -1 || e.Type != 2 || e.BattleID != 77 || e.MissionID != -1 || len(e.Items) != 1 || e.Items[0].ID != 5 ||
		e.Items[0].Count != -3 || e.BanType != 0 || fmt.Sprint(e.CollectedReagents) != "[3 4]" {
		t.Fatalf("%+v", e)
	}
}

func TestAssembleEquipResult125(t *testing.T) {
	var got []byte
	EmitAssembleEquipResult125(func(b []byte) { got = append(got, b...) }, 0, 7, 1)
	cmd, n, err := wire.ReadHeader(bytes.NewReader(got))
	if err != nil || cmd != CmdAssembleEquipResult125 || int(n) != len(got)-wire.HeaderSize {
		t.Fatalf("header %d %d %v", cmd, n, err)
	}
	body := got[wire.HeaderSize:]
	r := wire.NewReader(body)
	if equip, family, result, key := r.U16(), r.U16(), r.U16(), r.UTF(); equip != 0 || family != 7 || result == 0 || key != "" || r.Remaining() != 16 {
		t.Fatalf("got %d %d %d %q, %d left", equip, family, result, key, r.Remaining())
	}
}

func TestSplitCraftedHat(t *testing.T) {
	for _, c := range [][3]int{{2005, 2, 1005}, {3, 3, 0}, {20000 + 2*200 + 1, 2, 1501}, {20000 + 5*200 + 147, 5, 1647}} {
		if r, h := splitHat(int16(c[0])); r != c[1] || h != c[2] {
			t.Fatalf("splitHat(%d) = %d %d, want %d %d", c[0], r, h, c[1], c[2])
		}
	}
}

// The 1.05.0 client gets no crafted equipment: not in the stuff, not worn.
func TestCraftedHiddenFrom105(t *testing.T) {
	social := "s"
	p := UserProfile{ID: 1, SocialID: &social, Stuff: []uint16{1005, 1501, 2001, 2546}, Recipes: []uint16{},
		WeaponRecordList: []Weapon{}, WormsGroup: []Worm{{OwnerID: 1, SocialOwnerID: &social, Hat: 20000 + 2*200 + 1, Artifact: 2546}}}
	var w wire.Writer
	p.Write(&w)
	if int(p.Size()) != len(w.B) {
		t.Fatalf("size %d, wrote %d", p.Size(), len(w.B))
	}
	r := wire.NewReader(w.B)
	r.U32()
	r.U32()
	r.U32()
	r.U32()
	r.U16() // one worm
	r.U32()
	r.U32()
	r.U32()
	r.U32()
	r.U32()
	if hat := r.U16(); hat != 2 {
		t.Fatalf("hat %d, want the race only (2)", hat)
	}
	r.UTF()
	if a := r.U16(); a != 0 {
		t.Fatalf("artifact %d, want 0", a)
	}
	r.U16() // no weapons
	if n := r.U16(); n != 2 || r.U16() != 1005 || r.U16() != 2001 {
		t.Fatalf("stuff: %d items", n)
	}
}

// The 1.25.0 hats after 1061 are unknown to the 1.05.0 client too: the worm shows the race only and
// the hat is left out of the stuff.
func TestNewHatsHiddenFrom105(t *testing.T) {
	social := "s"
	p := UserProfile{ID: 1, SocialID: &social, Stuff: []uint16{1061, 1064, 2001}, Recipes: []uint16{},
		WeaponRecordList: []Weapon{}, WormsGroup: []Worm{{OwnerID: 1, SocialOwnerID: &social, Hat: 1*500 + 1064}}}
	var w wire.Writer
	p.Write(&w)
	if int(p.Size()) != len(w.B) {
		t.Fatalf("size %d, wrote %d", p.Size(), len(w.B))
	}
	r := wire.NewReader(w.B)
	for range 4 {
		r.U32()
	}
	r.U16() // one worm
	for range 5 {
		r.U32()
	}
	if hat := r.U16(); hat != 1 {
		t.Fatalf("hat %d, want the race only (1)", hat)
	}
	r.UTF()
	r.U16() // artifact
	r.U16() // no weapons
	if n := r.U16(); n != 2 || r.U16() != 1061 || r.U16() != 2001 {
		t.Fatalf("stuff: %d items", n)
	}

	// A hat the 1.05.0 client knows stays.
	var k wire.Writer
	(&Worm{OwnerID: 1, SocialOwnerID: &social, Hat: 2*500 + 1061}).write(&k)
	kr := wire.NewReader(k.B)
	for range 5 {
		kr.U32()
	}
	if hat := kr.U16(); hat != 2*500+1061 {
		t.Fatalf("known hat %d", hat)
	}
}

// The 1.25.0 client reads the temporal stuff after the stuff; without it a purchase never finishes.
func TestShopResult125(t *testing.T) {
	var got []byte
	res := ShopResult{Result: 0, Weapons: []Weapon{{1061, -1}}, Stuff: []int16{3}}
	res.Emit125(func(b []byte) { got = append(got, b...) })
	cmd, n, err := wire.ReadHeader(bytes.NewReader(got))
	if err != nil || cmd != CmdShopResult || int(n) != len(got)-wire.HeaderSize {
		t.Fatalf("header %d %d %v", cmd, n, err)
	}
	body := got[wire.HeaderSize:]
	if !wire.IsSecure(body) {
		t.Fatal("not signed")
	}
	r := wire.NewReader(body[:len(body)-16])
	if r.U16() != 0 || r.U16() != 1 || r.U16() != weaponSize {
		t.Fatalf("% x", body)
	}
	if id, count := r.U32(), int32(r.U32()); id != 1061 || count != -1 {
		t.Fatalf("weapon %d %d", id, count)
	}
	if r.U16() != 1 || r.U16() != 3 || r.U16() != 0 {
		t.Fatalf("stuff and temporal stuff: % x", body)
	}
	if r.Remaining() != 0 {
		t.Fatalf("%d bytes left", r.Remaining())
	}
}

// The 1.25.0 wardrobe sends the hat and artifact of each worm and gets a result for each.
func TestSelectStuff125(t *testing.T) {
	var w wire.Writer
	w.U16(2)
	for _, x := range [][3]int{{7, 1064, 0}, {9, 0, 2019}} {
		w.U16(8)
		w.U32(uint32(x[0]))
		w.I16(x[1])
		w.I16(x[2])
	}
	got := ReadSelectStuff125(wire.NewReader(w.B))
	if len(got) != 2 || got[0] != (SelectStuff125{7, 1064, 0}) || got[1] != (SelectStuff125{9, 0, 2019}) {
		t.Fatalf("read %v", got)
	}

	var msg []byte
	EmitSelectStuffResult125(func(b []byte) { msg = append(msg, b...) }, []SelectStuffResult125{{7, 0, 1064, 1, 0}})
	cmd, n, err := wire.ReadHeader(bytes.NewReader(msg))
	if err != nil || cmd != CmdSelectStuffResult || int(n) != len(msg)-wire.HeaderSize {
		t.Fatalf("header %d %d %v", cmd, n, err)
	}
	body := msg[wire.HeaderSize:]
	if !wire.IsSecure(body) {
		t.Fatal("not signed")
	}
	r := wire.NewReader(body[:len(body)-16])
	if r.U16() != 1 || r.U16() != 12 || r.U32() != 7 || r.U16() != 0 || r.U16() != 1064 || r.U16() != 1 || r.U16() != 0 || r.Remaining() != 0 {
		t.Fatalf("% x", body)
	}
}
