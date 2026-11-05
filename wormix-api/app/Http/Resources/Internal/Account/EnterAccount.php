<?php

namespace App\Http\Resources\Internal\Account;

use App\Helpers\Wormix\WormixTrashHelper;
use App\Models\Wormix\UserProfile;
use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\JsonResource;

class EnterAccount extends JsonResource
{
    private string $session_key;

    public function __construct($resource, $session_key)
    {
        $this->session_key = $session_key;
        parent::__construct($resource);
    }
    public function toArray(Request $request): array
    {

        return [
            'UserProfileStructure' => new  UserProfileStructure($this->user_profile),

            'UserProfileStructures' => UserProfileStructure::collection(UserProfile::query()->where('user_id', '!=', $this->id)->get()),

            'AvailableSearchKeys' => WormixTrashHelper::getSearchKeys($this->id),

            'Friends' => UserProfile::query()->where('user_id', '!=', $this->id)->count(),
            'OnlineFriends' => 0,
            'IsBonusDay' => false,

            'DailyBonusStructure' => new DailyBonusStructure($this->login_sequence),
            'Reagents' => $this->user_profile->reagents,

            'SessionKey' => $this->session_key
        ];
    }
}
