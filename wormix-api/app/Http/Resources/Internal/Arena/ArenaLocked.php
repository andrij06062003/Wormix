<?php

namespace App\Http\Resources\Internal\Arena;

use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\JsonResource;

class ArenaLocked extends JsonResource
{
    /**
     * Transform the resource into an array.
     *
     * @return array<string, mixed>
     */
    public function toArray(Request $request): array
    {
        return [
            'Delay' => (config('wormix.game.missions.delay') - (time() - $this->last_battle_time)) * 1000,
            'CurrentMission' => $this->mission_id,
            'ErrorCode' => 0
        ];
    }
}
