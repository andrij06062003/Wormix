"""Differential test: sends the same request sequence to the reference (PHP) API and
the Go API, each running on its own copy of the same database, and compares status
codes and JSON bodies (after normalizing tokens, keys, timestamps and host names).
Random outcomes (bots, search-the-house, crate reagents) are compared by shape.

usage: python difftest.py http://ref:8000 http://go:8000 vars.json
exit code 0 when no differences.
"""
import copy
import os
import json
import sys
import uuid

from harness import norm, png, send, shape, shape_eq

REF, GO, VARS = sys.argv[1], sys.argv[2], sys.argv[3]
SIDES = {"ref": REF, "go": GO}
HOSTS = [REF.split("//")[1], GO.split("//")[1]]
base_vars = json.load(open(VARS))
V = {s: copy.deepcopy(base_vars) for s in SIDES}
for s in SIDES:
    V[s]["tcp"] = {}

failures = []
count = 0


def r(x, v):
    return x(v) if callable(x) else x


NO_EXPECT = os.environ.get("DIFF_NO_EXPECT") == "1"  # dev mode: only compare ref vs go


def step(name, method, path, mode="exact", capture=None, expect=None, go_json=None, **kw):
    """mode: exact | shape | status | raw | go (an intended change: only the Go answer is checked, against go_json)"""
    global count
    count += 1
    res = {}
    for side, base in SIDES.items():
        v = V[side]
        args = {k: r(x, v) for k, x in kw.items()}
        res[side] = send(base, method, r(path, v), **args)
        if capture:
            capture(v, res[side][3])
    (sa, ca, ba, ja), (sb, cb, bb, jb) = res["ref"], res["go"]
    problems = []
    if mode == "go":
        if sb != 200 or jb != go_json:
            problems.append(f"go answered {sb} {jb!r}, want 200 {go_json!r}")
    elif sa != sb:
        problems.append(f"status ref={sa} go={sb}")
    if expect is not None and sa != expect and not NO_EXPECT:
        problems.append(f"ref status {sa} != expected {expect}")
    if mode == "raw":
        if ba != bb:
            problems.append(f"body differs ({len(ba)} vs {len(bb)} bytes)")
        if ca.split(";")[0] != cb.split(";")[0]:
            problems.append(f"content-type ref={ca} go={cb}")
    elif mode not in ("status", "go"):
        if ("json" in ca) != ("json" in cb):
            problems.append(f"content-type ref={ca} go={cb}")
        if ja is None and jb is None:
            if ba != bb and sa != 500:
                problems.append(f"non-json body differs: {ba[:100]!r} vs {bb[:100]!r}")
        else:
            na, nb = norm(ja, HOSTS), norm(jb, HOSTS)
            if mode == "shape":
                na, nb = shape(na), shape(nb)
            if (na != nb) if mode != "shape" else not shape_eq(na, nb):
                problems.append("json differs:\n      ref=" + json.dumps(na, ensure_ascii=False)[:1500]
                                + "\n      go =" + json.dumps(nb, ensure_ascii=False)[:1500])
    tag = "ok  " if not problems else "FAIL"
    print(f"{tag} [{count}] {name}: {sa}")
    for p in problems:
        print("     ", p)
    if problems:
        failures.append(name)
    return res


def u(login):
    return lambda v: v["users"][login]


def bearer(login):
    return lambda v: {"Authorization": "Bearer " + v["users"][login]["token"]}


def internal(login, extra=None):
    def h(v):
        x = {"X-TCP-SESSION": v["users"][login].get("tcp", "x"), "X-SESSION-KEY": v["users"][login].get("session_key", "")}
        x.update(extra or {})
        return x
    return h


def cap(login, key, path):
    def f(v, j):
        x = j
        for p in path:
            x = x[p] if x is not None and p in x else None
        if x is not None:
            v["users"].setdefault(login, {})[key] = x
    return f


ACH = {"X-TCP-SESSION": "achievements", "Accept": "application/json"}

# ------------------------------------------------------------------ routing / errors
step("unknown path", "GET", "/api/nothing", expect=404)
step("wrong method", "GET", "/api/account/sign_in", expect=405)
step("root 404", "GET", "/", expect=404)
step("trailing slash", "POST", "/api/account/sign_in/", json_body={})
step("internal without tcp header", "POST", "/api/internal/login", json_body={}, expect=404)
step("internal auth: no key", "POST", "/api/internal/account/buy/battles", json_body={"MoneyType": 1},
     headers={"X-TCP-SESSION": "x"}, expect=403)
step("internal auth: malformed key", "POST", "/api/internal/account/buy/battles", json_body={"MoneyType": 1},
     headers={"X-TCP-SESSION": "x", "X-SESSION-KEY": "1.2.3"}, expect=403)
step("internal auth: wrong key", "POST", "/api/internal/account/buy/battles", json_body={"MoneyType": 1},
     headers={"X-TCP-SESSION": "x", "X-SESSION-KEY": "1.deadbeef"}, expect=403)
step("internal auth: unknown user", "POST", "/api/internal/account/buy/battles", json_body={"MoneyType": 1},
     headers={"X-TCP-SESSION": "x", "X-SESSION-KEY": "999.abc"}, expect=403)

# ------------------------------------------------------------------ public account API
step("sign_up empty", "POST", "/api/account/sign_up", json_body={}, expect=422)
step("sign_up short pw", "POST", "/api/account/sign_up", json_body={"login": "x", "password": "123", "password_confirmation": "123"}, expect=422)
step("sign_up mismatch", "POST", "/api/account/sign_up", json_body={"login": "x", "password": "12345678", "password_confirmation": "12345679"}, expect=422)
step("sign_up duplicate", "POST", "/api/account/sign_up", json_body={"login": "alice", "password": "12345678", "password_confirmation": "12345678"}, expect=422)
step("sign_up types", "POST", "/api/account/sign_up", json_body={"login": 5, "password": ["a"], "password_confirmation": None}, expect=422)
step("sign_up form", "POST", "/api/account/sign_up", form={"login": "  ", "password": "x"}, expect=422)
step("sign_up eve (trimmed login)", "POST", "/api/account/sign_up",
     json_body={"login": "  eve ", "password": " pass word ", "password_confirmation": " pass word "},
     capture=lambda v, j: v["users"].update({"eve": {"id": j["id"], "token": j["auth_token"], "password": " pass word "}}))
step("sign_up unicode", "POST", "/api/account/sign_up",
     json_body={"login": "Вася/<&>", "password": "пароль123", "password_confirmation": "пароль123"},
     capture=lambda v, j: v["users"].update({"vasya": {"id": j["id"], "token": j["auth_token"]}}))
step("sign_in eve", "POST", "/api/account/sign_in", json_body={"login": "eve", "password": " pass word "},
     capture=cap("eve", "token", ["auth_token"]))
step("sign_in old password (phase A)", "POST", "/api/account/sign_in", json_body=lambda v: {"login": "bob", "password": v["users"]["bob"]["password"]},
     capture=cap("bob", "token", ["auth_token"]))
step("sign_in wrong password", "POST", "/api/account/sign_in", json_body={"login": "bob", "password": "nope"}, expect=403)
step("sign_in unknown login", "POST", "/api/account/sign_in", json_body={"login": "nobody", "password": "x"}, expect=422)
step("sign_in missing", "POST", "/api/account/sign_in", json_body={"login": None}, expect=422)
step("account: old token from phase A (alice)", "GET", "/api/account", headers=bearer("alice"), expect=200)
step("account: no token", "GET", "/api/account", expect=403)
step("account: bad token", "GET", "/api/account", headers={"Authorization": "Bearer 1|xxxxxxxx"}, expect=403)
step("account: token without id", "GET", "/api/account", headers={"Authorization": "Bearer abc"}, expect=403)
step("account: eve", "GET", "/api/account", headers=bearer("eve"))
step("account: vasya (unicode)", "GET", "/api/account", headers=bearer("vasya"))
step("account info 2", "GET", "/api/account/info/2", headers=bearer("eve"))
step("account info 999", "GET", "/api/account/info/999", headers=bearer("eve"), expect=404)
step("account info abc no auth", "GET", "/api/account/info/abc", expect=404)
step("account info no auth", "GET", "/api/account/info/2", expect=403)

step("update: social+profile", "POST", "/api/account", headers=bearer("eve"),
     json_body={"social_data": {"first_name": " Eve ", "last_name": "Smith", "photo": "hack.png"},
                "user_profile": {"money": 100000, "real_money": "500", "rating": 30, "reaction_rate": 4, "invested_award_points": 9}})
step("update: battles", "POST", "/api/account", headers=bearer("eve"), json_body={"battle_info": {"battles_count": 5}})
step("update: invalid", "POST", "/api/account", headers=bearer("eve"),
     json_body={"social_data": {"first_name": "x" * 101}, "user_profile": {"money": -1, "rating": "abc"},
                "worm_data": {"level": 31, "armor": "a", "race": 99}, "battle_info": {"battles_count": 1000},
                "user": {"login": ["a"], "password": "short", "password_confirmation": "other"}}, expect=422)
step("update: worm without race -> 500", "POST", "/api/account", headers=bearer("eve"),
     json_body={"worm_data": {"level": 10, "armor": 5, "attack": 5}}, mode="status", expect=500)
step("update: worm data level 10 race 2", "POST", "/api/account", headers=bearer("eve"),
     json_body={"worm_data": {"level": 10, "armor": 5, "attack": 7, "race": 2, "experience": 99, "hat": 5}})
step("update: worm level 30 strings", "POST", "/api/account", headers=bearer("vasya"),
     json_body={"worm_data": {"level": "30", "armor": "0", "attack": "1", "race": "0"}})
step("update: login taken", "POST", "/api/account", headers=bearer("eve"), json_body={"user": {"login": "alice"}}, expect=422)
step("update: password w/o confirmation", "POST", "/api/account", headers=bearer("eve"), json_body={"user": {"password": "newpassword"}}, expect=422)
step("update: login+password", "POST", "/api/account", headers=bearer("eve"),
     json_body={"user": {"login": "eve2", "password": "newpassword", "password_confirmation": "newpassword"}})
step("sign_in eve2 new password", "POST", "/api/account/sign_in", json_body={"login": "eve2", "password": "newpassword"},
     capture=cap("eve", "token", ["auth_token"]))
step("account: eve after updates", "GET", "/api/account", headers=bearer("eve"))

PNG_OK, PNG_SMALL = png(120, 110), png(50, 50)
step("photo: missing", "POST", "/api/account/update_photo", headers=bearer("eve"), form={"x": "1"}, files={}, expect=422)
step("photo: text file", "POST", "/api/account/update_photo", headers=bearer("eve"), files={"photo": ("a.txt", "text/plain", b"hello")}, expect=422)
step("photo: too small", "POST", "/api/account/update_photo", headers=bearer("eve"), files={"photo": ("a.png", "image/png", PNG_SMALL)}, expect=422)
step("photo: string field", "POST", "/api/account/update_photo", headers=bearer("eve"), form={"photo": "abc"}, files={}, expect=422)
step("photo: ok", "POST", "/api/account/update_photo", headers=bearer("eve"), files={"photo": ("me.png", "image/png", PNG_OK)},
     capture=cap("eve", "photo", ["data", "photo"]))
step("photo: replace", "POST", "/api/account/update_photo", headers=bearer("eve"), files={"photo": ("me2.png", "image/png", PNG_OK)},
     capture=cap("eve", "photo", ["data", "photo"]))
step("photo: get uploaded", "GET", lambda v: "/api/photos/users/" + v["users"]["eve"]["photo"], mode="raw")
step("photo: get bot", "GET", "/api/photos/bot/1.jpeg", mode="raw")
step("photo: get bot upper", "GET", "/api/photos/BOT/2.png", mode="raw")
step("photo: missing -> default", "GET", "/api/photos/users/none.png", mode="raw")
step("photo: other storage -> default", "GET", "/api/photos/x/1.jpeg", mode="raw")

# VK API emulation (the Flash client posts form data)
step("vk: getUserSettings", "POST", "/api/", form={"method": "getUserSettings", "api_id": "1"})
step("vk: getAppFriends", "POST", "/api/", form={"method": "getAppFriends"})
step("vk: getProfiles users", "POST", "/api/", form={"method": "getProfiles", "uids": "[1,2]"})
step("vk: getProfiles bots", "POST", "/api/", form={"method": "getProfiles", "uids": "[-1,-2,3]"}, mode="shape")
step("vk: getProfiles no uids", "POST", "/api/", form={"method": "getProfiles"}, mode="status", expect=500)
step("vk: getUserBalance json", "POST", "/api", json_body={"method": "getUserBalance"})
# Intended change from the PHP API (which answered VK methods it doesn't know with an empty body, so the
# client waited for an answer forever): they are refused at once, as if the player had cancelled.
VK_DENIED = {"error": {"error_code": 10007, "error_msg": "Operation denied by user"}}
for vk_method in ("wall.post", "notes.add", "photos.getUploadServer", "wall.getPhotoUploadServer", "video.save"):
    step(f"vk: {vk_method} refused", "POST", "/api", mode="go", go_json=VK_DENIED, form={"method": vk_method})
step("vk: getAppFriends (again, for friends.get)", "POST", "/api/", form={"method": "getAppFriends"},
     capture=lambda v, j: v.__setitem__("app_friends", j))
# friends.get (the "send a gift to a friend" list) answers like getAppFriends: every registered player.
step("vk: friends.get = everyone", "POST", "/api/", mode="go", go_json=V["go"]["app_friends"], form={"method": "friends.get"})
step("vk: no method", "POST", "/api", form={"x": "1"}, expect=422)
step("vk: method array", "POST", "/api", json_body={"method": [1]}, expect=422)
# The method is read from the body only (as in PHP), so this is an unknown method: refused.
step("vk: query method", "POST", "/api?method=getUserSettings", mode="go", go_json=VK_DENIED, form={})

# ------------------------------------------------------------------ game login (internal)
for login in ["eve", "dave", "carol", "frank"]:
    step(f"game auth key {login}", "POST", "/api/account/game", headers=bearer(login) if login == "eve" else
         (lambda L: (lambda v: {"Authorization": "Bearer " + v["users"][L]["token"]}))(login),
         capture=cap(login, "auth_key", ["auth_key"]), mode="shape")
step("login: invalid", "POST", "/api/internal/login", json_body={"Id": "x", "tcp_session": "nope"}, headers={"X-TCP-SESSION": "not-a-uuid"}, expect=422)
step("login: wrong auth key", "POST", "/api/internal/login", json_body={"Id": 4, "AuthKey": "nope"}, headers={"X-TCP-SESSION": str(uuid.uuid4())})
step("login: unknown user", "POST", "/api/internal/login", json_body={"Id": 999, "AuthKey": "nope"}, headers={"X-TCP-SESSION": str(uuid.uuid4())})
step("login: already logged in (alice, phase A)", "POST", "/api/internal/login",
     json_body=lambda v: {"Id": 1, "AuthKey": v["users"]["alice"]["auth_key"]}, headers={"X-TCP-SESSION": str(uuid.uuid4())})
for login, uid in [("eve", None), ("dave", 4), ("carol", 3), ("frank", 5)]:
    tcp = str(uuid.uuid4())
    for s in SIDES:
        V[s]["users"][login]["tcp"] = tcp
    step(f"login: {login}", "POST", "/api/internal/login",
         json_body=(lambda L: lambda v: {"Id": v["users"][L]["id"], "ReferrerId": 0, "AuthKey": v["users"][L]["auth_key"], "Ids": [], "SocialCode": 0})(login),
         headers={"X-TCP-SESSION": tcp}, capture=cap(login, "session_key", ["data", "SessionKey"]))

# Phase-A session keys still valid (cache table compatibility)
step("phase A session key (bob)", "POST", "/api/internal/pvp/profile", json_body={}, headers=internal("bob"))
step("phase A session key (alice) buy battles", "POST", "/api/internal/account/buy/battles", json_body={"MoneyType": 1}, headers=internal("alice"))

E = internal("eve")
step("arena: locked (frank)", "POST", "/api/internal/game/get_arena", json_body={"ReturnUsersProfiles": "1"}, headers=internal("frank"))
step("start: no battles", "POST", "/api/internal/game/start_battle", json_body={"MissionId": 0}, headers=internal("frank"), expect=422)
# ------------------------------------------------------------------ account / shop
step("distribute: invalid", "POST", "/api/internal/account/distribute_points", json_body={"Armor": -1, "Attack": "x"}, headers=E, expect=422)
step("distribute: too many", "POST", "/api/internal/account/distribute_points", json_body={"Armor": 30, "Attack": 30}, headers=E)
step("distribute: ok", "POST", "/api/internal/account/distribute_points", json_body={"Armor": 1, "Attack": 1}, headers=internal("dave"))
step("buy items: invalid", "POST", "/api/internal/account/buy/items", json_body={"ShopItems": [{"Id": 99999, "Count": "x"}, {}, 5]}, headers=E, expect=422)
step("buy items: not array", "POST", "/api/internal/account/buy/items", json_body={"ShopItems": "x"}, headers=E, expect=422)
step("buy items: money + rubies", "POST", "/api/internal/account/buy/items",
     json_body={"ShopItems": [{"Id": 6, "Count": 5, "MoneyType": 1}, {"Id": 1003, "Count": -1, "MoneyType": 0}, {"Id": 7, "Count": 2, "MoneyType": 1}]}, headers=E)
step("buy items: again (stack)", "POST", "/api/internal/account/buy/items",
     json_body={"ShopItems": [{"Id": 6, "Count": 2, "MoneyType": 1}]}, headers=E)
step("buy items: hidden", "POST", "/api/internal/account/buy/items", json_body={"ShopItems": [{"Id": 1, "Count": 1, "MoneyType": 1}]}, headers=E)
step("buy items: -1 on finite", "POST", "/api/internal/account/buy/items", json_body={"ShopItems": [{"Id": 6, "Count": -1, "MoneyType": 1}]}, headers=E)
step("buy items: not enough", "POST", "/api/internal/account/buy/items", json_body={"ShopItems": [{"Id": 6, "Count": 100000, "MoneyType": 1}]}, headers=internal("dave"))
step("buy items: zero price", "POST", "/api/internal/account/buy/items", json_body={"ShopItems": [{"Id": 6, "Count": 1, "MoneyType": 0}]}, headers=E)
step("select stuff: owned hat", "POST", "/api/internal/account/select_stuff", json_body={"StuffId": 1003}, headers=E)
step("select stuff: not owned", "POST", "/api/internal/account/select_stuff", json_body={"StuffId": 1025}, headers=E)
step("select stuff: invalid", "POST", "/api/internal/account/select_stuff", json_body={"StuffId": 5}, headers=E, expect=422)
# artifacts (1.25.0 stage 1): shop, wardrobe (select_stuff 2001+ / 2000 = take off)
step("artifact: buy 2001 (money)", "POST", "/api/internal/account/buy/items", json_body={"ShopItems": [{"Id": 2001, "Count": -1, "MoneyType": 1}]}, headers=E)
step("artifact: buy weapon + 2002 (rubies)", "POST", "/api/internal/account/buy/items",
     json_body={"ShopItems": [{"Id": 6, "Count": 1, "MoneyType": 1}, {"Id": 2002, "Count": -1, "MoneyType": 0}]}, headers=E)
step("artifact: hidden boss drop", "POST", "/api/internal/account/buy/items", json_body={"ShopItems": [{"Id": 2007, "Count": -1, "MoneyType": 1}]}, headers=E)
step("artifact: profile (worn 2002)", "POST", "/api/internal/pvp/profile", json_body={}, headers=E)
step("artifact: select owned 2001", "POST", "/api/internal/account/select_stuff", json_body={"StuffId": 2001}, headers=E)
step("artifact: select not owned 2010", "POST", "/api/internal/account/select_stuff", json_body={"StuffId": 2010}, headers=E)
step("artifact: select string id", "POST", "/api/internal/account/select_stuff", json_body={"StuffId": "2002"}, headers=E)
step("artifact: take off (2000)", "POST", "/api/internal/account/select_stuff", json_body={"StuffId": 2000}, headers=E)
step("artifact: unknown 2999", "POST", "/api/internal/account/select_stuff", json_body={"StuffId": 2999}, headers=E, expect=422)
step("artifact: unknown 1500 + not int", "POST", "/api/internal/account/select_stuff", json_body={"StuffId": 1500.5}, headers=E, expect=422)
step("artifact: hat still works", "POST", "/api/internal/account/select_stuff", json_body={"StuffId": 1003}, headers=E)
step("artifact: profile after", "POST", "/api/internal/pvp/profile", json_body={}, headers=E)
step("artifact: frank keeps his (negative balance shown as 0)", "POST", "/api/internal/pvp/profile", json_body={}, headers=internal("frank"))
step("race: same", "POST", "/api/internal/account/buy/race", json_body={"RaceId": 2, "MoneyType": 1}, headers=E)
step("race: too low level", "POST", "/api/internal/account/buy/race", json_body={"RaceId": 6, "MoneyType": 1}, headers=E)
step("race: buy rubies", "POST", "/api/internal/account/buy/race", json_body={"RaceId": 3, "MoneyType": 0}, headers=E)
step("race: invalid", "POST", "/api/internal/account/buy/race", json_body={"RaceId": 1, "MoneyType": 2}, headers=E, expect=422)
step("battles: full", "POST", "/api/internal/account/buy/battles", json_body={"MoneyType": 1}, headers=E)
step("battles: buy frank rubies", "POST", "/api/internal/account/buy/battles", json_body={"MoneyType": 0}, headers=internal("frank"))
step("reaction: 4", "POST", "/api/internal/account/buy/reaction", json_body={"ReactionRateCount": 4}, headers=E)
step("reaction: 9", "POST", "/api/internal/account/buy/reaction", json_body={"ReactionRateCount": 9}, headers=E)
step("reaction: 200", "POST", "/api/internal/account/buy/reaction", json_body={"ReactionRateCount": 200}, headers=E, expect=422)
step("reaction: poor dave", "POST", "/api/internal/account/buy/reaction", json_body={"ReactionRateCount": 120}, headers=internal("dave"))
step("mission unlock: too high level", "POST", "/api/internal/account/buy/mission", json_body={"MissionId": 7}, headers=internal("dave"))
step("mission unlock: eve 1", "POST", "/api/internal/account/buy/mission", json_body={"MissionId": 1}, headers=E)
step("mission unlock: invalid", "POST", "/api/internal/account/buy/mission", json_body={"MissionId": 999}, headers=E, expect=422)

# ------------------------------------------------------------------ arena
step("arena: no profiles", "POST", "/api/internal/game/get_arena", json_body={"ReturnUsersProfiles": False}, headers=E)
step("arena: bots", "POST", "/api/internal/game/get_arena", json_body={"ReturnUsersProfiles": True}, headers=E, mode="shape")
step("arena: invalid", "POST", "/api/internal/game/get_arena", json_body={"ReturnUsersProfiles": "yes"}, headers=E, expect=422)


def battle(name, login, mission, result_delta, typ=0, exp=5, items=None, reagents=None, shape_start=True):
    step(f"{name}: start", "POST", "/api/internal/game/start_battle", json_body={"MissionId": mission}, headers=internal(login),
         capture=cap(login, "battle", ["data", "BattleId"]), mode="shape" if shape_start else "exact")
    step(f"{name}: end", "POST", "/api/internal/game/end_battle",
         json_body=lambda v: {"Result": v["users"][login].get("battle", 0) + result_delta, "Type": typ, "ExpBonus": exp,
                              "BattleId": v["users"][login].get("battle", 0), "MissionId": mission,
                              "Items": items or [], "Signature": [1, 2], "BanType": 0, "BanNote": "",
                              "CollectedReagents": reagents or []}, headers=internal(login))


battle("battle win", "dave", 0, 1, items=[{"Id": 6, "Count": 1}, {"Id": 1, "Count": 1}])
battle("battle draw high", "dave", 0, 0, typ=1, exp=10)
battle("battle lose", "dave", 0, -1)
battle("lesson -1 win", "carol", -1, 1, shape_start=False)
battle("lesson -2 win", "carol", -2, 1, shape_start=False)
battle("boss 1 win", "eve", 1, 1)
battle("boss 1 again (twice run)", "eve", 1, 1)
battle("win low with bad reagent", "eve", 0, 1, typ=2, reagents=[999])
step("start: invalid mission", "POST", "/api/internal/game/start_battle", json_body={"MissionId": 9}, headers=E, expect=422)
step("start: level mismatch", "POST", "/api/internal/game/start_battle", json_body={"MissionId": 2}, headers=internal("dave"), expect=422)
step("end: wrong battle id", "POST", "/api/internal/game/end_battle",
     json_body={"Result": 5, "Type": 0, "ExpBonus": 0, "BattleId": 5, "MissionId": 0, "Items": [], "CollectedReagents": []}, headers=E)
step("end: invalid", "POST", "/api/internal/game/end_battle", json_body={"Type": 7, "ExpBonus": 11}, headers=E, expect=422)
for i in range(5):
    battle(f"drain {i}", "dave", 0, -1)
step("arena: dave after drain", "POST", "/api/internal/game/get_arena", json_body={"ReturnUsersProfiles": False}, headers=internal("dave"))

# ------------------------------------------------------------------ house / info / team
step("pump: self", "POST", "/api/internal/house/pump_reaction", json_body={"FriendId": 6}, headers=E)
step("pump: eve->dave", "POST", "/api/internal/house/pump_reaction", json_body={"FriendId": 4}, headers=E)
step("pump: again", "POST", "/api/internal/house/pump_reaction", json_body={"FriendId": 4}, headers=E)
step("pump: invalid", "POST", "/api/internal/house/pump_reaction", json_body={"FriendId": 999}, headers=E, expect=422)
step("pumps: dave back", "POST", "/api/internal/house/pump_reactions", json_body={"FriendsIds": [6, 1, 2]}, headers=internal("dave"))
step("pumps: invalid", "POST", "/api/internal/house/pump_reactions", json_body={"FriendsIds": [6, "x", 999]}, headers=internal("dave"), expect=422)
step("who pumped dave", "POST", "/api/internal/info/pumped_reaction", json_body={"TodayOnly": False}, headers=internal("dave"))
step("who pumped bob", "POST", "/api/internal/info/pumped_reaction", json_body={}, headers=internal("bob"))
step("rating", "POST", "/api/internal/info/rating", json_body={}, headers=E)
step("team: add (level 1 limit)", "POST", "/api/internal/team/add", json_body={"ProfileId": 2, "MoneyType": 1}, headers=internal("dave"))
step("team: add eve<-alice", "POST", "/api/internal/team/add", json_body={"ProfileId": 1, "MoneyType": 1}, headers=E)
step("team: add eve<-bob rubies", "POST", "/api/internal/team/add", json_body={"ProfileId": 2, "MoneyType": 0}, headers=E)
step("team: add duplicate", "POST", "/api/internal/team/add", json_body={"ProfileId": 2, "MoneyType": 0}, headers=E)
step("team: reorder", "POST", "/api/internal/team/reorder", json_body={"ReorderedWormGroup": [2, 6, 1]}, headers=E)
step("team: reorder unknown", "POST", "/api/internal/team/reorder", json_body={"ReorderedWormGroup": [3]}, headers=E)
step("team: reorder too many", "POST", "/api/internal/team/reorder", json_body={"ReorderedWormGroup": [1, 2, 3, 4, 5]}, headers=E, expect=422)
step("pvp profile eve (team)", "POST", "/api/internal/pvp/profile", json_body={}, headers=E)
step("team: remove self", "POST", "/api/internal/team/remove", json_body={"ProfileId": 6}, headers=E, expect=422)
step("team: remove bob", "POST", "/api/internal/team/remove", json_body={"ProfileId": 2}, headers=E)

# ------------------------------------------------------------------ craft
step("craft: missing prev weapon", "POST", "/api/internal/craft/upgrade_weapon", json_body={"RecipeId": 1}, headers=internal("dave"))
step("give eve level+rubies", "POST", "/api/account", headers=bearer("eve"), json_body={"user_profile": {"real_money": 100000}})
step("buy grenade launcher inf", "POST", "/api/internal/account/buy/items", json_body={"ShopItems": [{"Id": 5, "Count": -1, "MoneyType": 0}]}, headers=E)
step("craft: upgrade 1", "POST", "/api/internal/craft/upgrade_weapon", json_body={"RecipeId": 1}, headers=E)
step("craft: upgrade 1 again", "POST", "/api/internal/craft/upgrade_weapon", json_body={"RecipeId": 1}, headers=E)
step("craft: upgrade 2", "POST", "/api/internal/craft/upgrade_weapon", json_body={"RecipeId": 2}, headers=E)
step("craft: downgrade 2", "POST", "/api/internal/craft/downgrade_weapon", json_body={"RecipeId": 2}, headers=E)
step("craft: downgrade not owned", "POST", "/api/internal/craft/downgrade_weapon", json_body={"RecipeId": 3}, headers=E)
step("craft: invalid", "POST", "/api/internal/craft/upgrade_weapon", json_body={"RecipeId": 9999}, headers=E, expect=422)

# ------------------------------------------------------------------ pvp
step("pvp: start friend battle", "POST", "/api/internal/pvp/start_battle", json_body={"BattleId": 777, "Wager": 0, "Reagents": []}, headers=E)
step("pvp: player end_battle (items)", "POST", "/api/internal/game/end_battle",
     json_body={"Result": 1, "Type": 3, "ExpBonus": 0, "BattleId": 777, "MissionId": 0, "Items": [{"Id": 6, "Count": 2}], "CollectedReagents": [1]}, headers=E)
step("pvp: end", "POST", "/api/internal/pvp/end_battle", json_body={"BattleId": 777, "Result": 1}, headers=E)
step("pvp: end again ignored", "POST", "/api/internal/pvp/end_battle", json_body={"BattleId": 777, "Result": 1}, headers=E)
step("pvp: wager with reagents", "POST", "/api/internal/pvp/start_battle", json_body={"BattleId": 778, "Wager": 300, "Reagents": [1, 2]}, headers=E)
step("pvp: items with reagent", "POST", "/api/internal/game/end_battle",
     json_body={"Result": 1, "Type": 3, "ExpBonus": 0, "BattleId": 778, "MissionId": 0, "Items": [], "CollectedReagents": [2, 5]}, headers=E)
step("pvp: win wager", "POST", "/api/internal/pvp/end_battle", json_body={"BattleId": 778, "Result": 1}, headers=E)
step("pvp: wager random reagents", "POST", "/api/internal/pvp/start_battle", json_body={"BattleId": 779, "Wager": 50}, headers=internal("dave"), mode="shape")
step("pvp: lose", "POST", "/api/internal/pvp/end_battle", json_body={"BattleId": 779, "Result": -1}, headers=internal("dave"))
step("pvp: not enough money", "POST", "/api/internal/pvp/start_battle", json_body={"BattleId": 780, "Wager": 99999999}, headers=internal("dave"), expect=422)
step("pvp: invalid", "POST", "/api/internal/pvp/start_battle", json_body={"BattleId": 0, "Wager": -1, "Reagents": ["x"]}, headers=E, expect=422)
step("pvp: draw ignored (other battle)", "POST", "/api/internal/pvp/end_battle", json_body={"BattleId": 1, "Result": 0}, headers=E)

# ------------------------------------------------------------------ achievements
step("ach: login bad key", "POST", "/api/internal/achievements/login", json_body={"id": 6, "auth_key": "x"}, headers=ACH)
step("ach: login unknown", "POST", "/api/internal/achievements/login", json_body={"id": 999, "auth_key": "x"}, headers=ACH)
step("ach: login eve", "POST", "/api/internal/achievements/login", json_body=lambda v: {"id": 6, "auth_key": v["users"]["eve"]["auth_key"]}, headers=ACH,
     capture=lambda v, j: v["users"]["eve"].update({"ach": j["session_id"], "ach_t": j["login_time"]}))
step("ach: get bad session", "POST", "/api/internal/achievements/get", json_body={"session_id": "zz", "profile_id": "6"}, headers=ACH, expect=401)
step("ach: get eve (empty)", "POST", "/api/internal/achievements/get", json_body=lambda v: {"session_id": v["users"]["eve"]["ach"], "profile_id": "6"}, headers=ACH)
step("ach: get alice via phase-A session", "POST", "/api/internal/achievements/get", json_body=lambda v: {"session_id": v["users"]["alice"]["ach_session"], "profile_id": "1"}, headers=ACH)
inc = lambda dt, idx, rises: (lambda v: {"session_id": v["users"]["eve"]["ach"], "time_scale": v["users"]["eve"]["ach_t"] + dt, "indexes": idx, "rises": rises})
step("ach: increase", "POST", "/api/internal/achievements/increase", json_body=inc(10, [0, 0, 1, 100, 200, 5], [30, 30, 1000, 3, 5, -1]), headers=ACH)
step("ach: increase resend", "POST", "/api/internal/achievements/increase", json_body=inc(10, [0], [999]), headers=ACH)
step("ach: increase old seq", "POST", "/api/internal/achievements/increase", json_body=inc(5, [0], [999]), headers=ACH)
step("ach: increase more", "POST", "/api/internal/achievements/increase", json_body=inc(20, [1, 2, 3, 4], [5000, 70000, 600, 1200]), headers=ACH)
step("ach: get eve", "POST", "/api/internal/achievements/get", json_body=lambda v: {"session_id": v["users"]["eve"]["ach"], "profile_id": "6"}, headers=ACH)
step("ach: choose unknown item", "POST", "/api/internal/achievements/choose_bonus", json_body=lambda v: {"session_id": v["users"]["eve"]["ach"], "item_id": 5}, headers=ACH)
step("ach: choose level 0", "POST", "/api/internal/achievements/choose_bonus", json_body=lambda v: {"session_id": v["users"]["eve"]["ach"], "item_id": 62}, headers=ACH)
step("ach: choose level 1", "POST", "/api/internal/achievements/choose_bonus", json_body=lambda v: {"session_id": v["users"]["eve"]["ach"], "item_id": 64}, headers=ACH)
step("ach: choose again", "POST", "/api/internal/achievements/choose_bonus", json_body=lambda v: {"session_id": v["users"]["eve"]["ach"], "item_id": 62}, headers=ACH)
step("ach: choose level 3", "POST", "/api/internal/achievements/choose_bonus", json_body=lambda v: {"session_id": v["users"]["eve"]["ach"], "item_id": 1034}, headers=ACH)

# ------------------------------------------------------------------ resets (last)
step("reset params: invalid", "POST", "/api/internal/account/reset/parameters", json_body={"MoneyType": 5}, headers=E, expect=422)
step("reset params: rubies", "POST", "/api/internal/account/reset/parameters", json_body={"MoneyType": 0}, headers=E)
step("reset params: poor", "POST", "/api/internal/account/reset/parameters", json_body={"MoneyType": 1}, headers=internal("frank"))
step("enter account data via login of bob after /game", "POST", "/api/account/game", headers=bearer("bob"),
     capture=cap("bob", "auth_key", ["auth_key"]), mode="shape")
for s in SIDES:
    V[s]["users"]["bob"]["tcp"] = "11111111-2222-4333-8444-555555555555"
step("login bob (EnterAccount with teams/pumps)", "POST", "/api/internal/login",
     json_body=lambda v: {"Id": 2, "AuthKey": v["users"]["bob"]["auth_key"]}, headers={"X-TCP-SESSION": "11111111-2222-4333-8444-555555555555"},
     capture=cap("bob", "session_key", ["data", "SessionKey"]))
step("reset account bob", "POST", "/api/internal/account/reset/account", json_body={"ConfirmCode": ""}, headers=internal("bob"))
step("bob session gone", "POST", "/api/internal/pvp/profile", json_body={}, headers=internal("bob"), expect=403)
step("account bob after reset", "GET", "/api/account", headers=bearer("bob"))
step("sign_out eve", "POST", "/api/account/sign_out", headers=bearer("eve"))
step("account after sign_out", "GET", "/api/account", headers=bearer("eve"), expect=403)
step("vk: getProfiles users (final)", "POST", "/api/", form={"method": "getProfiles", "uids": "[1]"})
step("HEAD account", "HEAD", "/api/account", mode="status")
# carol's house search has a random reward: keep it last so it doesn't leak into other responses
step("search: self", "POST", "/api/internal/house/search", json_body={"SessionKey": "", "FriendId": 3, "KeyNum": 0}, headers=internal("carol"))
step("search: carol->alice", "POST", "/api/internal/house/search", json_body={"FriendId": 1}, headers=internal("carol"), mode="shape")
step("search: again same day", "POST", "/api/internal/house/search", json_body={"FriendId": 1}, headers=internal("carol"))
step("search: string id", "POST", "/api/internal/house/search", json_body={"FriendId": "1"}, headers=internal("carol"))

print(f"\n{count} steps, {len(failures)} failed")
if failures:
    print("FAILED:", failures)
sys.exit(1 if failures else 0)
