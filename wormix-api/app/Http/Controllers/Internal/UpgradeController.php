<?php

namespace App\Http\Controllers\Internal;

use App\Http\Controllers\Controller;
use App\Http\Requests\Internal\Craft\DowngradeWeaponRequest;
use App\Http\Requests\Internal\Craft\UpgradeWeaponRequest;
use App\Http\Resources\Internal\Craft\DowngradeWeaponResult;
use App\Http\Resources\Internal\Craft\UpgradeWeaponResult;
use App\Models\Wormix\Craft;
use App\Models\Wormix\Reagent;
use App\Models\Wormix\UserProfile;
use App\Models\Wormix\UserWeapon;
use App\Models\Wormix\WormData;
use Illuminate\Database\Eloquent\Collection;
use Illuminate\Support\Facades\Log;

class UpgradeController extends Controller
{
    private const UPGRADE_BASE = 300;

    public function upgradeWeapon(UpgradeWeaponRequest $request)
    {
        $craft = Craft::query()
            ->where('id', $request->json('RecipeId'))
            ->get()
            ->first();

        $user_profile = UserProfile::query()
            ->where('user_id', $request->json('internal_user_id'))
            ->get()
            ->first();

        $wormData = WormData::query()
            ->where('owner_id', $request->json('internal_user_id'))
            ->get()
            ->first();

        if(
            ($craft->prev_upgrade_id < self::UPGRADE_BASE &&
            UserWeapon::query()
                ->where('weapon_id', $craft->prev_upgrade_id)
                ->where('count', -1)
            ->count() == 0) ||
            (
                $craft->prev_upgrade_id > self::UPGRADE_BASE &&
                !in_array(@$craft->prev_upgrade->id, $user_profile->recipes)
            ) ||
            in_array($craft->id, $user_profile->recipes)
            ||
            $wormData->level < $craft->required_level
            ||
            in_array(
                @Craft::query()->where('upgrade_id', $craft->prev_upgrade->id)
                ->where('id', '!=', $request->json('RecipeId'))
                ->get()
                ->first()->id
                , $user_profile->recipes)
        ){
            return [
                'data' => new UpgradeWeaponResult(Collection::empty(), UpgradeWeaponResult::Error, $request->json('RecipeId'))
            ];
        }


        $reagents2craft = Reagent::query()
            ->select('reagent_id', 'reagent_price')
            ->whereIn('reagent_id', array_map(function ($x) {return $x[0];}, $craft->reagents))
            ->get()
            ->pluck('reagent_price', 'reagent_id')
            ->toArray();

        $maxReagentId = max(array_map(function ($x) {return $x[0];}, $craft->reagents));


        $changedReagents =  $user_profile->reagents;
        if(count($changedReagents) < $maxReagentId + 1){
            $oldReagents = $changedReagents;
            $changedReagents = array_fill(0, $maxReagentId + 1, 0);
            foreach($oldReagents as $k => $v) {
                $changedReagents[$k] = $v;
            }
        }

//        Log::debug("PREPARED REAGENT DATA",
//            [
//                'reagents' => $reagents2craft,
//                'craft_reagents' => $craft->reagents,
//                'max_id' => $maxReagentId,
//                'user_reagents' => $changedReagents
//            ]
//        );

        $sum = 0;

        foreach($craft->reagents as $reagent){
            $sum += max(0, ($reagent[1] - $changedReagents[$reagent[0]]) * $reagents2craft[(string)$reagent[0]]);
            $changedReagents[$reagent[0]] = max(0,  $changedReagents[$reagent[0]] - $reagent[1]);
        }

        $sum = (int)(round($sum / 100));

//        Log::debug("NeedSum", [
//            'sum' => $sum,
//        ]);

        if($user_profile->real_money < $sum)
            return [
                'data' => new UpgradeWeaponResult(Collection::empty(), UpgradeWeaponResult::NotEnoughMoney, $request->json('RecipeId'))
            ];

        $user_profile->real_money -= $sum;
        $user_profile->recipes = array_merge($user_profile->recipes, [$request->json('RecipeId')]);
        $user_profile->reagents = $changedReagents;
        $user_profile->save();

        return [
            'data' => new UpgradeWeaponResult(Collection::empty(), UpgradeWeaponResult::Success, $request->json('RecipeId'))
        ];
    }

    public function downgradeWeapon(DowngradeWeaponRequest $request)
    {
        $craft = Craft::query()
            ->where('id', $request->json('RecipeId'))
            ->get()
            ->first();

        $user_profile = UserProfile::query()
            ->where('user_id', $request->json('internal_user_id'))
            ->get()
            ->first();

        $recipes = $user_profile->recipes;
        $reagents = $user_profile->reagents;

        //Add cross craft checks
        if(
            !in_array($craft->id, $recipes) ||
            $user_profile->real_money < config('wormix.game.buy.downgrade')
        )
            return [
                'data' => new DowngradeWeaponResult(Collection::empty(), DowngradeWeaponResult::Error, $request->json('RecipeId'))
            ];

        foreach($craft->reagents as $reagent){
            $reagents[$reagent[0]] += (int)($reagent[1] * 0.8);
        }

        unset($recipes[array_search($request->json('RecipeId'), $recipes)]);
        $recipes = array_values($recipes);

        $user_profile->recipes = $recipes;
        $user_profile->reagents = $reagents;
        $user_profile->real_money -= config('wormix.game.buy.downgrade');
        $user_profile->save();

        return [
            'data' => new DowngradeWeaponResult(Collection::empty(), DowngradeWeaponResult::Success, $request->json('RecipeId'))
        ];
    }
}
