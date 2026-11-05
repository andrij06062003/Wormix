<?php

namespace App\Http\Controllers\Account;

use App\Helpers\Wormix\WormixBotHelper;
use App\Helpers\Wormix\WormixTrashHelper;
use App\Http\Controllers\Controller;
use App\Http\Requests\Account\UpdateAccountRequest;
use App\Http\Requests\Account\UpdatePhotoRequest;
use App\Http\Resources\Account\UserResource;
use App\Models\User;
use App\Models\UserSocialData;
use App\Modules\UserSession;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\File;
use Illuminate\Support\Str;

class AccountController extends Controller
{
    public function getAccount(Request $request)
    {
        return $this->getAccountResource($request->user());
    }

    public function getAccountInfo(User $user)
    {
        return $this->getAccountResource($user);
    }

    private function getAccountResource(User $user)
    {
        return new UserResource($user->load([
            'user_profile',
            'worm_data',
            'social_data',
            'battle_info',
            'user_profile.teammates.teammate'
        ]));
    }

    public function updateAccount(UpdateAccountRequest $request)
    {
        if($request->json('social_data') !== null){
            $social_data = $request->user()->social_data;
            $social_data->fill($request->json('social_data'));
            $social_data->save();
        }

        if($request->json('user_profile') !== null){
            $user_profile = $request->user()->user_profile;
            $user_profile->fill($request->json('user_profile'));
            $user_profile->save();
        }

        if($request->json('worm_data') !== null){
            $worm_data = $request->user()->worm_data;

            $wormHatAndRace = WormixTrashHelper::getRaceAndHatIds($worm_data->hat);
            $wormHatAndRace[0] = $request->json('worm_data.race');

            $worm_data->fill($request->json('worm_data'));
            $strip_params = WormixBotHelper::stripData(
                (int)(($worm_data->armor + $worm_data->attack) / 2),
                $worm_data->level,
                $worm_data->armor,
                $worm_data->attack
            );
            $worm_data->armor = $strip_params['armor'];
            $worm_data->attack = $strip_params['attack'];
            $worm_data->experience = 0;
            $worm_data->hat = WormixTrashHelper::getHatByRaceAndHatIds($wormHatAndRace[1], $wormHatAndRace[0]);
            $worm_data->save();

            return [
                'worm_data' => $worm_data,
            ];
        }

        if($request->json('user') !== null){
            $user = $request->user();
            if($request->json('user.login') !== null){
                if(User::query()
                        ->where('login', $request->json('user.login'))
                        ->where('id', '!=', $user->id)->count() > 0
                ){
                    return response([
                        'message' => 'User login must be unique'
                    ], 422);
                }
                $user->login = $request->json('user.login');
            }

            if($request->json('user.password') !== null) {
                $user->password = $request->json('user.password');
                $user->tokens()->where('id', '!=', $user->currentAccessToken()->id)->delete();
            }

            if($user->isDirty([
                'login',
                'password'
            ]))
                $user->save();

        }

        if($request->json('battle_info.battles_count') !== null){
            $battle_info = $request->user()->battle_info;
            $battle_info->battles_count = $request->json('battle_info.battles_count');
            $battle_info->save();
        }

        return [
            'success' => true,
        ];
    }

    public function updateAccountPhoto(UpdatePhotoRequest $request)
    {
        $user_social_data = $request->user()->social_data;
        $user_photo = $request->file('photo');

        $new_photo_name = Str::uuid()->toString().".".$user_photo->extension();

        if($user_social_data->photo !== null){
            $old_photo_path = resource_path('images/users/'.$user_social_data->photo);
            if(File::exists($old_photo_path))
                File::delete($old_photo_path);
        }

        $user_social_data->photo = $new_photo_name;
        $user_social_data->save();

        $user_photo->move(resource_path('images/users/'), $new_photo_name);

        return [
            'data' => $user_social_data
        ];
    }

    /**
     * @throws \Exception
     */
    public function startGame(Request $request)
    {
        $user_session = new UserSession($request->user()->id);
        return response([
            'auth_key' => $user_session->setAuthKey()
        ]);
    }
}
