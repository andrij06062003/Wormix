#!/bin/sh
# End-to-end verification of the Go API against the reference PHP API.
# Run inside a Docker-in-Docker environment where the repo is at /work, e.g.:
#   docker exec wormix-go-rewrite sh /work/services/api/test/run.sh
# Needs images wormix-api:ref (PHP) and wormix-api-go:test (docker build -f services/api/Dockerfile -t wormix-api-go:test .)
#
# 1. schema+seed: both APIs on empty volumes -> compare schema and every table
# 2. phase A: the PHP API creates accounts and plays (tmp/phaseA.sqlite, tmp/vars.json)
# 3a. upgrade: a DB built by the pre-artifacts API ($OLD_IMAGE), started with each API -> compare
# 3. migrations on a ref DB lacking the last migration: both APIs -> compare
# 4. differential: PHP and Go each on a copy of the phase-A DB, same requests, compare
#    responses (difftest.py), then the resulting databases (dbcompare.py)
# 5. the same differential test with APP_ENV=dev
set -eu
cd /work/services/api/test
REF_IMAGE=${REF_IMAGE:-wormix-api:refdev}
# an API image from before the artifacts migration (builds a DB the upgrade must handle)
OLD_IMAGE=${OLD_IMAGE:-wormix-api:ref}
GO_IMAGE=${GO_IMAGE:-wormix-api-go:test}
NET=apitest-net
TMP=/work/services/api/test/tmp
PY="docker run --rm --network $NET -v /work:/work -w /work/services/api/test python:3.12-alpine python"
mkdir -p "$TMP"
rc=0

cleanup() {
  docker rm -f apitest-ref apitest-go apitest-refA >/dev/null 2>&1 || true
  for v in apitest-a apitest-b apitest-c apitest-d1 apitest-d2 apitest-e1 apitest-e2 apitest-m1 apitest-m2 apitest-o apitest-u1 apitest-u2; do docker volume rm "$v" >/dev/null 2>&1 || true; done
  docker network rm $NET >/dev/null 2>&1 || true
}
cleanup
trap cleanup EXIT
docker network create $NET >/dev/null

put_db() { # volume file
  docker volume create "$1" >/dev/null
  docker run --rm -v "$1":/data -v /work:/work alpine sh -c "cp $2 /data/database.sqlite"
}
get_db() { # volume file
  docker run --rm -v "$1":/data -v /work:/work alpine sh -c "cp /data/database.sqlite $2; if [ -f /data/database.sqlite-wal ]; then cp /data/database.sqlite-wal $2-wal; else rm -f $2-wal; fi"
}
start() { # name image volume [docker run options...]
  n=$1; i=$2; v=$3; shift 3
  docker run -d --name "$n" --network $NET -v "$v":/data "$@" "$i" >/dev/null
  $PY wait.py "http://$n:8000"
}

echo "=== 1. schema + seed on empty volumes"
docker volume create apitest-e1 >/dev/null; docker volume create apitest-e2 >/dev/null
start apitest-ref "$REF_IMAGE" apitest-e1
start apitest-go "$GO_IMAGE" apitest-e2
docker stop apitest-ref apitest-go >/dev/null
get_db apitest-e1 "$TMP/empty-ref.sqlite"; get_db apitest-e2 "$TMP/empty-go.sqlite"
docker rm apitest-ref apitest-go >/dev/null
$PY dbcompare.py tmp/empty-ref.sqlite tmp/empty-go.sqlite || rc=1

echo "=== 2. phase A with the PHP API"
docker volume create apitest-a >/dev/null
start apitest-refA "$REF_IMAGE" apitest-a
$PY setup_phase_a.py http://apitest-refA:8000 tmp/vars.json
docker stop apitest-refA >/dev/null; docker rm apitest-refA >/dev/null
get_db apitest-a "$TMP/phaseA.sqlite"
$PY tweak_db.py tmp/phaseA.sqlite

echo "=== 3a. upgrade: DB built by the API before artifacts ($OLD_IMAGE), then started with each API"
docker volume create apitest-o >/dev/null
start apitest-refA "$OLD_IMAGE" apitest-o
$PY setup_phase_a.py http://apitest-refA:8000 tmp/vars-old.json
docker stop apitest-refA >/dev/null; docker rm apitest-refA >/dev/null
get_db apitest-o "$TMP/old.sqlite"
put_db apitest-u1 "$TMP/old.sqlite"; put_db apitest-u2 "$TMP/old.sqlite"
start apitest-ref "$REF_IMAGE" apitest-u1
start apitest-go "$GO_IMAGE" apitest-u2
docker stop apitest-ref apitest-go >/dev/null
get_db apitest-u1 "$TMP/up-ref.sqlite"; get_db apitest-u2 "$TMP/up-go.sqlite"
docker rm apitest-ref apitest-go >/dev/null
$PY dbcompare.py tmp/up-ref.sqlite tmp/up-go.sqlite --normalize || rc=1
$PY -c "
import sqlite3
for f in ['tmp/up-ref.sqlite','tmp/up-go.sqlite']:
    c = sqlite3.connect(f)
    print(f, c.execute('select migration, batch from migrations order by id desc limit 1').fetchall(), c.execute('select count(*) from wormix_weapons where id > 2000').fetchall())"

echo "=== 3. migrations on a ref DB without the last migration"
cp "$TMP/phaseA.sqlite" "$TMP/premig.sqlite"
$PY -c "
import sqlite3; c = sqlite3.connect('tmp/premig.sqlite')
c.execute('delete from wormix_users_weapons where weapon_id > 2000')
c.execute('delete from wormix_weapons where id > 2000')
c.execute('alter table wormix_worms_data drop column artifact')
c.execute(\"delete from migrations where migration = '2026_11_06_120000_add_wormix_artifacts'\")
c.commit()"
put_db apitest-m1 "$TMP/premig.sqlite"; put_db apitest-m2 "$TMP/premig.sqlite"
start apitest-ref "$REF_IMAGE" apitest-m1
start apitest-go "$GO_IMAGE" apitest-m2
docker stop apitest-ref apitest-go >/dev/null
get_db apitest-m1 "$TMP/mig-ref.sqlite"; get_db apitest-m2 "$TMP/mig-go.sqlite"
docker rm apitest-ref apitest-go >/dev/null
$PY dbcompare.py tmp/mig-ref.sqlite tmp/mig-go.sqlite --normalize || rc=1
$PY -c "
import sqlite3
for f in ['tmp/mig-ref.sqlite','tmp/mig-go.sqlite']:
    print(f, sqlite3.connect(f).execute('select migration, batch from migrations order by id desc limit 2').fetchall())"

echo "=== 4. differential test on copies of the phase-A DB"
put_db apitest-b "$TMP/phaseA.sqlite"; put_db apitest-c "$TMP/phaseA.sqlite"
start apitest-ref "$REF_IMAGE" apitest-b
start apitest-go "$GO_IMAGE" apitest-c
$PY difftest.py http://apitest-ref:8000 http://apitest-go:8000 tmp/vars.json || rc=1
docker stop apitest-ref apitest-go >/dev/null
get_db apitest-b "$TMP/diff-ref.sqlite"; get_db apitest-c "$TMP/diff-go.sqlite"
docker rm apitest-ref apitest-go >/dev/null
# carol searched a house (random reward), dave's last PvP crate reagents are random.
$PY dbcompare.py tmp/diff-ref.sqlite tmp/diff-go.sqlite --normalize \
  --mask wormix_user_profiles.money@user_id=3 --mask wormix_user_profiles.real_money@user_id=3 \
  --mask wormix_user_profiles.reagents@user_id=3 --mask wormix_users_battle_info.awards@user_id=4 || rc=1

echo "=== 5. differential test with APP_ENV=dev (dev starters, dev achievements)"
put_db apitest-d1 "$TMP/phaseA.sqlite"; put_db apitest-d2 "$TMP/phaseA.sqlite"
start apitest-ref "$REF_IMAGE" apitest-d1 -e APP_ENV=dev
start apitest-go "$GO_IMAGE" apitest-d2 -e APP_ENV=dev
docker run --rm --network $NET -e DIFF_NO_EXPECT=1 -v /work:/work -w /work/services/api/test python:3.12-alpine \
  python difftest.py http://apitest-ref:8000 http://apitest-go:8000 tmp/vars.json > "$TMP/dev.log" 2>&1 || rc=1
grep -E "^FAIL|steps" "$TMP/dev.log" -A4 | head -60
docker stop apitest-ref apitest-go >/dev/null
get_db apitest-d1 "$TMP/dev-ref.sqlite"; get_db apitest-d2 "$TMP/dev-go.sqlite"
$PY dbcompare.py tmp/dev-ref.sqlite tmp/dev-go.sqlite --normalize \
  --mask wormix_user_profiles.money@user_id=3 --mask wormix_user_profiles.real_money@user_id=3 \
  --mask wormix_user_profiles.reagents@user_id=3 --mask wormix_users_battle_info.awards@user_id=4 || rc=1

echo "=== result: $([ $rc = 0 ] && echo PASS || echo FAIL)"
exit $rc
