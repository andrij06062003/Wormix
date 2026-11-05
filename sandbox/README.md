# Dedicated sandbox (1 CPU / 2 GB)

Runs the whole Wormix stack (api, core, relay, redis, cloudflared) inside one
Docker-in-Docker container capped at 1 CPU and 2 GB RAM.

The API (`wormix-api/`) and game files (`wormix-client/`) are included in
this repo. Override with `API_DIR` / `CLIENT_DIR` to use other copies. Put your
`CLOUDFLARE_TUNNEL_TOKEN` in `Wormix/.env` (see `.env.example`).

```
cd sandbox
./manage.sh start | stop | restart | status | logs [internal] | backup | exec <cmd>
```

The game is published on `127.0.0.1:32700` of the host.
