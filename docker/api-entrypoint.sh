#!/bin/sh
# Prepares and starts wormix-api inside the container.
# Everything that must survive restarts lives in /data (a Docker volume):
#   /data/database.sqlite   accounts, money, items, progress
#   /data/app_key           Laravel app key (generated once)
#   /data/photos            uploaded profile photos
set -e
cd /app

[ -f .env ] || cp .env.example .env

# Discover packages (skipped at build time because it needs the app to boot).
php artisan package:discover --ansi >/dev/null

# App key: from the environment, or generated once and kept in /data.
if [ -z "$APP_KEY" ]; then
    if [ ! -s /data/app_key ]; then
        php artisan key:generate --show > /data/app_key
    fi
    APP_KEY="$(cat /data/app_key)"
    export APP_KEY
fi

# Uploaded files persist in /data/photos.
mkdir -p /data/photos
if [ ! -L storage/app ]; then
    if [ -d storage/app ]; then cp -rn storage/app/. /data/photos/ 2>/dev/null || true; rm -rf storage/app; fi
    ln -s /data/photos storage/app
fi

fresh=0
if [ ! -f "$DB_DATABASE" ]; then
    touch "$DB_DATABASE"
    fresh=1
fi

php artisan migrate --force

# Weapons, hats, missions, recipes...: only into a new database (running it twice reports duplicates).
if [ "$fresh" = "1" ]; then
    php artisan game:init
fi

exec php artisan serve --host=0.0.0.0 --port=8000
