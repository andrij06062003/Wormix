#!/usr/bin/env python3
"""Differential test driver, realistic part: game flows against a fresh reference API (wormix-api:ref).

Registers accounts through the public API (/api/account/sign_up, /api/account/game for the game auth key),
logs in to the main server like the client does, and plays: arena, mission battle, shop, team, house, craft,
wipe, re-login, a friend challenge with a PvP battle, and the achievements server. Every server message is
recorded byte for byte, except values that differ between runs by nature and are replaced by placeholders:
  - the API session key in EnterAccount (random) and the MD5 that covers it (checked, then dropped);
  - PvP battle ids (taken from the clock);
  - replies whose content the API randomizes (arena bots, mission reagents, house search): their layout is
    checked instead and only the deterministic fields are recorded.

  python3 diff_real.py <core host> <api base url, e.g. http://api:8000/api/>
"""
import hashlib
import json
import struct
import sys
import time
import urllib.error
import urllib.request

from diff_fake import Conn, msg, utf, u16, u32, parse_frames

CORE = sys.argv[1] if len(sys.argv) > 1 else "core"
API = sys.argv[2] if len(sys.argv) > 2 else "http://api:8000/api/"
SECRET = b"ygfA69i3Uz6bDO8"
out = []
battle_ids = {}


def emit(line):
    out.append(line)


def api(path, body=None, token=None):
    req = urllib.request.Request(API + path, data=json.dumps(body or {}).encode(), method="POST",
                                 headers={"Content-Type": "application/json", "Accept": "application/json",
                                          **({"Authorization": "Bearer " + token} if token else {})})
    with urllib.request.urlopen(req, timeout=30) as r:
        return json.loads(r.read())


class R:
    def __init__(self, b, p=0):
        self.b, self.p = b, p

    def take(self, n):
        v = self.b[self.p:self.p + n]
        assert len(v) == n, "truncated"
        self.p += n
        return v

    def u8(self): return self.take(1)[0]
    def u16(self): return struct.unpack(">H", self.take(2))[0]
    def u32(self): return struct.unpack(">I", self.take(4))[0]
    def utf(self): return self.take(self.u16())

    def profile(self):
        start = self.p
        self.take(16)
        for _ in range(self.u16()):
            self.take(22); self.utf(); self.take(2)   # + Artifact (artifacts port)
        self.take(8 * self.u16())
        self.take(2 * self.u16())
        self.take(4); self.utf()
        self.take(2 * self.u16())
        return self.b[start:self.p]


def frames(data):
    res, p = [], 0
    while len(data) - p >= 7:
        _, rid, rlen = struct.unpack_from(">BHI", data, p)
        n = rlen & 0x3FFFFFFF
        res.append((rid & 0x7FFF, data[p + 7:p + 7 + n]))
        p += 7 + n
    return res


def secure_ok(body):
    return hashlib.md5(body[:-16] + SECRET).digest() == body[-16:]


def norm_ids(body):
    """Replace PvP battle ids; an MD5 suffix over a replaced id is checked and replaced too."""
    hexs = body.hex()
    found = False
    for real, name in battle_ids.items():
        if struct.pack(">I", real).hex() in hexs:
            found = True
            hexs = hexs.replace(struct.pack(">I", real).hex(), name)
    if found and len(body) > 16 and secure_ok(body):
        hexs = hexs[:-32] + "<md5 ok>"
    return hexs


def describe(data):
    """Record a reply: byte for byte, except the per-run values described in the module docstring."""
    lines = []
    valid = parse_frames(data)
    if any("INVALID" in l or "tail" in l for l in valid):
        return valid
    for cmd, body in frames(data):
        if cmd == 10001:   # EnterAccount: session key is random
            assert secure_ok(body), "EnterAccount md5"
            r = R(body)
            r.take(2); r.profile()
            for _ in range(r.u16()):
                r.take(2); r.profile()
            r.take(2 + 12 + 4)
            kpos = r.p
            key = r.utf()
            assert b"." in key
            body = body[:kpos] + b"<KEY>" + body[r.p:-16]
            lines.append(f"  cmd {cmd} len {len(key)} body {body.hex().replace('3c4b45593e', '<KEY>')} (md5 ok)")
        elif cmd == 10004:   # ArenaResult: bot profiles are random
            r = R(body)
            n = r.u16()
            for _ in range(n):
                size = r.u16(); prof = r.profile(); assert len(prof) == size
            rest = body[r.p:]
            assert len(rest) == 7
            lines.append(f"  cmd {cmd} arena with {n} profiles, then {rest.hex()}")
        elif cmd == 10006:   # StartBattleResult: reagents are random
            assert secure_ok(body)
            r = R(body)
            awards = [r.take(10).hex() for _ in range(r.u16())]
            bid = r.u32()
            nreag = r.u16(); r.take(nreag)
            assert r.p == len(body) - 16
            lines.append(f"  cmd {cmd} awards {awards} battle {bid} reagents ({nreag}) (md5 ok)")
        elif cmd == 10021:   # SearchTheHouseResult: what's found is random
            assert secure_ok(body) and len(body) == 26
            lines.append(f"  cmd {cmd} search result keys {body[5]} friend {body[6:10].hex()} (md5 ok)")
        else:
            lines.append(f"  cmd {cmd} len {len(body)} body {norm_ids(body)}")
    return lines


class Player:
    def __init__(self, n):
        r = api("account/sign_up", {"login": f"difftest{n}", "password": "password123", "password_confirmation": "password123"})
        self.id, self.token = r["id"], r["auth_token"]
        self.main = None

    def login(self):
        self.auth_key = api("account/game", token=self.token)["auth_key"]
        self.main = Conn()
        body = u32(self.id) + u32(0) + utf(self.auth_key) + u16(0) + bytes([1])
        self.main.send(msg(1, body))
        return self.main.collect()


def step(p, name, *messages):
    emit(f"== {name}")
    for m in messages:
        p.main.send(m)
    for line in describe(p.main.collect()):
        emit(line)
    if p.main.closed:
        emit("  CONNECTION CLOSED; logging in again")
        p.main.close()
        time.sleep(0.3)
        for line in describe(p.login()):
            emit("  " + line)


def wait_for_stack(timeout=240):
    deadline = time.time() + timeout
    while True:
        try:
            Conn().close()
            try:
                urllib.request.urlopen(API.rsplit("/api/", 1)[0] + "/", timeout=5)
            except urllib.error.HTTPError:
                pass
            return
        except OSError:
            if time.time() > deadline:
                raise
            time.sleep(2)


def main():
    wait_for_stack()
    a, b, c = Player(1), Player(2), Player(3)
    emit(f"== accounts {a.id} {b.id} {c.id}")
    emit("== login A")
    for line in describe(a.login()):
        emit(line)
    emit("== login B")
    for line in describe(b.login()):
        emit(line)

    step(a, "need money", msg(2, u32(100) + u32(1)))
    step(a, "arena", msg(4, b"\x01"))
    step(a, "start mission 0", msg(6, u16(0)))
    end = (utf("result") + u32(1001 + a.id) + utf("type") + u32(0) + utf("exp") + u32(0) + utf("id") + u32(1000 + a.id)
           + u16(0) + u16(0) + u16(0) + u16(0) + utf("") + u16(0) + b"\x00" * 16)
    step(a, "end battle (win)", msg(84, end))
    step(a, "arena again", msg(4, b"\x00"))
    step(a, "start mission 1", msg(6, u16(1)))
    end1 = (utf("result") + u32(1001 + a.id + 1) + utf("type") + u32(0) + utf("exp") + u32(0) + utf("id") + u32(1001 + a.id)
            + u16(1) + u16(1) + b"\x08\x00" + u32(1) + u32(1) + u16(0) + u16(0) + utf("") + u16(0) + b"\x00" * 16)
    step(a, "end battle (mission 1, used a weapon)", msg(84, end1))
    step(a, "shop: buy weapon 1 x3 for money", msg(3, u16(1) + u16(12) + u32(1) + u32(3) + u32(1)))
    step(a, "shop: too expensive", msg(3, u16(1) + u16(12) + u32(53) + u32(1) + u32(1)))
    step(a, "shop: real money", msg(3, u16(1) + u16(12) + u32(16) + u32(1) + u32(0)))
    step(a, "buy battle", msg(11, u32(1)))
    step(a, "distribute points", msg(14, u32(1) + u32(1)))
    step(a, "reset parameters", msg(15, u32(1)))
    step(a, "select stuff", msg(25, u16(1)))
    step(a, "change race", msg(36, b"\x01\x01"))
    step(a, "buy reaction", msg(49, b"\x01"))
    step(a, "unlock mission", msg(58, u16(2)))
    step(a, "team: add B", msg(12, u32(b.id) + u16(1)))
    step(a, "team: reorder", msg(22, u16(2) + u32(a.id) + u32(b.id)))
    step(a, "team: remove B", msg(13, u32(b.id)))
    step(a, "house: search B", msg(81, utf(f"{a.id}") + u32(b.id) + b"\x01"))
    step(a, "house: pump B", msg(83, u32(b.id)))
    step(a, "house: pump B and C", msg(82, u16(2) + u32(b.id) + u32(c.id) + b"\x00" * 16))
    step(b, "who pumped (B)", msg(46, b"\x00"))
    step(b, "who pumped today (B)", msg(46, b"\x01"))
    step(a, "craft: upgrade", msg(86, u16(1)))
    step(a, "craft: downgrade", msg(88, u16(1)))
    step(a, "wipe without code", msg(48, utf("0000")))
    emit("== re-login A (profile after the changes)")
    a.main.close()
    time.sleep(0.5)
    for line in describe(a.login()):
        emit(line)

    # Friend challenge and a short battle on the PvP server.
    step(a, "challenge B", msg(1002, utf("wormixtest.ru") + u32(49198) + u32(3) + u32(42) + utf(str(b.id))))
    offer = b.main.collect()
    for cmd, body in frames(offer):
        if cmd == 2002:
            bid = struct.unpack(">I", body[-4:])[0]
            battle_ids[bid] = "<BATTLE>"
    emit("== B gets the offer")
    for line in describe(offer):
        emit(line)
    b.main.send(msg(1003, struct.pack(">hII", 0, a.id, bid)))
    b.main.collect()
    emit("== A gets the result")
    for line in describe(a.main.collect()):
        emit(line)
    pa, pb = Conn(49198), Conn(49198)
    for p, conn, first, enemy in ((a, pa, True, b), (b, pb, False, a)):
        conn.send(msg(1001, struct.pack(">I?", p.id, first) + utf(p.auth_key) + struct.pack(">HI", 1, enemy.id)
                      + utf("wormixtest.ru") + struct.pack(">iIB", 49197, bid, 1)))
        emit(f"== PvP login {'A' if first else 'B'}")
        for line in describe(conn.collect()):
            emit(line)
    body = struct.pack(">IIHHIBH", 10, 12, 1, 1, bid, 1, 3) + struct.pack(">III", 10, 1, 4096)
    body += hashlib.md5(body + SECRET).digest()
    pa.send(msg(1007, body))
    time.sleep(0.5)
    emit("== B gets A's move")
    for line in describe(pb.collect()):
        emit(line)
    pb.send(msg(17, struct.pack(">I", b.id) + utf("gl") + struct.pack(">I", bid)))
    time.sleep(0.5)
    emit("== A gets B's chat")
    for line in describe(pa.collect()):
        emit(line)
    drop = struct.pack(">IIi", b.id, bid, 1)
    pb.send(msg(1006, drop + hashlib.md5(drop + SECRET).digest()))
    emit("== B surrenders: A")
    time.sleep(0.5)
    for line in describe(pa.collect()):
        emit(line)
    emit("== B surrenders: B")
    for line in describe(pb.collect()):
        emit(line)
    pa.close(); pb.close()
    time.sleep(1.5)
    emit("== re-login B (after the battle)")
    b.main.close()
    time.sleep(0.5)
    for line in describe(b.login()):
        emit(line)

    emit("== achievements server")
    ach = Conn(49199)
    ach.send(msg(3001, utf("wormix") + utf("vk") + utf(str(a.id)) + utf(a.auth_key) + b"\x01"))
    for line in parse_frames(ach.collect(ping=False, idle=1.0, limit=2)):
        emit(line)
    emit(f"  closed: {ach.closed}")
    print("\n".join(out))


if __name__ == "__main__":
    main()
