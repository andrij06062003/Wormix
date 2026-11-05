#!/bin/bash
# Starts the sandbox's own Docker daemon, then the deployer as the main process.
# The Wormix containers inside are started again here; the deployer only touches them when a new
# commit on its branch has passed CI.
set -euo pipefail

log() { echo "$(date '+%Y-%m-%d %H:%M:%S') [${CD_ENV:-sandbox}] $*"; }

log "starting the sandbox Docker daemon"
dockerd-entrypoint.sh dockerd > /var/log/dockerd.log 2>&1 &
DOCKERD_PID=$!

for _ in $(seq 1 60); do
    docker info > /dev/null 2>&1 && break
    sleep 1
done
if ! docker info > /dev/null 2>&1; then
    log "ERROR: the Docker daemon did not start:"
    cat /var/log/dockerd.log
    exit 1
fi
log "Docker daemon ready"

# Bring the game back. stop() below stops its containers on shutdown, and Docker's "unless-stopped"
# then treats them as stopped on purpose, so they would not start again by themselves.
docker ps -aq --filter label=com.docker.compose.project=wormix | xargs -r docker start > /dev/null \
    && log "game containers started" || log "WARNING: could not start the game containers"

DEPLOYER_PID=""
stop() {
    log "stopping"
    [ -n "$DEPLOYER_PID" ] && kill -TERM "$DEPLOYER_PID" 2> /dev/null || true
    # Stop the game cleanly (SQLite flushed) but keep containers, so a restart is quick.
    docker ps -q --filter label=com.docker.compose.project=wormix | xargs -r docker stop -t 30 > /dev/null || true
    kill -TERM "$DOCKERD_PID" 2> /dev/null || true
    wait "$DOCKERD_PID" 2> /dev/null || true
    exit 0
}
trap stop TERM INT

# A deployer that came with a deployed release (self-update) wins over the one in the image.
DEPLOYER=/usr/local/bin/wormix-deployer.sh
[ -f "${CD_STATE:-/state}/deployer.sh" ] && DEPLOYER="${CD_STATE:-/state}/deployer.sh"

while :; do
    sh "$DEPLOYER" &
    DEPLOYER_PID=$!
    wait "$DEPLOYER_PID" || true
    log "deployer exited; restarting it in 30s"
    sleep 30 &
    wait $! || true
    # A broken self-updated deployer falls back to the one in the image.
    DEPLOYER=/usr/local/bin/wormix-deployer.sh
done
