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
c.commit()
print("tweaked")
