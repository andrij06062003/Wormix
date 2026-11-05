# Sandboxes and continuous deployment

Two environments run side by side on the host. Each is one Docker-in-Docker container with its own
Docker daemon, database, images and deployer:

| Sandbox | Branch | Local address | Public address |
|---|---|---|---|
| `wormix-prod` | `main` | http://127.0.0.1:32700 | the prod Cloudflare tunnel (`prod.env`) |
| `wormix-dev`  | `dev`  | http://127.0.0.1:32701 | a separate dev tunnel (`dev.env`), optional |

## How a change ships

```
feature branch --PR--> dev  --(CI passes)--> wormix-dev redeploys
               dev  --PR--> main --(CI passes)--> wormix-prod redeploys
```

1. Open a pull request into `dev`. CI (`.github/workflows/ci.yml`, GitHub-hosted runners) lints the
   scripts and compose files, builds every image, starts the stack and checks that the game page and
   the API answer through the relay.
2. Merge it. CI runs again for the push to `dev`.
3. The deployer inside `wormix-dev` polls GitHub every 30 s. Once CI for that push has passed, it:
   downloads the commit into an immutable `releases/<sha>` folder, builds images tagged `<sha>`
   (the running game is untouched if that fails), stops the API and backs up its `/data`, recreates
   the stack, and health checks it (all containers stable, game page and API answer). If the check
   fails, it brings the previous release back from its still-tagged images.
4. Promote with a pull request `dev` → `main`; `wormix-prod` deploys the same way.

Nothing listens for webhooks: the deployers only make outbound requests to GitHub, so nothing is
exposed and pull requests from forks can never run anything on the host. A commit whose CI failed is
never deployed. `deployer.sh` updates itself from the release it just deployed.

Database migrations are not rolled back with the code. Pre-deploy backups are in `backups/<env>/`
(last 10 kept).

## Setup (once, on the host)

```bash
cd sandbox
cp prod.env.example prod.env    # prod tunnel token, optional GITHUB_TOKEN
cp dev.env.example dev.env      # dev tunnel token (a different tunnel) or empty
./manage.sh up                  # = docker compose up -d --build
```

`wormix-prod` reuses the Docker volume of the earlier single sandbox
(`wormix-dedicated_wormix-dind-data`), so its database is kept.
Re-run `./manage.sh up` after changing this folder's `Dockerfile`, `entrypoint.sh` or
`docker-compose.yml`; nothing else needs the host.

## Everyday commands

```bash
./manage.sh status                 # both sandboxes, their containers, deployed commits
./manage.sh logs dev               # deployer log of dev (add -f to follow)
./manage.sh logs prod api -f       # a service inside prod
./manage.sh backup prod            # copy the database to backups/prod/manual-...
./manage.sh shell dev              # shell inside the dev sandbox
```

## Securing it

Whoever can push to `main` or `dev` controls what runs in the sandboxes, and the sandboxes are
privileged containers. A repository admin should protect both branches on GitHub
(Settings > Branches): require a pull request and the CI checks (`Lint`, `Build and smoke test`,
`Sandbox image`) to pass, and block force pushes.
