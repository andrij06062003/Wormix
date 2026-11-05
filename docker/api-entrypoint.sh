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

# Ensure DB_DATABASE is explicitly in .env so PHP CLI server workers inherit it
if ! grep -q "^DB_DATABASE=" .env; then
    echo "DB_DATABASE=/data/database.sqlite" >> .env
fi

# Ensure APP_ENV and CD_ENV are in .env so PHP CLI server workers inherit them
if [ -n "$APP_ENV" ]; then
    if grep -q "^APP_ENV=" .env; then
        sed -i "s/^APP_ENV=.*/APP_ENV=$APP_ENV/" .env
    else
        echo "APP_ENV=$APP_ENV" >> .env
    fi
fi
if [ -n "$CD_ENV" ]; then
    if grep -q "^CD_ENV=" .env; then
        sed -i "s/^CD_ENV=.*/CD_ENV=$CD_ENV/" .env
    else
        echo "CD_ENV=$CD_ENV" >> .env
    fi
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

ln -sf /data/database.sqlite /app/database/database.sqlite

php artisan migrate --force

# Weapons, hats, missions, recipes...: only into a new database (running it twice reports duplicates).
if [ "$fresh" = "1" ]; then
    php artisan game:init
fi

exec php artisan serve --host=0.0.0.0 --port=8000
