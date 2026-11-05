<?php

namespace App\Listeners;

use App\Events\InternalLoginEvent;
use App\Helpers\Wormix\WormixTrashHelper;
use App\Models\Wormix\LoginSequence;
use App\Models\Wormix\UserBattleInfo;
use App\Models\Wormix\UserWeapon;
use Carbon\Carbon;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Support\Facades\Log;

class LoginEventListener
{
    /**
     * Create the event listener.
     */
    public function __construct()
    {
        //
    }

    /**
     * Handle the event.
     */
    public function handle(object $event): void
    {
        if($event instanceof InternalLoginEvent)
            $this->handleInternalLoginEvent($event);
    }

    private function handleInternalLoginEvent(InternalLoginEvent $event): void
    {
        $user = $event->getUser();

        //Update login sequence
        $new_account = $user->login_sequence->last_login == null;
        $old_login_date = $new_account ? now() : Carbon::createFromTimeString($user->login_sequence->last_login);
        $login_sequence = $user->login_sequence;

        $days = max(0, (int)$old_login_date->diff(now())->totalDays);

        if($days === 1){
            $login_sequence->login_sequence += 1;
        }elseif($days > 1){
            $login_sequence->login_sequence  = 1;
        }

        if($login_sequence->login_sequence > 5)
            $login_sequence->login_sequence = 1;

        $user->login_sequence->last_login = date("Y-m-d");
        $login_sequence->gift_accepted = false;

        //Not add gifts if already taken
        if($days === 0 && ! $new_account){
            LoginSequence::withoutEvents(function () use ($login_sequence) {
                $login_sequence->gift_accepted = true;
                $login_sequence->save();
            });
        }
        else{
            $login_sequence->save();
        }

        //Check current hat
        $race_and_hat = WormixTrashHelper::getRaceAndHatIds($user->worm_data->hat);
        $current_hat = UserWeapon::query()->where('id', $race_and_hat[1])->get()->first();
        if($current_hat != null && @$current_hat->expire_at < time()){
            //Reset expired hat
            $new_hat = WormixTrashHelper::getHatByRaceAndHatIds(0, $race_and_hat[0]);
            $user_worm = $user->worm_data;
            $user_worm->hat = $new_hat;
            $user_worm->save();
        }

        $battle_info = UserBattleInfo::query()->where('user_id', $user->id)
            ->get()
            ->first();

        //Clear mission id before boss fights
        if($user->worm_data->level > 5 && $battle_info->last_mission_id < 0){
            $battle_info->last_mission_id = 0;
            $battle_info->save();
        }
        //Destroy expired items
        UserWeapon::destroy(
            UserWeapon::query()
                ->select('id')
                ->where('expire_at', '!=', -1)
                ->where('expire_at', '<', time())
                ->get()
        );
    }
}
