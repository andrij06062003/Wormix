# Game host: serves play.html + game files, forwards /api/, relays game connections to wormix-core.
FROM mcr.microsoft.com/dotnet/sdk:8.0 AS build
WORKDIR /src
COPY wormix-relay.csproj .
RUN dotnet restore
COPY Program.cs appsettings.json ./
RUN dotnet publish -c Release -o /out --no-restore

FROM mcr.microsoft.com/dotnet/aspnet:8.0
WORKDIR /app
COPY --from=build /out .
COPY wwwroot ./wwwroot
ENV Urls=http://0.0.0.0:32700 \
    ClientRoot=/client \
    Target=core \
    ApiUrl=http://api:8000
EXPOSE 32700
ENTRYPOINT ["dotnet", "wormix-relay.dll"]
