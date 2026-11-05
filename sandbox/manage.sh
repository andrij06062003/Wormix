#!/bin/bash
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

case "$1" in
    start)
        echo "Starting Wormix dedicated container..."
        docker compose up -d
        ;;
    stop)
        echo "Stopping Wormix dedicated container..."
        docker compose down
        ;;
    restart)
        echo "Restarting Wormix dedicated container..."
        docker compose restart
        ;;
    status)
        echo "=== Host Container Status ==="
        docker compose ps
        echo ""
        echo "=== Internal Services Status ==="
        docker exec wormix-sandbox docker compose -f /workspace/wormix-relay/docker-compose.yml ps 2>/dev/null || echo "Sandbox not running"
        ;;
    logs)
        shift
        if [ "$1" = "internal" ]; then
            shift
            docker exec -it wormix-sandbox docker compose -f /workspace/wormix-relay/docker-compose.yml logs "$@"
        else
            docker compose logs "$@"
        fi
        ;;
    backup)
        BACKUP_DIR="./backups/backup-$(date +%F-%H%M%S)"
        mkdir -p "$BACKUP_DIR"
        echo "Backing up database to $BACKUP_DIR..."
        docker exec wormix-sandbox docker compose -f /workspace/wormix-relay/docker-compose.yml cp api:/data "$BACKUP_DIR"
        echo "Backup complete: $BACKUP_DIR"
        ;;
    exec)
        shift
        docker exec -it wormix-sandbox "$@"
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|logs|backup|exec}"
        exit 1
        ;;
esac
