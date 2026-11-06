package app

import (
	"errors"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
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
	u.SetRelation("social_data", c.First(defSocial, `where "users_vk_data"."user_id" in (?)`, id))
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

func (c *Ctx) updateAccount() Resp {
	c.Validate([]R{
		{"social_data.first_name", "string|max:100"},
		{"social_data.last_name", "string|max:100"},
		{"user_profile.money", "integer|min:0"},
		{"user_profile.real_money", "integer|min:0"},
		{"user_profile.rating", "integer|min:0"},
		{"user_profile.reaction_rate", "integer|min:0"},
		{"worm_data.level", "integer|min:1|max:30"},
		{"worm_data.armor", "integer|min:0|max:60"},
		{"worm_data.attack", "integer|min:0|max:60"},
		{"worm_data.race", "integer|exists:wormix_races,race_id"},
		{"battle_info.battles_count", "integer|min:0|max:999"},
		{"user.login", "string|max:100"},
		{"user.password", "nullable|string|min:8"},
		{"user.password_confirmation", "nullable|string|min:8|required_unless:user.password,null|same:user.password"},
	})
	u := c.user
	if c.JSON("social_data") != nil {
		sd := must(c.userSocial(u))
		sd.Fill(c.JSON("social_data"), "first_name", "last_name", "nickname")
		c.Save(sd)
	}
	if c.JSON("user_profile") != nil {
		p := must(c.userProfile(u))
		p.Fill(c.JSON("user_profile"), "money", "real_money", "rating", "reaction_rate")
		c.Save(p)
	}
	if c.JSON("worm_data") != nil {
		wd := must(c.userWorm(u))
		_, hatID := raceAndHat(php.IntArg(wd.Get("hat")))
		race := c.JSON("worm_data.race")
		wd.Fill(c.JSON("worm_data"), "level", "armor", "attack")
		armor, attack := stripData(
			php.ToInt(php.Div(php.Add(wd.Get("armor"), wd.Get("attack")), int64(2))),
			php.IntArg(wd.Get("level")), php.IntArg(wd.Get("armor")), php.IntArg(wd.Get("attack")))
		wd.Set("armor", armor)
		wd.Set("attack", attack)
		wd.Set("experience", int64(0))
		if race == nil {
			php.Fail("getHatByRaceAndHatIds(): Argument #2 ($raceId) must be of type int, null given")
		}
		wd.Set("hat", hatByRaceAndHat(hatID, php.IntArg(race)))
		c.saveWorm(wd)
		return jsonResp(200, php.Assoc("worm_data", wd.ToArray()))
	}
	if c.JSON("user") != nil {
		if login := c.JSON("user.login"); login != nil {
			if c.Count(`select count(*) as aggregate from "users" where "login" = ? and "id" != ?`, login, u.Get("id")) > 0 {
				return jsonResp(422, php.Assoc("message", "User login must be unique"))
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
	if bc := c.JSON("battle_info.battles_count"); bc != nil {
		bi := must(c.userBattleInfo(u))
		bi.Set("battles_count", bc)
		c.Save(bi)
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
	case "getAppFriends":
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
	return Resp{Status: 200, Raw: []byte{}, ContentType: "text/html; charset=UTF-8"}
}

func (c *Ctx) vkProfile(userID, first, last, nickname, photo any) *php.Arr {
	path := "users"
	if nickname == "bot" {
		path = "bot"
	}
	p := ""
	if photo != nil {
		p = c.url("api/photos/" + path + "/" + php.ToString(photo))
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
		php.Fail("foreach() argument must be of type array|object, %s given", php.TypeName(dec))
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
