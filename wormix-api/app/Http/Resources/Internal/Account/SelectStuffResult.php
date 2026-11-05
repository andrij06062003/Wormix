<?php

namespace App\Http\Resources\Internal\Account;

use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\JsonResource;

class SelectStuffResult extends JsonResource
{
    public const Success = 0;
    public const Error = 1;

    private int $result;
    private int $stuffId;

    public function __construct($resource, int $result, int $stuffId)
    {
        $this->result = $result;
        $this->stuffId = $stuffId;
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
            'StuffId' => $this->stuffId
        ];
    }
}
