<?php

namespace App\Http\Resources\Internal\Account;

use App\Models\Wormix\LoginSequence;
use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\JsonResource;

class DailyBonusStructure extends JsonResource
{
    /**
     * Transform the resource into an array.
     *
     * @return array<string, mixed>
     */
    public function toArray(Request $request): array
    {
        if(!$this->gift_accepted){
            return [
                'LoginSequence' =>  $this->login_sequence,
                'DailyBonusType' => $this->bonus_type,
                'DailyBonusCount' => $this->bonus_count,
            ];
        }

        return [
            'LoginSequence' => 0,
            'DailyBonusType' => 0,
            'DailyBonusCount' =>0,
        ];

    }
}
