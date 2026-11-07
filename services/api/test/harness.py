"""HTTP + normalization helpers shared by the differential tests (stdlib only)."""
import json
import re
import struct
import urllib.error
import urllib.request
import uuid
import zlib


def png(w, h):
    """A valid RGB PNG of w x h."""
    rows = b""
    for y in range(h):
        rows += b"\x00" + bytes(((x * 3 + y) % 256 for x in range(w * 3)))
    def chunk(t, d):
        return struct.pack(">I", len(d)) + t + d + struct.pack(">I", zlib.crc32(t + d) & 0xFFFFFFFF)
    return (b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 2, 0, 0, 0))
            + chunk(b"IDAT", zlib.compress(rows)) + chunk(b"IEND", b""))


def multipart(fields, files):
    b = uuid.uuid4().hex
    out = b""
    for k, v in fields.items():
        out += f"--{b}\r\nContent-Disposition: form-data; name=\"{k}\"\r\n\r\n{v}\r\n".encode()
    for k, (fname, ctype, data) in files.items():
        out += (f"--{b}\r\nContent-Disposition: form-data; name=\"{k}\"; filename=\"{fname}\"\r\n"
                f"Content-Type: {ctype}\r\n\r\n").encode() + data + b"\r\n"
    out += f"--{b}--\r\n".encode()
    return out, f"multipart/form-data; boundary={b}"


def send(base, method, path, json_body=None, form=None, raw=None, files=None, headers=None):
    headers = dict(headers or {})
    data = None
    if files is not None:
        data, ct = multipart(form or {}, files)
        headers["Content-Type"] = ct
    elif form is not None:
        data = "&".join(f"{urllib.request.quote(str(k))}={urllib.request.quote(str(v))}" for k, v in form.items()).encode()
        headers.setdefault("Content-Type", "application/x-www-form-urlencoded")
    elif raw is not None:
        data = raw if isinstance(raw, bytes) else raw.encode()
    elif json_body is not None:
        data = json.dumps(json_body).encode()
        headers.setdefault("Content-Type", "application/json")
    req = urllib.request.Request(base + path, data=data, method=method, headers=headers)
    try:
        resp = urllib.request.urlopen(req, timeout=60)
        status, body, h = resp.status, resp.read(), resp.headers
    except urllib.error.HTTPError as e:
        status, body, h = e.code, e.read(), e.headers
    parsed = None
    ctype = h.get("Content-Type", "")
    if "json" in ctype:
        try:
            parsed = json.loads(body)
        except Exception:
            parsed = ("<invalid json>", body[:200])
    return status, ctype, body, parsed


ISO = re.compile(r"^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d\.\d{6}Z$")
HEX64 = re.compile(r"^[0-9a-f]{64}$")
HEX32 = re.compile(r"^[0-9a-f]{32}$")
TOKEN = re.compile(r"^\d+\|[A-Za-z0-9]{40}[0-9a-f]{8}$")
SKEY = re.compile(r"^-?\d+\.[0-9a-f]{64}$")
UUIDF = re.compile(r"[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\.(png|jpg)")


def norm(v, hosts=()):
    """Replace values that legitimately differ between runs."""
    if isinstance(v, dict):
        return {k: norm(x, hosts) for k, x in v.items()}
    if isinstance(v, list):
        return [norm(x, hosts) for x in v]
    if isinstance(v, bool):
        return v
    if isinstance(v, int) and 1_700_000_000 <= v <= 2_100_000_000:
        return "<unixtime>"
    if isinstance(v, int) and 1_700_000_000_000 <= abs(v) <= 2_100_000_000_000:
        return "<unixtime-ms>"
    if isinstance(v, str):
        if ISO.match(v):
            return "<iso-ts>"
        if HEX64.match(v):
            return "<hex64>"
        if HEX32.match(v):
            return "<hex32>"
        if TOKEN.match(v):
            return "<sanctum-token>"
        if SKEY.match(v):
            return "<session-key>"
        for h in hosts:
            v = v.replace(h, "<host>")
        v = UUIDF.sub("<uuid-file>", v)
    return v


def shape(v):
    """Type structure only (for random content)."""
    if isinstance(v, dict):
        return {k: shape(x) for k, x in v.items()}
    if isinstance(v, list):
        shapes = []
        for x in v:
            s = shape(x)
            if s not in shapes:
                shapes.append(s)
        return ["list-of", shapes]
    return type(v).__name__


def shape_eq(a, b):
    """Shapes equal, where an empty list matches a list of any element shape."""
    if isinstance(a, list) and isinstance(b, list) and len(a) == 2 and a[:1] == ["list-of"] and b[:1] == ["list-of"]:
        if not a[1] or not b[1]:
            return True
        return len(a[1]) == len(b[1]) and all(shape_eq(x, y) for x, y in zip(a[1], b[1]))
    if isinstance(a, dict) and isinstance(b, dict):
        return a.keys() == b.keys() and all(shape_eq(a[k], b[k]) for k in a)
    return a == b
