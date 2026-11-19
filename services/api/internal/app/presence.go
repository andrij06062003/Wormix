package app

import (
	"sort"
	"sync"
	"time"
)

// Who is in the game. The game's page (wwwroot/play.html) asks GET api/chat for new messages every 10 seconds
// while a player is logged in (more often while the chat is open), so a player the server heard from lately has
// the game open. The page shows them with a green dot in the chat and the profile. Kept in memory: after a
// restart everybody is back within one poll, only "last seen" of the others is forgotten.
const (
	// OnlineWindow: a player counts as in the game this long after their page's last request. Browsers poll a
	// hidden tab less often (Chrome: about once a minute after 5 minutes), so it is well over the 10 s poll.
	OnlineWindow  = 90 * time.Second
	onlineListMax = 200 // players in the "in the game" list of one answer
)

type presence struct {
	mu   sync.Mutex
	seen map[int64]time.Time
}

func newPresence() *presence { return &presence{seen: map[int64]time.Time{}} }

// touch records that the player's page has just asked something.
func (p *presence) touch(id int64, now time.Time) {
	p.mu.Lock()
	p.seen[id] = now
	p.mu.Unlock()
}

// lastSeen is when the player's page last asked (zero: not since the server started).
func (p *presence) lastSeen(id int64) time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.seen[id]
}

func (p *presence) online(id int64, now time.Time) bool {
	t := p.lastSeen(id)
	return !t.IsZero() && now.Sub(t) < OnlineWindow
}

// onlineIDs: the players in the game, the most recently seen first, at most `limit`.
func (p *presence) onlineIDs(now time.Time, limit int) []int64 {
	p.mu.Lock()
	type e struct {
		id int64
		at time.Time
	}
	var list []e
	for id, at := range p.seen {
		if now.Sub(at) < OnlineWindow {
			list = append(list, e{id, at})
		}
	}
	p.mu.Unlock()
	sort.Slice(list, func(i, j int) bool {
		if !list[i].at.Equal(list[j].at) {
			return list[i].at.After(list[j].at)
		}
		return list[i].id < list[j].id
	})
	if len(list) > limit {
		list = list[:limit]
	}
	out := make([]int64, len(list))
	for i, x := range list {
		out[i] = x.id
	}
	return out
}
