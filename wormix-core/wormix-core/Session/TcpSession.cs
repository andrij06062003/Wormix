using System.Net.Sockets;
using StackExchange.Redis;
using wormix_core.Extensions;
using wormix_core.Handlers;
using wormix_core.Pragmatix.Flox.Serialization.Internals;
using wormix_core.Server;

namespace wormix_core.Session;

public class TcpSession
{
    /// <summary>
    /// Server base object
    /// </summary>
    public readonly TcpServer Server;

    /// <summary>
    /// Current TcpClient
    /// </summary>
    protected TcpClient? sessionClient { get; private set; }

    /// <summary>
    /// Map of commands handlers
    /// </summary>
    protected Dictionary<uint, GameMessageHandler> handlers { get; private set; }

    /// <summary>
    /// Session ID
    /// </summary>
    protected Guid Id;

    /// <summary>
    /// Main session thread
    /// </summary>
    private readonly Thread _sessionThread;


    /// <summary>
    /// Cache redis connection (null for sessions that don't use Redis, see <see cref="UsesRedis"/>)
    /// </summary>
    private readonly ConnectionMultiplexer? _redisConnection;

    private readonly IDatabase? _cache;

    /// <summary>
    /// Redis cache database
    /// </summary>
    public IDatabase Cache => _cache ?? throw new InvalidOperationException($"{GetType().Name} has no Redis connection");

    /// <summary>
    /// Session auth token
    /// </summary>
    private string _token;

    /// <summary>
    /// Serializes writes to the socket: messages can be sent to a session from other sessions' threads
    /// (PvP offers, battle relay), so whole messages must go out one at a time.
    /// </summary>
    private readonly object _sendLock = new();

    /// <summary>
    /// Stream handed to serializers. Collects the pieces a serializer writes and sends each complete message at once.
    /// </summary>
    private readonly MessageWriteStream _writeStream;

    private int _stopped;

    /// <summary>
    /// Whether this kind of session needs its own Redis connection (the PvP server doesn't).
    /// </summary>
    protected virtual bool UsesRedis => true;

    public TcpSession(TcpServer server)
    {
        if (UsesRedis)
        {
            _redisConnection = ConnectionMultiplexer.Connect($"{Config.RedisConfig?.Host}:{Config.RedisConfig?.Port}");
            _redisConnection.ConnectionFailed += RedisConnectionOnConnectionFailed;
            _cache = _redisConnection.GetDatabase();
        }

        handlers = new();
        _token = string.Empty;
        Server = server;
        _writeStream = new MessageWriteStream(this);
        _sessionThread = new Thread(MessageLoop) { IsBackground = true };
    }

    private void RedisConnectionOnConnectionFailed(object? sender, ConnectionFailedEventArgs e)
    {
        ColorPrint.WriteLine($"{this}\t[{Id}] Redis connection was closed!", ConsoleColor.Red);
        CloseSession();
    }


    /// <summary>
    /// Setup client session
    /// </summary>
    /// <param name="id">Session ID</param>
    /// <param name="client">Session TcpCLient object</param>
    public void SetupSession(Guid id, TcpClient client)
    {
        Id = id;
        sessionClient = client;
        client.NoDelay = true;
    }

    /// <summary>
    /// Start session thread
    /// </summary>
    public void StartSession()
    {
        _sessionThread.Start();
    }

    /// <summary>
    /// Close TCP session
    /// </summary>
    public void CloseSession()
    {
        try { sessionClient?.Client.Close(); } catch { /* already closed */ }
    }

    /// <summary>
    /// Stop tcp session & terminate thread
    /// </summary>
    public void StopSession()
    {
        // Called both by the session's own loop and by other sessions (e.g. a second login); run it once.
        if (Interlocked.Exchange(ref _stopped, 1) == 1)
            return;
        try { sessionClient?.Close(); } catch { /* already closed */ }
        if (_cache != null)
        {
            try { Facades.Cache.ClearCache(this); } catch { /* Redis gone */ }
        }
        if(_sessionThread.IsAlive && _sessionThread != Thread.CurrentThread)
            _sessionThread.Interrupt();
        OnDisconnected();
    }

    /// <summary>
    /// Get game session token
    /// </summary>
    /// <returns>Session key</returns>
    public string GetToken() => _token;

    /// <summary>
    /// Set game session token
    /// </summary>
    /// <param name="token">Session key</param>
    /// <returns></returns>
    public string SetToken(string token) => _token = token;

    /// <summary>
    /// Send one or more complete messages to the client. Safe to call from any thread.
    /// </summary>
    /// <returns>false if the connection is gone</returns>
    public bool SendMessage(byte[] message)
    {
        var socket = sessionClient?.Client;
        if (socket == null)
            return false;
        lock (_sendLock)
        {
            try
            {
                var sent = 0;
                while (sent < message.Length)
                    sent += socket.Send(message, sent, message.Length - sent, SocketFlags.None);
                return true;
            }
            catch (Exception ex) when (ex is SocketException or ObjectDisposedException)
            {
                return false;
            }
        }
    }

    /// <summary>
    /// Stream for serializers to write responses to. Each complete message is sent as one piece.
    /// </summary>
    public Stream GetStream() => _writeStream;

    /// <summary>
    /// Send message to specific client
    /// </summary>
    /// <param name="sessionId">Client session GUID</param>
    /// <param name="message">client message</param>
    public void SendTo(Guid sessionId, byte[] message)
    {
        var session = Server.FindSession(sessionId);
        if(session == null)
            return;
        session.SendMessage(message);
    }

    /// <summary>
    /// On TcpClient connected handler
    /// </summary>
    protected virtual void OnConnected()
    {
        //Init game commands handlers
        if (handlers.Count == 0)
            handlers = GetHandlers();
    }

    /// <summary>
    /// TcpClient disconnected handler
    /// </summary>
    protected virtual void OnDisconnected()
    {
        _redisConnection?.Close();
    }

    /// <summary>
    /// TcpClient on message handler
    /// </summary>
    /// <param name="dataStream">TcpClient NetworkStream object</param>
    protected virtual void OnMessage(Stream dataStream)
    {

    }

    /// <summary>
    /// Handle a message without a registered handler. Return true if it was handled.
    /// </summary>
    protected virtual bool HandleRaw(uint commandId, byte[] body) => false;

    /// <summary>
    /// Main data receive loop
    /// </summary>
    private void MessageLoop()
    {
        try
        {
            OnConnected();
            while (true)
            {
                if (sessionClient == null)
                    break;
                try
                {
                    byte[] peakByte = new byte[1];
                    if (sessionClient.Client.Receive(peakByte, SocketFlags.Peek) == 0)
                        break;
                }
                catch
                {
                    break;
                }

                if (sessionClient.Available == 0)
                    continue;
                OnMessage(sessionClient.GetStream());
            }
        }
        catch (ThreadInterruptedException)
        {
            // StopSession from another thread
        }
        catch (Exception ex)
        {
            ColorPrint.WriteLine($"[{this}] Session error: {ex.Message}", ConsoleColor.Red);
        }

        Server.UnregisterSession(Id);
    }

    /// <summary>
    /// Get session GUID
    /// </summary>
    /// <returns>Session GUID</returns>
    public Guid GetSessionId() => Id;

    /// <summary>
    /// Return game message handlers map
    /// </summary>
    /// <returns></returns>
    protected virtual Dictionary<uint, GameMessageHandler> GetHandlers()
    {
        return new();
    }

    /// <summary>
    /// Process game message
    /// </summary>
    /// <param name="dataStream">TcpClient NetworkStream object</param>
    /// <param name="log">Log binary data</param>
    protected void ProcessMessage(Stream dataStream, bool log = true)
    {
        try
        {
            BinaryCommandHeader cmd = new BinaryCommandHeader();
            cmd.Read(dataStream);

            if (log)
            {
                ColorPrint.WriteLine($"[{this}] New data: {sessionClient?.Available}", ConsoleColor.Green);
                ColorPrint.WriteLine($"[{this}] CMD ID: {cmd.GetCommandId()}", ConsoleColor.Green);
                ColorPrint.WriteLine($"[{this}] Length: {cmd.GetLength()}", ConsoleColor.Green);
            }

            if (cmd.GetLength() > MaxMessageLength)
                throw new ArgumentException($"Message too long ({cmd.GetLength()} bytes)");

            // The body may arrive in several TCP segments: read all of it.
            byte[] data = new byte[cmd.GetLength()];
            if (cmd.GetLength() != 0)
            {
                dataStream.ReadExactly(data);
                if(log)
                    ColorPrint.WriteLine($"[{this}] RAW:\n{HexDump.HexDump.Format(data)}", ConsoleColor.Yellow);
            }

            if (handlers.ContainsKey(cmd.GetCommandId()))
            {
                if(log)
                    ColorPrint.WriteLine($"[{this}] Processing via: {handlers[cmd.GetCommandId()]}", ConsoleColor.DarkGreen);
                handlers[cmd.GetCommandId()].Handle(data, cmd);
            }
            else if (!HandleRaw(cmd.GetCommandId(), data))
            {
                if(log)
                    ColorPrint.WriteLine($"[{this}] Unknown CMD ID", ConsoleColor.Red);
            }
        }
        catch (ArgumentException ex)
        {
            ColorPrint.WriteLine($"[{this}] Argument exception: {ex.Message}", ConsoleColor.Red);
            ColorPrint.WriteLine($"[{this}] Closing client connection", ConsoleColor.Red);
            CloseSession();
        }
        catch (Exception ex)
        {
            ColorPrint.WriteLine($"[{this}] Unknown error: {ex.Message}", ConsoleColor.Red);
            ColorPrint.WriteLine($"[{this}] Closing client connection", ConsoleColor.Red);
            CloseSession();
        }
    }

    /// <summary>
    /// Largest message a client may send (the biggest real ones, PvP move batches, are a few KB).
    /// </summary>
    private const uint MaxMessageLength = 1024 * 1024;
}
