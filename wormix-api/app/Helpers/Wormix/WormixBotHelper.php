<?php

namespace App\Helpers\Wormix;

use App\Models\User;
use App\Models\Wormix\Level;
use App\Models\Wormix\Race;
use App\Models\Wormix\Weapon;
use App\Models\Wormix\WormData;
use Illuminate\Database\Eloquent\Collection;

class WormixBotHelper
{
    public const BOT_BASE = -1000;

    public static function GenerateBots(WormData $userWorm)
    {
        $bot_base_id = User::query()->select('id')->count();

        $user_profile = $userWorm->owner->user_profile;

        $random_stuff = Weapon::query()
            ->where('id', '>', WormixTrashHelper::STUFF_START_INDEX)
            ->where('required_level', '<=', $userWorm->level)
            ->where('required_rating', '<=', $user_profile->rating)
            ->where('hide_in_shop', 0)
            ->where('price', '!=', 0)
            ->get()
            ->pluck('id')
            ->toArray();


        $random_race = Race::query()
            ->where('required_level', '<=', $userWorm->level)
            ->get()->pluck('race_id')->toArray();


        $bots = Collection::empty();

        $bots_count = (
            $userWorm->level >= 5 ||
            $userWorm->owner->battle_info->mission_id < -2 ||
            $userWorm->owner->battle_info->mission_id >= 0
        ) ? 6 : 4;

        for($i = 0; $i < $bots_count; $i++){
            $worm_group = [];
            $random_level = rand(
                max(1, $userWorm->level - 1),
                min($userWorm->level + 1, 30)
            );

            $level = Level::query()->where('id', $random_level)->get()->first();
            $random_worms_count = rand(1, $level->max_worms_count);

            for($j = 0; $j < $random_worms_count; $j++){

                $random_armor = rand(0, $random_level * 2);
                $random_attack = $random_level * 2 - $random_armor;
                $hat = count($random_stuff) === 0 ? 0 : $random_stuff[array_rand($random_stuff)];
                if(rand(0, 10) % 5 === 0)
                    $hat = 0;

                $race = count($random_race) === 0 ? 0 : $random_race[array_rand($random_race)];

                $worm_group[] = [
                    'Armor' => $random_armor,
                    'Attack' => $random_attack,
                    'Experience' => 0,
                    'Level' => $random_level,
                    'Hat' => WormixTrashHelper::getHatByRaceAndHatIds($hat, $race),
                    'OwnerId' => $bot_base_id + ($i+$j+1) * 10,
                    'SocialOwnerId' => (string)(0 - $bot_base_id + self::BOT_BASE - ($i+1+$j)*10)
                ];
            }

            $profile = [
                'Id' => $bot_base_id + ($i+1) * 10,
                'Rating' => rand(0, $user_profile->rating),
                'Money' => rand(0, $user_profile->money),
                'RealMoney' => rand(0, $user_profile->real_money),
                'Recipes' => [],
                'SocialId' => (string)(0 - $bot_base_id + self::BOT_BASE - ($i+1)*10),
                'Stuff' => $hat == 0 ? [] : [$hat],
                'WeaponRecordList' => [
                    ['Id' => 1, 'Count' => -1]
                ],
                'WormsGroup' => $worm_group
            ];
            $bots->add($profile);
        }
        return $bots;
    }

    public static function stripData(
        int $level,
        int $toLevel,
        int $armor,
        int $attack
    )
    {
        $levelPoints = $level * 2;
        $toLevelPoints = $toLevel * 2;

        if($armor + $attack < $levelPoints)
            $armor += $levelPoints - ($armor + $attack);


        $armorPoints = (int)(($armor / $levelPoints) * $toLevelPoints);
        $attackPoints = $toLevelPoints - $armorPoints;

        return [
            'armor' => $armorPoints,
            'attack' => $attackPoints
        ];
    }
}
