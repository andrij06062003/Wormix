using System.Collections.Concurrent;
using wormix_core.Session;

namespace wormix_core.Pvp;

/// <summary>
/// Players currently logged in to the main server, by user id. Replaces the VK friends/online lookup:
/// a battle request reaches whoever is logged in with that id.
/// </summary>
public static class Online
{
    private static readonly ConcurrentDictionary<uint, MainServerSession> Players = new();

    public static void Register(MainServerSession session) => Players[session.UserId] = session;

    /// <summary>
    /// Remove the session, unless the player has logged in again from a newer session meanwhile.
    /// </summary>
    public static void Unregister(MainServerSession session)
    {
        if (session.UserId != 0)
            Players.TryRemove(new KeyValuePair<uint, MainServerSession>(session.UserId, session));
    }

    public static MainServerSession? Find(uint userId) => Players.TryGetValue(userId, out var s) ? s : null;

    public static int Count => Players.Count;
}
