# wormix-core (game server) by n30np14gu3, built from GitHub at a fixed, tested commit.
FROM mcr.microsoft.com/dotnet/sdk:8.0 AS build
ARG WORMIX_CORE_REPO=https://github.com/n30np14gu3/wormix-core.git
ARG WORMIX_CORE_REF=e9be18d
RUN git clone "$WORMIX_CORE_REPO" /src && git -C /src checkout "$WORMIX_CORE_REF"
RUN dotnet publish /src/wormix-core/wormix-core.csproj -c Release -o /out

FROM mcr.microsoft.com/dotnet/runtime:8.0
WORKDIR /app
COPY --from=build /out .
COPY docker/wormix-core.json /app/config.json
EXPOSE 49197 49198 49298
# wormix-core waits on Console.ReadLine() to stay alive, so compose gives it stdin + tty.
ENTRYPOINT ["dotnet", "/app/wormix-core.dll", "/app/config.json"]
