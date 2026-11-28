package app

import (
	"testing"
	"time"
)

// A player whose page polled the chat lately is in the game: in the chat's `online` list and in their profile.
func TestPresence(t *testing.T) {
	srv, _, v := newServer(t)
	alice := map[string]string{"Authorization": "Bearer " + v.Users["alice"].Token}
	bob := map[string]string{"Authorization": "Bearer " + v.Users["bob"].Token}

	if st, j := call(t, srv, "GET", "/api/account/info/1", nil, bob); st != 200 || j["data"].(map[string]any)["online"] != false ||
		j["data"].(map[string]any)["last_seen_at"] != nil {
		t.Fatalf("alice before her page polled: %d %v", st, j)
	}
	call(t, srv, "GET", "/api/chat?after=0", nil, alice)
	st, j := call(t, srv, "GET", "/api/chat?after=0", nil, bob)
	if st != 200 {
		t.Fatalf("chat: %d %v", st, j)
	}
	online := j["online"].([]any)
	if len(online) != 2 || online[0] != float64(2) || online[1] != float64(1) {
		t.Fatalf("online %v, want bob (the latest) then alice", online)
	}
	if u := j["users"].(map[string]any)["1"]; u == nil {
		t.Fatalf("no name and photo for alice, who is online: %v", j["users"])
	}
	st, j = call(t, srv, "GET", "/api/account/info/1", nil, bob)
	if d := j["data"].(map[string]any); st != 200 || d["online"] != true || d["last_seen_at"] == nil {
		t.Fatalf("alice after her page polled: %d %v", st, j)
	}
	if _, j := call(t, srv, "GET", "/api/account/info/3", nil, bob); j["data"].(map[string]any)["online"] != false {
		t.Fatalf("carol never opened the game: %v", j)
	}
}

func TestPresenceWindow(t *testing.T) {
	p := newPresence()
	now := time.Now()
	p.touch(1, now.Add(-OnlineWindow-time.Second))
	p.touch(2, now.Add(-time.Minute))
	p.touch(3, now)
	if p.online(1, now) || !p.online(2, now) || !p.online(3, now) || p.online(4, now) {
		t.Fatal("online window")
	}
	if ids := p.onlineIDs(now, 10); len(ids) != 2 || ids[0] != 3 || ids[1] != 2 {
		t.Fatalf("onlineIDs %v", ids)
	}
	if ids := p.onlineIDs(now, 1); len(ids) != 1 || ids[0] != 3 {
		t.Fatalf("onlineIDs limit %v", ids)
	}
	if p.lastSeen(1).IsZero() || !p.lastSeen(4).IsZero() {
		t.Fatal("lastSeen")
	}
}
