using System.Net;
using System.Text;
using wormix_core.Extensions;
using wormix_core.Pragmatix.Wormix.Messages.Structures;

namespace wormix_core.Pvp;

/// <summary>
/// Calls to wormix-api's /api/internal/pvp/ routes. Each call acts as one player: the API checks the
/// player's session key ("{userId}.{key}", the main-server session token) in X-SESSION-KEY.
/// </summary>
public static class PvpApi
{
    private static readonly HttpClient Http = new() { Timeout = TimeSpan.FromSeconds(15) };

    public class ApiException(HttpStatusCode status, string body) : Exception($"API answered {(int)status}: {body}")
    {
        public HttpStatusCode Status { get; } = status;
        public JObject? Json { get; } = TryParse(body);

        private static JObject? TryParse(string body)
        {
            try { return JObject.Parse(body); } catch { return null; }
        }
    }

    private static JObject Post(string route, string token, object body)
    {
        using var request = new HttpRequestMessage(HttpMethod.Post, Config.Url + route);
        // The internal routes require an X-TCP-SESSION header; PvP calls don't belong to one TCP session.
        request.Headers.Add("X-TCP-SESSION", "pvp");
        request.Headers.Add("X-SESSION-KEY", token);
        request.Content = new StringContent(JsonConvert.SerializeObject(body), Encoding.UTF8, "application/json");

        using var response = Http.Send(request);
        var text = response.Content.ReadAsStringAsync().Result;
        if (response.StatusCode != HttpStatusCode.OK)
            throw new ApiException(response.StatusCode, text);
        return JObject.Parse(text);
    }

    /// <summary>
    /// The player's own profile, as shown to them and to their opponent.
    /// </summary>
    public static UserProfileStructure GetProfile(string token)
    {
        var result = Post("pvp/profile", token, new { });
        return result["data"]!.ToObject<UserProfileStructure>();
    }

    /// <summary>
    /// Records the battle for the player and takes the wager. Pass the reagents the first player got to the second
    /// player (null makes the API choose them), so both see the same crates.
    /// Throws <see cref="ApiException"/> with status 422 if the player can't afford the wager.
    /// </summary>
    public static (UserProfileStructure Profile, List<int> Reagents) StartBattle(string token, uint battleId, int wager, List<int>? reagents)
    {
        var result = Post("pvp/start_battle", token, new { BattleId = battleId, Wager = wager, Reagents = reagents });
        var data = result["data"]!;
        return (data["UserProfileStructure"]!.ToObject<UserProfileStructure>(),
            data["ReagentsForBattle"]?.ToObject<List<int>>() ?? new List<int>());
    }

    /// <summary>
    /// Pays out the result the PvP server decided: 1 win, 0 draw (or cancelled: wager returned), -1 loss.
    /// </summary>
    public static void EndBattle(string token, uint battleId, int result)
    {
        Post("pvp/end_battle", token, new { BattleId = battleId, Result = result });
    }

    /// <summary>
    /// Like <see cref="EndBattle"/> but never throws: used where nobody is waiting for the answer.
    /// </summary>
    public static void TryEndBattle(string token, uint userId, uint battleId, int result)
    {
        try
        {
            EndBattle(token, battleId, result);
        }
        catch (Exception ex)
        {
            ColorPrint.WriteLine($"[PvP] Can't record result {result} of battle {battleId} for {userId}: {ex.Message}", ConsoleColor.Red);
        }
    }
}
