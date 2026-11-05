# wormix-relay

Runs the original **Wormix 1.05.0** Flash client in a modern browser (Chrome, Edge, Firefox) with
[Ruffle](https://ruffle.rs), against your own server. No Flash Player needed.

**Hosting it for others (Docker + Cloudflare Tunnel): see [HOSTING.md](HOSTING.md).**
The rest of this file is about running everything on one Windows PC.

This folder is the **game host** plus the scripts that start everything:

| Part | Folder | Port |
|---|---|---|
| Redis (Garnet) | – | 6379 |
| API (accounts, money, items, database) | `../wormix-api` | 8000 |
| Game server | `wormix-core` (from [n30np14gu3/wormix-core](https://github.com/n30np14gu3/wormix-core), with PvP added here) | 49197, 49298, 49198 |
| **Game host** (this project) | `.` | **32700** |
| Game files | `../wormix-client` | served by the game host |

The game host does three things on one address, so players only ever need `http://<server>:32700`:

1. **Serves the game**: `wwwroot/play.html` (login/registration page + Ruffle) and the game files from `../wormix-client`.
2. **Forwards `/api/`** to the API.
3. **Relays the game's network connections.** Browsers can't open raw TCP sockets, so Ruffle sends them
   as WebSockets (`ws://<server>:32700/<port>`) and the host forwards them to wormix-core. The game has
   `wormixtest.ru` built in; `play.html` redirects those connections here, so the game files are unmodified.
   The host also reassembles complete game messages before passing them on: wormix-core writes each message
   header in pieces, and the game's network code drops a reply whose header arrives split
   (that froze battles on "Инициализация карты").

## Folder layout

```
D:\Vibecode\
  wormix-api\      your API (github.com/andrij06062003/Wormix_Test)
  wormix-client\   game files (not in git, see below)
  wormix-relay\    this folder
```

## First-time setup (Windows)

1. Install the tools:
   ```
   winget install PHP.PHP.8.3
   winget install Microsoft.DotNet.SDK.8
   dotnet tool install --global garnet-server
   ```
   plus [Composer](https://getcomposer.org/download/).
2. Set up the API: `powershell -ExecutionPolicy Bypass -File ..\wormix-api\setup.ps1 -NoServe`
3. Game files: from your Wormix client archive, copy the files in `public\` (not its subfolders) and
   `public\vk\1.05.0\` into `..\wormix-client\`, keeping the `vk\1.05.0\` path. They are Pragmatix's
   files and are not part of this repository.

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

`appsettings.json` (or command-line, e.g. `--Urls http://0.0.0.0:32700`):

| Setting | Default | Meaning |
|---|---|---|
| `Urls` | `http://127.0.0.1:32700` | Address of the game host |
| `ClientRoot` | `../wormix-client` | Game files folder |
| `Target` | `127.0.0.1` | Where wormix-core runs |
| `ApiUrl` | `http://127.0.0.1:8000` | Where the API runs |
| `AllowedPorts` | 49197, 49198, 49199, 49297–49300 | Game server ports the relay may connect to |
| `LogApi` | `false` | Print every API request (passwords are hidden) |

`configs/wormix-core.json` is the wormix-core config matching the ports built into the 1.05.0 client
(main 49197, PvP 49298, achievements 49198).

## Playing against other players

There are no VK friends any more, so PvP works with game ids (shown under the game, next to the login):

- **Friends list / call by id**: challenge any player who is online; they get an offer to accept or decline.
  Every registered player appears in the friends list.
- **Arena > PvP (wager battles)**: pick a wager and wait; the next player choosing the same wager is your
  opponent. The winner gets the wager back plus 80%, a draw returns it, and the rating changes.

The PvP server (`wormix-core/wormix-core/Pvp/`) relays the moves between the two games and decides each turn
and the winner. `tests/pvp/run.sh` tests it end to end with two simulated players in a throwaway Docker stack.
