<?php

namespace App\Http\Resources\Internal\Shop;

use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\JsonResource;

class BuyReactionRateResult extends JsonResource
{
    public const Success = 0;
    public const Error = 1;

    private int $result;

    private int $count;

    public function __construct($resource, int $result, int $count)
    {
        $this->result = $result;
        $this->count = $count;
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
            'Result' => $this->result,
            'ReactionRateCount' => $this->count
        ];
    }
}
