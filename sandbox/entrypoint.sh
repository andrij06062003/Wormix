#!/bin/bash
set -eo pipefail

echo "=================================================="
echo " Starting Wormix Dedicated Sandbox (1 CPU, 2 GB) "
echo "=================================================="

# Start dockerd in background using dind's dockerd-entrypoint
echo "[sandbox] Starting isolated Docker daemon..."
dockerd-entrypoint.sh dockerd > /var/log/dockerd.log 2>&1 &
DOCKERD_PID=$!

# Wait for dockerd to be responsive
echo "[sandbox] Waiting for internal Docker daemon to be ready..."
RETRIES=30
until docker info >/dev/null 2>&1 || [ $RETRIES -eq 0 ]; do
    sleep 1
    RETRIES=$((RETRIES - 1))
done

if [ $RETRIES -eq 0 ]; then
    echo "[sandbox] ERROR: Docker daemon failed to start. Logs:"
    cat /var/log/dockerd.log
    exit 1
fi

echo "[sandbox] Internal Docker daemon is ready."

# Graceful termination handler
cleanup() {
    echo "[sandbox] Caught shutdown signal. Stopping Wormix compose stack..."
    if [ -d "/workspace/wormix-relay" ]; then
        cd /workspace/wormix-relay && docker compose down || true
    fi
    echo "[sandbox] Stopping internal Docker daemon..."
    kill -TERM "$DOCKERD_PID" 2>/dev/null || true
    wait "$DOCKERD_PID" 2>/dev/null || true
    echo "[sandbox] Dedicated sandbox stopped cleanly."
    exit 0
}

trap cleanup SIGTERM SIGINT

cd /workspace/wormix-relay

echo "[sandbox] Starting Wormix stack via Docker Compose..."
docker compose up -d --build

echo "[sandbox] Wormix services status:"
docker compose ps

echo "[sandbox] Streaming logs from Wormix services (Ctrl+C / SIGTERM to stop)..."
docker compose logs -f &
LOGS_PID=$!

# Wait on background logs while trapping signals
wait "$LOGS_PID"
