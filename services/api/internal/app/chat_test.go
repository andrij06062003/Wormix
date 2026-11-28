package app

import (
	"fmt"
	"strings"
	"testing"
)

// The chat: the room is seen by everybody, a private message only by its two players; the poll returns only
// newer messages; empty, too long and too many messages are refused.
func TestChat(t *testing.T) {
	srv, d, v := newServer(t)
	alice := map[string]string{"Authorization": "Bearer " + v.Users["alice"].Token}
	bob := map[string]string{"Authorization": "Bearer " + v.Users["bob"].Token}
	send := func(h map[string]string, body map[string]any) (int, map[string]any) {
		t.Helper()
		return call(t, srv, "POST", "/api/chat", body, h)
	}
	list := func(h map[string]string, after any) []any {
		t.Helper()
		st, j := call(t, srv, "GET", fmt.Sprintf("/api/chat?after=%v", after), nil, h)
		if st != 200 {
			t.Fatalf("chat list: %d %v", st, j)
		}
		return j["messages"].([]any)
	}

	st, j := send(alice, map[string]any{"text": "  hello   everybody "})
	if st != 200 || j["text"] != "hello everybody" {
		t.Fatalf("room message: %d %v", st, j)
	}
	first := j["id"]
	if st, j := send(alice, map[string]any{"text": "hi bob", "to": 2}); st != 200 || j["to"] != float64(2) {
		t.Fatalf("private message: %d %v", st, j)
	}
	carol := map[string]string{"Authorization": "Bearer " + v.Users["carol"].Token}
	if n := len(list(bob, 0)); n != 2 {
		t.Fatalf("bob sees %d messages, want 2 (the room and his private one)", n)
	}
	if n := len(list(carol, 0)); n != 1 {
		t.Fatalf("carol sees %d messages, want the room's 1", n)
	}
	if n := len(list(alice, first)); n != 1 {
		t.Fatalf("poll after the first message: %d, want 1", n)
	}
	_, all := call(t, srv, "GET", "/api/chat", nil, bob)
	users := all["users"].(map[string]any)
	if users["1"] == nil || users["2"] == nil {
		t.Fatalf("users %v", users)
	}
	if _, err := d.Exec(`insert into chat_messages (from_user_id, to_user_id, text, created_at) values (2, 3, 'secret', datetime('now'))`); err != nil {
		t.Fatal(err)
	}
	for _, m := range list(alice, 0) {
		if m.(map[string]any)["text"] == "secret" {
			t.Fatal("alice sees a private message between others")
		}
	}

	for _, body := range []map[string]any{
		{"text": "   "},
		{"text": strings.Repeat("x", chatMaxLen+1)},
		{"text": "me", "to": 1},
		{"text": "nobody", "to": 999},
	} {
		if st, _ := send(alice, body); st != 422 {
			t.Fatalf("%v: %d, want 422", body, st)
		}
	}
	for i := 0; i < chatBurst-2; i++ {
		if st, j := send(alice, map[string]any{"text": "spam"}); st != 200 {
			t.Fatalf("message %d: %d %v", i, st, j)
		}
	}
	if st, _ := send(alice, map[string]any{"text": "spam"}); st != 429 {
		t.Fatalf("message over the limit: %d, want 429", st)
	}
	if st, _ := call(t, srv, "GET", "/api/chat", nil, nil); st != 403 {
		t.Fatalf("without a token: %d, want 403", st)
	}
}
