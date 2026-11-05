<?php

namespace App\Observers;

use App\Helpers\Wormix\WormixTrashHelper;
use App\Http\Controllers\Internal\AchievementsController;
use App\Models\User;
use App\Models\UserSocialData;
use App\Models\Wormix\Level;
use App\Models\Wormix\LoginSequence;
use App\Models\Wormix\UserBattleInfo;
use App\Models\Wormix\UserProfile;
use App\Models\Wormix\UserTeam;
use App\Models\Wormix\UserWeapon;
use App\Models\Wormix\Weapon;
use App\Models\Wormix\WormData;

class UserObserver
{
    /**
     * Determine if running in a development environment.
     */
    private function isDevEnvironment(): bool
    {
        return app()->environment('dev', 'development', 'local')
            || env('CD_ENV') === 'dev'
            || env('WORMIX_DEV') === true
            || env('WORMIX_DEV') === 'true'
            || env('WORMIX_DEV') === '1';
    }

    /**
     * Handle the User "created" event.
     */
    public function created(User $user): void
    {
        $isDev = $this->isDevEnvironment();

        $realMoney = $isDev
            ? (int) config('wormix.starter.dev.real_money', 3000)
            : (int) config('wormix.starter.real_money', 3);

        $level = $isDev
            ? (int) config('wormix.starter.dev.level', 30)
            : (int) config('wormix.starter.level', 1);

        //Create user profile
        $user_profile = new UserProfile();
        $user_profile->user_id = $user->id;
        $user_profile->money = config('wormix.starter.money');
        $user_profile->real_money = $realMoney;
        $user_profile->save();

        //Create user worm data
        $worm_data = new WormData();
        $worm_data->owner_id = $user->id;
        $worm_data->level = $level;
        $worm_data->hat = config('wormix.starter.race');
        $worm_data->save();

        //Create user default teammate
        $teammate = new UserTeam();
        $teammate->user_id = $user->id;
        $teammate->teammate_id = $user->id;
        $teammate->save();

        //Create user social data
        $social_data = new UserSocialData();
        $social_data->user_id = $user->id;
        $social_data->first_name = $user->login;
        $social_data->save();

        //Create user battle info
        $battle_info = new UserBattleInfo();
        $battle_info->user_id = $user->id;
        $battle_info->battles_count = config('wormix.starter.missions');
        $battle_info->save();

        //Create user login sequence info
        $login_sequence = new LoginSequence();
        $login_sequence->user_id = $user->id;
        $login_sequence->last_login = date("Y-m-d");
        $login_sequence->gift_accepted = 1;
        $login_sequence->save();

        //Add starter user weapons
        $starter_weapons = Weapon::query()->where('is_starter', 1)->get();
        foreach($starter_weapons as $w){
            $weapon = new UserWeapon();
            $weapon->owner_id = $user->id;
            $weapon->weapon_id = $w->id;
            $weapon->save();
        }

        //Add level awards weapons for dev environment
        if ($isDev && $level > 1) {
            $levels = Level::query()->whereBetween('id', [2, $level])->get();
            foreach ($levels as $l) {
                if (!empty($l->awards)) {
                    WormixTrashHelper::addWeaponsAwards($l->awards, $worm_data);
                }
            }
        }

        //Grant dev achievement points for dev environment
        if ($isDev) {
            AchievementsController::grantDevAchievements($user->id);
        }
    }

    /**
     * Handle the User "updated" event.
     */
    public function updated(User $user): void
    {
        //
    }

    /**
     * Handle the User "deleted" event.
     */
    public function deleted(User $user): void
    {
        //
    }

    /**
     * Handle the User "restored" event.
     */
    public function restored(User $user): void
    {
        //
    }

    /**
     * Handle the User "force deleted" event.
     */
    public function forceDeleted(User $user): void
    {
        //
    }
}
