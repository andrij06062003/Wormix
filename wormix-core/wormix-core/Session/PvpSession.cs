using wormix_core.Extensions;
using wormix_core.Pvp;
using wormix_core.Server;

namespace wormix_core.Session;

/// <summary>
/// A player's connection to the PvP server for one battle. See <see cref="PvpBattle"/>.
/// </summary>
public class PvpSession(TcpServer server) : TcpSession(server)
{
    /// <summary>
    /// The battle this connection logged in to.
    /// </summary>
    public PvpBattle? Battle;

    protected override bool UsesRedis => false;

    protected override void OnConnected()
    {
        base.OnConnected();
        ColorPrint.WriteLine($"[PvP] [{Id}] {sessionClient?.Client.RemoteEndPoint} connected", ConsoleColor.Cyan);
    }

    protected override void OnDisconnected()
    {
        ColorPrint.WriteLine($"[PvP] [{Id}] disconnected", ConsoleColor.Cyan);
        Battle?.OnDisconnected(this);
        Battle = null;
        base.OnDisconnected();
    }

    protected override void OnMessage(Stream dataStream)
    {
        // Moves arrive many times a second: don't dump them to the log.
        ProcessMessage(dataStream, log: false);
    }

    protected override bool HandleRaw(uint commandId, byte[] body)
    {
        try
        {
            switch (commandId)
            {
                case PvpCmd.Ping:
                    SendMessage(Wire.Message(PvpCmd.Pong, _ => { }));
                    return true;
                case PvpCmd.BattleLogin:
                    Battles.Login(this, body);
                    return true;
                case PvpCmd.BattleReconnect:
                    Battles.Reconnect(this, body);
                    return true;
                default:
                    if (Battle == null)
                        return false;
                    Battle.OnMessage(this, commandId, body);
                    return true;
            }
        }
        catch (EndOfStreamException)
        {
            ColorPrint.WriteLine($"[PvP] [{Id}] truncated command {commandId}", ConsoleColor.Red);
            return true;
        }
    }

    public void SendLoginError(int code)
    {
        SendMessage(Wire.Message(PvpCmd.LoginError, bw => bw.WriteInt32Be(code)));
    }
}
