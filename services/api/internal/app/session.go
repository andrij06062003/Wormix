package app

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/andrij06062003/Wormix/services/api/internal/php"
)

// The game sessions (auth key, session key, TCP session) and the achievement sessions
// live in Laravel's database cache store (table `cache`, prefix ""), serialized with
// PHP serialize(). They are kept there in the same format so that sessions survive a
// restart and a switch between the PHP and the Go API in either direction.

const cacheForever = 315360000

func (c *Ctx) cacheGet(key string) (any, error) {
	row := c.First(&Def{Table: "cache", Key: "key"}, `where "key" = ? limit 1`, key)
	if row == nil {
		return nil, nil
	}
	if time.Now().Unix() >= php.ToInt(row.Get("expiration")) {
		c.exec(`delete from "cache" where "key" = ? and "expiration" <= ?`, key, time.Now().Unix())
		return nil, nil
	}
	v, err := php.Unserialize(php.ToString(row.Get("value")))
	if err != nil {
		return nil, &php.Error{Msg: "unserialize(): Error at offset"}
	}
	return v, nil
}

func (c *Ctx) cachePut(key string, v any, seconds int64) {
	c.exec(`insert into "cache" ("expiration", "key", "value") values (?, ?, ?) on conflict ("key") do update set "expiration" = "excluded"."expiration", "key" = "excluded"."key", "value" = "excluded"."value"`,
		time.Now().Unix()+seconds, key, php.Serialize(v))
}

func (c *Ctx) cacheForget(key string) {
	c.exec(`delete from "cache" where "key" = ?`, key)
}

// userSession is App\Modules\UserSession.
type userSession struct {
	c    *Ctx
	user *Model
	data *php.Arr
}

func (s *userSession) key() string { return "user_session_" + php.ToString(s.user.Get("id")) }

func (s *userSession) blank() *php.Arr {
	return php.Assoc("id", s.user.Get("id"), "auth_key", "", "session_key", "", "tcp_session", "", "logged_in", false)
}

// loadSession is `new UserSession($id)`; nil when the user does not exist.
func (c *Ctx) loadSession(id int64) *userSession {
	u := c.First(defUser, `where "id" = ?`, id)
	if u == nil {
		return nil
	}
	s := &userSession{c: c, user: u}
	raw, err := c.cacheGet(s.key())
	if err != nil {
		c.cacheForget(s.key())
		raw = nil
	}
	if raw == nil {
		s.data = s.blank()
		return s
	}
	str, ok := raw.(string)
	if !ok {
		php.Fail("json_decode(): Argument #1 ($json) must be of type string, %s given", php.TypeName(raw))
	}
	dec, _ := php.JSONDecode([]byte(str))
	a, ok := dec.(*php.Arr)
	if !ok || a.Val("id") == nil || a.Val("auth_key") == nil || a.Val("session_key") == nil || a.Val("tcp_session") == nil {
		s.data = s.blank()
		return s
	}
	s.data = a
	return s
}

func (s *userSession) save() {
	s.c.cachePut(s.key(), php.MustJSON(s.data), cacheForever)
}

func randomHash(prefix string) string {
	h := sha256.Sum256(append([]byte(prefix), php.RandomBytes(32)...))
	return hex.EncodeToString(h[:])
}

func (s *userSession) setAuthKey() string {
	s.data.Set("logged_in", false)
	s.data.Set("auth_key", randomHash(php.ToString(s.user.Get("id"))))
	s.data.Set("session_key", "")
	s.save()
	return php.ToString(s.data.Val("auth_key"))
}

func (s *userSession) authKey() string    { return php.ToString(s.data.Val("auth_key")) }
func (s *userSession) sessionKey() string { return php.ToString(s.data.Val("session_key")) }
func (s *userSession) tcpSession() string { return php.ToString(s.data.Val("tcp_session")) }
func (s *userSession) isLoggedIn() bool   { return php.ToBool(s.data.Val("logged_in")) }

func (s *userSession) setSessionKey() string {
	s.data.Set("session_key", randomHash(s.authKey()))
	s.save()
	return s.sessionKey()
}

func (s *userSession) loggedIn() {
	s.data.Set("logged_in", true)
	s.save()
}

func (s *userSession) setTcpSession(id string) {
	s.data.Set("tcp_session", id)
	s.save()
}
