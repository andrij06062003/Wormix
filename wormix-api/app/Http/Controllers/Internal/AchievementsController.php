<?php

namespace App\Http\Controllers\Internal;

use App\Helpers\Wormix\WormixTrashHelper;
use App\Http\Controllers\Controller;
use App\Models\User;
use App\Models\Wormix\UserAchievement;
use App\Models\Wormix\UserProfile;
use App\Models\Wormix\UserWeapon;
use App\Modules\UserSession;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\DB;

/**
 * Backend of the achievements server (the client's "wormix_achieve" HTTP connection).
 * The relay decodes the client's binary commands and calls these endpoints with JSON.
 *
 * The achievement session lives in the cache under achieve_session_{id}. An unknown
 * session answers 401, and the relay turns that into the "401" text that makes the
 * client log in again.
 */
class AchievementsController extends Controller
{
    private const SESSION_TTL = 6 * 60 * 60;

    // ShopResult codes the client understands
    private const RESULT_SUCCESS = 0;
    private const RESULT_ERROR = 1;
    private const RESULT_MIN_REQUIREMENTS = 2;

    private static ?array $config = null;

    public static function config(): array
    {
        if (self::$config === null)
            self::$config = json_decode(file_get_contents(resource_path('game/achievements.json')), true);
        return self::$config;
    }

    public static function grantDevAchievements(int $userId): void
    {
        $config = self::config();
        foreach ($config['achievements'] as $id => $ach) {
            $levels = $ach['levels'] ?? [];
            if (empty($levels))
                continue;
            $maxVal = end($levels)['max'];
            UserAchievement::updateOrCreate(
                ['user_id' => $userId, 'achievement_id' => (int)$id],
                ['value' => $maxVal]
            );
        }
    }

    public function login(Request $request)
    {
        $userId = (int)$request->json('id');
        $authKey = (string)$request->json('auth_key');

        try {
            $userSession = new UserSession($userId);
        } catch (\Exception) {
            return response(['ok' => false, 'code' => 1]);
        }
        $expected = $userSession->getAuthKey();
        if ($expected === '' || !hash_equals($expected, $authKey))
            return response(['ok' => false, 'code' => 1]);

        $sessionId = bin2hex(random_bytes(16));
        $loginTime = time();
        $this->putSession($sessionId, [
            'user_id' => $userId,
            'last_seq' => $loginTime,
            'last_result' => null,
        ]);

        return response(['ok' => true, 'session_id' => $sessionId, 'login_time' => $loginTime]);
    }

    public function get(Request $request)
    {
        if ($this->getSession($request) === null)
            return $this->unauthorized();

        $profileId = (string)$request->json('profile_id');
        $userId = (int)$profileId;
        $values = UserAchievement::query()
            ->where('user_id', $userId)
            ->where('value', '>', 0)
            ->orderBy('achievement_id')
            ->get(['achievement_id', 'value']);

        $profile = UserProfile::query()->find($userId);

        return response([
            'profile_id' => $profileId,
            'indexes' => $values->pluck('achievement_id')->all(),
            'values' => $values->pluck('value')->map(fn($v) => min($v, self::config()['max_value']))->all(),
            'invested' => $profile?->invested_award_points ?? 0,
        ]);
    }

    public function increase(Request $request)
    {
        $sessionId = (string)$request->json('session_id');
        $session = $this->getSession($request);
        if ($session === null)
            return $this->unauthorized();

        $timeScale = (int)$request->json('time_scale');

        // The client resends the same command after 30s without an answer:
        // answer it again without counting it twice.
        if ($timeScale === $session['last_seq'] && $session['last_result'] !== null)
            return response($session['last_result']);

        $userId = $session['user_id'];
        $profile = UserProfile::query()->find($userId);
        if ($profile === null)
            return $this->unauthorized();

        if ($timeScale <= $session['last_seq']) {
            // Old or out-of-order command: change nothing, just echo the sequence.
            return response($this->increaseResult([], [], [], [], $timeScale, $profile->invested_award_points));
        }

        $indexes = (array)$request->json('indexes', []);
        $rises = (array)$request->json('rises', []);
        $config = self::config();

        $rise = [];
        foreach ($indexes as $i => $index) {
            $index = (int)$index;
            $value = (int)($rises[$i] ?? 0);
            if (!self::isCounter($index) || $value <= 0)
                continue;
            $rise[$index] = ($rise[$index] ?? 0) + $value;
        }

        $result = DB::transaction(function () use ($rise, $userId, $config, $timeScale) {
            $profile = UserProfile::query()->lockForUpdate()->find($userId);
            $outIndexes = [];
            $outValues = [];
            $thresholdIndexes = [];
            $thresholdOld = [];

            foreach ($rise as $index => $value) {
                $row = UserAchievement::query()->firstOrNew([
                    'user_id' => $userId,
                    'achievement_id' => $index,
                ]);
                $old = (int)$row->value;
                $new = min($config['max_value'], $old + $value);
                if ($new === $old) {
                    $outIndexes[] = $index;
                    $outValues[] = $old;
                    continue;
                }
                $row->value = $new;
                $row->save();

                $crossed = false;
                foreach ($config['achievements'][(string)$index]['levels'] ?? [] as $level) {
                    if ($old < $level['max'] && $new >= $level['max']) {
                        $crossed = true;
                        $this->grantLevelAward($profile, $level['award'], (int)$level['count']);
                    }
                }
                if ($crossed) {
                    $thresholdIndexes[] = $index;
                    $thresholdOld[] = $old;
                }

                $outIndexes[] = $index;
                $outValues[] = $new;
            }
            $profile->save();

            return $this->increaseResult($outIndexes, $outValues, $thresholdIndexes, $thresholdOld,
                $timeScale, $profile->invested_award_points);
        });

        $session['last_seq'] = $timeScale;
        $session['last_result'] = $result;
        $this->putSession($sessionId, $session);

        return response($result);
    }

    public function chooseBonus(Request $request)
    {
        $session = $this->getSession($request);
        if ($session === null)
            return $this->unauthorized();

        $userId = $session['user_id'];
        $itemId = (int)$request->json('item_id');
        $config = self::config();

        $result = DB::transaction(function () use ($userId, $itemId, $config) {
            $profile = UserProfile::query()->lockForUpdate()->find($userId);
            if ($profile === null)
                return self::RESULT_ERROR;

            $level = null;
            foreach ($config['award_items'] as $l => $items) {
                if (in_array($itemId, $items, true)) {
                    $level = $l;
                    break;
                }
            }
            if ($level === null)
                return self::RESULT_ERROR;

            $invested = (int)$profile->invested_award_points;
            $available = intdiv($this->allPoints($userId), $config['points_per_award']) - $invested;
            // One award per 1000 points, and the next level opens after taking one from the previous.
            if ($available <= 0 || $level > $invested)
                return self::RESULT_MIN_REQUIREMENTS;

            $owned = UserWeapon::query()
                ->where('owner_id', $userId)
                ->where('weapon_id', $itemId)
                ->first();
            if ($owned !== null && ($itemId > WormixTrashHelper::STUFF_START_INDEX || $owned->count === -1))
                return self::RESULT_ERROR;

            if ($owned === null) {
                $owned = new UserWeapon();
                $owned->owner_id = $userId;
                $owned->weapon_id = $itemId;
            }
            $owned->count = -1;
            $owned->save();

            $profile->invested_award_points = $invested + 1;
            $profile->save();

            return self::RESULT_SUCCESS;
        });

        return response(['result' => $result, 'item_id' => $itemId]);
    }

    /**
     * Achievements (0..28, with levels and rewards) and the statistics page counters
     * (100..127 in the client's Stats class: bets, missions, kills...; no rewards) travel the same way.
     */
    private static function isCounter(int $index): bool
    {
        $config = self::config();
        return isset($config['achievements'][(string)$index])
            || ($index >= $config['stats_first'] && $index <= $config['stats_last']);
    }

    private function allPoints(int $userId): int
    {
        $levelsById = self::config()['achievements'];
        $points = 0;
        foreach (UserAchievement::query()->where('user_id', $userId)->get() as $row) {
            foreach ($levelsById[(string)$row->achievement_id]['levels'] ?? [] as $level) {
                if ($row->value >= $level['max'])
                    $points += $level['points'];
            }
        }
        return $points;
    }

    /**
     * Same rewards the client shows and adds locally in AchievLevelUpMessage.
     */
    private function grantLevelAward(UserProfile $profile, string $award, int $count): void
    {
        switch ($award) {
            case 'fuzy':
                $profile->money += $count;
                break;
            case 'ruby':
                $profile->real_money += $count;
                break;
            case 'reaction':
                $profile->reaction_rate += $count;
                break;
        }
    }

    private function increaseResult(array $indexes, array $values, array $thresholdIndexes, array $thresholdOld,
                                    int $timeSequence, int $invested): array
    {
        return [
            'indexes' => $indexes,
            'values' => $values,
            'threshold_indexes' => $thresholdIndexes,
            'threshold_old_values' => $thresholdOld,
            'time_sequence' => $timeSequence,
            'invested' => $invested,
        ];
    }

    private function getSession(Request $request): ?array
    {
        $sessionId = (string)$request->json('session_id');
        if (!preg_match('/^[0-9a-f]{32}$/', $sessionId))
            return null;
        $session = Cache::get('achieve_session_' . $sessionId);
        if (!is_array($session) || User::query()->where('id', $session['user_id'] ?? 0)->doesntExist())
            return null;
        return $session;
    }

    private function putSession(string $sessionId, array $session): void
    {
        Cache::put('achieve_session_' . $sessionId, $session, self::SESSION_TTL);
    }

    private function unauthorized()
    {
        return response(['message' => 'Unauthorized'], 401);
    }
}
