package app

import (
	"fmt"
	"testing"
)

// account/game gives the page the other players, the 1.61.7 client's friends: not the player.
func TestGameFriends(t *testing.T) {
	srv, _, v := newServer(t)
	alice, bob := v.Users["alice"], v.Users["bob"]
	st, j := call(t, srv, "POST", "/api/account/game", nil, map[string]string{"Authorization": "Bearer " + alice.Token})
	if st != 200 || j["auth_key"] == nil {
		t.Fatalf("account/game: %d %v", st, j)
	}
	friends, _ := j["friends"].([]any)
	has := func(id int64) bool {
		for _, f := range friends {
			if f == fmt.Sprint(id) {
				return true
			}
		}
		return false
	}
	if !has(bob.ID) || has(alice.ID) {
		t.Fatalf("friends %v: want bob (%d), not alice (%d)", friends, bob.ID, alice.ID)
	}
}
