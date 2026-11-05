using wormix_core.Extensions;
using wormix_core.Session;

namespace wormix_core.Pvp;

/// <summary>
/// Command ids of the PvP connection (from the 1.05.0 client).
/// </summary>
public static class PvpCmd
{
    // client -> server
    public const uint Ping = 16;
    public const uint Chat = 17;              // also server -> client (forwarded)
    public const uint BattleLogin = 1001;
    public const uint DropPlayer = 1006;      // client leaves: surrender, cheat detected, desync
    public const uint Action = 1007;          // also server -> client (forwarded)
    public const uint EndTurn = 1014;         // also server -> client (forwarded)
    public const uint BattleReconnect = 1015;
    public const uint EndTurnResponse = 1017;
    public const uint RetryRequestClient = 1018;

    // server -> client
    public const uint Pong = 10017;
    public const uint RetryRequestServer = 1016;
    public const uint StartTurn = 1019;
    public const uint EndBattle = 1021;
    public const uint LoginSuccess = 2001;
    public const uint LoginError = 2004;
    public const uint SystemMessage = 2021;
}

/// <summary>
/// One player of a battle.
/// </summary>
public class PvpPlayer(uint userId, bool firstTurn, string token)
{
    public readonly uint UserId = userId;
    public readonly bool FirstTurn = firstTurn;
    public byte PlayerNum => (byte)(FirstTurn ? 1 : 2);

    /// <summary>
    /// Main-server session token, for recording the result in the API.
    /// </summary>
    public string Token = token;

    public PvpSession? Session;
    public bool EverConnected;
    public DateTime? DisconnectedAt;
    public bool Left;   // ended the battle on their side (surrender etc.): don't wait for them any more

    public bool Connected => Session != null;
}

/// <summary>
/// A battle between two players on the PvP server.
///
/// The game simulation runs in both clients; the server relays the active player's moves to the other player
/// and referees the turns:
///   1. the active player sends moves (Action) and finally EndTurn: both are forwarded to the other player;
///   2. the other player replays them and answers EndTurnResponse;
///   3. when both agree on who is eliminated, the server either ends the battle (PvpEndBattle) or starts the
///      next turn for the other player (StartTurn to both).
/// It also handles reconnects (resending what a player missed), surrender, disconnects and inactivity.
/// </summary>
public class PvpBattle
{
    public static readonly TimeSpan NoShowTimeout = TimeSpan.FromSeconds(60);
    public static readonly TimeSpan ReconnectTimeout = TimeSpan.FromSeconds(45);
    public static readonly TimeSpan TurnTimeout = TimeSpan.FromSeconds(150);
    public static readonly TimeSpan ResponseTimeout = TimeSpan.FromSeconds(120);
    public static readonly TimeSpan EndTurnResendInterval = TimeSpan.FromSeconds(5);

    public readonly uint Id;
    public readonly int Wager;
    public readonly PvpPlayer[] Players;
    public readonly DateTime Created = DateTime.UtcNow;

    private readonly object _lock = new();

    private int _turn = 1;
    private int _active;                                   // index of the player whose turn it is
    private readonly List<(ushort Num, byte[] Message)> _turnCommands = new();   // active player's messages this turn
    private ushort _lastCommandNum;
    private (List<uint> Eliminated, int State)? _activeEnd, _passiveEnd;
    private DateTime _activeEndAt, _lastEndTurnSent, _lastActivity = DateTime.UtcNow;
    private byte[]? _lastStartTurn;

    public bool Over { get; private set; }
    public DateTime OverAt { get; private set; }

    /// <summary>
    /// Result per user id: 1 won, 0 draw/cancelled, -1 lost. Set when the battle is over.
    /// </summary>
    public readonly Dictionary<uint, int> Results = new();

    public PvpBattle(uint id, int wager, PvpPlayer first, PvpPlayer second)
    {
        Id = id;
        Wager = wager;
        Players = new[] { first, second };
        _active = first.FirstTurn ? 0 : 1;
    }

    public bool HasPlayer(uint userId) => Players.Any(p => p.UserId == userId);

    private PvpPlayer? PlayerOf(PvpSession session) => Players.FirstOrDefault(p => p.Session == session);
    private PvpPlayer Other(PvpPlayer p) => Players[0] == p ? Players[1] : Players[0];
    private PvpPlayer ActivePlayer => Players[_active];
    private PvpPlayer PassivePlayer => Players[1 - _active];

    private void Log(string text) => ColorPrint.WriteLine($"[PvP {Id}] {text}", ConsoleColor.Cyan);

    private static void Send(PvpPlayer p, byte[] message) => p.Session?.SendMessage(message);

    // ---- Login / reconnect -------------------------------------------------------------------------------

    /// <summary>
    /// First login of a player (BattleLogin). Returns false if the player isn't part of this battle.
    /// </summary>
    public bool Login(PvpSession session, uint userId)
    {
        lock (_lock)
        {
            var p = Players.FirstOrDefault(x => x.UserId == userId);
            if (p == null)
                return false;
            var reconnect = p.EverConnected;
            Attach(p, session);
            Log($"{userId} logged in{(reconnect ? " again" : "")}");

            if (Over)
            {
                SendEndBattle(p);
                return true;
            }
            if (reconnect)
                Send(Other(p), SystemMessage(PvpSystemType.Reconnected, p.UserId));
            if (p == PassivePlayer)
                ResendTurnCommands(p, 0);   // moves the active player made before this player connected
            return true;
        }
    }

    /// <summary>
    /// A player whose PvP connection dropped comes back (BattleReconnect) and says how far it got.
    /// </summary>
    public bool Reconnect(PvpSession session, uint userId, int turnNum, ushort lastCommandNum)
    {
        lock (_lock)
        {
            var p = Players.FirstOrDefault(x => x.UserId == userId);
            if (p == null)
                return false;
            Attach(p, session);
            Log($"{userId} reconnected at turn {turnNum} (server turn {_turn}), last command {lastCommandNum}");
            if (Over)
            {
                SendEndBattle(p);
                return true;
            }
            Send(Other(p), SystemMessage(PvpSystemType.Reconnected, p.UserId));

            if (turnNum < _turn && _lastStartTurn != null)
            {
                Send(p, _lastStartTurn);   // missed the start of the current turn
            }
            else if (p == PassivePlayer)
            {
                if (turnNum == _turn)
                    ResendTurnCommands(p, lastCommandNum);
                else if (turnNum == _turn + 1 && _activeEnd != null && _passiveEnd == null)
                {
                    // It already answered EndTurn but the answer was lost with the connection.
                    // Both had the same moves, so go with the active player's view.
                    _passiveEnd = _activeEnd;
                    TryFinishTurn();
                }
            }
            else if (_activeEnd == null && (turnNum == _turn || turnNum == _turn + 1))
            {
                // The active player may have sent moves (or its EndTurn) that never arrived: ask for everything after the last one we have.
                var from = Math.Max(_lastCommandNum + 1, _turn == 1 ? 1 : 2);
                Send(p, Wire.Message(PvpCmd.RetryRequestServer, bw =>
                {
                    bw.WriteUInt32Be(Id);
                    bw.WriteUInt16Be(2);
                    bw.WriteInt16Be(from);
                    bw.WriteInt16Be(-1);
                    bw.WriteUInt16Be((ushort)_turn);
                }, secure: true));
            }
            return true;
        }
    }

    private void Attach(PvpPlayer p, PvpSession session)
    {
        if (p.Session != null && p.Session != session)
        {
            var old = p.Session;
            old.Battle = null;
            old.CloseSession();
        }
        p.Session = session;
        p.EverConnected = true;
        p.DisconnectedAt = null;
        session.Battle = this;
        session.SendMessage(Wire.Message(PvpCmd.LoginSuccess, bw =>
        {
            bw.WriteUTF8("");
            bw.Write(p.PlayerNum);
        }));
    }

    public void OnDisconnected(PvpSession session)
    {
        lock (_lock)
        {
            var p = PlayerOf(session);
            if (p == null)
                return;
            p.Session = null;
            if (Over || p.Left)
                return;
            p.DisconnectedAt = DateTime.UtcNow;
            Log($"{p.UserId} disconnected");
            Send(Other(p), SystemMessage(PvpSystemType.Disconnected, p.UserId));
        }
    }

    // ---- Messages during the battle ----------------------------------------------------------------------

    public void OnMessage(PvpSession session, uint cmd, byte[] body)
    {
        lock (_lock)
        {
            var p = PlayerOf(session);
            if (p == null)
                return;
            try
            {
                switch (cmd)
                {
                    case PvpCmd.Action: OnAction(p, body); break;
                    case PvpCmd.EndTurn: OnEndTurn(p, body); break;
                    case PvpCmd.EndTurnResponse: OnEndTurnResponse(p, body); break;
                    case PvpCmd.RetryRequestClient: OnRetryRequest(p, body); break;
                    case PvpCmd.DropPlayer: OnDropPlayer(p, body); break;
                    case PvpCmd.Chat:
                        if (body.Length <= 1024)
                            Send(Other(p), Wire.Raw(PvpCmd.Chat, body));
                        break;
                    default:
                        Log($"unexpected command {cmd} from {p.UserId}");
                        break;
                }
            }
            catch (EndOfStreamException)
            {
                Log($"truncated command {cmd} from {p.UserId}");
            }
        }
    }

    private void OnAction(PvpPlayer p, byte[] body)
    {
        // firstFrame u32, lastFrame u32, turnNum u16, commandNum u16, battleId u32, playerNum u8, actions...
        var r = new WireReader(body);
        r.U32(); r.U32();
        var turn = r.U16();
        var num = r.U16();
        if (Over || p != ActivePlayer || turn != _turn || _activeEnd != null)
            return;
        Store(num, Wire.Raw(PvpCmd.Action, body));
    }

    private void OnEndTurn(PvpPlayer p, byte[] body)
    {
        // turnNum u16, commandNum u16, battleId i32, playerNum u8, eliminated u32[], battleState i32
        var r = new WireReader(body);
        var turn = r.U16();
        var num = r.U16();
        r.U32(); r.U8();
        var eliminated = r.U32Array();
        var state = r.S32();
        if (Over || p != ActivePlayer || turn != _turn || _activeEnd != null)
            return;
        Store(num, Wire.Raw(PvpCmd.EndTurn, body));
        _activeEnd = (eliminated, state);
        _activeEndAt = _lastEndTurnSent = DateTime.UtcNow;
        TryFinishTurn();
    }

    private void Store(ushort num, byte[] message)
    {
        _lastActivity = DateTime.UtcNow;
        if (_turnCommands.Any(c => c.Num == num))
            return;   // a resend of something we already have
        _turnCommands.Add((num, message));
        _turnCommands.Sort((a, b) => a.Num.CompareTo(b.Num));
        _lastCommandNum = Math.Max(_lastCommandNum, num);
        Send(PassivePlayer, message);
    }

    private void OnEndTurnResponse(PvpPlayer p, byte[] body)
    {
        // turnNum u16, commandNum u16, battleId i32, eliminated u32[], battleState i32
        var r = new WireReader(body);
        var turn = r.U16();
        r.U16(); r.U32();
        var eliminated = r.U32Array();
        var state = r.S32();
        if (Over || p != PassivePlayer || turn != _turn || _passiveEnd != null)
            return;
        _passiveEnd = (eliminated, state);
        TryFinishTurn();
    }

    private void OnRetryRequest(PvpPlayer p, byte[] body)
    {
        // commandNum u16 (last one it has), turnNum u16, battleId i32
        var r = new WireReader(body);
        var have = r.U16();
        var turn = r.U16();
        if (!Over && p == PassivePlayer && turn == _turn)
            ResendTurnCommands(p, have);
    }

    private void ResendTurnCommands(PvpPlayer p, int after)
    {
        foreach (var c in _turnCommands.Where(c => c.Num > after))
            Send(p, c.Message);
    }

    private void OnDropPlayer(PvpPlayer p, byte[] body)
    {
        // playerId u32, battleId u32, type i32 (PvpEndBattle.*)
        var r = new WireReader(body);
        r.U32(); r.U32();
        var type = r.S32();
        if (Over)
            return;
        p.Left = true;
        var other = Other(p);
        if (type == PvpDropType.Desync)
        {
            Log($"{p.UserId} reports desync: draw");
            Finish(new() { [p.UserId] = 0, [other.UserId] = 0 }, desync: true);
            return;
        }
        Log($"{p.UserId} left the battle (type {type}): {other.UserId} wins");
        Send(other, SystemMessage(type == PvpDropType.IAmCheater ? PvpSystemType.Cheater : PvpSystemType.Surrendered, p.UserId));
        Finish(new() { [p.UserId] = -1, [other.UserId] = 1 });
    }

    private void TryFinishTurn()
    {
        if (_activeEnd is not { } a || _passiveEnd is not { } b)
            return;

        if (!a.Eliminated.ToHashSet().SetEquals(b.Eliminated))
        {
            Log($"desync at turn {_turn}: eliminated [{string.Join(",", a.Eliminated)}] vs [{string.Join(",", b.Eliminated)}]");
            Finish(Players.ToDictionary(x => x.UserId, _ => 0), desync: true);
            return;
        }
        if (a.State != b.State)
            Log($"state hash differs at turn {_turn} ({a.State} vs {b.State}); continuing");

        var out_ = Players.Where(x => a.Eliminated.Contains(x.UserId)).ToList();
        if (out_.Count == 2)
        {
            Finish(Players.ToDictionary(x => x.UserId, _ => 0));
            return;
        }
        if (out_.Count == 1)
        {
            Finish(Players.ToDictionary(x => x.UserId, x => out_[0] == x ? -1 : 1));
            return;
        }

        // Next turn: the other player.
        _turn++;
        _active = 1 - _active;
        _turnCommands.Clear();
        _lastCommandNum = 0;
        _activeEnd = _passiveEnd = null;
        _lastActivity = DateTime.UtcNow;
        var activeId = ActivePlayer.UserId;
        _lastStartTurn = Wire.Message(PvpCmd.StartTurn, bw =>
        {
            bw.WriteUInt32Be(Id);
            bw.WriteUInt32Be(activeId);
            bw.WriteUInt16Be((ushort)_turn);
            bw.WriteUInt16Be(0);
            bw.WriteUInt16Be(0);   // dropped players
        }, secure: true);
        foreach (var x in Players)
            Send(x, _lastStartTurn);
    }

    // ---- Timeouts ----------------------------------------------------------------------------------------

    /// <summary>
    /// Called about once a second.
    /// </summary>
    public void Tick()
    {
        lock (_lock)
        {
            if (Over)
                return;
            var now = DateTime.UtcNow;

            // Someone never showed up, or didn't come back after a disconnect.
            var gone = Players.Where(p =>
                (!p.EverConnected && now - Created > NoShowTimeout) ||
                (p.DisconnectedAt is { } at && now - at > ReconnectTimeout)).ToList();
            if (gone.Count == 2)
            {
                Log("both players gone: cancelled");
                Finish(Players.ToDictionary(x => x.UserId, _ => 0));
                return;
            }
            // One is gone; the other wins if it's here (if it's away too, wait for it to come back or time out).
            if (gone.Count == 1 && Other(gone[0]).Connected)
            {
                var other = Other(gone[0]);
                Log($"{gone[0].UserId} didn't come back: {other.UserId} wins");
                Send(other, SystemMessage(PvpSystemType.DroppedByReconnectionTimeout, gone[0].UserId));
                Finish(new() { [gone[0].UserId] = -1, [other.UserId] = 1 });
                return;
            }

            if (_activeEnd == null)
            {
                // The active player stopped playing (connected, but nothing for a long time).
                if (ActivePlayer.Connected && now - _lastActivity > TurnTimeout)
                {
                    Log($"{ActivePlayer.UserId} inactive: {PassivePlayer.UserId} wins");
                    Send(PassivePlayer, SystemMessage(PvpSystemType.DroppedByCommandTimeout, ActivePlayer.UserId));
                    Finish(new() { [ActivePlayer.UserId] = -1, [PassivePlayer.UserId] = 1 });
                }
            }
            else if (_passiveEnd == null && PassivePlayer.Connected)
            {
                if (now - _activeEndAt > ResponseTimeout)
                {
                    Log($"{PassivePlayer.UserId} never confirmed the turn: {ActivePlayer.UserId} wins");
                    Send(ActivePlayer, SystemMessage(PvpSystemType.DroppedByResponseTimeout, PassivePlayer.UserId));
                    Finish(new() { [PassivePlayer.UserId] = -1, [ActivePlayer.UserId] = 1 });
                }
                else if (now - _lastEndTurnSent > EndTurnResendInterval)
                {
                    // The client ignores EndTurn until it's ready to replay the turn (e.g. still building the map
                    // on turn 1), so repeat it. A client that already has it ignores the copy; one that misses moves
                    // asks for them (RetryRequestClient).
                    _lastEndTurnSent = now;
                    var endTurn = _turnCommands.LastOrDefault().Message;
                    if (endTurn != null)
                        Send(PassivePlayer, endTurn);
                }
            }
        }
    }

    // ---- End -----------------------------------------------------------------------------------------------

    private void Finish(Dictionary<uint, int> results, bool desync = false)
    {
        Over = true;
        OverAt = DateTime.UtcNow;
        foreach (var (id, r) in results)
            Results[id] = r;
        Log($"over: {string.Join(", ", results.Select(r => $"{r.Key}={r.Value}"))}{(desync ? " (desync)" : "")}");

        _endBattle = Wire.Message(PvpCmd.EndBattle, bw =>
        {
            bw.Write((byte)(desync ? 1 : 0));
            bw.WriteUInt32Be(Id);
            bw.WriteUInt16Be((ushort)_turn);
            bw.WriteUInt16Be(_lastCommandNum);
            bw.WriteUInt16Be((ushort)Players.Length);
            foreach (var p in Players)
                bw.WriteUInt32Be(p.UserId);
            // results: 0 won, 3 draw, 1 lost
            bw.WriteUInt16Be((ushort)Players.Length);
            foreach (var p in Players)
                bw.Write((byte)(Results[p.UserId] switch { 1 => 0, 0 => 3, _ => 1 }));
            // signatures (not checked by this server): empty
            bw.WriteUInt16Be((ushort)Players.Length);
            foreach (var _ in Players)
            {
                bw.WriteUInt16Be(0);
                bw.WriteUInt16Be(0);
            }
        }, secure: true);
        foreach (var p in Players.Where(p => !p.Left))
            SendEndBattle(p);

        // Pay out (wager, rating) in the API, off the battle's lock.
        var settle = Players.Select(p => (p.UserId, Token: Online.Find(p.UserId)?.GetToken() is { Length: > 0 } t ? t : p.Token, Result: Results[p.UserId])).ToList();
        var battleId = Id;
        Task.Run(() =>
        {
            foreach (var s in settle)
                PvpApi.TryEndBattle(s.Token, s.UserId, battleId, s.Result);
        });
    }

    private byte[]? _endBattle;

    private void SendEndBattle(PvpPlayer p)
    {
        if (_endBattle != null)
            Send(p, _endBattle);
    }

    /// <summary>
    /// Cancel a battle that never started (e.g. the second player couldn't afford the wager after all).
    /// </summary>
    public void Cancel()
    {
        lock (_lock)
        {
            if (!Over)
                Finish(Players.ToDictionary(x => x.UserId, _ => 0));
        }
    }

    private static byte[] SystemMessage(int type, uint profileId) => Wire.Message(PvpCmd.SystemMessage, bw =>
    {
        bw.WriteInt16Be(type);
        bw.WriteUInt32Be(profileId);
    });
}

public static class PvpSystemType
{
    public const int DroppedByReconnectionTimeout = 1;
    public const int DroppedByCommandTimeout = 2;
    public const int DroppedByResponseTimeout = 3;
    public const int Surrendered = 4;
    public const int Cheater = 5;
    public const int Disconnected = 6;
    public const int Reconnected = 7;
}

public static class PvpDropType
{
    public const int Surrender = 1;
    public const int IAmCheater = 4;
    public const int Desync = 5;
}
