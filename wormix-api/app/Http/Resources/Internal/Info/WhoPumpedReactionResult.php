<?php

namespace App\Http\Resources\Internal\Info;

use App\Http\Resources\Internal\Account\ProfileDoubleKeyStructure;
use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\JsonResource;

class WhoPumpedReactionResult extends JsonResource
{
    /**
     * Transform the resource into an array.
     *
     * @return array<string, mixed>
     */
    public function toArray(Request $request): array
    {
        return [
            'TodayPumped' => ProfileDoubleKeyStructure::collection(
                $this
                    ->house_actions()
                    ->where('action_type', 0)
                    ->where('created_at', '>', date('Y-m-d'))
                    ->get()
            ),
            'YesterdayPumped' => ProfileDoubleKeyStructure::collection(
                $this
                    ->house_actions()
                    ->where('action_type', 0)
                    ->where('created_at', '>', date('Y-m-d', strtotime('-1 day')))
                    ->where('created_at', '<', date('Y-m-d'))
                    ->get()
            ),
            'TwoDaysAgoPumped' => ProfileDoubleKeyStructure::collection(
                $this
                    ->house_actions()
                    ->where('action_type', 0)
                    ->where('created_at', '>', date('Y-m-d', strtotime('-2 day')))
                    ->where('created_at', '<', date('Y-m-d', strtotime('-1 day')))
                    ->get()
            )
        ];
    }
}
