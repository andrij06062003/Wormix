# Hosting the Wormix server (Docker + Cloudflare Tunnel)

Runs the whole server on one machine with Docker and publishes it at an HTTPS address through a
Cloudflare Tunnel. Players just open that address in Chrome, Edge or Firefox. No Flash, no downloads,
and no ports opened on the router.

Works on Linux or Windows (Docker Desktop), on x86-64 or ARM64 (e.g. Raspberry Pi 4/5 with a 64-bit OS).

## What runs

| Container | What | Reached from the internet? |
|---|---|---|
| `cloudflared` | Cloudflare Tunnel: carries `https://<your hostname>` to `relay` | connects out to Cloudflare |
| `relay` | Game host: page, game files, `/api/` forwarding, game connections | only through the tunnel |
| `core` | Game server: main, PvP and achievements (`services/core/`, Go; originally ported from [n30np14gu3/wormix-core](https://github.com/n30np14gu3/wormix-core)) | no |
| `api` | Accounts, money, items, battles (`services/api/`, Go; SQLite database in a Docker volume) | `/api/` only, `/api/internal/` is blocked |

The game host is `services/relay/`. All three are Go programs, each built by its own `services/*/Dockerfile`.

## 1. Get the files

```
git clone https://github.com/andrij06062003/Wormix.git wormix-relay
```

The repository contains the server and the game files (`wormix-client/`). `wormix-client` must contain `vk/1.25.0/preloader.swf`, `vk/1.25.0/Wormix.swf`,
`vk/1.25.0/data/...` and the maps and sounds in `vk/storage/` (see `wormix-client/README.md`).

## 2. Create the Cloudflare Tunnel

You need a domain on Cloudflare (free plan is fine).

1. Cloudflare dashboard > **Zero Trust** > **Networks** > **Tunnels** > **Create a tunnel** > **Cloudflared**.
2. Name it (e.g. `wormix`). On the install step choose **Docker** and copy the token: the long string after `--token`.
3. **Public Hostname** tab > **Add a public hostname**:
   - Subdomain/domain: e.g. `wormix.yourdomain.com`
   - Service: **HTTP**, URL: **`relay:32700`**

   WebSockets are on by default; leave them on (the game needs them).

## 3. Configure and start

```bash
cd wormix-relay
cp .env.example .env
# edit .env: paste the token into CLOUDFLARE_TUNNEL_TOKEN= (keep COMPOSE_PROFILES=tunnel: it starts cloudflared)
docker compose up -d --build
```

The first build takes a few minutes. On first start the API creates the database and loads the game data
(weapons, hats, missions...).

Check it:

```bash
docker compose ps                  # all four should be "running"
docker compose logs -f api core    # Ctrl+C to stop watching
```

Open `https://wormix.yourdomain.com`, register an account, and play.
On the server itself it also answers at `http://127.0.0.1:32700`.

## Everyday commands

```bash
docker compose logs -f relay       # game host log
docker compose restart core        # restart the game server
docker compose down                # stop everything (the database is kept)
docker compose up -d --build       # start / apply updates
```

Update after a `git pull`: `docker compose up -d --build`.

## Back up the database

All accounts and progress are in the `wormix_api-data` volume:

```bash
docker compose cp api:/data ./backup-$(date +%F)
```

Restore: `docker compose cp ./backup-YYYY-MM-DD/. api:/data && docker compose restart api`.

## Who can play

Anyone with the address can open the page and register. To limit it to friends, add a
**Cloudflare Access** application for the hostname (Zero Trust > Access > Applications) with a rule such as
"emails ending in ..." or a list of your friends' emails. Cloudflare then asks for an email code before the
game loads.

## Troubleshooting

| Symptom | Check |
|---|---|
| Page doesn't open | `docker compose logs cloudflared`: token wrong? Public hostname service must be `http://relay:32700`. |
| Login screen works, game stays at 0% | `wormix-client` folder missing or wrong layout (`docker compose exec relay ls /client/vk/1.25.0`). |
| Game loads but can't connect | `docker compose logs relay core`: look for "Can't reach core". |
| Battle stuck on "Инициализация карты" | Make sure you run the current `relay` (it reassembles game messages). |
| Want the game's own log | Open `https://<hostname>/?debug` and set `DebugLog=true` for `relay` (environment), then `docker compose logs -f relay`. |

## Prod and dev with automatic deploys

On the dedicated host the stack runs in two sandboxes, `wormix-prod` (follows `main`) and
`wormix-dev` (follows `dev`), each redeploying itself when CI passes for a merge to its branch.
See [sandbox/README.md](sandbox/README.md).
