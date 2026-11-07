# wormix-core (game server) in Go: main, PvP and achievements servers. Build context: repository root.
#   docker build -f services/core/Dockerfile -t wormix-core .
FROM golang:1.27-alpine AS build
WORKDIR /src/services/core
COPY services/core/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/wormix-core ./cmd/wormix-core

FROM alpine:3
WORKDIR /app
COPY --from=build /out/wormix-core /app/wormix-core
COPY docker/wormix-core.json /app/config.json
# Runs as root like the original image, so an enabled policy server can bind port 843.
# Ports built into the 1.05.0 preloader: main 49197, PvP 49198, achievements 49199.
EXPOSE 49197 49198 49199
ENTRYPOINT ["/app/wormix-core"]
CMD ["/app/config.json"]
