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
| `core` | Game server ([n30np14gu3/wormix-core](https://github.com/n30np14gu3/wormix-core)) | no |
| `api` | Accounts, money, items, battles (SQLite database in a Docker volume) | `/api/` only, `/api/internal/` is blocked |
| `redis` | Live sessions for the game server | no |

## 1. Get the files

Three folders side by side:

```
wormix/
  wormix-relay/    git clone https://github.com/andrij06062003/Wormix.git wormix-relay
  wormix-api/      git clone -b sqlite-setup-script https://github.com/andrij06062003/Wormix_Test.git wormix-api
  wormix-client/   the game files (get them from the server owner; they are not in git)
```

`wormix-client` must contain `preloader.swf` and `vk/1.05.0/preloader.swf`, `vk/1.05.0/Wormix.swf`,
`vk/1.05.0/data/...`: the root files of the client archive's `public/` folder plus `public/vk/1.05.0/`.

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
# edit .env: paste the token into CLOUDFLARE_TUNNEL_TOKEN=
docker compose up -d --build
```

The first build takes a few minutes. On first start the API creates the database and loads the game data
(weapons, hats, missions...).

Check it:

```bash
docker compose ps                  # all five should be "running"
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

Update after a `git pull` in `wormix-relay` or `wormix-api`: `docker compose up -d --build`.

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
| Login screen works, game stays at 0% | `wormix-client` folder missing or wrong layout (`docker compose exec relay ls /client/vk/1.05.0`). |
| Game loads but can't connect | `docker compose logs relay core`: look for "Can't reach core". |
| Battle stuck on "Инициализация карты" | Make sure you run the current `relay` (it reassembles game messages). |
| Want the game's own log | Open `https://<hostname>/?debug` and set `DebugLog=true` for `relay` (environment), then `docker compose logs -f relay`. |
