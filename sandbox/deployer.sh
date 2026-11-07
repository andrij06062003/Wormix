#!/bin/sh
# Wormix continuous deployment. Runs inside a sandbox container (sandbox/entrypoint.sh starts it
# once the sandbox's own Docker daemon is up) and deploys one branch into that sandbox only:
# the prod sandbox follows main, the dev sandbox follows dev (CD_BRANCH).
#
# Pull-based: nothing listens for connections. Every CD_POLL_SECONDS it asks GitHub for the head
# of CD_BRANCH (an ETag request; a 304 is cheap). A new commit is deployed once the CI workflow
# run started by the push to that branch has passed:
#   1. download the commit's tarball into releases/<sha> (an immutable release folder);
#   2. build its images, tagged with the commit (the running game is untouched if this fails);
#   3. stop the API, back up /data, recreate the stack from the new release;
#   4. health check; if it fails, recreate the previous release from its still-tagged images.
# A failed download or build is retried (MAX_ATTEMPTS); a release that builds but does not come
# up is not retried, it would fail the same way. Push a fix instead.
#
# The deployer only controls the sandbox's Docker daemon, never the host's.
set -u

REPO="${CD_REPO:-andrij06062003/Wormix}"
BRANCH="${CD_BRANCH:-main}"
ENV_NAME="${CD_ENV:-$BRANCH}"
INTERVAL="${CD_POLL_SECONDS:-30}"
REQUIRE_CI="${CD_REQUIRE_CI:-1}"
API="${CD_GITHUB_API:-https://api.github.com}"
STATE="${CD_STATE:-/state}"
BACKUPS="${CD_BACKUPS:-/backups}"
KEEP_RELEASES="${CD_KEEP_RELEASES:-3}"
KEEP_BACKUPS="${CD_KEEP_BACKUPS:-10}"
HEALTH_TIMEOUT="${CD_HEALTH_TIMEOUT:-180}"
HEALTH_URL="${CD_HEALTH_URL:-http://127.0.0.1:32700}"
CI_WORKFLOW=.github/workflows/ci.yml
PROJECT=wormix              # compose project name (docker-compose.yml "name:"); keeps the api-data volume
MAX_ATTEMPTS=3
RELEASES="$STATE/releases"

log() { echo "$(date '+%Y-%m-%d %H:%M:%S') [$ENV_NAME] $*" >&2; }

gh_curl() {
    if [ -n "${GITHUB_TOKEN:-}" ]; then
        curl -sS --max-time 120 -H "Authorization: Bearer $GITHUB_TOKEN" "$@"
    else
        curl -sS --max-time 120 "$@"
    fi
}

header() { grep -i "^$1:" "$STATE/headers" 2>/dev/null | tail -n 1 | cut -d' ' -f2- | tr -d '\r'; }

# GET an API path into a file; prints the HTTP status.
api_get() {
    if [ -n "${3:-}" ]; then
        gh_curl -o "$2" -D "$STATE/headers" -w '%{http_code}' -H "Accept: application/vnd.github+json" \
            -H "If-None-Match: $3" "$API$1"
    else
        gh_curl -o "$2" -D "$STATE/headers" -w '%{http_code}' -H "Accept: application/vnd.github+json" "$API$1"
    fi
}

rate_limited() {
    local reset wait
    reset=$(header x-ratelimit-reset)
    wait=$(( ${reset:-0} - $(date +%s) ))
    [ "$wait" -gt 3600 ] && wait=3600
    [ "$wait" -lt "$INTERVAL" ] && wait=$INTERVAL
    log "GitHub rate limit reached; next check in ${wait}s (set GITHUB_TOKEN for a higher limit)"
    sleep "$wait"
}

# success | pending | failure of the CI workflow for a commit. Only runs of $CI_WORKFLOW started by
# a push to $BRANCH of $REPO count: a pull request (from a fork too) can carry the same commit,
# and its runs prove nothing about this branch. A re-run counts by its latest attempt.
ci_state() {
    local code
    code=$(api_get "/repos/$REPO/actions/runs?head_sha=$1&event=push&branch=$BRANCH&per_page=100" \
        "$STATE/runs.json")
    case $code in
        200) ;;
        403|429) rate_limited; echo pending; return ;;
        *) echo pending; return ;;
    esac
    jq -r --arg sha "$1" --arg branch "$BRANCH" --arg repo "$REPO" --arg workflow "$CI_WORKFLOW" '
        [.workflow_runs[]
         | select(.head_sha == $sha and .event == "push" and .head_branch == $branch
             and (.path | split("@")[0]) == $workflow
             and ((.head_repository.full_name // "") | ascii_downcase) == ($repo | ascii_downcase))]
        | if length == 0 then "pending"
          elif any(.[]; .status != "completed") then "pending"
          elif all(.[]; .conclusion == "success") then "success"
          else "failure" end' "$STATE/runs.json" 2>/dev/null || echo pending
}

compose() { (cd "$1" && shift && docker compose "$@"); }

# Settings the stack reads from .env. Secrets come from the sandbox's environment, never from git.
write_env() {
    (
        umask 077
        {
            echo "IMAGE_TAG=$2"
            echo "CLOUDFLARE_TUNNEL_TOKEN=${CLOUDFLARE_TUNNEL_TOKEN:-}"
            [ -n "${CLOUDFLARE_TUNNEL_TOKEN:-}" ] && echo "COMPOSE_PROFILES=tunnel"
            echo "APP_KEY=${APP_KEY:-}"
            echo "APP_ENV=${APP_ENV:-${CD_ENV:-production}}"
            echo "CD_ENV=${CD_ENV:-}"
            echo "CLIENT_DIR=./wormix-client"
        } > "$1/.env"
    )
}

container() {
    docker ps -aq --filter "label=com.docker.compose.project=$PROJECT" \
        --filter "label=com.docker.compose.service=$1" | head -n 1
}

# Stops the API (so SQLite is not written to) and copies /data out. The caller restarts it.
backup() {
    local api dir
    api=$(container api)
    [ -z "$api" ] && { log "no api container yet; skipping backup"; return 0; }
    dir="$BACKUPS/pre-deploy-$1-$(date +%Y%m%d-%H%M%S)"
    docker stop -t 30 "$api" > /dev/null || return 1
    mkdir -p "$dir" && docker cp "$api:/data" "$dir/" || { rm -rf "$dir"; return 1; }
    log "backed up the database to $dir"
    ls -1dt "$BACKUPS"/pre-deploy-* 2>/dev/null | tail -n +$((KEEP_BACKUPS + 1)) | xargs -r rm -rf
}

# The relay serves the game page and reaches the API (any answer from Laravel, even a 404, proves
# the API is up and migrated; the relay answers 502-504 while it is not).
answers() {
    local api
    curl -fsS -o /dev/null --max-time 5 "$HEALTH_URL/play.html" || return 1
    api=$(curl -sS -o /dev/null -w '%{http_code}' --max-time 10 "$HEALTH_URL/api/photos/0/health" || true)
    case $api in 000|502|503|504) return 1 ;; esac
}

# Every service of the release is running, none has restarted, and the game answers.
healthy() {
    local dir="$1" waited=0 ids id
    while [ "$waited" -lt "$HEALTH_TIMEOUT" ]; do
        ids=$(compose "$dir" ps -aq)
        for id in $ids; do
            [ "$(docker inspect -f '{{.RestartCount}}' "$id")" -gt 0 ] && return 1
            [ "$(docker inspect -f '{{.State.Status}}' "$id")" = exited ] && return 1
        done
        if [ -n "$ids" ] && answers; then
            sleep 10   # still running a little later: not a crash loop that answered once
            for id in $ids; do
                [ "$(docker inspect -f '{{.State.Running}}/{{.RestartCount}}' "$id")" = true/0 ] || return 1
            done
            return 0
        fi
        sleep 3
        waited=$((waited + 3))
    done
    return 1
}

prune() {
    local keep tag old
    keep=$(ls -1t "$RELEASES" | head -n "$KEEP_RELEASES")
    for old in $(ls -1t "$RELEASES" | tail -n +$((KEEP_RELEASES + 1))); do
        [ "$old" = "$(cat "$STATE/current" 2>/dev/null)" ] && continue
        rm -rf "${RELEASES:?}/$old"
    done
    # Images built by the deployer are tagged with a short commit; drop those of removed releases.
    docker images --format '{{.Repository}}:{{.Tag}}' | grep -E '^wormix-(api|core|relay):[0-9a-f]{7}$' |
        while read -r image; do
            tag=${image##*:}
            echo "$keep" | grep -qx "$tag" || docker rmi "$image" > /dev/null 2>&1 || true
        done
    docker builder prune -f --filter until=168h > /dev/null 2>&1 || true
}

# 0 deployed, 1 failed (retried), 2 this commit cannot be deployed (not retried).
deploy() {
    local sha="$1" short dir prev code api
    short=$(echo "$sha" | cut -c1-7)
    dir="$RELEASES/$short"
    prev=$(cat "$STATE/current" 2>/dev/null || true)
    log "deploying $short"

    rm -rf "$dir" && mkdir -p "$dir"
    code=$(gh_curl -L -o "$STATE/source.tar.gz" -w '%{http_code}' "$API/repos/$REPO/tarball/$sha")
    # -m: extraction time as mtime, so BuildKit never reuses a stale file with a repeated mtime.
    if [ "$code" != 200 ] || ! tar -xzmf "$STATE/source.tar.gz" -C "$dir" --strip-components=1; then
        log "could not download $short (HTTP $code)"
        rm -rf "$dir"
        return 1
    fi
    rm -f "$STATE/source.tar.gz"
    [ -f "$dir/docker-compose.yml" ] || { log "$short has no docker-compose.yml"; return 2; }
    write_env "$dir" "$short"

    if ! compose "$dir" build --pull; then
        log "build of $short failed; the running version is unchanged"
        return 1
    fi

    if ! backup "$short"; then
        log "backup failed; not deploying $short"
        api=$(container api) && [ -n "$api" ] && docker start "$api" > /dev/null
        return 1
    fi

    compose "$dir" up -d --force-recreate --remove-orphans || true
    if healthy "$dir"; then
        echo "$short" > "$STATE/current"
        log "deployed $short"
        prune
        return 0
    fi

    log "$short did not become healthy; rolling back"
    compose "$dir" ps -a 2>&1 | sed 's/^/    /'
    compose "$dir" logs --tail 30 2>&1 | sed 's/^/    /'
    if [ -n "$prev" ] && [ -d "$RELEASES/$prev" ]; then
        if compose "$RELEASES/$prev" up -d --force-recreate --remove-orphans && healthy "$RELEASES/$prev"; then
            log "previous release $prev restored"
        else
            log "ERROR: previous release $prev did not come back either; check the sandbox"
        fi
        # Database migrations are not undone; the pre-deploy backup is in $BACKUPS if needed.
    else
        log "no previous release to restore"
    fi
    rm -rf "$dir"
    return 2
}

# After a deploy, switch to the deployer that came with the release if it changed.
self_update() {
    local new
    new="$RELEASES/$(cat "$STATE/current")/sandbox/deployer.sh"
    if [ -f "$new" ] && ! cmp -s "$new" "$0"; then
        sh -n "$new" || { log "new deployer.sh has a syntax error; keeping the current one"; return; }
        cp "$new" "$STATE/deployer.sh.new" && mv "$STATE/deployer.sh.new" "$STATE/deployer.sh"
        log "deployer updated; restarting it"
        exec sh "$STATE/deployer.sh"
    fi
}

mkdir -p "$STATE" "$RELEASES" "$BACKUPS"
deployed=$(cat "$STATE/deployed" 2>/dev/null || true)
log "watching $REPO@$BRANCH every ${INTERVAL}s (deployed: $(echo "${deployed:-none}" | cut -c1-7), CI required: $REQUIRE_CI)"

etag="" sha="" given_up="" waiting="" attempts=0
while :; do
    code=$(api_get "/repos/$REPO/commits/$BRANCH" "$STATE/commit.json" "$etag")
    case $code in
        200) etag=$(header etag); sha=$(jq -r '.sha // empty' "$STATE/commit.json") ;;
        304) ;;
        403|429) rate_limited; continue ;;
        *) log "GitHub answered HTTP $code for $REPO@$BRANCH" ;;
    esac

    if [ -n "$sha" ] && [ "$sha" != "$deployed" ] && [ "$sha" != "$given_up" ]; then
        short=$(echo "$sha" | cut -c1-7)
        ci=success
        [ "$REQUIRE_CI" = 1 ] && ci=$(ci_state "$sha")
        case $ci in
            pending)
                [ "$waiting" != "$sha" ] && log "new commit $short; waiting for CI" && waiting=$sha ;;
            failure)
                log "CI failed for $short; not deploying it"
                given_up=$sha ;;
            success)
                deploy "$sha"
                result=$?
                if [ "$result" = 0 ]; then
                    deployed=$sha attempts=0
                    echo "$sha" > "$STATE/deployed"
                    echo "$(date '+%Y-%m-%d %H:%M:%S') $sha" >> "$STATE/history.log"
                    self_update
                elif [ "$result" = 2 ]; then
                    given_up=$sha
                else
                    attempts=$((attempts + 1))
                    if [ "$attempts" -ge "$MAX_ATTEMPTS" ]; then
                        log "giving up on $short after $attempts attempts; waiting for a new commit"
                        given_up=$sha attempts=0
                    fi
                fi ;;
        esac
    fi
    sleep "$INTERVAL"
done
