# wormix-api (Go). Build context: repository root.
#   docker build -f services/api/Dockerfile .
FROM golang:1.27-alpine AS build
WORKDIR /src/services/api
ENV CGO_ENABLED=0 GOWORK=off
COPY services/api/go.mod services/api/go.sum ./
RUN go mod download
COPY services/api/ ./
RUN go build -trimpath -ldflags="-s -w" -o /out/wormix-api .

FROM alpine:3
# Runs as root like the PHP image did, so existing /data volumes (root-owned) stay writable.
RUN apk add --no-cache ca-certificates tzdata && mkdir -p /data
COPY --from=build /out/wormix-api /usr/local/bin/wormix-api
# Everything that must survive restarts lives in /data:
#   /data/database.sqlite   accounts, money, items, progress (and game/achievement sessions in the cache table)
#   /data/photos/users      uploaded profile photos
#   /data/app_key           kept for compatibility with the PHP API (unused by this API)
ENV DB_DATABASE=/data/database.sqlite
VOLUME ["/data"]
EXPOSE 8000
ENTRYPOINT ["wormix-api"]
