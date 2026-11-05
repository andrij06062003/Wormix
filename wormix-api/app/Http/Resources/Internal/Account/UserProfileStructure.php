<?php

namespace App\Http\Resources\Internal\Account;

use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\JsonResource;

class UserProfileStructure extends JsonResource
{
    /**
     * Transform the resource into an array.
     *
     * @return array<string, mixed>
     */
    public function toArray(Request $request): array
    {
        return [
            'Id' => $this->user_id,
            'SocialId' => (string)$this->user_id,

            // The game server reads these as unsigned numbers and can't read a negative balance. Every login
            // includes all players' profiles, so one negative balance would block login for everyone (stuck at 85%).
            'Money' => max(0, (int)$this->money),
            'RealMoney' => max(0, (int)$this->real_money),

            'ReactionRate' => $this->reaction_rate,
            'Rating' => $this->rating,

            'WeaponRecordList' => WeaponRecordList::collection($this->weapons),

            'WormsGroup' => WormStructure::collection($this->teammates()->orderBy('order')->get()),

            'Recipes' => $this->recipes,

            'Stuff' => $this->weapons()->where('weapon_id', '>', 1000)->get()->pluck('weapon_id')
        ];
    }
}
