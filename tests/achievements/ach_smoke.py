"""Achievements end-to-end test: speaks the game's binary HTTP protocol to the relay.

    python3 tests/achievements/ach_smoke.py [relay url]      (default http://127.0.0.1:32700)

Signs up a throwaway account, so run it against a test stack, not the live server.
"""
import hashlib, json, random, struct, sys, urllib.request

BASE = (sys.argv[1] if len(sys.argv) > 1 else "http://127.0.0.1:32700").rstrip("/")
KEY = b"ygfA69i3Uz6bDO8"
URL = BASE + "/wormix_achieve/"

def utf(s): b = s.encode(); return struct.pack(">H", len(b)) + b
def secure(b): return b + hashlib.md5(b + KEY).digest()

def header(cid, length):
    flags, rid, rlen = 87, cid, length
    if random.random() < .5: flags |= 128
    else: rid |= 0x8000
    if random.random() < .5: flags |= 32
    else: rlen |= 0x40000000
    if random.random() < .5: flags |= 8
    else: rlen |= 0x80000000
    return struct.pack(">BHI", flags, rid, rlen)

def post(cmds, sid=None):
    data = b"".join(header(c, len(b)) + b for c, b in cmds)
    req = urllib.request.Request(URL, data=data, method="POST")
    if sid: req.add_header("X-PRAGMATIX-CORP-SID", sid)
    return urllib.request.urlopen(req).read()

def parse(data):
    out, p = [], 0
    while p < len(data):
        flags, rid, rlen = struct.unpack_from(">BHI", data, p); p += 7
        assert flags & 87 == 87
        assert bool(flags & 128) != bool(rid & 0x8000)
        assert bool(flags & 32) != bool(rlen & 0x40000000)
        assert bool(flags & 8) != bool(rlen & 0x80000000)
        n = rlen & 0x3FFFFFFF
        out.append((rid & 0x7FFF, data[p:p + n])); p += n
    return out

class R:
    def __init__(s, b): s.b, s.p = b, 0
    def take(s, n): v = s.b[s.p:s.p + n]; s.p += n; return v
    def u8(s): return s.take(1)[0]
    def u16(s): return struct.unpack(">H", s.take(2))[0]
    def i16(s): return struct.unpack(">h", s.take(2))[0]
    def i32(s): return struct.unpack(">i", s.take(4))[0]
    def u32(s): return struct.unpack(">I", s.take(4))[0]
    def utf(s): return s.take(s.u16()).decode()
    def bytes_(s): return [s.u8() for _ in range(s.u16())]
    def ushorts(s): return [s.u16() for _ in range(s.u16())]

def check_md5(body):
    assert hashlib.md5(body[:-16] + KEY).digest() == body[-16:], "bad md5 from server"

def signup():
    login = "achtest%d" % random.randrange(10**9)
    body = json.dumps({"login": login, "password": "test-pass-1", "password_confirmation": "test-pass-1"}).encode()
    req = urllib.request.Request(BASE + "/api/account/sign_up", data=body, method="POST",
                                 headers={"Content-Type": "application/json", "Accept": "application/json"})
    return json.loads(urllib.request.urlopen(req).read())["auth_token"]

TOKEN = signup()

def api(path, method="GET"):
    req = urllib.request.Request(BASE + "/api/account" + path, method=method, data=b"" if method == "POST" else None,
                                 headers={"Authorization": "Bearer " + TOKEN, "Accept": "application/json"})
    return json.loads(urllib.request.urlopen(req).read())

def login(uid, key):
    body = utf("wormix") + utf("vk") + utf(uid) + utf(key) + b"\x01"
    [(cid, b)] = parse(post([(3001, body)]))
    return cid, b

def increase(sid, ts, pairs):
    body = utf(sid) + struct.pack(">H", len(pairs)) + bytes(i for i, _ in pairs) \
        + struct.pack(">H", len(pairs)) + b"".join(struct.pack(">h", v) for _, v in pairs) + struct.pack(">I", ts)
    [(cid, b)] = parse(post([(3003, secure(body))], sid))
    assert cid == 13004, cid
    check_md5(b)
    r = R(b)
    res = dict(indexes=r.bytes_(), values=r.ushorts(), thr=r.bytes_(), thr_old=r.ushorts(), ts=r.u32(), invested=r.u8())
    return res

def get(sid, pid):
    [(cid, b)] = parse(post([(3002, secure(utf(pid) + b"\x00"))], sid))
    assert cid == 13003, cid
    check_md5(b)
    r = R(b)
    return dict(pid=r.utf(), indexes=r.bytes_(), values=r.ushorts(), invested=r.u8())

def choose(sid, item):
    [(cid, b)] = parse(post([(3005, secure(struct.pack(">i", item)))], sid))
    assert cid == 13006, cid
    check_md5(b)
    r = R(b)
    return r.i16(), r.i32()

me = api("")
uid = str(me.get("data", me).get("id", me.get("id")))
print("user", uid)

cid, b = login(uid, "wrong")
assert cid == 13002, cid
print("ok  bad auth key -> AchieveLoginError")

key = api("/game", "POST")["auth_key"]
cid, b = login(uid, key)
assert cid == 13001, cid
check_md5(b)
r = R(b); sid = r.utf(); t = r.u32()
print("ok  login ->", sid[:8] + "...", t)

g = get(sid, uid)
assert g == dict(pid=uid, indexes=[], values=[], invested=0), g
print("ok  get (empty)")

[(cid, _)] = parse(post([(16, b"")], sid))
assert cid == 10017
print("ok  ping -> pong")

res = increase(sid, t + 1, [(23, 5)])
assert res == dict(indexes=[23], values=[5], thr=[23], thr_old=[0], ts=t + 1, invested=0), res
again = increase(sid, t + 1, [(23, 5)])
assert again == res, again
print("ok  increase GAME_VISITS +5 (level 1), resend is not counted twice")

res = increase(sid, t + 2, [(23, 15), (0, 10), (99, 3), (5, -4)])
assert res["indexes"] == [23, 0] and res["values"] == [20, 10] and res["thr"] == [23] and res["thr_old"] == [5], res
print("ok  increase +15 crosses levels 2 and 3, unknown/negative entries ignored")

assert choose(sid, 62) == (2, 62)
print("ok  choose bonus without 1000 points -> MIN_REQUIREMENTS")

res = increase(sid, t + 3, [(1, 30000), (3, 1000)])
assert res["values"] == [30000, 1000] and res["thr"] == [1, 3], res
assert choose(sid, 64) == (2, 64)      # level 2 item locked until one award is taken
assert choose(sid, 62) == (0, 62)
assert choose(sid, 63) == (2, 63)      # only one award per 1000 points
print("ok  choose bonus: level lock, success, point limit")

g = get(sid, uid)
assert g["invested"] == 1 and dict(zip(g["indexes"], g["values"])) == {0: 10, 1: 30000, 3: 1000, 23: 20}, g
print("ok  get after progress", g)

assert post([(3002, secure(utf(uid) + b"\x00"))], "0" * 32).startswith(b"401")
print("ok  unknown session -> 401 (client logs in again)")

assert post([(3002, utf(uid) + b"\x00" + b"\x00" * 16)], sid).startswith(b"400")
print("ok  bad md5 -> 400")
print("ALL OK")
