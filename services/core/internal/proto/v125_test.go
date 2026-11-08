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
