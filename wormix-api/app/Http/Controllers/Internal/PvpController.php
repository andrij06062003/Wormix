<?php

namespace App\Http\Controllers\Internal;

use App\Http\Controllers\Controller;
use App\Http\Requests\Internal\Pvp\PvpEndBattleRequest;
use App\Http\Requests\Internal\Pvp\PvpStartBattleRequest;
use App\Http\Resources\Internal\Account\UserProfileStructure;
use App\Models\Wormix\Reagent;
use App\Models\Wormix\UserBattleInfo;
use App\Models\Wormix\UserProfile;
use App\Models\Wormix\UserTeam;
use App\Models\Wormix\WormData;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Log;

/**
 * PvP battles. The game server (wormix-core) finds the opponents and referees the battle; this records it per player:
 *  - start_battle: takes the wager and remembers the battle, so the player's own EndBattle (used weapons) is accepted;
 *  - end_battle:   pays out the result the game server decided (the client's own claim is not trusted).
 *
 * Wager battles pay like the 1.05.0 client shows them: the winner gets the wager back plus 80% of it, a draw returns
 * the wager, and the rating changes by the client's formula (BattleAwardSettings.getRating). Friend battles have no wager.
 */
class PvpController extends Controller
{
    public const BATTLE_TYPE_PVP = 2;

    public function profile(Request $request)
    {
        $profile = UserProfile::query()->findOrFail($request->json('internal_user_id'));
        return ['data' => new UserProfileStructure($profile)];
    }

    public function startBattle(PvpStartBattleRequest $request)
    {
        $userId = $request->json('internal_user_id');
        $wager = (int)$request->json('Wager');

        return DB::transaction(function () use ($request, $userId, $wager) {
            $profile = UserProfile::query()->lockForUpdate()->findOrFail($userId);
            $battleInfo = UserBattleInfo::query()->lockForUpdate()->findOrFail($userId);

            if ($profile->money < $wager)
                return response(['message' => 'Not enough money'], 422);

            $reagents = $request->json('Reagents');
            if ($reagents === null) {
                // Up to 3 reagents in the battle's crates, the same for both players.
                $reagents = $wager > 0
                    ? Reagent::query()->pluck('reagent_id')->shuffle()->take(rand(0, 3))->values()->toArray()
                    : [];
            }

            $profile->money -= $wager;
            $profile->save();

            $battleInfo->current_battle_id = $request->json('BattleId');
            $battleInfo->battle_type = self::BATTLE_TYPE_PVP;
            $battleInfo->mission_id = 0;
            $battleInfo->awards = [
                'reagents' => $reagents,
                'awards' => [],
                'wager' => $wager,
                'settled' => false,       // result paid out (end_battle)
                'items_done' => false,    // player's EndBattle (used weapons) processed
            ];
            $battleInfo->save();

            return [
                'data' => [
                    'UserProfileStructure' => new UserProfileStructure($profile->refresh()),
                    'ReagentsForBattle' => $reagents,
                ]
            ];
        });
    }

    public function endBattle(PvpEndBattleRequest $request)
    {
        $userId = $request->json('internal_user_id');
        $result = (int)$request->json('Result');

        return DB::transaction(function () use ($request, $userId, $result) {
            $battleInfo = UserBattleInfo::query()->lockForUpdate()->findOrFail($userId);
            $awards = $battleInfo->awards ?? [];

            if ((int)$battleInfo->battle_type !== self::BATTLE_TYPE_PVP
                || (int)$battleInfo->current_battle_id !== (int)$request->json('BattleId')
                || ($awards['settled'] ?? true)) {
                Log::debug('PvP result for a battle that is not open', [
                    'user' => $userId, 'battle' => $request->json('BattleId'), 'current' => $battleInfo->current_battle_id,
                ]);
                return ['Status' => 'IGNORED'];
            }

            $wager = (int)($awards['wager'] ?? 0);
            if ($wager > 0) {
                $profile = UserProfile::query()->lockForUpdate()->findOrFail($userId);
                if ($result === 1)
                    $profile->money += $wager + (int)floor($wager * 0.8);
                elseif ($result === 0)
                    $profile->money += $wager;
                $profile->rating = max(0, $profile->rating + $this->ratingDelta($userId, $profile->rating, $result, $wager));
                $profile->save();
            }

            $awards['settled'] = true;
            $battleInfo->awards = $awards;
            $battleInfo->save();

            return ['Status' => 'OK'];
        });
    }

    /**
     * Rating change of a wager battle, as the client computes it (BattleAwardSettings.getRating).
     */
    private function ratingDelta(int $userId, int $rating, int $result, int $wager): int
    {
        if ($result === 0)
            return 0;

        $level = (int)(WormData::query()->where('owner_id', $userId)->value('level') ?? 1);
        // The team (the player's own worm included) is shown at the player's level.
        $teamLevel = $level * max(1, UserTeam::query()->where('user_id', $userId)->count());
        $basket = max(1, intdiv($level, 3));

        if ($result < 0) {
            $delta = -min($rating, (int)ceil($basket * $basket / 4) + (int)floor(($teamLevel - $level) / 6));
            if ($rating > 20000)
                $delta -= intdiv($rating - 20000, 200);
            return $delta;
        }

        $delta = 1 + (int)ceil($basket * $basket / 3) + (int)floor(($teamLevel - $level) / 3);
        if ($wager === 50)
            $delta = (int)round(1.1 * $delta);
        elseif ($wager === 300)
            $delta = (int)round(1.25 * $delta);
        return $delta;
    }
}
