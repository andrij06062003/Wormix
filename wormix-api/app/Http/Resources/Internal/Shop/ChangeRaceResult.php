<?php

namespace App\Http\Resources\Internal\Shop;

use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\JsonResource;

class ChangeRaceResult extends JsonResource
{
    public const Success = 0;
    public const Error = 1;
    public const MinRequirementsError = 2;
    public const NotEnoughMoney = 3;

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
