# WORMIX LOCAL SERVER API

## Install

You need PHP 8.2+ and [Composer](https://getcomposer.org/download/). No database server is needed:
the API uses SQLite by default, so the whole database is one file, `database/database.sqlite`.

### Windows: one command
```shell
powershell -ExecutionPolicy Bypass -File .\setup.ps1
```
It creates `.env`, installs packages, creates the database, loads the game data and starts the API
at `http://127.0.0.1:8000/api/`. Run the same command again later to just start the server.
Add `-NoServe` to only set up, or `-Port 8080` to use another port.

To reset all accounts and game data, delete `database/database.sqlite` and run the script again.

### Manual
```shell
cp .env.example .env
composer install
php artisan key:generate
php artisan migrate        # answer "yes" to create database/database.sqlite
php artisan game:init      # only once, on an empty database
php artisan serve
```

To use MySQL instead of SQLite, set `DB_CONNECTION=mysql` and the `DB_*` values in `.env`.

## Base configuration
Base configuration is [wormix.php](./config/wormix.php)

## Api documentation
Postman API file is [wormix.postman_collection.json](./resources/api/wormix.postman_collection.json)

## Files in resources/game
* [gifts.json](./resources/game/gifts.json) - login sequence gifts
* [hats.json](./resources/game/hats.json) - array of game hats
* [items.messages_ru.xml](./resources/game/items.messages_ru.xml) - canonical names of items
* [level_awards.json](./resources/game/level_awards.json) - awards for level up
* [missions_awards.json](./resources/game/missions_awards.json) - awards for mission completed
* [names.json](./resources/game/names.json) - bot names array for random generation
* [races.json](./resources/game/races.json) - game races config
* [recipes.json](./resources/game/recipes.json) - game craft recipes config
* [weapons.json](./resources/game/weapons.json) - game weapons config
* [weapons_start.json](./resources/game/weapons_start.json) - starter weapons ids
