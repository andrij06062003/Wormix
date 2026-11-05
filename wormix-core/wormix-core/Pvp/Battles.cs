using System.Collections.Concurrent;
using wormix_core.Extensions;
using wormix_core.Session;

namespace wormix_core.Pvp;

/// <summary>
/// Battles in progress (and recently finished ones, so a late login still gets the result).
/// </summary>
public static class Battles
{
    private static readonly ConcurrentDictionary<uint, PvpBattle> All = new();
    private static readonly TimeSpan KeepFinished = TimeSpan.FromMinutes(2);

    // Battle ids must not repeat after a restart (the API keeps the last one per player). Mission battles use
    // small ids (1000 + user id + mission), so start PvP ids high, from the clock, below int.MaxValue.
    private static int _nextId = 1_000_000_000 + (int)(DateTimeOffset.UtcNow.ToUnixTimeSeconds() % 100_000_000);

    private static readonly Timer TickTimer = new(_ => Tick(), null, TimeSpan.FromSeconds(1), TimeSpan.FromSeconds(1));

    static Battles() { }   // start the tick timer on first use, not lazily at some later field access

    public static uint NextId() => (uint)Interlocked.Increment(ref _nextId);

    public static PvpBattle Create(uint id, int wager, PvpPlayer first, PvpPlayer second)
    {
        var battle = new PvpBattle(id, wager, first, second);
        All[id] = battle;
        ColorPrint.WriteLine($"[PvP {id}] created: {first.UserId} vs {second.UserId}, wager {wager}", ConsoleColor.Cyan);
        return battle;
    }

    public static PvpBattle? Find(uint id) => All.TryGetValue(id, out var b) ? b : null;

    /// <summary>
    /// Whether the player is in a battle that hasn't ended.
    /// </summary>
    public static bool IsInBattle(uint userId) => All.Values.Any(b => !b.Over && b.HasPlayer(userId));

    private static void Tick()
    {
        foreach (var battle in All.Values)
        {
            try
            {
                battle.Tick();
                if (battle.Over && DateTime.UtcNow - battle.OverAt > KeepFinished)
                    All.TryRemove(battle.Id, out _);
            }
            catch (Exception ex)
            {
                ColorPrint.WriteLine($"[PvP {battle.Id}] tick error: {ex.Message}", ConsoleColor.Red);
            }
        }
        Lobby.Tick();
    }

    /// <summary>
    /// BattleLogin (1001) on the PvP server.
    /// </summary>
    public static void Login(PvpSession session, byte[] body)
    {
        // id u32, firstTurn bool, authKey utf, enemyIds u32[], mainHost utf, mainPort i32, battleId u32, socialId u8
        var r = new WireReader(body);
        var userId = r.U32();
        r.Bool();
        var authKey = r.Utf();
        r.U32Array();
        r.Utf();
        r.S32();
        var battleId = r.U32();
        LoginOrReconnect(session, userId, authKey, battleId, b => b.Login(session, userId));
    }

    /// <summary>
    /// BattleReconnect (1015) on the PvP server.
    /// </summary>
    public static void Reconnect(PvpSession session, byte[] body)
    {
        // id u32, authKey utf, mainHost utf, mainPort i32, battleId u32, turnNum u16, playerNum u8, lastCommandNum u16, socialId u8
        var r = new WireReader(body);
        var userId = r.U32();
        var authKey = r.Utf();
        r.Utf();
        r.S32();
        var battleId = r.U32();
        var turn = r.U16();
        r.U8();
        var lastCommand = r.U16();
        LoginOrReconnect(session, userId, authKey, battleId, b => b.Reconnect(session, userId, turn, lastCommand));
    }

    private static void LoginOrReconnect(PvpSession session, uint userId, string authKey, uint battleId, Func<PvpBattle, bool> join)
    {
        // The player must be logged in to the main server with the same key (the PvP connection carries no other proof).
        var main = Online.Find(userId);
        if (main == null || string.IsNullOrEmpty(authKey) || main.AuthKey != authKey)
        {
            ColorPrint.WriteLine($"[PvP {battleId}] login of {userId} refused: not logged in to the main server with this key", ConsoleColor.Yellow);
            session.SendLoginError(BattleLoginErrorCode.IncorrectKey);
            return;
        }
        var battle = Find(battleId);
        if (battle == null || !join(battle))
        {
            ColorPrint.WriteLine($"[PvP {battleId}] login of {userId} refused: no such battle for this player", ConsoleColor.Yellow);
            session.SendLoginError(BattleLoginErrorCode.InternalServerError);
        }
    }
}

public static class BattleLoginErrorCode
{
    public const int IncorrectKey = 1;
    public const int AlreadyInGame = 2;
    public const int InternalServerError = 3;
}
