<?php

namespace App\Http\Controllers\Internal;

use App\Helpers\Wormix\WormixTrashHelper;
use App\Http\Controllers\Controller;
use App\Http\Requests\Internal\Arena\EndBattleRequest;
use App\Http\Requests\Internal\Arena\GetArenaRequest;
use App\Http\Requests\Internal\Arena\StartBattleRequest;
use App\Http\Resources\Internal\Arena\ArenaLocked;
use App\Http\Resources\Internal\Arena\ArenaResult;
use App\Models\Wormix\Mission;
use App\Models\Wormix\Reagent;
use App\Models\Wormix\UserBattleInfo;
use App\Models\Wormix\UserProfile;
use App\Models\Wormix\UserWeapon;
use App\Models\Wormix\WormData;
use Illuminate\Support\Facades\Log;

class ArenaController extends Controller
{
    private const BATTLE_BASE = 1000;

    public function getArena(GetArenaRequest $request)
    {
        $battle_info = UserBattleInfo::query()
            ->where('user_id', $request->json('internal_user_id'))
            ->get()
            ->first();

        //Add battles (one battle per x minutes)
        if($battle_info->battles_count < config('wormix.game.missions.max')){
            $battle_info->battles_count += min(
                (int)((time() - $battle_info->last_battle_time) / config('wormix.game.missions.delay')),
                config('wormix.game.missions.max')
            );
            $battle_info->battles_count = min(
                $battle_info->battles_count,
                config('wormix.game.missions.max')
            );
            $battle_info->save();
        }

        if($battle_info->battles_count === 0){
            return [
                'type' => 'ArenaLocked',
                'data' => new ArenaLocked($battle_info)
            ];
        }
        return [
            'type' => 'ArenaResult',
            'data' => new ArenaResult($battle_info, $request->json('ReturnUsersProfiles'))
        ];
    }

    public function startBattle(StartBattleRequest $request)
    {
        $battle_info = UserBattleInfo::query()
            ->where('user_id', $request->json('internal_user_id'))
            ->get()
            ->first();

        if($battle_info->battles_count <= 0)
            return response([
                'message' => 'Not enough battles'
            ], 422);

        if($request->json('MissionId') !== 0){
            if(
                //Check for bosses
                ($request->json('MissionId') > ($battle_info->last_mission_id + 1))||

                //Check for lessons
                ($request->json('MissionId') !== ($battle_info->last_mission_id - 1) && $battle_info->last_mission_id < -1)
            ){
                return response([
                    'message' => 'Try to start invalid mission'
                ], 422);
            }

            $mission = Mission::query()->where('mission_id', $request->json('MissionId'))->get()->first();
            $worm_data = WormData::query()->where('owner_id', $request->json('internal_user_id'))->get()->first();
            if($worm_data->level < $mission->required_level){
                return response([
                    'message' => 'Mission required level mismatch'
                ], 422);
            }
        }

        $battle_info->last_battle_time = time();

        if($request->json('MissionId') >= 0)
            $battle_info->battles_count -= 1;

        $battle_info->current_battle_id = self::BATTLE_BASE + $battle_info->user_id + $request->json('MissionId');

        //Random reagents generation
        srand(time());
        $reagents = [];
        if($request->json('MissionId') >= 0)
            $reagents = Reagent::query()->select('reagent_id')->pluck('reagent_id')->random(rand(0, 3))->toArray();

        if($request->json('MissionId') === 0) {
            $battle_info->battle_type = 0;
            $awards = [];
        }
        else{
            $battle_info->battle_type = 1;
            $awards = Mission::query()
                ->where('mission_id', $request->json('MissionId'))
                ->get()
                ->first()
                ->awards;
        }

        $battle_info->mission_id = $request->json('MissionId');
        $battle_info->awards = [
            'reagents' => $reagents,
            'awards'   => $awards
        ];
        $battle_info->save();

        return [
            'data' => [
                'Awards' => $awards,
                'BattleId' => $battle_info->current_battle_id,
                'ReagentForBattle' => $reagents
            ]
        ];
    }

    public function endBattle(EndBattleRequest $request)
    {
        $battle_info =
            UserBattleInfo::query()->
            where('user_id', $request->json('internal_user_id'))
                ->get()
                ->first();

        $result = $request->json('Result') - $battle_info->current_battle_id;
        if(abs($result) >= 2){
            Log::debug("Invalid battle id", [
                'request' => $request->json('BattleId'),
                'current' => $battle_info->current_battle_id,
            ]);
            return response([]);
        }

        if($request->json('MissionId') !== $battle_info->mission_id) {
            Log::debug("Invalid mission id", [
                'request' => $request->json('MissionId'),
                'current' => $battle_info->mission_id,
            ]);
            return response([]);
        }

        $this->processBattleResult($request, $result, $battle_info);
        return [
            'Status' => 'OK'
        ];
    }

    private function processBattleResult(EndBattleRequest $request, int $result, UserBattleInfo $battleInfo)
    {
        $wormData = WormData::query()->where('owner_id', $battleInfo->user_id)->get()->first();
        $user_info = UserProfile::query()->where('user_id', $battleInfo->user_id)->get()->first();

        if($battleInfo->battle_type === 0)
            $wormData->experience += $request->json('ExpBonus');

        //Decrease used weapons
        foreach($request->json('Items') as $item){
            if($item['Count'] <= 0)
                continue;
            $user_weapon = UserWeapon::query()
                ->where('weapon_id', $item['Id'])
                ->where('count', '!=', -1)
                ->where('owner_id', $battleInfo->user_id)
                ->get()->first();

            $user_weapon->count -= $item['Count'];
            if($user_weapon->count < 0)
                $user_weapon->count = 0;
            $user_weapon->save();
        }

        $mission = null;
        $twiceRun = false;

        switch($result){
            case 0: //Draw
                if($battleInfo->battle_type === 0){
                    switch ($request->json('Type')){
                        case 0: //MyLevel draw
                            $user_info->money += config('wormix.game.missions.awards.draw.money.medium');
                            $wormData->experience += config('wormix.game.missions.awards.draw.experience.medium');
                            break;
                        case 1: //High level draw
                            $user_info->money += config('wormix.game.missions.awards.draw.money.high');
                            $wormData->experience += config('wormix.game.missions.awards.draw.experience.high');
                            break;
                        case 2: //Low level draw
                            $user_info->money += config('wormix.game.missions.awards.draw.money.low');
                            $wormData->experience += config('wormix.game.missions.awards.draw.experience.low');
                            break;
                    }
                }
                break;
            case 1: //Winner
                if($battleInfo->battle_type === 1){
                    $mission = Mission::query()->where('mission_id', $battleInfo->mission_id)
                        ->get()
                        ->first();
                    if($battleInfo->mission_id < 0) {
                        $battleInfo->last_mission_id = $battleInfo->mission_id;
                    }
                    else{
                        $twiceRun = $battleInfo->mission_id - $battleInfo->last_mission_id <= 0;
                        if(!$twiceRun){
                            $battleInfo->last_mission_id = $battleInfo->mission_id;
                        }
                        $battleInfo->last_boss_fight_time = time();
                    }
                }
                else{
                    switch ($request->json('Type')){
                        case 0: //MyLevel win
                            $user_info->money += config('wormix.game.missions.awards.win.money.medium');
                            $wormData->experience += config('wormix.game.missions.awards.win.experience.medium');
                            break;
                        case 1: //High level win
                            $user_info->money += config('wormix.game.missions.awards.win.money.high');
                            $wormData->experience += config('wormix.game.missions.awards.win.experience.high');
                            break;
                        case 2: //Low level win
                            $user_info->money += config('wormix.game.missions.awards.win.money.low');
                            $wormData->experience += config('wormix.game.missions.awards.win.experience.low');
                            break;
                    }
                }
                break;
            case -1: //Looser
                if($battleInfo->battle_type == 0){
                    $user_info->money += config('wormix.game.missions.awards.loose.money');
                    $wormData->experience += config('wormix.game.missions.awards.loose.experience');
                }
                break;
        }


        $user_info->save();
        $wormData->save();

        $valid = true;
        foreach($request->json('CollectedReagents') as $reagent){
            if(!in_array($reagent, $battleInfo->awards['reagents'])){
                $valid = false;
                break;
            }
        }

        if($valid)
            WormixTrashHelper::addReagents($user_info, $request->json('CollectedReagents'));

        if($mission !== null)
            $this->processAwards($mission, $twiceRun, $user_info, $wormData);

        $battleInfo->current_battle_id = 0;
        $battleInfo->mission_id = 0;
        $battleInfo->awards = [];
        $battleInfo->save();
    }

    private function processAwards(Mission $mission, bool $isDouble, UserProfile $userProfile, WormData $wormData):void
    {
        $awards = $mission->awards;
        if(count($awards) === 0)
            return;

        if(count($awards) === 1)
            $awards = $awards[0];
        elseif(!$isDouble)
            $awards = $awards[0];
        else
            $awards  = $awards[1];

        $userProfile->money += $awards['money'];
        $userProfile->real_money += $awards['real_money'];
        $userProfile->save();

        WormixTrashHelper::addWeaponsAwards($awards['weapons'], $wormData);

        $wormData->experience += $awards['experience'];
        $wormData->save();
    }
}

