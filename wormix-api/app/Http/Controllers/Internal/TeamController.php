<?php

namespace App\Http\Controllers\Internal;

use App\Http\Controllers\Controller;
use App\Http\Requests\Internal\Team\AddTeammateRequest;
use App\Http\Requests\Internal\Team\RemoveTeammateRequest;
use App\Http\Requests\Internal\Team\ReorderTeamRequest;
use App\Http\Resources\Internal\Team\TeamResult;
use App\Models\User;
use App\Models\Wormix\UserTeam;
use Illuminate\Database\Eloquent\Collection;
use Illuminate\Http\Request;

class TeamController extends Controller
{
    public function addTeammate(AddTeammateRequest $request)
    {
        $user = User::query()
            ->where('id', $request->json('internal_user_id'))
            ->with([
                'worm_data.level_model',
                'user_profile.teammates'
            ])
            ->get()
            ->first();

        if(
            UserTeam::query()
                ->where('user_id', $request->json('internal_user_id'))
                ->where('teammate_id', $request->json('ProfileId'))
                ->exists() ||
            $user->worm_data->level_model->max_worms_count <= count($user->user_profile->teammates) ||
            ($user->user_profile->money < config('wormix.game.buy.teammate.money') && $request->json('MoneyType') === 1) ||
            ($user->user_profile->real_money < config('wormix.game.buy.teammate.real_money') && $request->json('MoneyType') === 0)
        )
            return [
                'data' => new TeamResult(Collection::empty(), TeamResult::Error)
            ];

        $teammate = new UserTeam();
        $teammate->user_id = $request->json('internal_user_id');
        $teammate->teammate_id = $request->json('ProfileId');
        $teammate->order = count($user->user_profile->teammates) + 1;
        $teammate->save();

        $user_profile = $user->user_profile;
        if($request->json('MoneyType') === 0)
            $user_profile->money -= config('wormix.game.buy.teammate.money');
        else
            $user_profile->real_money -= config('wormix.game.buy.teammate.real_money');

        $user_profile->save();

        return [
            'data' => new TeamResult(Collection::empty(), TeamResult::Success)
        ];
    }

    public function deleteTeammate(RemoveTeammateRequest $request)
    {
        UserTeam::destroy(
            UserTeam::query()
                ->where('user_id', $request->json('internal_user_id'))
                ->where('teammate_id', $request->json('ProfileId'))
                ->select('id')
                ->get()
                ->pluck('id')
                ->toArray()
        );
        return [
            'data' => new TeamResult(Collection::empty(), TeamResult::Success)
        ];
    }

    public function reorderTeam(ReorderTeamRequest $request)
    {
        foreach($request->json('ReorderedWormGroup') as $key => $profileId){
            $teammate = UserTeam::query()
                ->where('user_id', $request->json('internal_user_id'))
                ->where('teammate_id', $profileId)
                ->get()
                ->first();
            if($teammate === null)
                return [
                    'data' => new TeamResult(Collection::empty(), TeamResult::Error)
                ];
            $teammate->order = $key;
            $teammate->save();
        }
        return [
            'data' => new TeamResult(Collection::empty(), TeamResult::Success)
        ];
    }
}
