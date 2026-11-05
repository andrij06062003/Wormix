using wormix_core.Extensions;
using wormix_core.Pragmatix.Wormix.Messages.Structures;
using wormix_core.Session;

namespace wormix_core.Pvp;

/// <summary>
/// Main-server side of PvP: finding an opponent before a battle starts on the PvP server.
///
/// Two ways, both from the 1.05.0 client:
///  - Challenge a player (friends list, or "call by id"): BattleRequest -> the other player gets BattleOffer ->
///    answers BattleOfferResult -> the challenger gets BattleRequestResult. The challenger can CancelBattle.
///    On VK this went to VK friends; here it reaches any player who is logged in, by their game id.
///  - Wager battle (Arena > PvP): WagerBattleRequest puts the player in a queue per wager amount; the next player
///    asking for the same wager is matched with them and both get WagerBattleRequestResult. WagerCancelBattle leaves the queue.
/// </summary>
public static class Lobby
{
    private static class Cmd
    {
        public const uint BattleRequest = 1002;        // client -> server
        public const uint BattleOfferResult = 1003;    // client -> server
        public const uint CancelBattle = 1004;         // both ways
        public const uint BattleOffer = 2002;          // server -> client
        public const uint BattleRequestResult = 2003;  // server -> client
        public const uint WagerRequest = 2005;         // client -> server
        public const uint WagerResult = 2006;          // server -> client
        public const uint WagerCancel = 2007;          // client -> server
        public const uint WagerError = 2008;           // server -> client
    }

    private static class RequestResult
    {
        public const int Agree = 0;
        public const int Disagree = 1;
        public const int Busy = 2;
        public const int EnemyIdNotFound = 3;
        public const int NotOnline = 4;
    }

    private static class WagerErrorCode
    {
        public const int Error = 1;
        public const int NotEnoughMoney = 3;
    }

    /// <summary>
    /// Wagers offered in the client's Arena > PvP tab (PvpWager.WAGER_1..3).
    /// </summary>
    private static readonly int[] Wagers = { 15, 50, 300 };

    private static readonly TimeSpan ChallengeTimeout = TimeSpan.FromSeconds(60);

    private class Challenge
    {
        public uint BattleId;
        public required MainServerSession From, To;
        public string Host = "";
        public int Port;
        public uint MapId;
        public int Seed;
        public UserProfileStructure FromProfile, ToProfile;
        public DateTime Created = DateTime.UtcNow;
    }

    private class WagerTicket
    {
        public required MainServerSession Session;
        public int Wager;
        public string Host = "";
        public int Port;
        public uint MapId;
        public bool Cancelled;
    }

    private static readonly object Lock = new();
    private static readonly Dictionary<uint, Challenge> Challenges = new();
    private static readonly List<WagerTicket> Queue = new();
    private static readonly List<WagerTicket> Matching = new();   // matched, battle being set up
    private static readonly Random Rng = new();

    private static void Log(string text) => ColorPrint.WriteLine($"[Lobby] {text}", ConsoleColor.Cyan);

    /// <summary>
    /// Handle a PvP command on the main server. Returns false for other commands.
    /// </summary>
    public static bool Handle(MainServerSession s, uint cmd, byte[] body)
    {
        if (cmd is not (Cmd.BattleRequest or Cmd.BattleOfferResult or Cmd.CancelBattle or Cmd.WagerRequest or Cmd.WagerCancel))
            return false;
        if (s.UserId == 0)
            return true;   // not logged in
        try
        {
            switch (cmd)
            {
                case Cmd.BattleRequest: OnBattleRequest(s, new WireReader(body)); break;
                case Cmd.BattleOfferResult: OnBattleOfferResult(s, new WireReader(body)); break;
                case Cmd.CancelBattle: OnCancelBattle(s); break;
                case Cmd.WagerRequest: OnWagerRequest(s, body); break;
                case Cmd.WagerCancel: OnWagerCancel(s); break;
            }
        }
        catch (EndOfStreamException)
        {
            Log($"truncated command {cmd} from {s.UserId}");
        }
        return true;
    }

    private static bool IsBusy(MainServerSession s) =>
        Challenges.Values.Any(c => c.From == s || c.To == s) ||
        Queue.Any(t => t.Session == s) ||
        Matching.Any(t => t.Session == s) ||
        Battles.IsInBattle(s.UserId);

    // ---- Challenges ------------------------------------------------------------------------------------

    private static void OnBattleRequest(MainServerSession from, WireReader r)
    {
        var host = r.Utf();
        var port = r.S32();
        var mapId = r.U32();
        var seed = r.S32();
        var enemyId = r.Utf();

        if (!uint.TryParse(enemyId.Trim(), out var toId) || toId == 0 || toId == from.UserId)
        {
            SendRequestResult(from, RequestResult.EnemyIdNotFound);
            return;
        }
        var to = Online.Find(toId);
        if (to == null)
        {
            SendRequestResult(from, RequestResult.NotOnline);
            return;
        }

        Challenge challenge;
        lock (Lock)
        {
            DropEverything(from);   // a stale request or queue place of the challenger's own
            if (IsBusy(to))
            {
                SendRequestResult(from, RequestResult.Busy);
                return;
            }
            challenge = new Challenge { BattleId = Battles.NextId(), From = from, To = to, Host = host, Port = port, MapId = mapId, Seed = seed };
            Challenges[challenge.BattleId] = challenge;
        }

        try
        {
            challenge.FromProfile = PvpApi.GetProfile(from.GetToken());
            challenge.ToProfile = PvpApi.GetProfile(to.GetToken());
        }
        catch (Exception ex)
        {
            Log($"can't load profiles for {from.UserId} -> {to.UserId}: {ex.Message}");
            lock (Lock)
                Challenges.Remove(challenge.BattleId);
            SendRequestResult(from, RequestResult.EnemyIdNotFound);
            return;
        }

        lock (Lock)
        {
            if (!Challenges.ContainsKey(challenge.BattleId))
                return;   // cancelled meanwhile
        }
        Log($"{from.UserId} challenges {to.UserId} (battle {challenge.BattleId})");
        to.SendMessage(Wire.Message(Cmd.BattleOffer, bw =>
        {
            bw.WriteUTF8(challenge.Host);
            bw.WriteInt32Be(challenge.Port);
            bw.WriteUInt32Be(challenge.MapId);
            bw.WriteInt32Be(challenge.Seed);
            bw.WriteProfile(challenge.FromProfile);   // enemy (for the one receiving the offer)
            bw.WriteProfile(challenge.ToProfile);     // their own
            bw.WriteUInt32Be(challenge.BattleId);
        }));
    }

    private static void OnBattleOfferResult(MainServerSession to, WireReader r)
    {
        var result = r.S16();
        r.U32();   // enemy id
        var battleId = r.U32();

        Challenge? c;
        lock (Lock)
        {
            if (!Challenges.TryGetValue(battleId, out c) || c.To != to)
                return;
            Challenges.Remove(battleId);
        }

        if (result != RequestResult.Agree)
        {
            Log($"{to.UserId} declined {c.From.UserId} (result {result})");
            SendRequestResult(c.From, result == RequestResult.Busy ? RequestResult.Busy : RequestResult.Disagree);
            return;
        }

        // The player who accepted is already loading the battle. The challenger moves first.
        Log($"{to.UserId} accepted {c.From.UserId}: battle {c.BattleId}");
        Battles.Create(c.BattleId, 0, new PvpPlayer(c.From.UserId, true, c.From.GetToken()), new PvpPlayer(to.UserId, false, to.GetToken()));
        // Register the battle with the API so the players' EndBattle (used weapons) is accepted. No wager in friend battles.
        foreach (var s in new[] { c.From, to })
        {
            try { PvpApi.StartBattle(s.GetToken(), c.BattleId, 0, new List<int>()); }
            catch (Exception ex) { Log($"can't register battle {c.BattleId} for {s.UserId}: {ex.Message}"); }
        }
        SendRequestResult(c.From, RequestResult.Agree, c.ToProfile, c.FromProfile, c.BattleId);
    }

    private static void OnCancelBattle(MainServerSession from)
    {
        List<Challenge> cancelled;
        lock (Lock)
        {
            cancelled = Challenges.Values.Where(c => c.From == from).ToList();
            foreach (var c in cancelled)
                Challenges.Remove(c.BattleId);
        }
        foreach (var c in cancelled)
        {
            Log($"{from.UserId} cancelled the challenge to {c.To.UserId}");
            SendCancel(c);
        }
    }

    private static void SendCancel(Challenge c) => c.To.SendMessage(Wire.Message(Cmd.CancelBattle, bw =>
    {
        bw.WriteUInt32Be(c.From.UserId);
        bw.WriteUInt32Be(c.To.UserId);
    }));

    private static void SendRequestResult(MainServerSession s, int result, UserProfileStructure enemy = default, UserProfileStructure me = default, uint battleId = 0)
    {
        s.SendMessage(Wire.Message(Cmd.BattleRequestResult, bw =>
        {
            bw.WriteInt16Be(result);
            if (result != RequestResult.Agree)
                return;
            bw.WriteProfile(enemy);
            bw.WriteProfile(me);
            bw.WriteUInt32Be(battleId);
        }));
    }

    // ---- Wager battles ---------------------------------------------------------------------------------

    private static void OnWagerRequest(MainServerSession s, byte[] body)
    {
        var r = new WireReader(body);
        var host = r.Utf();
        var port = r.S32();
        var wager = r.S32();
        var mapId = r.U32();

        if (!Wagers.Contains(wager))
        {
            SendWagerError(s, WagerErrorCode.Error);
            return;
        }
        try
        {
            if (PvpApi.GetProfile(s.GetToken()).Money < wager)
            {
                SendWagerError(s, WagerErrorCode.NotEnoughMoney);
                return;
            }
        }
        catch (Exception ex)
        {
            Log($"can't load profile of {s.UserId}: {ex.Message}");
            SendWagerError(s, WagerErrorCode.Error);
            return;
        }

        var me = new WagerTicket { Session = s, Wager = wager, Host = host, Port = port, MapId = mapId };
        WagerTicket? waiting;
        lock (Lock)
        {
            DropEverything(s);
            if (Battles.IsInBattle(s.UserId))
            {
                SendWagerError(s, WagerErrorCode.Error);
                return;
            }
            waiting = Queue.FirstOrDefault(t => t.Wager == wager && t.Session.UserId != s.UserId && Online.Find(t.Session.UserId) == t.Session);
            if (waiting == null)
            {
                Queue.Add(me);
                Log($"{s.UserId} waits for a {wager} wager battle ({Queue.Count(t => t.Wager == wager)} waiting)");
                return;
            }
            Queue.Remove(waiting);
            Matching.Add(waiting);
            Matching.Add(me);
        }

        try
        {
            StartWagerBattle(waiting, me);
        }
        finally
        {
            lock (Lock)
            {
                Matching.Remove(waiting);
                Matching.Remove(me);
            }
        }
    }

    private static void StartWagerBattle(WagerTicket first, WagerTicket second)
    {
        var battleId = Battles.NextId();
        var wager = first.Wager;

        // Take the wager from both. If the second can't pay any more, give the first theirs back.
        (UserProfileStructure Profile, List<int> Reagents) a, b;
        try
        {
            a = PvpApi.StartBattle(first.Session.GetToken(), battleId, wager, null);
        }
        catch (Exception ex)
        {
            Log($"can't start battle {battleId} for {first.Session.UserId}: {ex.Message}");
            SendWagerError(first.Session, IsNotEnoughMoney(ex) ? WagerErrorCode.NotEnoughMoney : WagerErrorCode.Error);
            Requeue(second);
            return;
        }
        try
        {
            b = PvpApi.StartBattle(second.Session.GetToken(), battleId, wager, a.Reagents);
        }
        catch (Exception ex)
        {
            Log($"can't start battle {battleId} for {second.Session.UserId}: {ex.Message}");
            PvpApi.TryEndBattle(first.Session.GetToken(), first.Session.UserId, battleId, 0);
            SendWagerError(second.Session, IsNotEnoughMoney(ex) ? WagerErrorCode.NotEnoughMoney : WagerErrorCode.Error);
            Requeue(first);
            return;
        }

        // Someone pressed Cancel while we were talking to the API: give both their wager back.
        lock (Lock)
        {
            if (first.Cancelled || second.Cancelled)
            {
                Log($"battle {battleId} cancelled before it started");
                PvpApi.TryEndBattle(first.Session.GetToken(), first.Session.UserId, battleId, 0);
                PvpApi.TryEndBattle(second.Session.GetToken(), second.Session.UserId, battleId, 0);
                if (!first.Cancelled) Queue.Add(first);
                if (!second.Cancelled) Queue.Add(second);
                return;
            }
        }

        var firstMoves = Rng.Next(2) == 0 ? first : second;
        var seed = Rng.Next(1000);
        Battles.Create(battleId, wager,
            new PvpPlayer(first.Session.UserId, firstMoves == first, first.Session.GetToken()),
            new PvpPlayer(second.Session.UserId, firstMoves == second, second.Session.GetToken()));
        Log($"wager {wager}: {first.Session.UserId} vs {second.Session.UserId}, battle {battleId}");

        // The map is the one the player who waited longer picked.
        SendWagerResult(first, a.Profile, b.Profile, firstMoves.Session.UserId, seed, battleId, a.Reagents, first);
        SendWagerResult(second, b.Profile, a.Profile, firstMoves.Session.UserId, seed, battleId, a.Reagents, first);
    }

    private static bool IsNotEnoughMoney(Exception ex) => ex is PvpApi.ApiException { Status: System.Net.HttpStatusCode.UnprocessableEntity };

    private static void Requeue(WagerTicket t)
    {
        lock (Lock)
        {
            if (!t.Cancelled && Online.Find(t.Session.UserId) == t.Session)
                Queue.Add(t);
        }
    }

    private static void SendWagerResult(WagerTicket to, UserProfileStructure me, UserProfileStructure enemy, uint firstPlayerId, int seed, uint battleId, List<int> reagents, WagerTicket mapOwner)
    {
        to.Session.SendMessage(Wire.Message(Cmd.WagerResult, bw =>
        {
            bw.WriteUTF8(mapOwner.Host);
            bw.WriteInt32Be(mapOwner.Port);
            bw.WriteUInt32Be(mapOwner.MapId);
            bw.WriteProfile(enemy);
            bw.WriteProfile(me);
            bw.WriteUInt32Be(firstPlayerId);
            bw.WriteInt32Be(seed);
            bw.WriteUInt32Be(battleId);
            bw.WriteUInt16Be((ushort)reagents.Count);
            foreach (var reagent in reagents)
                bw.Write((sbyte)reagent);
        }, secure: true));
    }

    private static void SendWagerError(MainServerSession s, int code) =>
        s.SendMessage(Wire.Message(Cmd.WagerError, bw => bw.WriteInt16Be(code)));

    private static void OnWagerCancel(MainServerSession s)
    {
        lock (Lock)
        {
            foreach (var t in Queue.Concat(Matching).Where(t => t.Session == s))
                t.Cancelled = true;
            if (Queue.RemoveAll(t => t.Session == s) > 0)
                Log($"{s.UserId} left the wager queue");
        }
    }

    // ---- Cleanup -------------------------------------------------------------------------------------------

    /// <summary>
    /// Forget everything the player was waiting for (call with <see cref="Lock"/> held).
    /// </summary>
    private static void DropEverything(MainServerSession s)
    {
        foreach (var t in Queue.Concat(Matching).Where(t => t.Session == s))
            t.Cancelled = true;
        Queue.RemoveAll(t => t.Session == s);
        foreach (var c in Challenges.Values.Where(c => c.From == s).ToList())
        {
            Challenges.Remove(c.BattleId);
            SendCancel(c);
        }
    }

    /// <summary>
    /// The player's main connection closed.
    /// </summary>
    public static void OnDisconnected(MainServerSession s)
    {
        if (s.UserId == 0)
            return;
        lock (Lock)
        {
            DropEverything(s);
            foreach (var c in Challenges.Values.Where(c => c.To == s).ToList())
            {
                Challenges.Remove(c.BattleId);
                SendRequestResult(c.From, RequestResult.NotOnline);
            }
        }
    }

    /// <summary>
    /// Expire unanswered challenges. Called about once a second.
    /// </summary>
    public static void Tick()
    {
        lock (Lock)
        {
            foreach (var c in Challenges.Values.Where(c => DateTime.UtcNow - c.Created > ChallengeTimeout).ToList())
            {
                Challenges.Remove(c.BattleId);
                Log($"challenge {c.From.UserId} -> {c.To.UserId} not answered");
                SendCancel(c);
                SendRequestResult(c.From, RequestResult.Disagree);
            }
        }
    }
}
