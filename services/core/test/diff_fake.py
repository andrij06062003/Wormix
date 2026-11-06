#!/usr/bin/env python3
"""Differential test driver, deterministic part: drives the main server against fakeapi.py.

Every case sets the fake API's answers, opens a connection (logs in when the case needs a session key), sends
client commands, then a Ping, and records everything the server sends until the Pong (or the connection closes),
plus every request the server made to the API. The transcript is printed; run it against the reference core
and against the Go core and diff the two (see run_diff.sh). Nothing in the transcript depends on timing or
randomness: message headers are decoded (their sync bits are random by design) and checked for validity.

  python3 diff_fake.py <core host> <fake api base url>
"""
import json
import re
import socket
import struct
import sys
import time
import urllib.request

CORE = sys.argv[1] if len(sys.argv) > 1 else "core"
FAKE = sys.argv[2] if len(sys.argv) > 2 else "http://fakeapi:8000"
MAIN, ACH = 49197, 49199
out = []


def emit(line):
    out.append(line)


# ---- wire --------------------------------------------------------------------------------------------------

def hdr(cmd, length):
    return struct.pack(">BHI", 87 | 128 | 32 | 8, cmd, length)


def msg(cmd, body=b""):
    return hdr(cmd, len(body)) + body


def utf(s):
    b = s.encode() if isinstance(s, str) else s
    return struct.pack(">H", len(b)) + b


def u16(v): return struct.pack(">H", v & 0xFFFF)
def u32(v): return struct.pack(">I", v & 0xFFFFFFFF)


def parse_frames(data):
    """Decode frames; header sync bits are random, so only record whether they are valid."""
    res, p = [], 0
    while p < len(data):
        if len(data) - p < 7:
            res.append(f"  tail {data[p:].hex()}")
            break
        flags, rid, rlen = struct.unpack_from(">BHI", data, p)
        valid = (flags & 87 == 87 and bool(flags & 128) != bool(rid & 0x8000)
                 and bool(flags & 32) != bool(rlen & 0x40000000) and bool(flags & 8) != bool(rlen & 0x80000000))
        if not valid:
            res.append(f"  INVALID HEADER, {len(data) - p} bytes left")
            break
        cmd, n = rid & 0x7FFF, rlen & 0x3FFFFFFF
        body = data[p + 7:p + 7 + n]
        res.append(f"  cmd {cmd} len {n} body {body.hex()}" + ("" if len(body) == n else f" (only {len(body)})"))
        p += 7 + n
    return res


class Conn:
    def __init__(self, port=MAIN):
        self.s = socket.create_connection((CORE, port), timeout=10)
        self.closed = False

    def send(self, data):
        try:
            self.s.sendall(data)
        except OSError:
            self.closed = True

    def collect(self, ping=True, idle=1.5, limit=10.0):
        """Send a Ping and read until the Pong, the end of the connection, or no data for `idle` seconds."""
        if ping:
            self.send(msg(16))
        data = b""
        deadline = time.time() + limit
        last = time.time()
        self.s.settimeout(0.2)
        while time.time() < deadline:
            try:
                chunk = self.s.recv(65536)
            except socket.timeout:
                if time.time() - last > idle and (not ping or data):
                    break
                continue
            except OSError:
                self.closed = True
                break
            if not chunk:
                self.closed = True
                break
            data += chunk
            last = time.time()
            if ping and data.endswith(b"") and has_pong(data):
                break
        return data

    def close(self):
        self.s.close()


def has_pong(data):
    p = 0
    while len(data) - p >= 7:
        _, rid, rlen = struct.unpack_from(">BHI", data, p)
        n = rlen & 0x3FFFFFFF
        if len(data) - p < 7 + n:
            return False
        if rid & 0x7FFF == 10017 and n == 0:
            return True
        p += 7 + n
    return False


# ---- fake API ----------------------------------------------------------------------------------------------

def set_api(**routes):
    data = json.dumps({k.replace("__", "/"): v for k, v in routes.items()}).encode()
    urllib.request.urlopen(urllib.request.Request(FAKE + "/__set", data=data, method="POST"), timeout=5).read()


def take_log():
    return json.loads(urllib.request.urlopen(FAKE + "/__log", timeout=5).read())


GUID = re.compile(r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")


def record_requests(sids):
    for r in take_log():
        h = dict(r["headers"])
        sid = h.get("x-tcp-session")
        if sid is not None and GUID.match(sid):
            if sid not in sids:
                sids[sid] = f"<session {len(sids) + 1}>"
            h["x-tcp-session"] = sids[sid]
        emit(f"  API {r['route']} headers {json.dumps(h, sort_keys=True)}")
        emit(f"      body {r['body']}")


def ok(data):
    return [200, json.dumps(data, ensure_ascii=False)]


# ---- canned answers -----------------------------------------------------------------------------------------

def worm(owner, social, level=1, hat=0):
    return {"OwnerId": owner, "SocialOwnerId": social, "Armor": 3, "Attack": 4, "Level": level, "Experience": 25, "Hat": hat}


def profile(uid, social=None, money=450, rating=0):
    return {"Id": uid, "SocialId": social if social is not None else str(uid), "Money": money, "RealMoney": 7,
            "Rating": rating, "WormsGroup": [worm(uid, str(uid), 2, 5), worm(uid, str(uid))],
            "WeaponRecordList": [{"Id": 1, "Count": -1}, {"Id": 7, "Count": 3}], "Stuff": [1, 2, 300],
            "ReactionRate": 12, "Recipes": [4, 5]}


def enter_account(uid, key="abc", **extra):
    d = {"UserProfileStructure": profile(uid), "UserProfileStructures": [profile(2), profile(3, money=10)],
         "AvailableSearchKeys": 5, "Friends": 2, "OnlineFriends": 0, "IsBonusDay": False,
         "DailyBonusStructure": {"LoginSequence": 3, "DailyBonusType": 1, "DailyBonusCount": 20},
         "Reagents": [1, 0, 2], "SessionKey": f"{uid}.{key}"}
    d.update(extra)
    return d


def login_body(uid, key="KEY", ids=(), social=1):
    return u32(uid) + u32(0) + utf(key) + u16(len(ids)) + b"".join(struct.pack("<I", i) for i in ids) + bytes([social])


LOGIN_OK = ok({"type": "EnterAccount", "old_session": None, "data": enter_account(1)})


def case(name, commands, api=None, login=True, port=MAIN, ping=True):
    """Run one case on a fresh connection. commands: list of raw messages (bytes)."""
    emit(f"== {name}")
    sids = {}
    take_log()
    c = Conn(port)
    if login:
        set_api(login=LOGIN_OK)
        c.send(msg(1, login_body(1)))
        emit("  login:")
        for line in parse_frames(c.collect()):
            emit("  " + line)
        record_requests(sids)
    if api:
        set_api(**api)
    for m in commands:
        c.send(m)
    data = c.collect(ping=ping)
    for line in parse_frames(data):
        emit(line)
    if c.closed:
        emit("  CONNECTION CLOSED")
    record_requests(sids)
    c.close()


# ---- cases -------------------------------------------------------------------------------------------------

def main():
    # wait for the core
    for _ in range(90):
        try:
            socket.create_connection((CORE, MAIN), 2).close()
            urllib.request.urlopen(FAKE + "/__log", timeout=2).read()
            break
        except OSError:
            time.sleep(1)
    time.sleep(2)

    # ---- login ----
    case("login ok", [], login=True)
    case("login with friend ids (read little-endian) and special characters in the key",
         [msg(1, login_body(5, 'k"\\\n\t\x01é€😀 ', ids=(1, 2, 0x01020304)))], login=False,
         api={"login": ok({"type": "EnterAccount", "data": enter_account(5)})})
    case("login with invalid UTF-8 in the key",
         [msg(1, login_body(5, b"a\xff\xe2\x82b\xc0\xafc\xed\xa0\x80d\xf0\x9f\x98"))], login=False,
         api={"login": ok({"type": "EnterAccount", "data": enter_account(5)})})
    case("login error", [msg(1, login_body(1))], login=False, api={"login": ok({"type": "LoginError", "data": {"Result": 1}})})
    case("login unknown type", [msg(1, login_body(1))], login=False, api={"login": ok({"type": "Nope", "data": {}})})
    case("login id 0", [msg(1, login_body(0))], login=False, api={"login": LOGIN_OK})
    case("login API 500", [msg(1, login_body(1))], login=False, api={"login": [500, "{}"]})
    case("login API bad JSON", [msg(1, login_body(1))], login=False, api={"login": [200, "<html>"]})
    case("login no type", [msg(1, login_body(1))], login=False, api={"login": ok({"data": {}})})
    case("login JSON null", [msg(1, login_body(1))], login=False, api={"login": [200, "null"]})
    case("login bonus day + lowercase keys",
         [msg(1, login_body(7))], login=False,
         api={"login": ok({"type": "EnterAccount", "data": {**{k[0].lower() + k[1:]: v for k, v in enter_account(7).items()},
                                                            "IsBonusDay": True,
                                                            "BonusDaysStructure": {"Money": 1, "RealMoney": 2, "BattlesCount": 3, "BonusMessage": "hi"}}})})
    case("login missing lists (struct defaults)", [msg(1, login_body(8))], login=False,
         api={"login": ok({"type": "EnterAccount", "data": {"UserProfileStructure": {"Id": 8, "SocialId": "8"}, "SessionKey": "8.x"}})})
    case("login minimal profile", [msg(1, login_body(8))], login=False,
         api={"login": ok({"type": "EnterAccount", "data": {"UserProfileStructure": {"Id": 8, "SocialId": "8", "WormsGroup": [], "WeaponRecordList": [], "Stuff": [], "Recipes": []},
                                                            "UserProfileStructures": [], "Reagents": [], "SessionKey": "8.x"}})})
    case("login non-ASCII social id (fixed buffer overflow)", [msg(1, login_body(9))], login=False,
         api={"login": ok({"type": "EnterAccount", "data": enter_account(9, UserProfileStructure=profile(9, social="Вася"))})})
    conv = [("short from string", {"Friends": " 12 "}), ("short from float", {"OnlineFriends": 2.5}),
            ("short from float 3.5", {"OnlineFriends": 3.5}), ("short overflow", {"Friends": 70000}),
            ("byte from bool", {"AvailableSearchKeys": True}), ("byte from string", {"AvailableSearchKeys": "7"}),
            ("byte negative", {"AvailableSearchKeys": -1}), ("bool from int", {"IsSecure": 2}),
            ("bool from string", {"IsSecure": " TRUE "}), ("bool from bad string", {"IsSecure": "yes"}),
            ("bool from null", {"IsSecure": None}), ("short from null", {"Friends": None}),
            ("list of uint from bool", {"Reagents": [True, False]}), ("list null", {"Reagents": None}),
            ("list as object", {"Reagents": {}}), ("big integer", {"Friends": 123456789012345678901234567890}),
            ("string from empty string", {"Friends": ""}), ("unknown member", {"Whatever": [1, {"a": None}]})]
    for name, extra in conv:
        case("login conversion: " + name, [msg(1, login_body(10))], login=False,
             api={"login": ok({"type": "EnterAccount", "data": enter_account(10, **extra)})})
    pconv = [("uint from string", {"Money": "77"}), ("uint from float", {"Money": 2.5}), ("uint from bool", {"Money": True}),
             ("uint from string with sign", {"Money": "+5"}), ("int from float", {"Rating": -5.5}),
             ("int from bool", {"Rating": False}), ("int from string", {"Rating": "-3"}),
             ("string from int", {"SocialId": 123}), ("string from float", {"SocialId": 1.5}),
             ("string from big float", {"SocialId": 1e20}), ("string from small float", {"SocialId": 0.00001}),
             ("string from bool", {"SocialId": True}), ("string from object", {"SocialId": {}}),
             ("ushort from bool", {"Stuff": [True]}), ("ushort from float", {"Recipes": [1.5, 2.5]}),
             ("worm hat from string", {"WormsGroup": [worm(10, "10", hat="-4")]}),
             ("worm null social", {"WormsGroup": [{"OwnerId": 10}]}),
             ("weapon count from string", {"WeaponRecordList": [{"Id": 1, "Count": "9"}]}),
             ("case-insensitive members", {"money": 5, "MONEY": 6, "socialid": "x"})]
    for name, extra in pconv:
        case("login profile conversion: " + name, [msg(1, login_body(10))], login=False,
             api={"login": ok({"type": "EnterAccount", "data": enter_account(10, UserProfileStructure={**profile(10), **extra})})})
    case("login negative uint", [msg(1, login_body(11))], login=False,
         api={"login": ok({"type": "EnterAccount", "data": enter_account(11, UserProfileStructure={**profile(11), "Money": -1})})})
    case("login null session key", [msg(1, login_body(12))], login=False,
         api={"login": ok({"type": "EnterAccount", "data": enter_account(12, SessionKey=None)})})

    # second login of the same user kicks the old session
    emit("== old session is closed by a new login")
    take_log()
    a = Conn()
    set_api(login=ok({"type": "EnterAccount", "old_session": None, "data": enter_account(1)}))
    a.send(msg(1, login_body(1)))
    a.collect()
    sid_a = take_log()[0]["headers"]["x-tcp-session"]
    b = Conn()
    set_api(login=ok({"type": "EnterAccount", "old_session": sid_a.upper(), "data": enter_account(1)}))
    b.send(msg(1, login_body(1)))
    for line in parse_frames(b.collect()):
        emit("  new: " + line)
    a.collect(ping=True, limit=3)
    emit(f"  old closed: {a.closed}")
    take_log()
    a.close(); b.close()
    case("login bad old_session", [msg(1, login_body(1))], login=False,
         api={"login": ok({"type": "EnterAccount", "old_session": "not-a-guid", "data": enter_account(1)})})

    # ---- framing ----
    case("bad header", [b"\x00\x00\x10\x00\x00\x00\x00"], login=False)
    case("too long", [hdr(16, 2 * 1024 * 1024)], login=False)
    case("unknown command", [msg(999, b"xyz")], login=False)
    case("two commands in one packet", [msg(16) + msg(2, u32(5) + u32(1))], login=False)
    case("truncated NeedMoney", [msg(2, b"\x00\x00")], login=False)
    case("ping and need money", [msg(2, u32(5) + u32(1))], login=False)
    case("no login: commands still go to the API without a key", [msg(11, u32(1))], login=False,
         api={"account/buy/battles": ok({"data": {"Result": 0}})})

    # ---- shop ----
    shop = msg(3, u16(2) + u16(12) + u32(5) + u32(2) + u32(1) + u16(12) + u32(9) + u32(1) + u32(0))
    case("shop", [shop], api={"account/buy/items": ok({"data": {"Result": 0, "Weapons": [{"Id": 5, "Count": 2}], "Stuff": [9]}})})
    case("shop empty list sends nothing", [msg(3, u16(0))])
    case("shop missing Result (struct defaults)", [shop], api={"account/buy/items": ok({"data": {"Weapons": [], "Stuff": []}})})
    case("shop missing lists", [shop], api={"account/buy/items": ok({"data": {"Result": 0}})})
    case("shop no data", [shop], api={"account/buy/items": ok({"type": "x"})})
    case("shop 403", [shop], api={"account/buy/items": [403, '{"message":"Access Denied"}']})

    # ---- arena / battles ----
    arena = msg(4, b"\x01")
    case("arena result", [arena], api={"game/get_arena": ok({"type": "ArenaResult", "data": {"UserProfileStructures": [profile(2), profile(3)], "BattlesCount": 5, "CurrentMission": 70000, "BossAvailable": True}})})
    case("arena locked", [arena], api={"game/get_arena": ok({"type": "ArenaLocked", "data": {"Delay": 3600, "MissionId": -2, "ErrorCode": 2}})})
    case("arena unknown type", [arena], api={"game/get_arena": ok({"data": {}})})
    case("arena null root", [arena], api={"game/get_arena": [200, "null"]})
    case("arena array root", [arena], api={"game/get_arena": [200, "[1]"]})
    case("arena data null", [arena], api={"game/get_arena": ok({"type": "ArenaLocked", "data": None})})
    case("arena trailing garbage", [arena], api={"game/get_arena": [200, '{"type":"ArenaLocked","data":{}} x']})
    case("arena duplicate keys", [arena], api={"game/get_arena": [200, '{"type":"ArenaLocked","data":{"Delay":1,"delay":2,"Delay":3}}']})
    case("start battle", [msg(6, u16(65535))], api={"game/start_battle": ok({"data": {"Awards": [{"Id": 1, "Count": 2}], "BattleId": 1077, "ReagentForBattle": [1, 2, 255]}})})
    case("start battle bad byte", [msg(6, u16(3))], api={"game/start_battle": ok({"data": {"Awards": [], "BattleId": 1, "ReagentForBattle": [256]}})})
    end_battle = msg(84, utf("r") + u32(1) + utf("t") + u32(0) + utf("e") + u32(5) + utf("b") + u32(1077) + u16(3)
                     + u16(2) + b"\x08\x00" + u32(5) + u32(1) + b"\x08\x00" + u32(6) + u32(0xFFFFFFFF)
                     + u16(3) + b"\x01\x02\x03" + u16(0) + utf("note\"x") + u16(2) + b"\x07\x80" + b"H" * 16)
    case("end battle", [end_battle], api={"game/end_battle": ok({"data": None})})
    case("end battle no hash", [end_battle[:-16]], api={"game/end_battle": ok({})})
    case("end battle empty answer", [end_battle], api={"game/end_battle": [200, ""]})
    case("end battle truncated", [msg(84, utf("r") + u32(1))], api={"game/end_battle": ok({})})

    # ---- account ----
    case("buy battle", [msg(11, u32(1))], api={"account/buy/battles": ok({"data": {"Result": 3}})})
    case("buy battle no data", [msg(11, u32(1))], api={"account/buy/battles": ok({})})
    case("buy battle data null", [msg(11, u32(1))], api={"account/buy/battles": ok({"data": None})})
    case("buy battle result as string/float/bool", [msg(11, u32(0)), msg(11, u32(0)), msg(11, u32(0))],
         api={"account/buy/battles": ok({"data": {"Result": " -7 "}})})
    case("buy battle result float", [msg(11, u32(0))], api={"account/buy/battles": ok({"data": {"Result": 2.5}})})
    case("buy battle result bool", [msg(11, u32(0))], api={"account/buy/battles": ok({"data": {"result": True}})})
    case("buy battle result overflow", [msg(11, u32(0))], api={"account/buy/battles": ok({"data": {"Result": 40000}})})
    case("add to group", [msg(12, u32(77) + u16(0xFFFF))], api={"team/add": ok({"data": {"Result": 0}})})
    case("remove from group", [msg(13, u32(77))], api={"team/remove": ok({"data": {"Result": 1}})})
    case("distribute points", [msg(14, u32(2) + u32(0xFFFFFFFF))], api={"account/distribute_points": ok({"data": {"Result": 2}})})
    case("reset parameters", [msg(15, u32(1))], api={"account/reset/parameters": ok({"data": {"Result": 0}})})
    case("reorder group", [msg(22, u16(3) + u32(1) + u32(5) + u32(9))], api={"team/reorder": ok({"data": {"Result": 0}})})
    case("reorder group negative count", [msg(22, u16(0xFFFF))], api={"team/reorder": ok({"data": {"Result": 0}})})
    case("select stuff", [msg(25, u16(300))], api={"account/select_stuff": ok({"data": {"Result": 0, "StuffId": 300, "IsSecure": True}})})
    case("select stuff no data", [msg(25, u16(300))], api={"account/select_stuff": ok({})})
    case("change race", [msg(36, b"\x02\x01")], api={"account/buy/race": ok({"data": {"Result": 2}})})
    case("change race no data", [msg(36, b"\x02\x01")], api={"account/buy/race": ok({})})
    case("buy reaction", [msg(49, b"\x05")], api={"account/buy/reaction": ok({"data": {"Result": 0, "ReactionRateCount": 5}})})
    case("unlock mission", [msg(58, u16(12))], api={"account/buy/mission": ok({"data": {"Result": 0}})})
    case("wipe without confirm code", [msg(48, utf("1234"))], api={"account/reset/account": ok({"data": {"Result": 0}})})
    case("wipe", [msg(52, u32(1) + u32(2) + u32(3)), msg(48, utf("1234"))], api={"account/reset/account": ok({"data": {"Result": 0}})})
    case("upgrade weapon", [msg(86, u16(4))], api={"craft/upgrade_weapon": ok({"data": {"RecipeId": 4, "Result": 0}})})
    case("downgrade weapon", [msg(88, u16(4))], api={"craft/downgrade_weapon": ok({"data": {"RecipeId": 4, "Result": 1}})})
    case("downgrade weapon no data", [msg(88, u16(4))], api={"craft/downgrade_weapon": ok({"type": "x"})})

    # ---- house / info ----
    case("search the house", [msg(81, utf("1.abc") + u32(2) + b"\x01")],
         api={"house/search": ok({"data": {"Result": 2, "Value": 30, "AvailableSearchKeys": 4, "FriendId": 2, "IsSecure": True}})})
    case("search the house odd key", [msg(81, utf(b"\xff\x00\x1f/\x7f") + u32(2) + b"\x01")],
         api={"house/search": ok({"data": {"Result": 0}})})
    case("pump reaction", [msg(83, u32(2))], api={"house/pump_reaction": ok({"data": {"Result": 1}})})
    pumps = msg(82, u16(2) + u32(2) + u32(3) + b"M" * 16)
    case("pump reactions (one)", [msg(82, u16(1) + u32(2) + b"M" * 16)], api={"house/pump_reactions": ok({"data": {"PumpedFriends": [{"FriendId": 2, "Result": 0}]}})})
    case("pump reactions (two: size quirk)", [pumps], api={"house/pump_reactions": ok({"data": {"PumpedFriends": [{"FriendId": 2, "Result": 0}, {"FriendId": 3, "Result": 1}]}})})
    case("pump reactions no data", [pumps], api={"house/pump_reactions": ok({})})
    who = {"TodayPumped": [{"LongId": 2, "StringId": "2"}], "YesterdayPumped": [], "TwoDaysAgoPumped": [{"LongId": 3, "StringId": "3"}, {"LongId": 4, "StringId": "4"}]}
    case("who pumped", [msg(46, b"\x00")], api={"info/pumped_reaction": ok({"data": who})})
    case("who pumped non-ASCII (header size quirk)", [msg(46, b"\x01")],
         api={"info/pumped_reaction": ok({"data": {**who, "TodayPumped": [{"LongId": 2, "StringId": "Пётр"}]}})})
    case("who pumped null string", [msg(46, b"\x01")],
         api={"info/pumped_reaction": ok({"data": {**who, "TodayPumped": [{"LongId": 2}]}})})

    # ---- lobby without a peer ----
    case("battle request to offline player", [msg(1002, utf("h") + u32(49198) + u32(3) + u32(1) + utf(" 4242 "))])
    case("battle request bad id", [msg(1002, utf("h") + u32(49198) + u32(3) + u32(1) + utf("x"))])
    case("battle request to self", [msg(1002, utf("h") + u32(49198) + u32(3) + u32(1) + utf("+1"))])
    case("battle request truncated", [msg(1002, utf("h"))])
    case("lobby command before login", [msg(1002, utf("h") + u32(49198) + u32(3) + u32(1) + utf("2"))], login=False)
    case("wager bad amount", [msg(2005, utf("h") + u32(49198) + u32(16) + u32(5))])
    case("wager not enough money", [msg(2005, utf("h") + u32(49198) + u32(300) + u32(5))],
         api={"pvp/profile": ok({"data": profile(1, money=10)})})
    case("wager API error", [msg(2005, utf("h") + u32(49198) + u32(15) + u32(5))], api={"pvp/profile": [500, "{}"]})
    case("wager queued, then cancel", [msg(2005, utf("h") + u32(49198) + u32(15) + u32(5)), msg(2007)],
         api={"pvp/profile": ok({"data": profile(1)})})

    # ---- PvP API requests (challenge + battle, wager match); battle ids, seeds and the first player vary ----
    emit("== pvp API requests")
    take_log()
    set_api(**{"pvp/profile": ok({"data": profile(1)}),
               "pvp/start_battle": ok({"data": {"UserProfileStructure": profile(1), "ReagentsForBattle": [3, 200]}}),
               "pvp/end_battle": ok({})})
    p1, p2 = Conn(), Conn()
    set_api(login=ok({"type": "EnterAccount", "data": enter_account(1)}))
    p1.send(msg(1, login_body(1)))
    p1.collect()
    set_api(login=ok({"type": "EnterAccount", "data": enter_account(2, key="def")}))
    p2.send(msg(1, login_body(2, key="KEY2")))
    p2.collect()
    p1.send(msg(1002, utf("h") + u32(49198) + u32(3) + u32(1) + utf("2")))
    p1.collect()
    offer = p2.collect(ping=True)
    bid = 0
    q = 0
    while len(offer) - q >= 7:
        _, rid, rlen = struct.unpack_from(">BHI", offer, q)
        n = rlen & 0x3FFFFFFF
        if rid & 0x7FFF == 2002:
            bid = struct.unpack(">I", offer[q + 7 + n - 4:q + 7 + n])[0]
        q += 7 + n
    emit(f"  offer received: {bid != 0}")
    p2.send(msg(1003, struct.pack(">hII", 0, 1, bid)))
    p2.collect()
    p1.collect()
    pv = Conn(49198)
    pv.send(msg(1001, u32(1) + b"\x01" + utf("KEY") + u16(0) + utf("h") + u32(49197) + u32(bid) + b"\x01"))
    pv.collect()
    pv.send(msg(1006, u32(1) + u32(bid) + u32(1) + b"S" * 16))
    pv.collect()
    time.sleep(1.5)
    wager = msg(2005, utf("h") + u32(49198) + u32(50) + u32(5) + b"S" * 16)
    p1.send(wager)
    p1.collect()
    p2.send(wager)
    p2.collect()
    p1.collect()
    time.sleep(1)
    for r in take_log():
        body = re.sub(r'"BattleId":\d+', '"BattleId":<id>', r["body"])
        tcp = r['headers'].get('x-tcp-session')
        tcp = "<guid>" if tcp and GUID.match(tcp) else tcp
        emit(f"  API {r['route']} key {r['headers'].get('x-session-key')} tcp {tcp} body {body}")
    pv.close(); p1.close(); p2.close()

    # ---- achievements server: no handlers ----
    emit("== achievements server")
    c = Conn(ACH)
    c.send(msg(3001, utf("wormix")))
    for line in parse_frames(c.collect(ping=False, idle=1.0, limit=2)):
        emit(line)
    emit(f"  closed: {c.closed}")
    c.close()

    print("\n".join(out))


if __name__ == "__main__":
    main()
