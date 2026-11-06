# Wormix server

Runs the original **Wormix 1.05.0** Flash client in a modern browser (Chrome, Edge, Firefox) with
[Ruffle](https://ruffle.rs), against your own server. No Flash Player needed.

**Hosting it for others (Docker + Cloudflare Tunnel): see [HOSTING.md](HOSTING.md).**
The rest of this file is about running everything on one Windows PC.

This folder is the **game host** plus the scripts that start everything:

All three server parts are written in Go (one module each, tied together by `go.work`):

| Part | Folder | Port |
|---|---|---|
| API (accounts, money, items, SQLite database) | `services/api` | 8000 |
| Game server: main, PvP, achievements (originally ported from [n30np14gu3/wormix-core](https://github.com/n30np14gu3/wormix-core)) | `services/core` | 49197, 49198, 49199 |
| **Game host** | `services/relay` | **32700** |
| Game files | `wormix-client` | served by the game host |

The game host does four things on one address, so players only ever need `http://<server>:32700`:

1. **Serves the game**: `wwwroot/play.html` (login/registration page + Ruffle) and the game files from `../wormix-client`.
2. **Forwards `/api/`** to the API.
3. **Relays the game's network connections.** Browsers can't open raw TCP sockets, so Ruffle sends them
   as WebSockets (`ws://<server>:32700/<port>`) and the host forwards them to wormix-core. The game has
   `wormixtest.ru` built in; `play.html` redirects those connections here, so the game files are unmodified.
   The host also reassembles complete game messages before passing them on: wormix-core writes each message
   header in pieces, and the game's network code drops a reply whose header arrives split
   (that froze battles on "Инициализация карты").
4. **Answers achievements** (`services/relay/achievements.go`). The game sends them as HTTP POSTs to
   `<server>:49199/wormix_achieve/`, not over a socket; `play.html` rewrites that URL to the host, which
   decodes the binary commands and stores progress through the API (`/api/internal/achievements/`).
   Level rewards (fuzzes, rubies, reaction) and the award items for every 1000 points are granted on the server.
   `tests/achievements/ach_smoke.py` tests it end to end.

## First-time setup (Windows)

1. Install Go: `winget install GoLang.Go`. That's the only tool needed; `start.ps1` builds the three parts.
2. The game files are in `wormix-client\` (the root files of the client archive's `public\` plus
   `public\vk\1.05.0\`). They are Pragmatix's files.

The database and uploaded photos go to `data\` (`-DataDir` to change). To keep the accounts of an older
PHP setup, copy its `wormix-api\database\database.sqlite` to `data\database.sqlite` before the first start:
the Go API reads the same database, passwords and logins.

## Play

```
powershell -ExecutionPolicy Bypass -File .\start.ps1
```

Starts every part in its own minimized window (restore one to see its log), then opens the game.
Register on the page that opens, and play. Parts that are already running are skipped.

- `-Lan`: let other computers open the game (`http://<this PC's address>:32700`), e.g. friends over Tailscale.
- `-GameLog`: open the game with `?debug`; the game's warnings and errors appear in the game host window.
- `-NoBrowser`: don't open the browser.

Stop everything:

```
powershell -ExecutionPolicy Bypass -File .\stop.ps1
```

## Settings

Game host: `services/relay/appsettings.json`, environment variables with the same names, or command line
(e.g. `--Urls http://0.0.0.0:32700`), in that order of precedence (lowest first):

| Setting | Default | Meaning |
|---|---|---|
| `Urls` | `http://127.0.0.1:32700` | Address of the game host |
| `ClientRoot` | `../wormix-client` | Game files folder |
| `Target` | `127.0.0.1` | Where wormix-core runs |
| `ApiUrl` | `http://127.0.0.1:8000` | Where the API runs |
| `AllowedPorts` | 49197, 49198, 49199, 49297–49300 | Game server ports the relay may connect to |
| `LogApi` | `false` | Print every API request (passwords are hidden) |

API: `DB_DATABASE` (default `/data/database.sqlite`), `PHOTOS_DIR` (default `photos` next to the database),
`LISTEN_ADDR` (default `0.0.0.0:8000`).

`configs/wormix-core.json` is the game server config matching the ports built into the 1.05.0 client
(main 49197, PvP 49198, achievements 49199; from the preloader's built-in server list).

## Playing against other players

There are no VK friends any more, so PvP works with game ids (shown under the game, next to the login):

- **Friends list / call by id**: challenge any player who is online; they get an offer to accept or decline.
  Every registered player appears in the friends list.
- **Arena > PvP (wager battles)**: pick a wager and wait; the next player choosing the same wager is your
  opponent. The winner gets the wager back plus 80%, a draw returns it, and the rating changes.

The PvP server (`services/core/internal/pvp/`) relays the moves between the two games and decides each turn
and the winner. `tests/pvp/run.sh` tests it end to end with two simulated players in a throwaway Docker stack.
