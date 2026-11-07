#!/usr/bin/env bash
# Runs the PvP end-to-end test against a throwaway stack (api, core) in its own compose project.
# Nothing is published on the host, so it doesn't touch a running server.
#
#   tests/pvp/run.sh          # build, test, remove everything
#   KEEP=1 tests/pvp/run.sh   # leave the stack running afterwards (docker compose -p wormix-pvptest down -v)
set -euo pipefail
cd "$(dirname "$0")/../.."

export COMPOSE_PROJECT_NAME=wormix-pvptest IMAGE_TAG=pvptest COMPOSE_PROFILES= CLOUDFLARE_TUNNEL_TOKEN=unused
compose() { docker compose -f docker-compose.yml "$@"; }
cleanup() { [ "${KEEP:-}" = 1 ] || compose down -v --remove-orphans >/dev/null 2>&1 || true; }
trap cleanup EXIT

compose up -d --build api core
network="${COMPOSE_PROJECT_NAME}_default"

set +e
docker run --rm --network "$network" -v "$PWD/tests/pvp:/t:ro" python:3.12-alpine python -u /t/pvp_smoke.py
status=$?
[ $status -eq 0 ] || compose logs --tail 80 core api
exit $status
