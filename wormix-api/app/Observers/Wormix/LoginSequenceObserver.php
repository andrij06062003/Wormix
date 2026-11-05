<?php

namespace App\Observers\Wormix;

use App\Models\Wormix\DailyBonus;
use App\Models\Wormix\LoginSequence;
use App\Models\Wormix\UserBattleInfo;
use App\Models\Wormix\UserProfile;
use Illuminate\Support\Facades\Log;

class LoginSequenceObserver
{
    /**
     * Handle the LoginSequence "created" event.
     */
    public function created(LoginSequence $loginSequence): void
    {
        //
    }

    /**
     * Handle the LoginSequence "updated" event.
     */
    public function updated(LoginSequence $loginSequence): void
    {
        $user_profile = UserProfile::query()->where('user_id', $loginSequence->user_id)->get()->first();
        $bonus = DailyBonus::query()->where('login_sequence', $loginSequence->login_sequence)->get()->first();
        $battle_info = UserBattleInfo::query()->where('user_id', $loginSequence->user_id)->get()->first();
        if($user_profile === null || $bonus === null || $battle_info == null)
            return;

        srand(time());

        LoginSequence::withoutEvents(function () use ($loginSequence, $user_profile, $bonus, $battle_info) {
            $amount = $bonus->random_gift ? rand($bonus->rand_min, $bonus->rand_max) : $bonus->bonus_value;
            $type = $bonus->random_gift ? rand(1, 4) : $bonus->bonus_type;
            switch ($type){
                case 1: //fuzzes
                    $user_profile->money += $amount;
                    $user_profile->save();
                    break;
                case 2: //rubies
                    $user_profile->real_money += $amount;
                    $user_profile->save();
                    break;

                case 3: //missions
                    $battle_info->battles_count += $amount;
                    $battle_info->save();
                    break;

                case 4: // reaction rate
                    $user_profile->reaction_rate += $amount;
                    $user_profile->save();
                    break;
            }
            $loginSequence->bonus_count = $amount;
            $loginSequence->bonus_type = $type;
            $loginSequence->save();
        });
    }

    /**
     * Handle the LoginSequence "deleted" event.
     */
    public function deleted(LoginSequence $loginSequence): void
    {
        //
    }

    /**
     * Handle the LoginSequence "restored" event.
     */
    public function restored(LoginSequence $loginSequence): void
    {
        //
    }

    /**
     * Handle the LoginSequence "force deleted" event.
     */
    public function forceDeleted(LoginSequence $loginSequence): void
    {
        //
    }
}
