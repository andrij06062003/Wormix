package pvp

import "testing"

type fakePlayer struct{ id uint32 }

func (p *fakePlayer) UserID() uint32            { return p.id }
func (p *fakePlayer) AuthKey() string           { return "" }
func (p *fakePlayer) Token() string             { return "" }
func (p *fakePlayer) SendMessage(_ []byte) bool { return true }

func withLobby(t *testing.T, ids ...uint32) map[uint32]*fakePlayer {
	t.Helper()
	ps := map[uint32]*fakePlayer{}
	for _, id := range ids {
		ps[id] = &fakePlayer{id}
		Register(ps[id])
	}
	t.Cleanup(func() {
		for _, p := range ps {
			Unregister(p)
		}
		queue, teams = nil, map[uint32]uint32{}
	})
	return ps
}

func enqueue(ps map[uint32]*fakePlayer, ids ...uint32) {
	for _, id := range ids {
		queue = append(queue, &wagerTicket{session: ps[id], wager: 15, mode: modeTeam})
	}
}

func sideOf(order []*wagerTicket) map[uint32]int {
	m := map[uint32]int{}
	for i, t := range order {
		m[t.session.UserID()] = i % 2
	}
	return m
}

func TestMatchTwoTeams(t *testing.T) {
	ps := withLobby(t, 1, 2, 3, 4)
	teams = map[uint32]uint32{1: 3, 3: 1, 2: 4, 4: 2}
	enqueue(ps, 1, 2, 3, 4)
	for i := 0; i < 20; i++ { // sides and order are random
		g := matchTeams(15)
		if len(g) != 4 {
			t.Fatalf("no match: %v", g)
		}
		s := sideOf(g)
		if s[1] != s[3] || s[2] != s[4] || s[1] == s[2] {
			t.Fatalf("teammates split: %v", s)
		}
	}
}

func TestMatchWaitsForTeammate(t *testing.T) {
	ps := withLobby(t, 1, 2, 3, 4, 5)
	teams = map[uint32]uint32{1: 5, 5: 1}
	enqueue(ps, 1, 2, 3, 4) // 1's teammate 5 isn't queued yet
	if g := matchTeams(15); g != nil {
		t.Fatalf("matched without the teammate: %v", sideOf(g))
	}
	enqueue(ps, 5)
	g := matchTeams(15)
	if g == nil {
		t.Fatal("no match once the teammate came")
	}
	s := sideOf(g)
	if s[1] != s[5] {
		t.Fatalf("teammates split: %v", s)
	}
	if _, ok := s[4]; ok {
		t.Fatalf("took 4 instead of the earlier 2 and 3: %v", s)
	}
}
