using System.Text;
using wormix_core.Extensions;
using wormix_core.Pragmatix.Flox.Secure;
using wormix_core.Pragmatix.Flox.Serialization.Internals;
using wormix_core.Pragmatix.Wormix.Messages.Structures;

namespace wormix_core.Pvp;

/// <summary>
/// Builds and reads the game's binary messages (big-endian, AS3 ByteArray layout).
/// </summary>
public static class Wire
{
    /// <summary>
    /// Build a complete message: header + body (+ the MD5 "security" suffix the client checks on some PvP messages).
    /// </summary>
    public static byte[] Message(uint commandId, Action<BinaryWriter> writeBody, bool secure = false)
    {
        using var body = new MemoryStream();
        using (var bw = new BinaryWriter(body, Encoding.UTF8, leaveOpen: true))
            writeBody(bw);
        var payload = body.ToArray();
        var hash = secure ? SerializeSecurityUtils.Secure(payload) : Array.Empty<byte>();

        using var output = new MemoryStream();
        var header = new BinaryCommandHeader();
        header.SetCommandId(commandId);
        header.SetLength((uint)(payload.Length + hash.Length));
        header.Write(output);
        output.Write(payload);
        output.Write(hash);
        return output.ToArray();
    }

    /// <summary>
    /// Raw message with an already-built body (used to forward a player's message to the other player unchanged).
    /// </summary>
    public static byte[] Raw(uint commandId, byte[] body) => Message(commandId, bw => bw.Write(body));

    public static void WriteInt32Be(this BinaryWriter bw, int value) => bw.WriteUInt32Be((uint)value);
    public static void WriteInt16Be(this BinaryWriter bw, int value) => bw.WriteUInt16Be((ushort)(short)value);

    /// <summary>
    /// UserProfileStructure as the client reads it inside PvP messages: a 2-byte length, then the structure.
    /// </summary>
    public static void WriteProfile(this BinaryWriter bw, UserProfileStructure profile)
    {
        using var ms = new MemoryStream();
        profile.Serialize(ms);
        bw.WriteUInt16Be((ushort)ms.Length);
        bw.Write(ms.ToArray());
    }

    /// <summary>
    /// Whether a message body ends with a valid MD5 suffix.
    /// </summary>
    public static bool IsSecure(byte[] body) => body.Length >= 16 && SerializeSecurityUtils.IsSecure(body);
}

/// <summary>
/// Reads a message body. Throws <see cref="EndOfStreamException"/> on a truncated body.
/// </summary>
public class WireReader(byte[] data)
{
    private int _pos;

    public int Remaining => data.Length - _pos;

    private ReadOnlySpan<byte> Take(int n)
    {
        if (_pos + n > data.Length)
            throw new EndOfStreamException();
        var span = data.AsSpan(_pos, n);
        _pos += n;
        return span;
    }

    public byte U8() => Take(1)[0];
    public bool Bool() => U8() != 0;
    public ushort U16() { var s = Take(2); return (ushort)(s[0] << 8 | s[1]); }
    public short S16() => (short)U16();
    public uint U32() { var s = Take(4); return (uint)(s[0] << 24 | s[1] << 16 | s[2] << 8 | s[3]); }
    public int S32() => (int)U32();
    public string Utf() { var n = U16(); return Encoding.UTF8.GetString(Take(n)); }

    public List<uint> U32Array()
    {
        var n = U16();
        var list = new List<uint>(n);
        for (var i = 0; i < n; i++)
            list.Add(U32());
        return list;
    }
}
