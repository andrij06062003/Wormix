#!/bin/sh
# End-to-end verification of the Go API against the reference PHP API.
# Run inside a Docker-in-Docker environment where the repo is at /work, e.g.:
#   docker exec wormix-go-rewrite sh /work/services/api/test/run.sh
# Needs images wormix-api:ref (PHP) and wormix-api-go:test (docker build -f services/api/Dockerfile -t wormix-api-go:test .)
#
# 1. schema+seed: both APIs on empty volumes -> compare schema and every table
# 2. phase A: the PHP API creates accounts and plays (tmp/phaseA.sqlite, tmp/vars.json)
# 3. migrations on a ref DB lacking the last migration: both APIs -> compare
# 4. differential: PHP and Go each on a copy of the phase-A DB, same requests, compare
#    responses (difftest.py), then the resulting databases (dbcompare.py)
set -eu
cd /work/services/api/test
REF_IMAGE=${REF_IMAGE:-wormix-api:ref}
GO_IMAGE=${GO_IMAGE:-wormix-api-go:test}
NET=apitest-net
TMP=/work/services/api/test/tmp
PY="docker run --rm --network $NET -v /work:/work -w /work/services/api/test python:3.12-alpine python"
mkdir -p "$TMP"
rc=0

cleanup() {
  docker rm -f apitest-ref apitest-go apitest-refA >/dev/null 2>&1 || true
  for v in apitest-a apitest-b apitest-c apitest-e1 apitest-e2 apitest-m1 apitest-m2; do docker volume rm "$v" >/dev/null 2>&1 || true; done
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
start() { # name image volume
  docker run -d --name "$1" --network $NET -v "$3":/data "$2" >/dev/null
  $PY wait.py "http://$1:8000"
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
docker stop apitest-refA >/dev/null
get_db apitest-a "$TMP/phaseA.sqlite"
$PY tweak_db.py tmp/phaseA.sqlite

echo "=== 3. migrations on a ref DB without the last migration"
cp "$TMP/phaseA.sqlite" "$TMP/premig.sqlite"
$PY -c "
import sqlite3; c = sqlite3.connect('tmp/premig.sqlite')
c.execute('drop table wormix_users_achievements')
c.execute('alter table wormix_user_profiles drop column invested_award_points')
c.execute(\"delete from migrations where migration = '2026_11_05_120000_create_wormix_users_achievements_table'\")
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
# carol searched a house (random reward), dave's last PvP crate reagents are random.
$PY dbcompare.py tmp/diff-ref.sqlite tmp/diff-go.sqlite --normalize \
  --mask wormix_user_profiles.money@user_id=3 --mask wormix_user_profiles.real_money@user_id=3 \
  --mask wormix_user_profiles.reagents@user_id=3 --mask wormix_users_battle_info.awards@user_id=4 || rc=1

echo "=== result: $([ $rc = 0 ] && echo PASS || echo FAIL)"
exit $rc
