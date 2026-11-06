# Game host: serves play.html + game files, forwards /api/, relays game connections to wormix-core.
# Build from the repository root:  docker build -f services/relay/Dockerfile .
FROM golang:1.27-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0 GOWORK=off
COPY services/relay/go.mod services/relay/go.sum ./
RUN go mod download
COPY services/relay/*.go ./
RUN go build -trimpath -ldflags="-s -w" -o /out/wormix-relay .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && adduser -D -H -u 10001 app
WORKDIR /app
COPY --from=build /out/wormix-relay ./
COPY services/relay/appsettings.json ./
COPY wwwroot ./wwwroot
ENV Urls=http://0.0.0.0:32700 \
    ClientRoot=/client \
    Target=core \
    ApiUrl=http://api:8000
USER app
EXPOSE 32700
ENTRYPOINT ["/app/wormix-relay"]
