package app

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/andrij06062003/Wormix/services/api/internal/db"
	"github.com/andrij06062003/Wormix/services/api/internal/php"
)

// TokenableType is the morph class Sanctum stored for users.
const TokenableType = `App\Models\User`

// BcryptCost is BCRYPT_ROUNDS from .env (12).
const BcryptCost = 12

func sha256hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func truncate72(p string) []byte {
	b := []byte(p)
	if len(b) > 72 {
		b = b[:72]
	}
	return b
}

// HashPassword is Hash::make (bcrypt, PHP's $2y$ prefix).
func HashPassword(plain string) string {
	h, err := bcrypt.GenerateFromPassword(truncate72(plain), BcryptCost)
	if err != nil {
		panic(err)
	}
	s := string(h)
	if strings.HasPrefix(s, "$2a$") {
		s = "$2y$" + s[4:]
	}
	return s
}

func isBcrypt(h string) bool {
	return len(h) == 60 && strings.HasPrefix(h, "$2y$")
}

// CheckPassword is Hash::check for the bcrypt hasher.
func CheckPassword(plain any, hashed any) bool {
	h, _ := hashed.(string)
	if h == "" {
		return false
	}
	if !isBcrypt(h) {
		php.Fail("This password does not use the Bcrypt algorithm.")
	}
	return bcrypt.CompareHashAndPassword([]byte(h), truncate72(php.ToString(plain))) == nil
}

// hashedCast is the 'hashed' cast on set.
func hashedCast(v any) any {
	if v == nil {
		return nil
	}
	s := php.ToString(v)
	if isBcrypt(s) {
		cost := php.ToInt(s[4:6])
		if cost > BcryptCost {
			php.Fail("Could not verify the hashed value's configuration.")
		}
		return s
	}
	if strings.HasPrefix(s, "$argon2") {
		php.Fail("Could not verify the hashed value's configuration.")
	}
	return HashPassword(s)
}

// bearerToken is Request::bearerToken().
func (c *Ctx) bearerToken() string {
	h := c.r.Header.Get("Authorization")
	pos := strings.LastIndex(strings.ToLower(h), "bearer ")
	if pos < 0 {
		return ""
	}
	h = h[pos+7:]
	if i := strings.IndexByte(h, ','); i >= 0 {
		h = h[:i]
	}
	return h
}

func ctypeDigit(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// authenticate is the Sanctum guard (auth('api')->check()).
func (c *Ctx) authenticate() bool {
	if c.user != nil {
		return true
	}
	tok := c.bearerToken()
	if tok == "" {
		return false
	}
	var t *Model
	if strings.Contains(tok, "|") {
		id, rest, _ := strings.Cut(tok, "|")
		if !ctypeDigit(id) || rest == "" || rest == "0" {
			return false
		}
		t = c.Find(defToken, id)
		if t == nil || subtle.ConstantTimeCompare([]byte(php.ToString(t.Get("token"))), []byte(sha256hex(rest))) != 1 {
			return false
		}
	} else {
		if tok == "0" {
			return false
		}
		t = c.First(defToken, `where "token" = ? limit 1`, sha256hex(tok))
		if t == nil {
			return false
		}
	}
	if exp := t.Get("expires_at"); exp != nil {
		et, err := asDateTime(exp)
		if err != nil {
			php.Fail("%v", err)
		}
		if et.Before(time.Now()) {
			return false
		}
	}
	if php.ToString(t.Get("tokenable_type")) != TokenableType {
		return false
	}
	u := c.Find(defUser, t.Get("tokenable_id"))
	if u == nil {
		return false
	}
	t.Set("last_used_at", time.Now())
	c.Save(t)
	c.user, c.token = u, t
	return true
}

// createToken is HasApiTokens::createToken($name) and returns the plain text token.
func (c *Ctx) createToken(user *Model, name string) (string, *Model) {
	entropy := php.RandomString(40)
	plain := entropy + fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte(entropy)))
	t := New(defToken)
	t.Set("name", name)
	t.Set("token", sha256hex(plain))
	t.Set("abilities", php.List("*"))
	t.Set("expires_at", nil)
	t.Set("tokenable_id", user.Get("id"))
	t.Set("tokenable_type", TokenableType)
	c.Save(t)
	return php.ToString(t.Get("id")) + "|" + plain, t
}

// ---- internal-auth middleware ----

func (c *Ctx) internalAuth() (Resp, bool) {
	deny := jsonResp(403, php.Assoc("message", "Access Denied"))
	key := c.r.Header.Get("X-SESSION-KEY")
	if key == "" || key == "0" {
		return deny, false
	}
	parts := strings.Split(key, ".")
	if len(parts) != 2 {
		return deny, false
	}
	id := php.ToInt(parts[0])
	s := c.loadSession(id)
	if s == nil || php.ToString(s.data.Val("session_key")) != parts[1] {
		return deny, false
	}
	c.req.json.Set("internal_user_id", id)
	return Resp{}, true
}

var _ = db.Now
