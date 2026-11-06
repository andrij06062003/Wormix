#!/usr/bin/env python3
"""End-to-end PvP test: two fake game clients speaking the Wormix 1.05.0 protocol.

Runs against a real API + wormix-core (see tests/pvp/run.sh). Covers:
  1. friend challenge: offer, accept, battle on the PvP server, turn handover, a win, the API result;
  2. wager battle: matchmaking queue, wager taken, chat relay, win paid out (+80%) and rating changed;
  3. surrender (PvpDropPlayer);
  4. a player who disconnects and doesn't come back loses (reconnect timeout);
  5. decline and cancel of a challenge;
  6. 2v2 wager battle: four players matched into two teams, turns in the client's order, a team win paid out;
  7. teams for 2v2: invite, teammate call (declined, then joined), the pair matched on one side, team ends on logout.
Exits non-zero on the first failure.
"""
import hashlib
import json
import os
import socket
import struct
import sys
import time
import urllib.error
import urllib.request
import uuid

API = os.environ.get("API", "http://api:8000/api/")
CORE = os.environ.get("CORE", "core")
MAIN_PORT, PVP_PORT = 49197, 49198   # as in the 1.05.0 preloader (AntWormixSettings)
SECRET = b"ygfA69i3Uz6bDO8"


# ---- wire format ------------------------------------------------------------------------------------------

def header(cmd, length):
    # flags: the constant bits (87) plus B/D/E set, so the command id and length need no sync bits
    return struct.pack(">BHI", 87 | 128 | 32 | 8, cmd, length)


def msg(cmd, body=b"", secure=False):
    if secure:
        body += hashlib.md5(body + SECRET).digest()
    return header(cmd, len(body)) + body


def utf(s):
    b = s.encode()
    return struct.pack(">H", len(b)) + b


class Reader:
    def __init__(self, data):
        self.d, self.p = data, 0

    def take(self, n):
        v = self.d[self.p:self.p + n]
        assert len(v) == n, "truncated message"
        self.p += n
        return v

    def u8(self): return self.take(1)[0]
    def u16(self): return struct.unpack(">H", self.take(2))[0]
    def s16(self): return struct.unpack(">h", self.take(2))[0]
    def u32(self): return struct.unpack(">I", self.take(4))[0]
    def s32(self): return struct.unpack(">i", self.take(4))[0]
    def utf(self): return self.take(self.u16()).decode()

    def profile(self):
        size = self.u16()
        start = self.p
        uid, money, real, rating = struct.unpack(">IIIi", self.take(16))
        self.p = start + size
        return {"id": uid, "money": money, "rating": rating}

    def secure_ok(self):
        return hashlib.md5(self.d[:-16] + SECRET).digest() == self.d[-16:]


class Conn:
    def __init__(self, name, port):
        self.name = name
        self.sock = socket.create_connection((CORE, port), timeout=10)
        self.buf = b""
        self.inbox = []

    def send(self, data):
        self.sock.sendall(data)

    def _read_one(self, timeout):
        self.sock.settimeout(timeout)
        while True:
            if len(self.buf) >= 7:
                _, rawcmd, rawlen = struct.unpack(">BHI", self.buf[:7])
                n = rawlen & 0x3FFFFFFF
                if len(self.buf) >= 7 + n:
                    cmd, body = rawcmd & 0x7FFF, self.buf[7:7 + n]
                    self.buf = self.buf[7 + n:]
                    return cmd, body
            chunk = self.sock.recv(65536)
            if not chunk:
                raise ConnectionError(f"{self.name}: connection closed")
            self.buf += chunk

    def expect(self, cmd, timeout=10):
        """Wait for a message with this command id (earlier ones of other kinds are kept for later)."""
        for i, (c, b) in enumerate(self.inbox):
            if c == cmd:
                del self.inbox[i]
                return b
        deadline = time.time() + timeout
        while True:
            left = deadline - time.time()
            if left <= 0:
                raise AssertionError(f"{self.name}: no command {cmd} within {timeout}s (got {[c for c, _ in self.inbox]})")
            try:
                c, b = self._read_one(left)
            except socket.timeout:
                continue
            if c == cmd:
                return b
            self.inbox.append((c, b))

    def nothing(self, cmd, wait=1.0):
        try:
            self.expect(cmd, wait)
        except AssertionError:
            return
        raise AssertionError(f"{self.name}: unexpected command {cmd}")

    def close(self):
        self.sock.close()


# ---- API ------------------------------------------------------------------------------------------------------

def api(path, body=None, token=None):
    req = urllib.request.Request(API + path, data=json.dumps(body or {}).encode(), method="POST",
                                 headers={"Content-Type": "application/json", "Accept": "application/json",
                                          **({"Authorization": "Bearer " + token} if token else {})})
    with urllib.request.urlopen(req, timeout=20) as r:
        return json.loads(r.read())


class Player:
    def __init__(self, name):
        self.name = name
        login = "t" + uuid.uuid4().hex[:10]
        r = api("account/sign_up", {"login": login, "password": "password123", "password_confirmation": "password123"})
        self.id, self.token = r["id"], r["auth_token"]
        self.main = None
        self.pvp = None
        self.login()

    def login(self):
        self.auth_key = api("account/game", token=self.token)["auth_key"]
        self.main = Conn(self.name + "/main", MAIN_PORT)
        body = struct.pack(">II", self.id, 0) + utf(self.auth_key) + struct.pack(">H", 0) + bytes([1])
        self.main.send(msg(1, body))
        enter = Reader(self.main.expect(10001, 20))   # EnterAccount
        self.profile = enter.profile()

    def fresh_profile(self):
        # Reconnecting gives the current profile from the database.
        self.main.close()
        time.sleep(0.3)
        self.login()
        return self.profile

    def pvp_login(self, battle_id, first, enemy):
        self.pvp = Conn(self.name + "/pvp", PVP_PORT)
        body = (struct.pack(">I?", self.id, first) + utf(self.auth_key) + struct.pack(">HI", 1, enemy.id)
                + utf("wormixtest.ru") + struct.pack(">iIB", MAIN_PORT, battle_id, 1))
        self.pvp.send(msg(1001, body))
        r = Reader(self.pvp.expect(2001))
        r.utf()
        return r.u8()


# ---- PvP turn helpers ------------------------------------------------------------------------------------------

def action(battle_id, turn, num, player_num):
    # firstFrame, lastFrame, turn, commandNum, battleId, playerNum, count, then frame 10 with one WORM_LEFT action
    actions = struct.pack(">III", 10, 1, 4096)
    body = struct.pack(">IIHHIBH", 10, 12, turn, num, battle_id, player_num, 3) + actions
    return msg(1007, body, secure=True)


def end_turn(battle_id, turn, num, player_num, eliminated, state):
    body = struct.pack(">HHiB", turn, num, battle_id, player_num) + struct.pack(">H", len(eliminated))
    body += b"".join(struct.pack(">I", e) for e in eliminated) + struct.pack(">i", state)
    return msg(1014, body, secure=True)


def end_turn_response(battle_id, turn, num, eliminated, state):
    body = struct.pack(">HHi", turn, num, battle_id) + struct.pack(">H", len(eliminated))
    body += b"".join(struct.pack(">I", e) for e in eliminated) + struct.pack(">i", state)
    return msg(1017, body, secure=True)


def play_turn(active, passive, battle_id, turn, eliminated=(), state=1234):
    first_num = 1 if turn == 1 else 2
    active.pvp.send(action(battle_id, turn, first_num, 1))
    got = Reader(passive.pvp.expect(1007))
    got.u32(); got.u32()
    assert got.u16() == turn and got.u16() == first_num and got.secure_ok(), "action not relayed intact"
    active.pvp.send(end_turn(battle_id, turn, first_num + 1, 1, list(eliminated), state))
    et = Reader(passive.pvp.expect(1014))
    assert et.u16() == turn and et.secure_ok(), "end turn not relayed intact"
    passive.pvp.send(end_turn_response(battle_id, turn, first_num + 1, list(eliminated), state))


def expect_start_turn(p, battle_id, turn, active_id):
    r = Reader(p.pvp.expect(1019))
    assert r.secure_ok(), "StartTurn hash"
    assert (r.u32(), r.u32(), r.u16()) == (battle_id, active_id, turn), "StartTurn content"


def expect_end_battle(p, battle_id):
    return expect_end_battle_within(p, battle_id, 15)


def expect_end_battle_within(p, battle_id, timeout):
    r = Reader(p.pvp.expect(1021, timeout))
    assert r.secure_ok(), "EndBattle hash"
    desync = r.u8()
    assert r.u32() == battle_id
    r.u16(); r.u16()
    players = [r.u32() for _ in range(r.u16())]
    results = [r.u8() for _ in range(r.u16())]
    return desync, dict(zip(players, results))   # 0 win, 1 loss, 3 draw


def challenge(a, b):
    """a challenges b, b accepts. Returns the battle id."""
    a.main.send(msg(1002, utf("wormixtest.ru") + struct.pack(">iIi", PVP_PORT, 3, 42) + utf(str(b.id))))
    offer = Reader(b.main.expect(2002))
    assert offer.utf() == "wormixtest.ru" and offer.s32() == PVP_PORT and offer.u32() == 3 and offer.s32() == 42
    enemy, me = offer.profile(), offer.profile()
    assert enemy["id"] == a.id and me["id"] == b.id, "offer profiles"
    battle_id = offer.u32()
    b.main.send(msg(1003, struct.pack(">hII", 0, a.id, battle_id)))
    res = Reader(a.main.expect(2003))
    assert res.s16() == 0, "challenge not accepted"
    enemy, me = res.profile(), res.profile()
    assert enemy["id"] == b.id and me["id"] == a.id and res.u32() == battle_id
    return battle_id


def wager_match(a, b, wager):
    """Both ask for a wager battle; returns (battle id, first player)."""
    req = utf("wormixtest.ru") + struct.pack(">iiI", PVP_PORT, wager, 5)
    a.main.send(msg(2005, req, secure=True))
    time.sleep(0.5)
    b.main.send(msg(2005, req, secure=True))
    results = {}
    for p in (a, b):
        r = Reader(p.main.expect(2006, 20))
        assert r.secure_ok(), "wager result hash"
        r.utf(); r.s32(); r.u32()
        enemy, me = r.profile(), r.profile()
        first, seed, battle_id = r.u32(), r.s32(), r.u32()
        assert enemy["id"] != p.id and me["id"] == p.id
        results[p.id] = (battle_id, first, me["money"])
    assert results[a.id][0] == results[b.id][0] and results[a.id][1] == results[b.id][1]
    battle_id, first = results[a.id][:2]
    return battle_id, (a if first == a.id else b), results


def play_turn_multi(active, others, battle_id, turn, eliminated=(), state=1234):
    """Like play_turn, with every other player watching and answering."""
    first_num = 1 if turn == 1 else 2
    active.pvp.send(action(battle_id, turn, first_num, 1))
    for p in others:
        got = Reader(p.pvp.expect(1007))
        got.u32(); got.u32()
        assert got.u16() == turn and got.u16() == first_num and got.secure_ok(), f"action not relayed to {p.name}"
    active.pvp.send(end_turn(battle_id, turn, first_num + 1, 1, list(eliminated), state))
    for p in others:
        assert Reader(p.pvp.expect(1014)).u16() == turn, f"end turn not relayed to {p.name}"
    for p in others:
        p.pvp.send(end_turn_response(battle_id, turn, first_num + 1, list(eliminated), state))


def team_wager_match(players, wager):
    """All four ask for a 2v2 wager battle. Returns (battle id, players in turn order, team by id, money by id)."""
    req = utf("wormixtest.ru") + struct.pack(">iiI", PVP_PORT, wager, 5) + bytes([1])   # mode 1: 2v2
    for p in players:
        p.main.send(msg(2005, req, secure=True))
        time.sleep(0.3)
    seen = None
    money = {}
    for p in players:
        r = Reader(p.main.expect(2006, 20))
        assert r.secure_ok(), "2v2 wager result hash"
        r.utf(); r.s32(); r.u32()
        enemy, me = r.profile(), r.profile()
        first, seed, battle_id = r.u32(), r.s32(), r.u32()
        reagents = [r.u8() for _ in range(r.u16())]
        seats = []
        for _ in range(r.u8()):
            team = r.u8()
            seats.append((r.profile()["id"], team))
        assert me["id"] == p.id and len(seats) == 4, seats
        assert seats[0][0] == first, "first player is the first seat"
        assert [t for _, t in seats] == [1, 2, 1, 2], f"teams must alternate: {seats}"
        my_team = dict(seats)[p.id]
        assert dict(seats)[enemy["id"]] != my_team, "the enemy profile is from the other team"
        if seen is None:
            seen = (battle_id, seats, reagents)
        assert seen == (battle_id, seats, reagents), "all four must get the same battle"
        money[p.id] = me["money"]
    battle_id, seats, _ = seen
    by_id = {p.id: p for p in players}
    return battle_id, [by_id[i] for i, _ in seats], dict(seats), money


def read_offer(p):
    """Returns (enemy id, seed, battle id, kind) of a BattleOffer."""
    r = Reader(p.main.expect(2002))
    r.utf(); r.s32(); r.u32()
    seed = r.s32()
    enemy, _ = r.profile(), r.profile()
    battle_id = r.u32()
    kind = r.u8() if r.p < len(r.d) else 0
    return enemy["id"], seed, battle_id, kind


def step(name):
    print(f"\n== {name}", flush=True)


def wait_for_stack(timeout=180):
    """Wait until the API answers HTTP and wormix-core listens (the API migrates its database on first start)."""
    deadline = time.time() + timeout
    while True:
        try:
            for port in (MAIN_PORT, PVP_PORT):
                socket.create_connection((CORE, port), 2).close()
            try:
                urllib.request.urlopen(API.rsplit("/api/", 1)[0] + "/", timeout=5)
            except urllib.error.HTTPError:
                pass   # any HTTP answer will do
            return
        except OSError:
            if time.time() > deadline:
                raise
            time.sleep(2)


def main():
    wait_for_stack()
    step("register two players and log in to the main server")
    a, b = Player("A"), Player("B")
    print(f"A={a.id} B={b.id}")

    step("friend challenge: A challenges B, B accepts")
    battle = challenge(a, b)
    assert a.pvp_login(battle, True, b) == 1
    assert b.pvp_login(battle, False, a) == 2
    print(f"battle {battle}: both logged in to the PvP server")

    step("turn 1 (A): moves relayed, then StartTurn for B")
    play_turn(a, b, battle, 1)
    expect_start_turn(a, battle, 2, b.id)
    expect_start_turn(b, battle, 2, b.id)

    step("chat is relayed")
    a.pvp.send(msg(17, struct.pack(">I", a.id) + utf("gl hf") + struct.pack(">I", battle)))
    chat = Reader(b.pvp.expect(17))
    assert chat.u32() == a.id and chat.utf() == "gl hf"

    step("turn 2 (B) eliminates A: B wins")
    play_turn(b, a, battle, 2, eliminated=[a.id])
    for p in (a, b):
        desync, results = expect_end_battle(p, battle)
        assert not desync and results == {a.id: 1, b.id: 0}, results
    a.pvp.close(); b.pvp.close()

    step("wager battle (15): matched from the queue, wager taken")
    before = {p.id: p.fresh_profile() for p in (a, b)}
    battle, first, res = wager_match(a, b, 15)
    for p in (a, b):
        assert res[p.id][2] == before[p.id]["money"] - 15, f"wager not taken from {p.name}"
    second = b if first is a else a
    first.pvp_login(battle, True, second)
    second.pvp_login(battle, False, first)

    step("first player wins: paid wager + 80%, rating up; loser keeps nothing")
    play_turn(first, second, battle, 1, eliminated=[second.id])
    for p in (a, b):
        _, results = expect_end_battle(p, battle)
        assert results == {first.id: 0, second.id: 1}, results
    first.pvp.close(); second.pvp.close()
    time.sleep(1.5)   # the result is recorded in the background
    after = {p.id: p.fresh_profile() for p in (a, b)}
    assert after[first.id]["money"] == before[first.id]["money"] + 12, (before[first.id], after[first.id])
    assert after[second.id]["money"] == before[second.id]["money"] - 15, (before[second.id], after[second.id])
    assert after[first.id]["rating"] > before[first.id]["rating"], "winner rating"
    print(f"winner money {before[first.id]['money']} -> {after[first.id]['money']}, rating {before[first.id]['rating']} -> {after[first.id]['rating']}")

    step("surrender: B gives up, A wins")
    battle = challenge(a, b)
    a.pvp_login(battle, True, b)
    b.pvp_login(battle, False, a)
    b.pvp.send(msg(1006, struct.pack(">IIi", b.id, battle, 1), secure=True))
    sysmsg = Reader(a.pvp.expect(2021))
    assert sysmsg.s16() == 4 and sysmsg.u32() == b.id, "surrender notice"
    _, results = expect_end_battle(a, battle)
    assert results == {a.id: 0, b.id: 1}, results
    a.pvp.close(); b.pvp.close()

    step("reconnect: B drops and comes back, then gets the moves it missed")
    battle = challenge(a, b)
    a.pvp_login(battle, True, b)
    b.pvp_login(battle, False, a)
    b.pvp.close()
    sysmsg = Reader(a.pvp.expect(2021))
    assert sysmsg.s16() == 6, "disconnect notice"
    a.pvp.send(action(battle, 1, 1, 1))
    b.pvp = Conn("B/pvp", PVP_PORT)
    body = struct.pack(">I", b.id) + utf(b.auth_key) + utf("wormixtest.ru") + struct.pack(">iIHBHB", MAIN_PORT, battle, 1, 2, 0, 1)
    b.pvp.send(msg(1015, body))
    b.pvp.expect(2001)
    assert Reader(a.pvp.expect(2021)).s16() == 7, "reconnect notice"
    got = Reader(b.pvp.expect(1007))
    got.u32(); got.u32()
    assert got.u16() == 1 and got.u16() == 1, "missed move resent"

    step("B surrenders to clean up; a fresh login with a wrong key is refused")
    b.pvp.send(msg(1006, struct.pack(">IIi", b.id, battle, 1), secure=True))
    expect_end_battle(a, battle)
    bad = Conn("bad/pvp", PVP_PORT)
    bad.send(msg(1001, struct.pack(">I?", a.id, True) + utf("wrong") + struct.pack(">HI", 1, b.id)
                 + utf("x") + struct.pack(">iIB", MAIN_PORT, battle, 1)))
    assert Reader(bad.expect(2004)).s32() == 1, "wrong key must be refused"
    bad.close(); a.pvp.close(); b.pvp.close()

    step("decline and cancel")
    a.main.send(msg(1002, utf("wormixtest.ru") + struct.pack(">iIi", PVP_PORT, 3, 1) + utf(str(b.id))))
    offer = Reader(b.main.expect(2002))
    offer.utf(); offer.s32(); offer.u32(); offer.s32(); offer.profile(); offer.profile()
    b.main.send(msg(1003, struct.pack(">hII", 1, a.id, offer.u32())))
    assert Reader(a.main.expect(2003)).s16() == 1, "decline"
    a.main.send(msg(1002, utf("wormixtest.ru") + struct.pack(">iIi", PVP_PORT, 3, 1) + utf(str(b.id))))
    b.main.expect(2002)
    a.main.send(msg(1004, struct.pack(">II", a.id, b.id)))
    cancel = Reader(b.main.expect(1004))
    assert cancel.u32() == a.id, "cancel forwarded"
    a.main.send(msg(1002, utf("wormixtest.ru") + struct.pack(">iIi", PVP_PORT, 3, 1) + utf("999999")))
    assert Reader(a.main.expect(2003)).s16() == 4, "offline player"

    step("2v2 wager battle (15): four players, two teams")
    c, d = Player("C"), Player("D")
    four = [a, b, c, d]
    before = {p.id: p.fresh_profile() for p in four}
    battle, order, teams, money = team_wager_match(four, 15)
    for p in four:
        assert money[p.id] == before[p.id]["money"] - 15, f"wager not taken from {p.name}"
    print("turn order:", [(p.name, teams[p.id]) for p in order])
    for seat, p in enumerate(order):
        num = p.pvp_login(battle, seat == 0, order[(seat + 1) % 4])
        assert num == seat + 1, f"{p.name}: player number {num}, want {seat + 1}"

    step("2v2 turn 1: moves go to all three others, then StartTurn for the second seat")
    play_turn_multi(order[0], order[1:], battle, 1)
    for p in four:
        expect_start_turn(p, battle, 2, order[1].id)

    step("2v2 chat reaches everyone")
    order[1].pvp.send(msg(17, struct.pack(">I", order[1].id) + utf("go team") + struct.pack(">I", battle)))
    for p in (order[0], order[2], order[3]):
        assert Reader(p.pvp.expect(17)).u32() == order[1].id

    step("2v2 turn 2: seat 2 kills seat 1; the client keeps its index, so seat 4 moves next")
    play_turn_multi(order[1], [order[0], order[2], order[3]], battle, 2, eliminated=[order[0].id])
    for p in four:
        expect_start_turn(p, battle, 3, order[3].id)

    step("2v2 turn 3: seat 4 kills seat 3: their team wins")
    play_turn_multi(order[3], order[:3], battle, 3, eliminated=[order[2].id])
    winners = {order[1].id, order[3].id}
    for p in four:
        desync, results = expect_end_battle(p, battle)
        assert not desync and results == {q.id: (0 if q.id in winners else 1) for q in four}, results
    for p in four:
        p.pvp.close()
    time.sleep(2)
    for p in four:
        got = p.fresh_profile()["money"]
        want = before[p.id]["money"] + (12 if p.id in winners else -15)
        assert got == want, f"{p.name}: money {got}, want {want}"
    c.main.close(); d.main.close()

    step("team: E invites F into a 2v2 team, F accepts")
    e, f, g, h = Player("E"), Player("F"), Player("G"), Player("H")
    e.main.send(msg(1002, utf("wormixtest.ru") + struct.pack(">iIi", PVP_PORT, 3, 1) + utf(str(f.id)) + bytes([1])))
    who, _, invite, kind = read_offer(f)
    assert (who, kind) == (e.id, 1), (who, kind)
    f.main.send(msg(1003, struct.pack(">hII", 0, e.id, invite)))
    assert Reader(e.main.expect(2003)).s16() == 0, "team invite accepted"

    team_req = utf("wormixtest.ru") + struct.pack(">iiI", PVP_PORT, 15, 5) + bytes([1])
    step("team: E looks for a 2v2 battle, F is called and declines: E stops waiting")
    e.main.send(msg(2005, team_req, secure=True))
    who, wager, call, kind = read_offer(f)
    assert (who, wager, kind) == (e.id, 15, 2), (who, wager, kind)
    f.main.send(msg(1003, struct.pack(">hII", 1, e.id, call)))
    assert Reader(e.main.expect(2008)).s16() == 10, "teammate declined"

    step("team: E looks again, F joins; two single players G and H make the other side")
    e.main.send(msg(2005, team_req, secure=True))
    _, _, call, kind = read_offer(f)
    assert kind == 2
    g.main.send(msg(2005, team_req, secure=True))
    h.main.send(msg(2005, team_req, secure=True))
    time.sleep(0.5)
    for p in (e, g, h):
        p.main.nothing(2006, 0.5)   # not before F has joined
    f.main.send(msg(1003, struct.pack(">hII", 0, e.id, call)))
    f.main.send(msg(2005, team_req, secure=True))
    sides = None
    for p in (e, f, g, h):
        r = Reader(p.main.expect(2006, 20))
        r.utf(); r.s32(); r.u32(); r.profile(); r.profile(); r.u32(); r.s32(); battle = r.u32()
        [r.u8() for _ in range(r.u16())]
        n = r.u8()
        seats = {}
        for _ in range(n):
            t = r.u8()
            seats[r.profile()["id"]] = t
        sides = sides or seats
        assert seats == sides
    assert sides[e.id] == sides[f.id] and sides[g.id] == sides[h.id] and sides[e.id] != sides[g.id], sides
    print("sides:", {p.name: sides[p.id] for p in (e, f, g, h)})
    for p in (e, f, g, h):
        p.pvp_login(battle, False, e)
    g.pvp.send(msg(1006, struct.pack(">IIi", g.id, battle, 1), secure=True))
    _, results = expect_end_battle(e, battle)
    assert results[e.id] == 0 and results[f.id] == 0 and results[g.id] == 1 and results[h.id] == 1, results
    for p in (e, f, g, h):
        p.pvp.close()

    step("team: E logs out, F is told the team is over")
    e.main.close()
    who, _, _, kind = read_offer(f)
    assert (who, kind) == (e.id, 3), (who, kind)
    for p in (f, g, h):
        p.main.close()

    step("no-show: B never comes back, A wins after the reconnect timeout (about 45 s)")
    battle = challenge(a, b)
    a.pvp_login(battle, True, b)
    b.pvp_login(battle, False, a)
    b.pvp.close()
    _, results = expect_end_battle_within(a, battle, 70)
    assert results == {a.id: 0, b.id: 1}, results

    print("\nALL PVP TESTS PASSED")


if __name__ == "__main__":
    try:
        main()
    except (AssertionError, ConnectionError, OSError) as e:
        print(f"\nFAILED: {e}", file=sys.stderr)
        sys.exit(1)
