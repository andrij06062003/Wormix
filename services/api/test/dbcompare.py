"""Compare two Wormix SQLite databases: schema (sqlite_master, in order) and the full
content of every table. Timestamps are ignored (only NULL vs non-NULL is compared).

usage: python dbcompare.py A.sqlite B.sqlite [--ignore table.column ...] [--mask table.col@keycol=value ...]
                           [--tables t1,t2] [--normalize]
--normalize replaces values that are random by design (bcrypt hashes, sha256/hex keys,
uuid photo names, unix times) with placeholders.
exit code 0 when equal.
"""
import re
import sqlite3
import sys

SUBS = [
    (re.compile(r"\$2y\$12\$[./A-Za-z0-9]{53}"), "<bcrypt>"),
    (re.compile(r"[0-9a-f]{64}"), "<h64>"),
    (re.compile(r"(?<![0-9a-f])[0-9a-f]{32}(?![0-9a-f])"), "<h32>"),
    (re.compile(r"[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\.(png|jpg)"), "<uuid-file>"),
]
NORMALIZE = False
MASKS = []


def normv(v):
    if not NORMALIZE:
        return v
    if isinstance(v, int) and 1_700_000_000 <= v <= 2_100_000_000:
        return "<time>"
    if isinstance(v, str):
        for rx, rep in SUBS:
            v = rx.sub(rep, v)
    return v

TS_COLS = {"created_at", "updated_at", "last_used_at", "failed_at"}


def schema(c):
    return [r for r in c.execute("select type, name, tbl_name, sql from sqlite_master where name != 'sqlite_sequence' order by rowid")]


def table_rows(c, t, ignore):
    cols = [r[1] for r in c.execute(f'pragma table_info("{t}")')]
    rows = []
    for r in c.execute(f'select * from "{t}" order by rowid'):
        row = []
        rowd = dict(zip(cols, r))
        for col, v in zip(cols, r):
            masked = any(mt == t and mc == col and str(rowd.get(kc)) == kv for mt, mc, kc, kv in MASKS)
            if f"{t}.{col}" in ignore or masked:
                v = "<ignored>"
            elif col in TS_COLS:
                v = None if v is None else "<ts>"
            row.append((col, normv(v), type(v).__name__))
        rows.append(row)
    return cols, rows


def main():
    a, b = sys.argv[1], sys.argv[2]
    ignore, tables = set(), None
    args = sys.argv[3:]
    while args:
        x = args.pop(0)
        if x == "--ignore":
            ignore.add(args.pop(0))
        elif x == "--normalize":
            global NORMALIZE
            NORMALIZE = True
        elif x == "--mask":
            spec = args.pop(0)
            tc, kv = spec.split("@")
            t, c = tc.split(".")
            k, val = kv.split("=")
            MASKS.append((t, c, k, val))
        elif x == "--tables":
            tables = args.pop(0).split(",")
    ca, cb = sqlite3.connect(a), sqlite3.connect(b)
    ok = True
    sa, sb = schema(ca), schema(cb)
    if sa != sb:
        ok = False
        print("SCHEMA DIFFERS")
        for x, y in zip(sa, sb):
            if x != y:
                print("  A:", x, "\n  B:", y)
        if len(sa) != len(sb):
            print("  lengths", len(sa), len(sb))
    names = tables or [r[0] for r in ca.execute("select name from sqlite_master where type='table' and name != 'sqlite_sequence' order by name")]
    for t in names:
        cola, ra = table_rows(ca, t, ignore)
        colb, rb = table_rows(cb, t, ignore)
        if cola != colb:
            ok = False
            print(f"COLUMNS DIFFER in {t}: {cola} vs {colb}")
            continue
        if ra != rb:
            ok = False
            print(f"DATA DIFFERS in {t}: {len(ra)} vs {len(rb)} rows")
            shown = 0
            for x, y in zip(ra, rb):
                if x != y and shown < 5:
                    print("  A:", x, "\n  B:", y)
                    shown += 1
        else:
            print(f"ok {t}: {len(ra)} rows")
    print("EQUAL" if ok else "DIFFERENT")
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
