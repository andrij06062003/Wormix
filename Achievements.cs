// Achievements server for the game's "wormix_achieve" connection.
//
// Unlike the main and PvP connections, achievements use HTTP: the game POSTs binary commands to
// http://<host>:49199/<prefix>/ (play.html rewrites that URL to this host) with the session id from
// the login reply in the X-PRAGMATIX-CORP-SID header, and reads binary commands back from the reply.
// This decodes them and calls wormix-api's /api/internal/achievements/ endpoints with JSON.
//
// Command format: 7-byte header (flags, command id, length; see wormix-core BinaryCommandHeader),
// then the body. Most bodies end with MD5(body + SecretKey) so the game can spot tampering.

using System.Buffers.Binary;
using System.Net;
using System.Net.Http.Json;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;

static class Achievements
{
    const string SecretKey = "ygfA69i3Uz6bDO8";   // SecuritySettings.SECRET_KEY in the 1.05.0 client
    const string SessionHeader = "X-PRAGMATIX-CORP-SID";

    // Client -> server
    const int Ping = 16, AchieveLogin = 3001, GetAchievements = 3002, IncreaseAchievements = 3003, ChooseBonusItem = 3005;
    // Server -> client
    const int Pong = 10017, AchieveLoginSuccess = 13001, AchieveLoginError = 13002, GetAchievementsResult = 13003,
        IncreaseAchievementResult = 13004, ChooseBonusItemResult = 13006;

    sealed class Unauthorized : Exception { }

    public static void MapAchievements(this WebApplication app, string apiBase, HttpClient http, ILogger log)
    {
        // The prefix comes from the preloader ("wormix_achieve") or the client default ("wormix_achievements").
        foreach (var prefix in new[] { "wormix_achieve", "wormix_achievements" })
            app.MapPost($"/{prefix}/{{**rest}}", (Delegate)((HttpContext ctx) => Serve(ctx, apiBase, http, log)));
    }

    static async Task<IResult> Serve(HttpContext ctx, string apiBase, HttpClient http, ILogger log)
    {
        {
            using var buffer = new MemoryStream();
            await ctx.Request.Body.CopyToAsync(buffer, ctx.RequestAborted);
            var input = buffer.ToArray();
            string? session = ctx.Request.Headers[SessionHeader];

            var output = new MemoryStream();
            try
            {
                var pos = 0;
                while (input.Length - pos >= 7)
                {
                    var (id, length) = ReadHeader(input, pos);
                    pos += 7;
                    if (input.Length - pos < length) throw new InvalidDataException("command is cut short");
                    var body = input.AsMemory(pos, length);
                    pos += length;
                    await Handle(id, body, session, output, apiBase, http, ctx.RequestAborted);
                }
            }
            catch (Unauthorized)
            {
                // The game looks for "401" in a non-binary reply, then logs in again.
                return Results.Text("401 Unauthorized");
            }
            catch (Exception ex) when (ex is InvalidDataException or EndOfStreamException or ArgumentOutOfRangeException)
            {
                log.LogWarning("Bad achievements request: {Message}", ex.Message);
                return Results.Text("400 Bad Request");
            }
            catch (HttpRequestException ex)
            {
                log.LogWarning("Can't reach API for achievements: {Message}", ex.Message);
                return Results.StatusCode(502);
            }
            return Results.Bytes(output.ToArray(), "application/octet-stream");
        }
    }

    static async Task Handle(int id, ReadOnlyMemory<byte> body, string? session, Stream output,
        string apiBase, HttpClient http, CancellationToken ct)
    {
        switch (id)
        {
            case Ping:
                WriteCommand(output, Pong, new Writer(), secure: false);
                break;

            case AchieveLogin:
            {
                var r = new Reader(body);
                r.Utf(); r.Utf();                 // application id, social network
                var userId = r.Utf();
                var authKey = r.Utf();
                var res = await Call(http, apiBase, "login", new { id = userId, auth_key = authKey }, ct);
                var w = new Writer();
                if (res.GetProperty("ok").GetBoolean())
                {
                    w.Utf(res.GetProperty("session_id").GetString()!);
                    w.U32(res.GetProperty("login_time").GetUInt32());
                    WriteCommand(output, AchieveLoginSuccess, w, secure: true);
                }
                else
                {
                    w.I32(res.GetProperty("code").GetInt32());
                    WriteCommand(output, AchieveLoginError, w, secure: false);
                }
                break;
            }

            case GetAchievements:
            {
                CheckSecure(body.Span);
                var r = new Reader(body);
                var profileId = r.Utf();
                var res = await Call(http, apiBase, "get", new { session_id = session, profile_id = profileId }, ct);
                var w = new Writer();
                w.Utf(res.GetProperty("profile_id").GetString()!);
                w.ByteArray(res.GetProperty("indexes"));
                w.U16Array(res.GetProperty("values"));
                w.U8(res.GetProperty("invested").GetInt32());
                WriteCommand(output, GetAchievementsResult, w, secure: true);
                break;
            }

            case IncreaseAchievements:
            {
                CheckSecure(body.Span);
                var r = new Reader(body);
                var bodySession = r.Utf();
                var indexes = new int[r.U16()];
                for (var i = 0; i < indexes.Length; i++) indexes[i] = r.U8();
                var rises = new int[r.U16()];
                for (var i = 0; i < rises.Length; i++) rises[i] = r.I16();
                var timeScale = r.U32();
                var res = await Call(http, apiBase, "increase", new
                {
                    session_id = string.IsNullOrEmpty(bodySession) ? session : bodySession,
                    indexes, rises, time_scale = timeScale
                }, ct);
                var w = new Writer();
                w.ByteArray(res.GetProperty("indexes"));
                w.U16Array(res.GetProperty("values"));
                w.ByteArray(res.GetProperty("threshold_indexes"));
                w.U16Array(res.GetProperty("threshold_old_values"));
                w.U32(res.GetProperty("time_sequence").GetUInt32());
                w.U8(res.GetProperty("invested").GetInt32());
                WriteCommand(output, IncreaseAchievementResult, w, secure: true);
                break;
            }

            case ChooseBonusItem:
            {
                CheckSecure(body.Span);
                var itemId = new Reader(body).I32();
                var res = await Call(http, apiBase, "choose_bonus", new { session_id = session, item_id = itemId }, ct);
                var w = new Writer();
                w.I16(res.GetProperty("result").GetInt32());
                w.I32(res.GetProperty("item_id").GetInt32());
                WriteCommand(output, ChooseBonusItemResult, w, secure: true);
                break;
            }

            default:
                throw new InvalidDataException($"unknown achievements command {id}");
        }
    }

    static async Task<JsonElement> Call(HttpClient http, string apiBase, string action, object payload, CancellationToken ct)
    {
        using var req = new HttpRequestMessage(HttpMethod.Post, $"{apiBase}/api/internal/achievements/{action}")
        {
            Content = JsonContent.Create(payload)
        };
        req.Headers.Add("Accept", "application/json");
        req.Headers.Add("X-TCP-SESSION", "achievements");
        using var resp = await http.SendAsync(req, ct);
        if (resp.StatusCode == HttpStatusCode.Unauthorized) throw new Unauthorized();
        resp.EnsureSuccessStatusCode();
        return await resp.Content.ReadFromJsonAsync<JsonElement>(ct);
    }

    // --- binary format ---

    static (int id, int length) ReadHeader(byte[] data, int pos)
    {
        uint flags = data[pos];
        uint rawId = BinaryPrimitives.ReadUInt16BigEndian(data.AsSpan(pos + 1));
        uint rawLength = BinaryPrimitives.ReadUInt32BigEndian(data.AsSpan(pos + 3));
        var valid = (flags & 87) == 87
            && ((flags & 128) != 0) != ((rawId & 0x8000) != 0)
            && ((flags & 32) != 0) != ((rawLength & 0x40000000) != 0)
            && ((flags & 8) != 0) != ((rawLength & 0x80000000) != 0);
        if (!valid) throw new InvalidDataException("invalid command header");
        return ((int)(rawId & 0x7FFF), (int)(rawLength & 0x3FFFFFFF));
    }

    static void WriteCommand(Stream output, int id, Writer body, bool secure)
    {
        var bytes = body.ToArray();
        if (secure) bytes = [.. bytes, .. Hash(bytes)];

        // Each sync bit is carried either in the flags byte or in the id/length field, picked at random.
        uint flags = 87, rawId = (uint)id, rawLength = (uint)bytes.Length;
        if (Random.Shared.Next(2) == 0) flags |= 128; else rawId |= 0x8000;
        if (Random.Shared.Next(2) == 0) flags |= 32; else rawLength |= 0x40000000;
        if (Random.Shared.Next(2) == 0) flags |= 8; else rawLength |= 0x80000000;

        Span<byte> header = stackalloc byte[7];
        header[0] = (byte)flags;
        BinaryPrimitives.WriteUInt16BigEndian(header[1..], (ushort)rawId);
        BinaryPrimitives.WriteUInt32BigEndian(header[3..], rawLength);
        output.Write(header);
        output.Write(bytes);
    }

    static byte[] Hash(ReadOnlySpan<byte> data) => MD5.HashData([.. data, .. Encoding.UTF8.GetBytes(SecretKey)]);

    static void CheckSecure(ReadOnlySpan<byte> body)
    {
        if (body.Length < 16 || !Hash(body[..^16]).AsSpan().SequenceEqual(body[^16..]))
            throw new InvalidDataException("bad command hash");
    }

    sealed class Reader(ReadOnlyMemory<byte> data)
    {
        ReadOnlyMemory<byte> rest = data;

        ReadOnlySpan<byte> Take(int n)
        {
            if (rest.Length < n) throw new EndOfStreamException();
            var s = rest.Span[..n];
            rest = rest[n..];
            return s;
        }

        public int U8() => Take(1)[0];
        public int U16() => BinaryPrimitives.ReadUInt16BigEndian(Take(2));
        public int I16() => BinaryPrimitives.ReadInt16BigEndian(Take(2));
        public int I32() => BinaryPrimitives.ReadInt32BigEndian(Take(4));
        public uint U32() => BinaryPrimitives.ReadUInt32BigEndian(Take(4));
        public string Utf() => Encoding.UTF8.GetString(Take(U16()));
    }

    sealed class Writer
    {
        readonly MemoryStream s = new();
        readonly byte[] b = new byte[4];

        public void U8(int v) => s.WriteByte((byte)v);
        public void I16(int v) { BinaryPrimitives.WriteInt16BigEndian(b, (short)v); s.Write(b, 0, 2); }
        public void U16(int v) { BinaryPrimitives.WriteUInt16BigEndian(b, (ushort)v); s.Write(b, 0, 2); }
        public void I32(int v) { BinaryPrimitives.WriteInt32BigEndian(b, v); s.Write(b, 0, 4); }
        public void U32(uint v) { BinaryPrimitives.WriteUInt32BigEndian(b, v); s.Write(b, 0, 4); }

        public void Utf(string v)
        {
            var bytes = Encoding.UTF8.GetBytes(v);
            U16(bytes.Length);
            s.Write(bytes);
        }

        public void ByteArray(JsonElement values)
        {
            U16(values.GetArrayLength());
            foreach (var v in values.EnumerateArray()) U8(v.GetInt32());
        }

        public void U16Array(JsonElement values)
        {
            U16(values.GetArrayLength());
            foreach (var v in values.EnumerateArray()) U16(v.GetInt32());
        }

        public byte[] ToArray() => s.ToArray();
    }
}
