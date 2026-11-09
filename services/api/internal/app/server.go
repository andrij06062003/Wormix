// Package app is the HTTP API: the Laravel routes, middlewares, controllers and
// model logic of wormix-api ported to Go.
package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/andrij06062003/Wormix/services/api/internal/db"
	"github.com/andrij06062003/Wormix/services/api/internal/php"
)

// App is the API server.
type App struct {
	DB        *db.DB
	PhotosDir string // user photos (were resources/images/users)
	Config    Config
	routes    []route
}

// Ctx is one request.
type Ctx struct {
	ctx    context.Context
	tx     *sql.Tx
	app    *App
	r      *http.Request
	req    *parsedRequest
	params map[string]string
	user   *Model
	token  *Model

	boundUser *Model // route model binding {user}
}

// Resp is a handler result.
type Resp struct {
	Status      int
	Body        any // PHP value encoded as JSON
	Raw         []byte
	ContentType string
	Headers     map[string]string
}

// httpError aborts a request with a JSON message (404 model binding, 403 auth, ...).
type httpError struct {
	status int
	body   any
}

func abort(status int, msg string) {
	panic(&httpError{status, php.Assoc("message", msg)})
}

type handler func(c *Ctx) Resp

type route struct {
	method   string
	segs     []string
	h        handler
	internal bool // internal-request middleware
	iauth    bool // internal-auth middleware
	sanctum  bool // auth:sanctum (ApiAuth) middleware
	noTx     bool
}

// New builds the app with all routes registered.
func NewApp(d *db.DB, photosDir string) *App {
	a := &App{DB: d, PhotosDir: photosDir, Config: LoadConfig()}
	a.registerRoutes()
	return a
}

func (a *App) add(method, path string, h handler, opts ...string) {
	r := route{method: method, h: h}
	path = strings.Trim(path, "/")
	if path != "" {
		r.segs = strings.Split(path, "/")
	}
	for _, o := range opts {
		switch o {
		case "internal":
			r.internal = true
		case "iauth":
			r.internal, r.iauth = true, true
		case "sanctum":
			r.sanctum = true
		case "notx":
			r.noTx = true
		}
	}
	a.routes = append(a.routes, r)
}

func (a *App) registerRoutes() {
	// routes/api.php (prefix api)
	a.add("POST", "api", (*Ctx).vkAPI)
	a.add("GET", "api/photos/{photo}/{name}", (*Ctx).getPhoto, "notx")

	// routes/internal/internal_api.php (prefix api/internal, middleware internal-request)
	a.add("POST", "api/internal/login", (*Ctx).internalLogin, "internal")
	a.add("POST", "api/internal/achievements/login", (*Ctx).achLogin, "internal")
	a.add("POST", "api/internal/achievements/get", (*Ctx).achGet, "internal")
	a.add("POST", "api/internal/achievements/increase", (*Ctx).achIncrease, "internal")
	a.add("POST", "api/internal/achievements/choose_bonus", (*Ctx).achChooseBonus, "internal")
	for _, x := range []struct {
		p string
		h handler
	}{
		{"account/buy/items", (*Ctx).buyItems},
		{"account/buy/race", (*Ctx).changeRace},
		{"account/buy/battles", (*Ctx).buyBattle},
		{"account/buy/mission", (*Ctx).unlockMission},
		{"account/buy/reaction", (*Ctx).buyReaction},
		{"account/reset/parameters", (*Ctx).resetParameters},
		{"account/reset/account", (*Ctx).resetAccount},
		{"account/distribute_points", (*Ctx).distributePoints},
		{"account/select_stuff", (*Ctx).selectStuff},
		{"game/get_arena", (*Ctx).getArena},
		{"game/start_battle", (*Ctx).startBattle},
		{"game/end_battle", (*Ctx).endBattle},
		{"info/rating", (*Ctx).getRating},
		{"info/pumped_reaction", (*Ctx).getPumpedReaction},
		{"team/add", (*Ctx).addTeammate},
		{"team/remove", (*Ctx).deleteTeammate},
		{"team/reorder", (*Ctx).reorderTeam},
		{"team/toggle", (*Ctx).toggleTeammate},
		{"house/search", (*Ctx).searchTheHouse},
		{"house/pump_reaction", (*Ctx).pumpReaction},
		{"house/pump_reactions", (*Ctx).pumpReactions},
		{"craft/upgrade_weapon", (*Ctx).upgradeWeapon},
		{"craft/assemble_equip", (*Ctx).assembleEquip},
		{"craft/downgrade_weapon", (*Ctx).downgradeWeapon},
		{"pvp/profile", (*Ctx).pvpProfile},
		{"pvp/start_battle", (*Ctx).pvpStartBattle},
		{"pvp/end_battle", (*Ctx).pvpEndBattle},
		{"pvp/coop_state", (*Ctx).pvpCoopState},
	} {
		a.add("POST", "api/internal/"+x.p, x.h, "iauth")
	}

	// routes/public/account_api.php (prefix api/account)
	a.add("POST", "api/account/sign_in", (*Ctx).signIn)
	a.add("POST", "api/account/sign_up", (*Ctx).signUp)
	a.add("POST", "api/account/sign_out", (*Ctx).signOut, "sanctum")
	a.add("GET", "api/account", (*Ctx).getAccount, "sanctum")
	a.add("POST", "api/account", (*Ctx).updateAccount, "sanctum")
	a.add("POST", "api/account/update_photo", (*Ctx).updateAccountPhoto, "sanctum")
	a.add("GET", "api/account/info/{user}", (*Ctx).getAccountInfo, "sanctum")
	a.add("POST", "api/account/game", (*Ctx).startGame, "sanctum")
}

func (rt *route) match(segs []string) (map[string]string, bool) {
	if len(segs) != len(rt.segs) {
		return nil, false
	}
	params := map[string]string{}
	for i, s := range rt.segs {
		if strings.HasPrefix(s, "{") {
			if segs[i] == "" {
				return nil, false
			}
			params[s[1:len(s)-1]] = segs[i]
			continue
		}
		if s != segs[i] {
			return nil, false
		}
	}
	return params, true
}

func jsonResp(status int, body any) Resp { return Resp{Status: status, Body: body} }

// ServeHTTP dispatches like Laravel's router + exception handler.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req := parseRequest(r)
	rawPath := r.URL.EscapedPath()
	path, err := url.PathUnescape(rawPath)
	if err != nil {
		path = r.URL.Path
	}
	path = strings.Trim(path, "/")
	var segs []string
	if path != "" {
		segs = strings.Split(path, "/")
	}

	var found *route
	var params map[string]string
	methodMismatch := false
	for i := range a.routes {
		rt := &a.routes[i]
		p, ok := rt.match(segs)
		if !ok {
			continue
		}
		if rt.method == req.method {
			found, params = rt, p
			break
		}
		methodMismatch = true
	}
	if found == nil {
		if methodMismatch {
			a.write(w, r, jsonResp(405, php.Assoc("message", "Method Not Allowed")))
			return
		}
		a.write(w, r, jsonResp(404, php.Assoc("message", "Not Found")))
		return
	}

	c := &Ctx{ctx: r.Context(), app: a, r: r, req: req, params: params}
	resp := a.run(c, found)
	a.write(w, r, resp)
}

type commitResp struct{ resp Resp }

func (e *commitResp) Error() string { return "response" }

func (a *App) run(c *Ctx, rt *route) (resp Resp) {
	defer func() {
		if p := recover(); p != nil {
			resp = a.renderPanic(c, p)
		}
	}()
	if rt.noTx {
		return rt.h(c)
	}
	// One write transaction per request. Handled outcomes (including 4xx answers and
	// validation errors, which happen after e.g. Sanctum's last_used_at update) commit;
	// a PHP-level crash (500) rolls back.
	err := a.DB.Tx(c.ctx, func(tx *sql.Tx) error {
		c.tx = tx
		resp = func() (out Resp) {
			defer func() {
				if p := recover(); p != nil {
					switch t := p.(type) {
					case *ValidationError:
						out = jsonResp(422, php.Assoc("message", t.Message, "errors", t.Errors))
					case *httpError:
						out = jsonResp(t.status, t.body)
					default:
						panic(p)
					}
				}
			}()
			return a.pipeline(c, rt)
		}()
		return nil
	})
	if err != nil {
		panic(err)
	}
	return resp
}

func (a *App) pipeline(c *Ctx, rt *route) Resp {
	if rt.internal {
		tcp := c.r.Header.Get("X-TCP-SESSION")
		if tcp == "" {
			return jsonResp(404, php.Assoc("message", "Not found"))
		}
		c.req.json.Set("tcp_session", tcp)
	}
	if rt.iauth {
		if r, ok := c.internalAuth(); !ok {
			return r
		}
	}
	// SubstituteBindings (api group) runs before the route's auth middleware.
	if v, ok := c.params["user"]; ok {
		c.boundUser = c.First(defUser, `where "users"."id" = ? limit 1`, v)
		if c.boundUser == nil {
			return jsonResp(404, php.Assoc("message", "Not Found"))
		}
	}
	if rt.sanctum {
		if !c.authenticate() {
			return jsonResp(403, php.Assoc("message", "Access Denied"))
		}
	}
	return rt.h(c)
}

func (a *App) renderPanic(c *Ctx, p any) Resp {
	var pe *php.Error
	if e, ok := p.(error); ok && errors.As(e, &pe) {
		log.Printf("500 %s %s: %s", c.r.Method, c.r.URL.Path, pe.Msg)
	} else {
		log.Printf("500 %s %s: %v\n%s", c.r.Method, c.r.URL.Path, p, debug.Stack())
	}
	if expectsJSON(c.r) {
		return jsonResp(500, php.Assoc("message", "Server Error"))
	}
	return Resp{Status: 500, Raw: []byte("<!DOCTYPE html>\n<html lang=\"en\"><head><meta charset=\"utf-8\"><title>Server Error</title></head><body>500 | Server Error</body></html>\n"), ContentType: "text/html; charset=UTF-8"}
}

func expectsJSON(r *http.Request) bool {
	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" && r.Header.Get("X-PJAX") == "" {
		return true
	}
	acc := r.Header.Get("Accept")
	if acc == "" {
		return false
	}
	first := strings.TrimSpace(strings.Split(acc, ",")[0])
	first, _, _ = strings.Cut(first, ";")
	return strings.Contains(first, "/json") || strings.Contains(first, "+json")
}

func (a *App) write(w http.ResponseWriter, r *http.Request, resp Resp) {
	for k, v := range resp.Headers {
		w.Header().Set(k, v)
	}
	var body []byte
	if resp.Raw != nil {
		body = resp.Raw
		ct := resp.ContentType
		if ct == "" {
			ct = "text/html; charset=UTF-8"
		}
		w.Header().Set("Content-Type", ct)
	} else {
		b, err := php.JSONEncode(resp.Body)
		if err != nil {
			resp.Status = 500
			b = []byte(`{"message":"Server Error"}`)
		}
		body = b
		w.Header().Set("Content-Type", "application/json")
	}
	if w.Header().Get("Cache-Control") == "" {
		w.Header().Set("Cache-Control", "no-cache, private")
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	if resp.Status == 0 {
		resp.Status = 200
	}
	w.WriteHeader(resp.Status)
	if r.Method != http.MethodHead {
		w.Write(body)
	}
}

// ---- request accessors ----

// JSON is $request->json($key) (dot notation).
func (c *Ctx) JSON(key string) any {
	v, _ := php.GetPath(c.req.json, key)
	return v
}

// HasJSON reports whether the json bag has the (dotted) key.
func (c *Ctx) HasJSON(key string) bool {
	_, ok := php.GetPath(c.req.json, key)
	return ok
}

// Post is $request->post($key): the request bag (json for JSON requests).
func (c *Ctx) Post(key string) any {
	src := c.req.form
	if c.req.isJSON {
		src = c.req.json
	}
	v, _ := php.GetPath(src, key)
	return v
}

// UserID is $request->json('internal_user_id').
func (c *Ctx) UserID() any { return c.JSON("internal_user_id") }

// url() helper: scheme://host[:port]/path from the request.
func (c *Ctx) url(path string) string {
	scheme := "http"
	if c.r.TLS != nil {
		scheme = "https"
	}
	host := strings.ToLower(c.r.Host)
	h, port := host, ""
	if strings.HasPrefix(host, "[") {
		if i := strings.LastIndex(host, "]"); i >= 0 {
			h = host[:i+1]
			port = strings.TrimPrefix(host[i+1:], ":")
		}
	} else if i := strings.LastIndexByte(host, ':'); i >= 0 {
		h, port = host[:i], host[i+1:]
	}
	if port == "" || (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		host = h
	} else {
		host = h + ":" + port
	}
	return scheme + "://" + host + "/" + strings.Trim(path, "/")
}

var _ = fmt.Sprintf
