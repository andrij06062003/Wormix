<?php

namespace App\Http\Controllers\Internal;

use App\Http\Controllers\Controller;
use App\Http\Requests\Internal\Account\ResetAccountRequest;
use App\Http\Requests\Internal\Account\ResetParametersRequest;
use App\Http\Resources\Internal\Account\ResetParametersResult;
use App\Models\Wormix\HouseAction;
use App\Models\Wormix\LoginSequence;
use App\Models\Wormix\UserBattleInfo;
use App\Models\Wormix\UserProfile;
use App\Models\Wormix\UserWeapon;
use App\Models\Wormix\Weapon;
use App\Models\Wormix\WormData;
use Illuminate\Database\Eloquent\Collection;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Log;

class ResetController extends Controller
{
    public function resetAccount(ResetAccountRequest $request)
    {
        $user_profile =
            UserProfile::query()->where('user_id', $request->json('internal_user_id'))
                ->get()
                ->first();

        $wormData = WormData::query()
            ->where('owner_id', $request->json('internal_user_id'))
            ->get()
            ->first();

        $battleInfo = UserBattleInfo::query()
            ->where('user_id', $request->json('internal_user_id'))
            ->get()
            ->first();

        $loginSequence = LoginSequence::query()
            ->where('user_id', $request->json('internal_user_id'))
            ->get()
            ->first();


        try{
            DB::beginTransaction();

            //Wipe battle info
            $battleInfo->battles_count = config('wormix.starter.missions');
            $battleInfo->last_mission_id = -1;
            $battleInfo->awards = [];
            $battleInfo->battle_type = 0;
            $battleInfo->mission_id = 0;
            $battleInfo->current_battle_id = 0;
            $battleInfo->last_boss_fight_time = 0;
            $battleInfo->last_battle_time = 0;
            $battleInfo->save();


            //Wipe worm data
            WormData::withoutEvents(function () use ($wormData) {
                $wormData->armor = 0;
                $wormData->attack = 0;
                $wormData->level = 1;
                $wormData->experience = 0;
                $wormData->hat = config('wormix.starter.race');
                $wormData->save();
            });

            //Wipe user profile
            $user_profile->money = config('wormix.starter.money');
            $user_profile->real_money = config('wormix.starter.real_money');
            $user_profile->rating = 0;
            $user_profile->reaction_rate = 0;
            $user_profile->reagents = [];
            $user_profile->recipes = [];
            $user_profile->save();

            //wipe login sequence
            $loginSequence->last_login = date("Y-m-d");
            $loginSequence->bonus_type = 0;
            $loginSequence->bonus_count = 0;
            $loginSequence->login_sequence = 1;
            $loginSequence->gift_accepted = 1;
            $loginSequence->save();

            HouseAction::destroy(
                HouseAction::query()
                    ->where('user_id', $request->json('internal_user_id'))
                    ->orWhere('to_user_id', $request->json('internal_user_id'))
                    ->select('id')
                    ->get()
                    ->pluck('id')
                    ->toArray()
            );

            UserWeapon::destroy(
                UserWeapon::query()
                    ->where('owner_id', $request->json('internal_user_id'))
                    ->select('id')
                    ->get()
                    ->pluck('id')
                    ->toArray()
            );

            DB::commit();
        }catch (\Exception $ex){
            DB::rollBack();
            Log::debug("Wipe error", [
                $ex->getMessage(),
                $ex->getFile(),
                $ex->getLine()
            ]);
            return [
                'data' => [
                    'Result' => 0
                ]
            ];
        }


        //Add starter weapons
        foreach(Weapon::query()->where('is_starter', 1)->get() as $w){
            $newWeapon = new UserWeapon();
            $newWeapon->weapon_id = $w->id;
            $newWeapon->count = -1;
            $newWeapon->owner_id = $request->json('internal_user_id');
            $newWeapon->save();
        }

        Cache::delete('user_session_' . $request->json('internal_user_id'));
        return [
            'data' => [
                'Result' => 0
            ]
        ];
    }

    public function resetParameters(ResetParametersRequest $request)
    {
        $user_profile = UserProfile::query()
            ->where('user_id', $request->json('internal_user_id'))
            ->get()
            ->first();
        $wormData = WormData::query()
            ->where('owner_id', $request->json('internal_user_id'))
            ->get()
            ->first();

        if(
            ($request->json('MoneyType') === 1 && $user_profile->money < config('wormix.game.buy.reset_stats.money')) ||
            $request->json('MoneyType') === 0 && $user_profile->real_money < config('wormix.game.buy.reset_stats.real_money')
        )
            return [
                'data' => new ResetParametersResult(Collection::empty(), ResetParametersResult::NotEnoughMoney)
            ];

        if($request->json('MoneyType') === 1)
            $user_profile->money -= config('wormix.game.buy.reset_stats.money');
        else
            $user_profile->real_money -= config('wormix.game.buy.reset_stats.real_money');

        $wormData->armor = 0;
        $wormData->attack = 0;
        $wormData->save();

        $user_profile->save();

        return [
            'data' => new ResetParametersResult(Collection::empty(), ResetParametersResult::Success)
        ];

    }
}
