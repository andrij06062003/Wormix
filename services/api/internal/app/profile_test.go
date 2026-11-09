package app

import (
	"strings"
	"testing"
	"time"
)

// The profile page: the name changes once in NameChangeDays; the login and the password only with the
// current password; the login is unique whatever its case.
func TestProfileChanges(t *testing.T) {
	srv, d, v := newServer(t)
	alice := v.Users["alice"]
	h := map[string]string{"Authorization": "Bearer " + alice.Token}
	post := func(body map[string]any) (int, map[string]any) {
		t.Helper()
		return call(t, srv, "POST", "/api/account", body, h)
	}
	name := func(first string) map[string]any {
		return map[string]any{"social_data": map[string]any{"first_name": first}}
	}

	// The name: the first change at once, the next one only after NameChangeDays.
	if st, j := post(name("Alice")); st != 200 {
		t.Fatalf("first name change: %d %v", st, j)
	}
	if st, j := post(name("Alice")); st != 200 {
		t.Fatalf("the same name again: %d %v", st, j)
	}
	st, j := post(name("Alicia"))
	if st != 429 || j["error"] != "name_cooldown" || j["next_change_at"] == nil {
		t.Fatalf("second change at once: %d %v, want 429", st, j)
	}
	_, acc := call(t, srv, "GET", "/api/account", nil, h)
	sd := acc["data"].(map[string]any)["social_data"].(map[string]any)
	if sd["first_name"] != "Alice" || sd["next_name_change_at"] == nil {
		t.Fatalf("account after the change: %v", sd)
	}
	old := time.Now().UTC().Add(-(NameChangeDays + 1) * 24 * time.Hour).Format(time.DateTime)
	d.Exec(`update users_vk_data set name_changed_at = ? where user_id = 1`, old)
	if st, j := post(map[string]any{"social_data": map[string]any{"first_name": "Alicia", "last_name": "Smith"}}); st != 200 {
		t.Fatalf("change after %d days: %d %v", NameChangeDays, st, j)
	}
	var first, last string
	d.QueryRow(`select first_name, last_name from users_vk_data where user_id = 1`).Scan(&first, &last)
	if first != "Alicia" || last != "Smith" {
		t.Fatalf("name %q %q", first, last)
	}
	if st, _ := post(name(strings.Repeat("x", 21))); st != 422 {
		t.Fatalf("21-letter name: %d, want 422", st)
	}

	// The login.
	login := func(l, current string) (int, map[string]any) {
		body := map[string]any{"user": map[string]any{"login": l}}
		if current != "" {
			body["current_password"] = current
		}
		return post(body)
	}
	if st, _ := login("alice2", ""); st != 403 {
		t.Fatalf("login without the password: %d, want 403", st)
	}
	if st, _ := login("alice2", "wrong-password"); st != 403 {
		t.Fatalf("login with a wrong password: %d, want 403", st)
	}
	if st, j := login("BOB", alice.Password); st != 422 || j["error"] != "login_taken" {
		t.Fatalf("taken login (another case): %d %v", st, j)
	}
	if st, j := login("a b", alice.Password); st != 422 || j["error"] != "login_format" {
		t.Fatalf("login with a space: %d %v", st, j)
	}
	if st, j := login("Аліса_2", alice.Password); st != 200 {
		t.Fatalf("login change: %d %v", st, j)
	}

	// The password.
	pw := func(current string) int {
		st, _ := post(map[string]any{"current_password": current,
			"user": map[string]any{"password": "new-secret-1", "password_confirmation": "new-secret-1"}})
		return st
	}
	if st := pw("wrong-password"); st != 403 {
		t.Fatalf("password with a wrong current one: %d, want 403", st)
	}
	if st := pw(alice.Password); st != 200 {
		t.Fatalf("password change: %d", st)
	}
	if st, j := call(t, srv, "POST", "/api/account/sign_in", map[string]any{"login": "Аліса_2", "password": "new-secret-1"}, nil); st != 200 {
		t.Fatalf("sign in with the new login and password: %d %v", st, j)
	}
}

// The game loads avatars from the address the page came from, not the API's internal one.
func TestProfilePhotoURLIsRelative(t *testing.T) {
	srv, d, _ := newServer(t)
	d.Exec(`update users_vk_data set photo = 'abc.jpg' where user_id = 1`)
	_, j := call(t, srv, "POST", "/api", map[string]any{"method": "getProfiles", "uids": "1"}, nil)
	for _, p := range j["response"].([]any) {
		p := p.(map[string]any)
		if p["uid"] == "1" && p["photo_medium"] != "/api/photos/users/abc.jpg" {
			t.Fatalf("photo url %v", p["photo_medium"])
		}
	}
}
