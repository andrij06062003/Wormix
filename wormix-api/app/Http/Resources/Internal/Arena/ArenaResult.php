<?php

namespace App\Http\Resources\Internal\Arena;

use App\Helpers\Wormix\WormixBotHelper;
use App\Models\Wormix\WormData;
use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\JsonResource;

class ArenaResult extends JsonResource
{

    private bool $returnAccounts;

    public function __construct($resource, bool $returnAccounts = true)
    {
        $this->returnAccounts = $returnAccounts;
        parent::__construct($resource);
    }
    /**
     * Transform the resource into an array.
     *
     * @return array<string, mixed>
     */
    public function toArray(Request $request): array
    {
        return [
            'UserProfileStructures' => $this->returnAccounts ?
                WormixBotHelper::GenerateBots(
                    WormData::query()
                        ->where('owner_id', $this->user_id)
                        ->get()
                        ->first()
                ) :
                []
            ,
            'BattlesCount' => $this->battles_count,
            'CurrentMission' => $this->last_mission_id,
            // No cooldown after a boss win: bosses are always available
            // (still limited by BattlesCount like any other battle).
            'BossAvailable' => true,
        ];
    }
}
