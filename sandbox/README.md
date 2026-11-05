# Dedicated sandbox (1 CPU / 2 GB)

Runs the whole Wormix stack (api, core, relay, redis, cloudflared) inside one
Docker-in-Docker container capped at 1 CPU and 2 GB RAM.

Expected layout (same as the main compose file):

```
<dir>/
  Wormix/          # this repo
  wormix-api/      # https://github.com/andrij06062003/Wormix_Test
  wormix-client/   # game files (not in git)
```

Override with `API_DIR` / `CLIENT_DIR` if yours differ. Put your
`CLOUDFLARE_TUNNEL_TOKEN` in `Wormix/.env` (see `.env.example`).

```
cd sandbox
./manage.sh start | stop | restart | status | logs [internal] | backup | exec <cmd>
```

The game is published on `127.0.0.1:32700` of the host.
