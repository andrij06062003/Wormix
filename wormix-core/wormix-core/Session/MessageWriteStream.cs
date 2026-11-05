namespace wormix_core.Session;

/// <summary>
/// Write-only stream that the message serializers write to. Serializers write a message in several pieces
/// (header, body, hash). This collects them and sends each complete message to the client in one piece,
/// so a message from another thread (a PvP offer, a battle move) can never land in the middle of it.
/// Message format: 7-byte header (flags, 2-byte command id, 4-byte length whose lower 30 bits are the body size), then the body.
/// </summary>
public class MessageWriteStream(TcpSession session) : Stream
{
    private const int HeaderSize = 7;
    private readonly MemoryStream _pending = new();
    private readonly object _lock = new();

    public override void Write(byte[] buffer, int offset, int count) => Write(buffer.AsSpan(offset, count));

    public override void Write(ReadOnlySpan<byte> buffer)
    {
        lock (_lock)
        {
            _pending.Write(buffer);
            var data = _pending.GetBuffer();
            var total = (int)_pending.Length;
            var complete = 0;
            while (total - complete >= HeaderSize)
            {
                var length = (int)((uint)(data[complete + 3] << 24 | data[complete + 4] << 16 | data[complete + 5] << 8 | data[complete + 6]) & 0x3FFFFFFF);
                if (total - complete < HeaderSize + length)
                    break;
                complete += HeaderSize + length;
            }
            if (complete == 0)
                return;

            session.SendMessage(data.AsSpan(0, complete).ToArray());

            var rest = data.AsSpan(complete, total - complete).ToArray();
            _pending.SetLength(0);
            _pending.Write(rest);
        }
    }

    public override void WriteByte(byte value) => Write(new[] { value });

    public override void Flush() { }
    public override bool CanRead => false;
    public override bool CanSeek => false;
    public override bool CanWrite => true;
    public override long Length => throw new NotSupportedException();
    public override long Position { get => throw new NotSupportedException(); set => throw new NotSupportedException(); }
    public override int Read(byte[] buffer, int offset, int count) => throw new NotSupportedException();
    public override long Seek(long offset, SeekOrigin origin) => throw new NotSupportedException();
    public override void SetLength(long value) => throw new NotSupportedException();
}
