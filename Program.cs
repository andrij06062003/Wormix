// Game host for running the Wormix Flash client in a modern browser with Ruffle.
//
// 1. Serves the client files (play.html, preloader.swf, vk/1.05.0/...) from "ClientRoot".
// 2. Relays the game's network connections: browsers can't open raw TCP sockets, so Ruffle sends
//    them over a WebSocket (its "socketProxy" setting). Those are forwarded, byte for byte, to
//    the wormix-core server ports:
//
//      ws://<host>:32700/49197  ->  tcp <Target>:49197
//
// Only ports listed in "AllowedPorts" can be reached, so the relay can't be used to connect anywhere else.
// Settings: appsettings.json, or command line, e.g.  --Urls http://0.0.0.0:32700 --Target 192.168.1.50

using System.Net.Sockets;
using System.Net.WebSockets;
using Microsoft.AspNetCore.StaticFiles;
using Microsoft.Extensions.FileProviders;

var builder = WebApplication.CreateBuilder(args);
builder.WebHost.UseUrls(builder.Configuration["Urls"] ?? "http://127.0.0.1:32700");

var app = builder.Build();
var target = app.Configuration["Target"] ?? "127.0.0.1";
var allowed = app.Configuration.GetSection("AllowedPorts").Get<int[]>() ?? new[] { 49197, 49198, 49199, 49297, 49298, 49299, 49300 };
var log = app.Logger;

app.UseWebSockets(new WebSocketOptions { KeepAliveInterval = TimeSpan.FromSeconds(30) });

var clientRoot = Path.GetFullPath(app.Configuration["ClientRoot"] ?? "../wormix-client", AppContext.BaseDirectory);
if (!Directory.Exists(clientRoot))
    clientRoot = Path.GetFullPath(app.Configuration["ClientRoot"] ?? "../wormix-client", builder.Environment.ContentRootPath);
if (Directory.Exists(clientRoot))
{
    // This project's own pages (wwwroot/play.html) first, then the game files from ClientRoot.
    var ownPages = Path.Combine(builder.Environment.ContentRootPath, "wwwroot");
    IFileProvider files = Directory.Exists(ownPages)
        ? new CompositeFileProvider(new PhysicalFileProvider(ownPages), new PhysicalFileProvider(clientRoot))
        : new PhysicalFileProvider(clientRoot);
    var types = new FileExtensionContentTypeProvider();
    types.Mappings[".swf"] = "application/x-shockwave-flash";
    app.UseDefaultFiles(new DefaultFilesOptions { FileProvider = files, DefaultFileNames = { "play.html" } });
    app.UseStaticFiles(new StaticFileOptions
    {
        FileProvider = files,
        ContentTypeProvider = types,
        ServeUnknownFileTypes = false,
        OnPrepareResponse = c => c.Context.Response.Headers.CacheControl = "no-cache"
    });
    log.LogInformation("Serving game client from {Root}", clientRoot);
}
else
{
    log.LogWarning("Client folder {Root} not found: only the relay is running", clientRoot);
}

// Debug: play.html?debug posts the game's log lines here, so they appear in this terminal.
// Off unless "DebugLog" is true (start.ps1 -GameLog turns it on), so the public can't fill the log.
var debugLog = app.Configuration.GetValue("DebugLog", false);
app.MapPost("/debug-log", async (HttpContext ctx) =>
{
    if (!debugLog) return Results.NotFound();
    using var reader = new StreamReader(ctx.Request.Body);
    var text = await reader.ReadToEndAsync();
    foreach (var line in text.Split('\n', StringSplitOptions.RemoveEmptyEntries))
        log.LogInformation("GAME {Line}", line.Length > 1000 ? line[..1000] + "..." : line);
    return Results.NoContent();
});

// 3. Forwards /api/... to wormix-api, so the game, the relay and the API share one address.
var apiBase = (app.Configuration["ApiUrl"] ?? "http://127.0.0.1:8000").TrimEnd('/');
var http = new HttpClient { Timeout = TimeSpan.FromSeconds(30) };
var logApi = app.Configuration.GetValue("LogApi", false);   // --LogApi true prints every API request
string[] skipHeaders = { "Host", "Connection", "Transfer-Encoding", "Keep-Alive", "Upgrade", "Content-Length", "Content-Type" };

app.Map("/api/{**rest}", async (HttpContext ctx) =>
{
    // /api/internal/ is for wormix-core only (it changes money, items, battle results).
    // wormix-core calls the API directly, so players never need it: don't expose it.
    if (ctx.Request.Path.StartsWithSegments("/api/internal", StringComparison.OrdinalIgnoreCase))
    {
        ctx.Response.StatusCode = 404;
        return;
    }
    var url = apiBase + ctx.Request.Path + ctx.Request.QueryString;
    using var req = new HttpRequestMessage(new HttpMethod(ctx.Request.Method), url);
    if (ctx.Request.ContentLength > 0 || ctx.Request.Headers.TransferEncoding.Count > 0)
    {
        using var buffer = new MemoryStream();
        await ctx.Request.Body.CopyToAsync(buffer, ctx.RequestAborted);
        var bytes = buffer.ToArray();
        if (logApi)
        {
            var text = System.Text.Encoding.UTF8.GetString(bytes);
            // Never log passwords or tokens.
            text = System.Text.RegularExpressions.Regex.Replace(text,
                "(\"(?:password|password_confirmation|auth_token|token)\"\\s*:\\s*\")[^\"]*\"", "$1***\"");
            log.LogInformation("API {Method} {Path} {Body}", ctx.Request.Method, ctx.Request.Path,
                text.Length > 300 ? text[..300] + "..." : text);
        }
        req.Content = new ByteArrayContent(bytes);
        if (ctx.Request.ContentType != null)
            req.Content.Headers.TryAddWithoutValidation("Content-Type", ctx.Request.ContentType);
    }
    foreach (var h in ctx.Request.Headers)
        if (!skipHeaders.Contains(h.Key, StringComparer.OrdinalIgnoreCase))
            req.Headers.TryAddWithoutValidation(h.Key, h.Value.ToArray());
    req.Headers.TryAddWithoutValidation("X-Forwarded-For", ctx.Connection.RemoteIpAddress?.ToString());

    HttpResponseMessage resp;
    try
    {
        resp = await http.SendAsync(req, HttpCompletionOption.ResponseHeadersRead, ctx.RequestAborted);
    }
    catch (Exception ex)
    {
        log.LogWarning("Can't reach API at {Url}: {Message}", url, ex.Message);
        ctx.Response.StatusCode = 502;
        return;
    }
    using (resp)
    {
        if (logApi && !resp.IsSuccessStatusCode)
            log.LogWarning("API answered {Status} for {Path}", (int)resp.StatusCode, ctx.Request.Path);
        ctx.Response.StatusCode = (int)resp.StatusCode;
        foreach (var h in resp.Headers.Concat(resp.Content.Headers))
            if (!skipHeaders.Contains(h.Key, StringComparer.OrdinalIgnoreCase) || h.Key == "Content-Type")
                ctx.Response.Headers[h.Key] = h.Value.ToArray();
        await resp.Content.CopyToAsync(ctx.Response.Body, ctx.RequestAborted);
    }
});

app.Map("/{port:int}", async (HttpContext ctx, int port) =>
{
    if (!ctx.WebSockets.IsWebSocketRequest)
        return Results.BadRequest("WebSocket expected");
    if (!allowed.Contains(port))
        return Results.StatusCode(403);

    using var tcp = new TcpClient { NoDelay = true };
    try
    {
        await tcp.ConnectAsync(target, port, ctx.RequestAborted);
    }
    catch (Exception ex)
    {
        log.LogWarning("Can't reach {Target}:{Port}: {Message}", target, port, ex.Message);
        return Results.StatusCode(502);
    }

    using var ws = await ctx.WebSockets.AcceptWebSocketAsync();
    var who = $"{ctx.Connection.RemoteIpAddress} -> {target}:{port}";
    log.LogInformation("Open   {Who}", who);

    var net = tcp.GetStream();
    using var cts = CancellationTokenSource.CreateLinkedTokenSource(ctx.RequestAborted);

    var toServer = Task.Run(async () =>
    {
        var buf = new byte[64 * 1024];
        while (!cts.IsCancellationRequested)
        {
            var r = await ws.ReceiveAsync(buf, cts.Token);
            if (r.MessageType == WebSocketMessageType.Close) break;
            await net.WriteAsync(buf.AsMemory(0, r.Count), cts.Token);
        }
    });

    // Server -> game: only send complete game messages.
    // wormix-core writes each message in several small pieces (flag byte, command id, length, body).
    // The game's 2012 network code breaks if a message *header* arrives split across deliveries
    // (it throws "End of file" / "invalid command header" and drops the data), so we reassemble here.
    // Message format: 7-byte header = 1 flag byte, 2-byte command id, 4-byte big-endian length
    // (lower 30 bits = body size), then the body.
    var toClient = Task.Run(async () =>
    {
        const int HeaderSize = 7;
        var buf = new byte[64 * 1024];
        var pending = new MemoryStream();
        while (!cts.IsCancellationRequested)
        {
            var n = await net.ReadAsync(buf, cts.Token);
            if (n == 0) break;
            pending.Write(buf, 0, n);

            var data = pending.GetBuffer();
            var total = (int)pending.Length;
            var complete = 0;
            while (total - complete >= HeaderSize)
            {
                var raw = (uint)(data[complete + 3] << 24 | data[complete + 4] << 16 | data[complete + 5] << 8 | data[complete + 6]);
                var bodyLength = (int)(raw & 0x3FFFFFFF);
                if (total - complete < HeaderSize + bodyLength) break;
                complete += HeaderSize + bodyLength;
            }
            if (complete == 0) continue;

            await ws.SendAsync(data.AsMemory(0, complete), WebSocketMessageType.Binary, true, cts.Token);

            var rest = total - complete;
            var next = new MemoryStream();
            if (rest > 0) next.Write(data, complete, rest);
            pending = next;
        }
    });

    try { await Task.WhenAny(toServer, toClient); } catch { /* connection dropped */ }
    cts.Cancel();
    try
    {
        if (ws.State == WebSocketState.Open)
            await ws.CloseAsync(WebSocketCloseStatus.NormalClosure, null, CancellationToken.None);
    }
    catch { /* already closed */ }

    log.LogInformation("Closed {Who}", who);
    return Results.Empty;
});

app.Run();
