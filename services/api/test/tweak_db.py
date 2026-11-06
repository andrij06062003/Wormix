"""Adjust the phase-A database so more code paths run (applied to both copies alike):
dave last logged in yesterday (daily bonus sequence), carol two days ago (sequence reset),
alice owns an expired one-day hat that she wears, frank's battles are used up."""
import sqlite3
import sys
import time
from datetime import datetime, timedelta, timezone

c = sqlite3.connect(sys.argv[1])
now = datetime.now(timezone.utc)
day = lambda n: (now - timedelta(days=n)).strftime("%Y-%m-%d 00:00:00")
c.execute("update wormix_login_sequence set last_login = ?, login_sequence = 2 where user_id = 4", (day(1),))
c.execute("update wormix_login_sequence set last_login = ? where user_id = 3", (day(2),))
c.execute("insert into wormix_users_weapons (owner_id, weapon_id, count, expire_at, created_at, updated_at) values (1, 1003, 1, ?, ?, ?)",
          (int(time.time()) - 100, day(0), day(0)))
c.execute("update wormix_worms_data set hat = 1003 where owner_id = 1")
c.execute("update wormix_users_battle_info set battles_count = 0, last_battle_time = ? where user_id = 5", (int(time.time()),))
# Artifacts (only when the DB has the artifacts migration): dave wears an expired one, carol one she
# doesn't own, frank one he owns for good; frank's balances are negative (must show as 0).
cols = [r[1] for r in c.execute("pragma table_info(wormix_worms_data)")]
if "artifact" in cols:
    c.execute("insert into wormix_users_weapons (owner_id, weapon_id, count, expire_at, created_at, updated_at) values (4, 2003, -1, ?, ?, ?)",
              (int(time.time()) - 50, day(0), day(0)))
    c.execute("insert into wormix_users_weapons (owner_id, weapon_id, count, expire_at, created_at, updated_at) values (5, 2001, -1, -1, ?, ?)", (day(0), day(0)))
    c.execute("update wormix_worms_data set artifact = 2003 where owner_id = 4")
    c.execute("update wormix_worms_data set artifact = 2005 where owner_id = 3")
    c.execute("update wormix_worms_data set artifact = 2001 where owner_id = 5")
    c.execute("update wormix_user_profiles set money = -10, real_money = -5 where user_id = 5")
c.commit()
print("tweaked")
