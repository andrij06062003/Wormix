<?php

namespace App\Http\Controllers;

use App\Http\Requests\VkApiRequest;
use App\Http\Resources\Vk\VkProfile;
use App\Models\User;
use App\Models\UserSocialData;
use Illuminate\Database\Eloquent\Collection;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\File;
use Illuminate\Support\Facades\Log;

class VkApiController extends Controller
{
    public function handleRequest(VkApiRequest $request){
        //Log::debug("VK_API_REQUEST", $request->toArray());
        return match ($request->post('method')) {
            'getUserSettings' => $this->getUserSettings(),
            'getAppFriends' => $this->getAppFriends(),
            'getProfiles' => $this->getProfiles($request),
            'getUserBalance' => $this->getUserBalance($request),
            default => "", //Other methods not needed
        };
    }

    private function getUserSettings()
    {
        return [
            'response' => 2371351
        ];
    }

    private function getAppFriends()
    {
        return [
            'response' => User::query()->select('id')->get()->pluck('id')->toArray(),
        ];
    }

    private function getProfiles(Request $request)
    {
        $uids_array = json_decode($request->post('uids'), true);
        $is_bots = false;

        foreach ($uids_array as $uid) {
            if($uid < 0){
                $is_bots = true;
                break;
            }
        }

        if(!$is_bots)
            return [
                'response' => VkProfile::collection(UserSocialData::all())
            ];

        $profiles = Collection::empty();
        $names = null;

        if(File::exists(resource_path('game/names.json')))
            $names = json_decode(File::get(resource_path('game/names.json')), true);

        if($names === null)
            $names = ["NaN Bot"];

        foreach($uids_array as $uid){
            $random_name = explode(" ", $names[array_rand($names)]);
            $bot_profile = new UserSocialData();
            $bot_profile->user_id = $uid;
            $bot_profile->first_name = $random_name[1];
            $bot_profile->last_name = $random_name[0];
            $bot_profile->nickname = "bot";

            if(File::exists(resource_path('images/bots'))){
                $file_list = File::files(resource_path('images/bots'));
                if(count($file_list) !== 0)
                    $bot_profile->photo = $file_list[array_rand($file_list)]->getFilename();
            }
            $profiles->add($bot_profile);
        }

        return [
            'response' => VkProfile::collection($profiles)
        ];
    }

    private function getUserBalance()
    {
        return [
            'response' => config('wormix.vk_balance') * 100
        ];
    }
}
