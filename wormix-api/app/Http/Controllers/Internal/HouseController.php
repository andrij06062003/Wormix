<?php

namespace App\Http\Controllers\Internal;

use App\Helpers\Wormix\WormixTrashHelper;
use App\Http\Controllers\Controller;
use App\Http\Requests\Internal\House\PumpReactionRateRequest;
use App\Http\Requests\Internal\House\PumpReactionsRateRequest;
use App\Http\Requests\Internal\House\SearchTheHouseRequest;
use App\Http\Resources\Internal\House\PumpReactionTheHouseResult;
use App\Http\Resources\Internal\House\SearchTheHouseResult;
use App\Models\Wormix\HouseAction;
use App\Models\Wormix\Reagent;
use App\Models\Wormix\UserProfile;
use Illuminate\Database\Eloquent\Collection;
use Illuminate\Support\Facades\Log;

class HouseController extends Controller
{
    public function pumpReactions(PumpReactionsRateRequest $request)
    {
        $result = [];

        foreach($request->json('FriendsIds') as $friendId) {
            if(HouseAction::query()
                ->where('user_id', $request->json('internal_user_id'))
                ->where('to_user_id', $friendId)
                ->where('action_type', 0)
                ->where('created_at', '>', now()->subDay())
                ->exists() ||
                !HouseAction::query()
                    ->where('user_id',  $friendId)
                    ->where('to_user_id', $request->json('internal_user_id'))
                    ->where('action_type', 0)
                    ->where('created_at', '>', now()->subDays(3))
                    ->exists()
            )
                continue;

            $result[] = [
                'FriendId' => $friendId,
                'Result' => 0
            ];

            $user_profile = UserProfile::query()
                ->where('user_id', $friendId)
                ->get()
                ->first();

            $user_profile->reaction_rate += 1;
            $user_profile->save();

            $action = new HouseAction();
            $action->action_type = 0;
            $action->user_id = $request->json('internal_user_id');
            $action->to_user_id = $friendId;
            $action->save();
        }

        return [
            'data' => [
                'PumpedFriends' => $result
            ]
        ];
    }

    public function pumpReaction(PumpReactionRateRequest $request)
    {
        if($request->json('internal_user_id') === $request->json('FriendId'))
            return new PumpReactionTheHouseResult(Collection::empty(), PumpReactionTheHouseResult::Error);

        $pump_reaction_friend = HouseAction::query()
            ->where('to_user_id', $request->json('FriendId'))
            ->where('user_id', $request->json('internal_user_id'))
            ->where('action_type', 0)
            ->where('created_at', '>', date('Y-m-d H:i:s', strtotime('-1 day')))
            ->get()
            ->first();

        if($pump_reaction_friend !== null){
            return  new PumpReactionTheHouseResult(Collection::empty(), PumpReactionTheHouseResult::TodayAlreadyPumped);
        }


        $to_user_profile = UserProfile::query()->where('user_id', $request->json('FriendId'))
            ->get()
            ->first();

        $to_user_profile->reaction_rate += 1;
        $to_user_profile->save();

        $new_action = new HouseAction();
        $new_action->user_id = $request->json('internal_user_id');
        $new_action->to_user_id = $request->json('FriendId');
        $new_action->action_type = 0;
        $new_action->save();

        return new PumpReactionTheHouseResult(Collection::empty(), PumpReactionTheHouseResult::Ok);
    }

    public function searchTheHouse(SearchTheHouseRequest $request)
    {
        $search_action = new HouseAction();
        $search_action->user_id = $request->json('internal_user_id');
        $search_action->to_user_id = $request->json('FriendId');
        $search_action->action_type = 1;

        if($request->json('internal_user_id') === $request->json('FriendId'))
            return new SearchTheHouseResult($search_action, SearchTheHouseResult::Error, 0);

        $search_keys = WormixTrashHelper::getSearchKeys($request->json('internal_user_id'));

        if($search_keys <= 0){
            return new SearchTheHouseResult($search_action, SearchTheHouseResult::KeyLimitExceed, 0);
        }

        if(WormixTrashHelper::isSearchedToday($request->json('internal_user_id'), $request->json('FriendId'))){
            return new SearchTheHouseResult($search_action, SearchTheHouseResult::Empty, 0);
        }

        $user_profile = UserProfile::query()->where('user_id', $request->json('internal_user_id'))->get()->first();
        $search_action->save();

        srand(time());

        switch (rand(0, 3)) {
            case 0: //Empty
                return [
                    'data' => new SearchTheHouseResult($search_action, SearchTheHouseResult::Empty, 0)
                ];

            case 1: //Money
                $money = rand(1, 20);
                $user_profile->money += $money;
                $user_profile->save();
                return [
                    'data' => new SearchTheHouseResult($search_action, SearchTheHouseResult::Money, $money)
                ];

            case 2: //Real money
                $money = rand(1, 5);
                $user_profile->real_money += $money;
                $user_profile->save();
                return [
                    'data' => new SearchTheHouseResult($search_action, SearchTheHouseResult::RealMoney, $money)
                ];

            case 3: //Reagent
                $reagents = Reagent::query()->select('reagent_id')->pluck('reagent_id')->toArray();
                $reagent = $reagents[array_rand($reagents)];
                WormixTrashHelper::addReagents($user_profile, [$reagent]);
                return [
                    'data' => new SearchTheHouseResult($search_action, SearchTheHouseResult::Reagent, $reagent)
                ];
        }

        return new SearchTheHouseResult($search_action, SearchTheHouseResult::Error, 0);
    }
}
