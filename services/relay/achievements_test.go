package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// signed appends the md5 of body + key, as the clients sign their commands.
func signed(body []byte, key string) []byte {
	h := md5.Sum(append(body[:len(body):len(body)], key...))
	return append(body, h[:]...)
}

// achAPI answers the achievements API with fixed results.
func achAPI(t *testing.T) *server {
	t.Helper()
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/login"):
			w.Write([]byte(`{"ok":true,"session_id":"s","login_time":7}`))
		case strings.HasSuffix(r.URL.Path, "/get"):
			w.Write([]byte(`{"profile_id":"p","indexes":[1],"values":[2],"invested":3}`))
		case strings.HasSuffix(r.URL.Path, "/increase"):
			w.Write([]byte(`{"indexes":[],"values":[],"threshold_indexes":[],"threshold_old_values":[],"time_sequence":5,"invested":0}`))
		case strings.HasSuffix(r.URL.Path, "/choose_bonus"):
			w.Write([]byte(`{"result":0,"item_id":9}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(api.Close)
	s, _ := testServer(t, api.URL)
	return s
}

// reply splits one written command into its id and body, and checks its hash with key.
func reply(t *testing.T, out []byte, key string) (int, []byte) {
	t.Helper()
	if len(out) < 7 {
		t.Fatalf("short reply %x", out)
	}
	id := int(binary.BigEndian.Uint16(out[1:]) & 0x7FFF)
	n := int(binary.BigEndian.Uint32(out[3:]) & 0x3FFFFFFF)
	if len(out) != 7+n {
		t.Fatalf("reply length %d, header says %d", len(out)-7, n)
	}
	body := out[7:]
	if key != "" {
		if len(body) < 16 {
			t.Fatalf("unsigned reply %x", out)
		}
		h := achHash(body[:len(body)-16], key)
		if !bytes.Equal(h[:], body[len(body)-16:]) {
			t.Fatalf("reply %d not signed with %q", id, key)
		}
		body = body[:len(body)-16]
	}
	return id, body
}

func TestAchievementsLogin125(t *testing.T) {
	s := achAPI(t)
	for _, tc := range []struct {
		extra []byte
		key   string
	}{{nil, achSecretKey}, {[]byte{1}, achSecretKey125}} {
		var wr writer
		for _, v := range []string{"app", "vk", "1", "auth"} {
			wr.utf(v)
		}
		wr.Write(tc.extra)
		var out bytes.Buffer
		if err := s.handleAchievement(context.Background(), cmdAchieveLogin, wr.Bytes(), nil, &out); err != nil {
			t.Fatal(err)
		}
		if id, _ := reply(t, out.Bytes(), tc.key); id != cmdAchieveLoginSuccess {
			t.Fatalf("id %d", id)
		}
	}
}

func TestGetAchievements125(t *testing.T) {
	s := achAPI(t)
	sid := "s"
	var wr writer
	wr.utf("p")
	wr.u8(0) // investedAwardPoints
	var out bytes.Buffer
	if err := s.handleAchievement(context.Background(), cmdGetAchievements, signed(wr.Bytes(), achSecretKey125), &sid, &out); err != nil {
		t.Fatal(err)
	}
	id, body := reply(t, out.Bytes(), achSecretKey125)
	var want writer
	want.utf("p")
	want.byteArray([]int{1})
	want.u16Array([]int{2})
	want.byteArray(nil)
	want.u8(3)
	if id != cmdGetAchievementsResult || !bytes.Equal(body, want.Bytes()) {
		t.Fatalf("got %d %x, want %x", id, body, want.Bytes())
	}

	// 1.05.0: no boolAchievements, its own salt.
	var old writer
	old.utf("p")
	out.Reset()
	if err := s.handleAchievement(context.Background(), cmdGetAchievements, signed(old.Bytes(), achSecretKey), &sid, &out); err != nil {
		t.Fatal(err)
	}
	if _, body := reply(t, out.Bytes(), achSecretKey); len(body) != len(want.Bytes())-2 {
		t.Fatalf("1.05.0 reply %x", body)
	}

	// A wrong hash is refused.
	if err := s.handleAchievement(context.Background(), cmdGetAchievements, signed(old.Bytes(), "x"), &sid, &out); err == nil {
		t.Fatal("bad hash accepted")
	}
}

func TestIncreaseAchievements125(t *testing.T) {
	s := achAPI(t)
	sid := "s"
	var wr writer
	wr.utf("s")
	wr.u16(1)
	wr.u8(4)
	wr.u16(1)
	wr.i16(1)
	wr.u16(2) // boolAchievements
	wr.i32(10)
	wr.i32(11)
	wr.u32(100)
	var out bytes.Buffer
	if err := s.handleAchievement(context.Background(), cmdIncreaseAchievements, signed(wr.Bytes(), achSecretKey125), &sid, &out); err != nil {
		t.Fatal(err)
	}
	id, body := reply(t, out.Bytes(), achSecretKey125)
	want := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5, 0}
	if id != cmdIncreaseAchievementResult || !bytes.Equal(body, want) {
		t.Fatalf("got %d %x", id, body)
	}
}

func TestBonusItems125(t *testing.T) {
	s := achAPI(t)
	sid := "s"
	var out bytes.Buffer
	if err := s.handleAchievement(context.Background(), cmdChooseBonusItem, signed([]byte{0, 0, 0, 9}, achSecretKey125), &sid, &out); err != nil {
		t.Fatal(err)
	}
	if id, body := reply(t, out.Bytes(), achSecretKey125); id != cmdChooseBonusItemResult || !bytes.Equal(body, []byte{0, 0, 0, 0, 0, 9}) {
		t.Fatalf("got %d %x", id, body)
	}
	out.Reset()
	if err := s.handleAchievement(context.Background(), cmdBuyResetBonusItems, signed([]byte{0, 0, 0, 3}, achSecretKey125), &sid, &out); err != nil {
		t.Fatal(err)
	}
	if id, body := reply(t, out.Bytes(), achSecretKey125); id != cmdBuyResetBonusItemsResult || !bytes.Equal(body, []byte{0, 1, 0, 0, 0, 3}) {
		t.Fatalf("got %d %x", id, body)
	}
}
