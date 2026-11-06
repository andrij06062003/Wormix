"""Phase A: build a "used" database with the REFERENCE (PHP) API only.

Creates accounts, logs them into the game, plays a bit through the internal API, and
writes the credentials it obtained (passwords, Sanctum tokens, game session keys,
achievement sessions) to vars.json so the Go API can be checked against them later.

usage: python setup_phase_a.py http://ref:8000 out/vars.json
"""
import json
import sys
import uuid

from harness import send

base, out = sys.argv[1], sys.argv[2]
V = {"users": {}}


def must(cond, msg):
    if not cond:
        print("SETUP FAILED:", msg)
        sys.exit(1)


for i, login in enumerate(["alice", "bob", "carol", "dave", "frank"], start=1):
    pw = f"secret-{login}-1"
    st, _, _, j = send(base, "POST", "/api/account/sign_up", {"login": login, "password": pw, "password_confirmation": pw})
    must(st == 200 and j["id"] == i, f"sign_up {login}: {st} {j}")
    u = {"id": i, "password": pw, "token": j["auth_token"]}
    st, _, _, j = send(base, "POST", "/api/account/game", headers={"Authorization": "Bearer " + u["token"]})
    must(st == 200, "game")
    u["auth_key"] = j["auth_key"]
    V["users"][login] = u

# A second token for alice: sign_in deletes older tokens, so the token kept is the sign_in one.
a = V["users"]["alice"]
st, _, _, j = send(base, "POST", "/api/account/sign_in", {"login": "alice", "password": a["password"]})
must(st == 200, "sign_in")
a["token"] = j["auth_token"]

# Game logins for alice, bob and carol (session keys end up in the cache table).
for login in ["alice", "bob", "carol"]:
    u = V["users"][login]
    tcp = str(uuid.uuid4())
    st, _, _, j = send(base, "POST", "/api/internal/login",
                       {"Id": u["id"], "ReferrerId": 0, "AuthKey": u["auth_key"], "Ids": [], "SocialCode": 0},
                       headers={"X-TCP-SESSION": tcp})
    must(st == 200 and j["type"] == "EnterAccount", f"internal login {login}: {j}")
    u["session_key"] = j["data"]["SessionKey"]
    u["tcp"] = tcp

H = lambda u: {"X-TCP-SESSION": u["tcp"], "X-SESSION-KEY": u["session_key"]}
# Some play: alice buys items and fights a battle; bob distributes nothing; reactions.
alice, bob = V["users"]["alice"], V["users"]["bob"]
st, _, _, j = send(base, "POST", "/api/internal/account/buy/items",
                   {"ShopItems": [{"Id": 6, "Count": 3, "MoneyType": 1}]}, headers=H(alice))
must(st == 200, f"buy {j}")
st, _, _, j = send(base, "POST", "/api/internal/game/start_battle", {"MissionId": 0}, headers=H(alice))
must(st == 200, f"start_battle {j}")
bid = j["data"]["BattleId"]
st, _, _, j = send(base, "POST", "/api/internal/game/end_battle",
                   {"Result": bid + 1, "Type": 0, "ExpBonus": 3, "BattleId": bid, "MissionId": 0,
                    "Items": [{"Id": 6, "Count": 1}], "Signature": [], "BanType": 0, "BanNote": "", "CollectedReagents": []},
                   headers=H(alice))
must(st == 200 and j == {"Status": "OK"}, f"end_battle {j}")
st, _, _, j = send(base, "POST", "/api/internal/house/pump_reaction", {"FriendId": 2}, headers=H(alice))
must(st == 200, "pump")

# Achievement session for alice.
st, _, _, j = send(base, "POST", "/api/internal/achievements/login", {"id": 1, "auth_key": alice["auth_key"]},
                   headers={"X-TCP-SESSION": "achievements", "Accept": "application/json"})
must(st == 200 and j["ok"], f"ach login {j}")
alice["ach_session"] = j["session_id"]
st, _, _, j = send(base, "POST", "/api/internal/achievements/increase",
                   {"session_id": alice["ach_session"], "time_scale": j["login_time"] + 5, "indexes": [0, 100], "rises": [60, 2]},
                   headers={"X-TCP-SESSION": "achievements", "Accept": "application/json"})
must(st == 200, f"ach increase {j}")

json.dump(V, open(out, "w"), indent=1)
print("phase A done")
