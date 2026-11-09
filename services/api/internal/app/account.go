package app

import (
	"errors"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/andrij06062003/Wormix/services/api/internal/assets"
	"github.com/andrij06062003/Wormix/services/api/internal/php"
)

// ---- relation loaders (hasOne lazy loads) ----

func (c *Ctx) hasOne(def *Def, fk string, id any) *Model {
	return c.First(def, `where "`+def.Table+`"."`+fk+`" = ? and "`+def.Table+`"."`+fk+`" is not null limit 1`, id)
}

// rel returns a cached relation of m or loads it (Eloquent caches lazy-loaded relations).
func (m *Model) rel(name string, load func() *Model) *Model {
	for _, r := range m.relations {
		if r.name == name {
			x, _ := r.val.(*Model)
			return x
		}
	}
	x := load()
	if x == nil {
		m.SetRelation(name, nil)
	} else {
		m.SetRelation(name, x)
	}
	return x
}

func (c *Ctx) userSocial(u *Model) *Model {
	return u.rel("social_data", func() *Model { return c.hasOne(defSocial, "user_id", u.Get("id")) })
}
func (c *Ctx) userProfile(u *Model) *Model {
	return u.rel("user_profile", func() *Model { return c.hasOne(defProfile, "user_id", u.Get("id")) })
}
func (c *Ctx) userWorm(u *Model) *Model {
	return u.rel("worm_data", func() *Model { return c.hasOne(defWorm, "owner_id", u.Get("id")) })
}
func (c *Ctx) userBattleInfo(u *Model) *Model {
	return u.rel("battle_info", func() *Model { return c.hasOne(defBattleInfo, "user_id", u.Get("id")) })
}
func (c *Ctx) userLoginSeq(u *Model) *Model {
	return u.rel("login_sequence", func() *Model { return c.hasOne(defLoginSeq, "user_id", u.Get("id")) })
}

func must(m *Model) *Model {
	if m == nil {
		php.Fail("Attempt to read property on null")
	}
	return m
}

// ---- AuthController ----

func (c *Ctx) respondWithToken(u *Model) Resp {
	plain, t := c.createToken(u, "auth_token-"+php.ToString(u.Get("id")))
	c.exec(`delete from "personal_access_tokens" where "personal_access_tokens"."tokenable_type" = ? and "personal_access_tokens"."tokenable_id" = ? and "personal_access_tokens"."tokenable_id" is not null and "id" != ?`,
		TokenableType, u.Get("id"), t.Get("id"))
	return jsonResp(200, php.Assoc("auth_token", plain, "id", u.Get("id")))
}

func (c *Ctx) signIn() Resp {
	c.Validate([]R{{"login", "required|string|exists:users,login"}, {"password", "required|string"}})
	u := must(c.First(defUser, `where "login" = ?`, c.JSON("login")))
	if !CheckPassword(c.JSON("password"), u.Get("password")) {
		return jsonResp(403, php.Assoc("message", "Access Denied"))
	}
	return c.respondWithToken(u)
}

func (c *Ctx) signUp() Resp {
	c.Validate([]R{
		{"login", "required|string|unique:users,login"},
		{"password", "required|min:8|string"},
		{"password_confirmation", "required|same:password"},
	})
	u := New(defUser)
	u.Set("password", hashedCast(c.JSON("password")))
	u.Set("login", c.JSON("login"))
	c.Save(u)
	c.userCreated(u)
	return c.respondWithToken(u)
}

// userCreated is UserObserver::created.
func (c *Ctx) userCreated(u *Model) {
	id := u.Get("id")
	level, realMoney := c.app.Config.starter()
	p := New(defProfile)
	p.Set("user_id", id)
	p.Set("money", c.app.Config.StarterMoney)
	p.Set("real_money", realMoney)
	c.Save(p)

	w := New(defWorm)
	w.Set("owner_id", id)
	w.Set("level", level)
	w.Set("hat", int64(0))
	c.Save(w)

	t := New(defTeam)
	t.Set("user_id", id)
	t.Set("teammate_id", id)
	c.Save(t)

	s := New(defSocial)
	s.Set("user_id", id)
	s.Set("first_name", u.Get("login"))
	c.Save(s)

	b := New(defBattleInfo)
	b.Set("user_id", id)
	b.Set("battles_count", int64(10))
	c.Save(b)

	l := New(defLoginSeq)
	l.Set("user_id", id)
	l.Set("last_login", time.Now().UTC().Format("2006-01-02")) // 'datetime:Y-m-d' cast: stored raw
	l.Set("gift_accepted", int64(1))
	c.Save(l)

	for _, wid := range c.Values(`select "id" from "wormix_weapons" where "is_starter" = ?`, int64(1)) {
		uw := New(defUserWeapon)
		uw.Set("owner_id", id)
		uw.Set("weapon_id", wid)
		c.Save(uw)
	}
	c.devStarter(w, level, id)
}

func (c *Ctx) signOut() Resp {
	c.Delete(c.token)
	return jsonResp(200, php.Assoc("message", "Logged out"))
}

// ---- AccountController ----

func (c *Ctx) accountResource(u *Model) Resp {
	id := u.Get("id")
	profile := c.First(defProfile, `where "wormix_user_profiles"."user_id" in (?)`, id)
	u.SetRelation("user_profile", profile)
	u.SetRelation("worm_data", c.First(defWorm, `where "wormix_worms_data"."owner_id" in (?)`, id))
	sd := c.First(defSocial, `where "users_vk_data"."user_id" in (?)`, id)
	if sd != nil {
		// For the profile page: when the name may be changed again (null: now).
		var next any
		if t := nextNameChange(sd); time.Now().UTC().Before(t) {
			next = t.Format(time.RFC3339)
		}
		sd.SetRaw("next_name_change_at", next)
	}
	u.SetRelation("social_data", sd)
	// For the profile page: whether the player has the game open, and when the server last heard from it.
	uid := php.ToInt(id)
	var seen any
	if t := c.app.presence.lastSeen(uid); !t.IsZero() {
		seen = t.UTC().Format(time.RFC3339)
	}
	u.SetRaw("online", c.app.presence.online(uid, time.Now()))
	u.SetRaw("last_seen_at", seen)
	u.SetRelation("battle_info", c.First(defBattleInfo, `where "wormix_users_battle_info"."user_id" in (?)`, id))
	if profile != nil {
		team := c.All(defTeam, `where "wormix_users_teams"."user_id" in (?)`, profile.Get("user_id"))
		for _, t := range team {
			t.SetRelation("teammate", c.First(defWorm, `where "wormix_worms_data"."owner_id" = ?`, t.Get("teammate_id")))
		}
		if team == nil {
			team = []*Model{}
		}
		profile.SetRelation("teammates", team)
	}
	return jsonResp(200, php.Assoc("data", u.ToArray()))
}

func (c *Ctx) getAccount() Resp { return c.accountResource(c.user) }

func (c *Ctx) getAccountInfo() Resp {
	return c.accountResource(c.boundUser)
}

// NameChangeDays: a player may change the name the game shows (the nickname) once in this many days.
const NameChangeDays = 60

// A login: 3 to 32 letters (any alphabet), digits, "_", "-" and ".".
var loginPattern = regexp.MustCompile(`^[\p{L}\p{N}_.\-]{3,32}$`)

// nextNameChange is when the player may change the name again (zero: now).
func nextNameChange(sd *Model) time.Time {
	at, err := time.Parse(time.DateTime, php.ToString(sd.Get("name_changed_at")))
	if err != nil {
		return time.Time{}
	}
	return at.Add(NameChangeDays * 24 * time.Hour)
}

func (c *Ctx) updateAccount() Resp {
	// Players change only their name, login and password here. Money, rubies, rating, the worm and
	// battles change only through the game.
	for _, key := range []string{"user_profile", "worm_data", "battle_info"} {
		if c.JSON(key) != nil {
			return jsonResp(403, php.Assoc("message", "Game data can't be changed here"))
		}
	}
	c.Validate([]R{
		{"social_data.first_name", "string|max:20"},
		{"social_data.last_name", "nullable|string|max:20"},
		{"user.login", "string|max:32"},
		{"user.password", "nullable|string|min:8"},
		{"user.password_confirmation", "nullable|string|min:8|required_unless:user.password,null|same:user.password"},
	})
	u := c.user
	// The login and the password change only with the current password, so a game left open somewhere
	// is not enough to take the account over.
	if c.JSON("user.login") != nil || c.JSON("user.password") != nil {
		if !CheckPassword(php.ToString(c.JSON("current_password")), u.Get("password")) {
			return jsonResp(403, php.Assoc("message", "Wrong current password", "error", "current_password"))
		}
	}
	if c.JSON("social_data") != nil {
		sd := must(c.userSocial(u))
		first, last := php.ToString(c.JSON("social_data.first_name")), php.ToString(c.JSON("social_data.last_name"))
		if first == "" {
			first, last = php.ToString(sd.Get("first_name")), php.ToString(sd.Get("last_name"))
		}
		if first != php.ToString(sd.Get("first_name")) || last != php.ToString(sd.Get("last_name")) {
			// The name (what other players know you by) changes once in NameChangeDays.
			if next := nextNameChange(sd); time.Now().UTC().Before(next) {
				return jsonResp(429, php.Assoc("message", "The name can be changed once in "+strconv.Itoa(NameChangeDays)+" days",
					"error", "name_cooldown", "next_change_at", next.Format(time.RFC3339)))
			}
			sd.Set("first_name", first)
			if last == "" {
				sd.Set("last_name", nil)
			} else {
				sd.Set("last_name", last)
			}
			sd.Set("name_changed_at", time.Now().UTC().Format(time.DateTime))
			c.Save(sd)
		}
	}
	if c.JSON("user") != nil {
		if login := c.JSON("user.login"); login != nil {
			if !loginPattern.MatchString(php.ToString(login)) {
				return jsonResp(422, php.Assoc("message", "The login is 3 to 32 letters, digits, _ - or .", "error", "login_format"))
			}
			if c.Count(`select count(*) as aggregate from "users" where lower("login") = lower(?) and "id" != ?`, login, u.Get("id")) > 0 {
				return jsonResp(422, php.Assoc("message", "User login must be unique", "error", "login_taken"))
			}
			u.Set("login", login)
		}
		if pw := c.JSON("user.password"); pw != nil {
			u.Set("password", hashedCast(pw))
			c.exec(`delete from "personal_access_tokens" where "personal_access_tokens"."tokenable_type" = ? and "personal_access_tokens"."tokenable_id" = ? and "personal_access_tokens"."tokenable_id" is not null and "id" != ?`,
				TokenableType, u.Get("id"), c.token.Get("id"))
		}
		if u.IsDirty("login", "password") {
			c.Save(u)
		}
	}
	return jsonResp(200, php.Assoc("success", true))
}

func (c *Ctx) photoPath(name string) string {
	return filepath.Join(c.app.PhotosDir, "users", name)
}

func (c *Ctx) updateAccountPhoto() Resp {
	c.Validate([]R{{"photo", "required|image|mimes:jpeg,png,jpg|max:4096|dimensions:min_width=100,min_height=100"}},
		M{"photo.required", "Photo is required."},
		M{"photo.image", "Photo is invalid."},
		M{"photo.mimes", "Photo format is invalid."},
		M{"photo.max", "Photo is too large."},
		M{"photo.dimensions", "Photo dimensions invalid.(min: 100x100)"},
	)
	sd := must(c.userSocial(c.user))
	f, _ := c.req.files.Val("photo").(*UploadedFile)
	if f == nil {
		php.Fail("Call to a member function extension() on null")
	}
	name := php.UUID4() + "." + guessExtension(f.Data)
	if old := sd.Get("photo"); old != nil {
		p := c.photoPath(php.ToString(old))
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			os.Remove(p)
		}
	}
	sd.SetRaw("photo", name)
	c.Save(sd)
	if err := os.MkdirAll(filepath.Join(c.app.PhotosDir, "users"), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(c.photoPath(name), f.Data, 0o644); err != nil {
		panic(err)
	}
	return jsonResp(200, php.Assoc("data", sd.ToArray()))
}

func (c *Ctx) startGame() Resp {
	s := c.loadSession(php.ToInt(c.user.Get("id")))
	if s == nil {
		php.Fail("Can't find user")
	}
	return jsonResp(200, php.Assoc("auth_key", s.setAuthKey()))
}

// ---- FileController ----

func (c *Ctx) getPhoto() Resp {
	photo, name := c.params["photo"], c.params["name"]
	var data []byte
	var err error
	switch strings.ToLower(photo) {
	case "bot":
		data, err = fs.ReadFile(assets.FS, "images/bots/"+name)
		if strings.Contains(name, "/") || name == "." || name == ".." {
			err = errors.New("invalid")
		}
	case "users":
		p := c.photoPath(name)
		if st, e := os.Stat(p); e == nil && st.IsDir() {
			php.Fail("File does not exist at path %s.", p)
		}
		data, err = os.ReadFile(p)
	default:
		err = errors.New("default")
	}
	if err != nil {
		data, _ = fs.ReadFile(assets.FS, "images/default.png")
	}
	return Resp{Status: 200, Raw: data, ContentType: mimeType(data), Headers: map[string]string{"Cache-Control": "max-age=120, no-transform, public"}}
}

// ---- VkApiController ----

func (c *Ctx) vkAPI() Resp {
	c.Validate([]R{{"method", "required|string"}},
		M{"method.required", "Method name is required."},
		M{"method.string", "Method name must be a string."})
	switch m, _ := c.Post("method").(string); m {
	case "getUserSettings":
		return jsonResp(200, php.Assoc("response", int64(2371351)))
	case "getAppFriends", "friends.get": // friends.get: the "send a gift to a friend" list; everyone is a friend
		ids := php.NewArr()
		for _, id := range c.Values(`select "id" from "users"`) {
			ids.Append(id)
		}
		return jsonResp(200, php.Assoc("response", ids))
	case "getProfiles":
		return c.vkGetProfiles()
	case "getUserBalance":
		return jsonResp(200, php.Assoc("response", int64(1000)))
	}
	// Everything else is a VK feature that isn't here (wall posts, notes, photo/video uploads...). The PHP API
	// answered with an empty body, which the client can't parse, so it never called back and the game sat on
	// "Saving..." forever. Refuse at once instead, the way VK reports a player pressing "Cancel" (10007): the
	// client then carries on.
	return jsonResp(200, php.Assoc("error", php.Assoc("error_code", int64(10007), "error_msg", "Operation denied by user")))
}

func (c *Ctx) vkProfile(userID, first, last, nickname, photo any) *php.Arr {
	path := "users"
	if nickname == "bot" {
		path = "bot"
	}
	p := ""
	if photo != nil {
		// A path on the game's own address, not c.url(): the API sits behind the game host, which doesn't pass
		// the player's Host on, so c.url() named the API's internal address (http://127.0.0.1:8000), which a
		// player's game can't load (another computer, another origin, http under https).
		p = "/api/photos/" + path + "/" + php.ToString(photo)
	}
	if last == nil {
		last = ""
	}
	return php.Assoc(
		"uid", php.ToString(userID),
		"first_name", first,
		"last_name", last,
		"nickname", "",
		"sex", int64(1),
		"bdate", "23.11.2000",
		"country", int64(1),
		"city", int64(1),
		"timezone", int64(1),
		"photo", p,
		"photo_medium", p,
		"photo_big", p,
		"has_mobile", int64(1),
	)
}

func (c *Ctx) vkGetProfiles() Resp {
	raw := c.Post("uids")
	if _, ok := raw.(*php.Arr); ok {
		php.Fail("json_decode(): Argument #1 ($json) must be of type string, array given")
	}
	dec, _ := php.JSONDecode([]byte(php.ToString(raw)))
	uids, ok := dec.(*php.Arr)
	if !ok {
		// The 1.25.0 client sends the ids as VK does, "1,2,3", not as a JSON array.
		uids = php.NewArr()
		for _, part := range strings.Split(php.ToString(raw), ",") {
			if id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64); err == nil {
				uids.Append(id)
			}
		}
		if uids.Len() == 0 {
			php.Fail("foreach() argument must be of type array|object, %s given", php.TypeName(dec))
		}
	}
	isBots := false
	for _, uid := range uids.Values() {
		if php.Lt(uid, int64(0)) {
			isBots = true
			break
		}
	}
	out := php.NewArr()
	if !isBots {
		for _, sd := range c.All(defSocial, "") {
			out.Append(c.vkProfile(sd.Get("user_id"), sd.Get("first_name"), sd.Get("last_name"), sd.Get("nickname"), sd.Get("photo")))
		}
		return jsonResp(200, php.Assoc("response", out))
	}
	var names []any
	if a, ok := loadGameJSON("names.json").(*php.Arr); ok {
		names = a.Values()
	}
	if len(names) == 0 {
		names = []any{"NaN Bot"}
	}
	bots := assets.BotImages()
	for _, uid := range uids.Values() {
		parts := strings.Split(php.ToString(names[rand.IntN(len(names))]), " ")
		if len(parts) < 2 {
			php.Fail("Undefined array key 1")
		}
		var photo any
		if len(bots) > 0 {
			photo = bots[rand.IntN(len(bots))]
		}
		out.Append(c.vkProfile(uid, parts[1], parts[0], "bot", photo))
	}
	return jsonResp(200, php.Assoc("response", out))
}

func loadGameJSON(name string) any {
	b, err := assets.Game(name)
	if err != nil {
		return nil
	}
	v, _ := php.JSONDecode(b)
	return v
}
