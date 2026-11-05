<?php

namespace App\Helpers\Wormix;

use App\Models\Wormix\HouseAction;
use App\Models\Wormix\UserProfile;
use App\Models\Wormix\UserWeapon;
use App\Models\Wormix\Weapon;
use App\Models\Wormix\WormData;
use Illuminate\Support\Facades\Log;

class WormixTrashHelper
{
    private const RACE_BASE = 500;
    private const RACE_LIMIT = 50;

    public const STUFF_START_INDEX = 1000;

    /**
     * @param int $hat worm_structure.hat
     * @return int[] 0-race id, 1-hat id
     */
    public static function getRaceAndHatIds(int $hat) : array
    {
        $hatId = 0;
        if($hat < self::RACE_LIMIT){
            $raceId = $hat;
        }
        else{
            $hatId = self::STUFF_START_INDEX + $hat % self::RACE_BASE;
            $raceId = (int)(($hat - self::STUFF_START_INDEX) / self::RACE_BASE);
        }
        return  [
            $raceId,
            $hatId
        ];
    }

    /**
     * @param int $hatId
     * @param int $raceId
     * @return int worm_structure.hat
     */
    public static function getHatByRaceAndHatIds(int $hatId, int $raceId) : int
    {
        return $hatId === 0 ? $raceId : ($raceId * self::RACE_BASE + $hatId);
    }

    /**
     * @param int $user_id
     * @return int
     */
    public static function getSearchKeys(int $user_id) : int
    {
        return max(
            0,
            (config('wormix.game.search_keys_per_day') -
                HouseAction::query()
                    ->where('action_type', 1)
                    ->where('user_id', $user_id)
                    ->where('created_at', '>=', now()->subDay())
                    ->count()
            )
        );
    }

    /**
     * @param int $user_id
     * @param int $to_user_id
     * @return bool
     */
    public static function isSearchedToday(int $user_id, int $to_user_id):bool
    {
        return HouseAction::query()
                ->where('user_id', $user_id)
                ->where('to_user_id', $to_user_id)
                ->where('action_type', 1)
                ->where('created_at', '>=', now()->subDay())->count() > 0;
    }

    public static function addReagents(UserProfile $profile, array $reagents):void
    {
        if(count($reagents) === 0)
            return;
        //Log::debug("Add reagents...");
        $userReagents = $profile->reagents;
        $maxReagent = max($reagents);
        if($maxReagent + 1 > count($userReagents)){
           $old_reagents  = $userReagents;
            $userReagents = array_fill(0, $maxReagent + 1, 0);
           for($i = 0; $i < count($old_reagents); $i++){
               $userReagents[$i] = $old_reagents[$i];
           }
        }

        for($i = 0; $i < count($reagents); $i++){
            $userReagents[$reagents[$i]] += 1;
        }

        $profile->reagents = $userReagents;
        $profile->save();
    }

    public static function addWeaponsAwards(array $awards, WormData $wormData):void
    {
        if(count($awards) === 0)
            return;

        //Add awards weapons
        $awards_weapons_ids = array_map(function ($x) { return $x[0];}, $awards);
        $weapons = Weapon::query()
            ->whereIn('id', $awards_weapons_ids)
            ->whereNotIn('id',
                UserWeapon::query()
                    ->where('owner_id', $wormData->owner_id)
                    ->whereIn('weapon_id', $awards_weapons_ids)
                    ->where('count', '!=', '-1')
                    ->select('weapon_id')
                    ->pluck('weapon_id')
                    ->toArray()
            )->get();

        //Bad coding
        $weapons_ids = $weapons->pluck('id')->toArray(); //Owned weapons
        $new_weapons = array_values(
            array_filter($awards, function($x) use ($weapons_ids)  {
                return in_array($x[0], $weapons_ids);
            }
            )
        );
        foreach($new_weapons as $weapon){
            $old_weapon = UserWeapon::query()->where('weapon_id', $weapon[0])->get()->first();
            $user_weapon = $old_weapon === null ? new UserWeapon() : $old_weapon;
            $user_weapon->owner_id = $wormData->owner_id;
            $user_weapon->weapon_id = $weapon[0];
            $user_weapon->count = $weapon[1];
            if($weapon[0] >= self::STUFF_START_INDEX)
                $user_weapon->expire_at = time() + 24 * 60 * 60;
            $user_weapon->save();
        }
    }
}
