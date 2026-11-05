<?php

namespace App\Http\Resources\Internal\House;

use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\JsonResource;

class PumpReactionTheHouseResult extends JsonResource
{
    public const Ok = 0;
    public const TodayAlreadyPumped = 1;
    public const Error = 2;
    public const DayLimitPump = 3;

    private int $result;

    public function __construct($resource, int $result)
    {
        $this->result = $result;
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
            'Result' => $this->result
        ];
    }
}
