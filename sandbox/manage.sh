#!/bin/bash
# Day-to-day commands for the two sandboxes. ENV is prod or dev.
#   ./manage.sh up                    build the sandbox image and start both environments
#   ./manage.sh status [ENV]          sandboxes, the Wormix containers inside, deployed commit
#   ./manage.sh logs ENV [deployer|SERVICE] [docker compose logs options]
#   ./manage.sh backup ENV            copy the database out (to backups/ENV/manual-...)
#   ./manage.sh shell ENV             a shell inside the sandbox
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

envs() { if [ -n "${1:-}" ]; then echo "$1"; else echo prod dev; fi; }
check() { case "${1:-}" in prod|dev) ;; *) echo "ENV must be prod or dev" >&2; exit 1 ;; esac; }
inner() { local e=$1; shift; docker exec "wormix-$e" sh -c 'cd /state/releases/$(cat /state/current) && docker compose "$@"' sh "$@"; }

case "${1:-}" in
    up) docker compose up -d --build ;;
    status)
        docker compose ps
        for e in $(envs "${2:-}"); do
            echo; echo "== $e: deployed $(docker exec "wormix-$e" cat /state/current 2>/dev/null || echo nothing yet)"
            inner "$e" ps 2>/dev/null || true
        done ;;
    logs)
        check "${2:-}"; e=$2; shift 2
        if [ "${1:-deployer}" = deployer ]; then
            [ $# -gt 0 ] && shift
            docker compose logs "wormix-$e" "$@"
        else
            inner "$e" logs "$@"
        fi ;;
    backup)
        check "${2:-}"
        dir="/backups/manual-$(date +%Y%m%d-%H%M%S)"
        docker exec "wormix-$2" sh -c "mkdir -p $dir && docker cp \$(docker ps -q --filter label=com.docker.compose.project=wormix --filter label=com.docker.compose.service=api):/data $dir/"
        echo "backed up to backups/$2/${dir#/backups/}" ;;
    shell) check "${2:-}"; docker exec -it "wormix-$2" bash ;;
    *) sed -n '2,7p' "$0"; exit 1 ;;
esac
