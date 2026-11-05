<?php

namespace App\Http\Resources\Internal\House;

use App\Helpers\Wormix\WormixTrashHelper;
use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\JsonResource;

class SearchTheHouseResult extends JsonResource
{
    public const Empty = 0;
    public const RealMoney = 1;
    public const Money = 2;
    public const Error = 3;
    public const NoFiveDay = 4;
    public const KeyLimitExceed = 5;
    public const Reagent = 7;


    private int $result;
    private int $value;


    public function __construct($resource, int $result, int $value)
    {
        parent::__construct($resource);
        $this->result = $result;
        $this->value = $value;
    }

    /**
     * Transform the resource into an array.
     *
     * @return array<string, mixed>
     */
    public function toArray(Request $request): array
    {
        return [
            'Result' => $this->result,
            'Value' => $this->value,
            'AvailableSearchKeys' => WormixTrashHelper::getSearchKeys($this->user_id),
            'FriendId' => $this->to_user_id
        ];
    }
}
