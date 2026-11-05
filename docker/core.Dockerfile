# wormix-core (game server): main, PvP and achievements servers. Source: wormix-core/ in this repository
# (from n30np14gu3/wormix-core, with the PvP server implemented here).
FROM mcr.microsoft.com/dotnet/sdk:8.0 AS build
WORKDIR /src
COPY wormix-core/wormix-core/wormix-core.csproj wormix-core/
RUN dotnet restore wormix-core/wormix-core.csproj
COPY wormix-core/wormix-core/ wormix-core/
RUN dotnet publish wormix-core/wormix-core.csproj -c Release -o /out --no-restore

FROM mcr.microsoft.com/dotnet/runtime:8.0
WORKDIR /app
COPY --from=build /out .
COPY docker/wormix-core.json /app/config.json
# Ports built into the 1.05.0 preloader: main 49197, PvP 49198, achievements 49199.
EXPOSE 49197 49198 49199
# wormix-core waits on Console.ReadLine() to stay alive, so compose gives it stdin + tty.
ENTRYPOINT ["dotnet", "/app/wormix-core.dll", "/app/config.json"]
