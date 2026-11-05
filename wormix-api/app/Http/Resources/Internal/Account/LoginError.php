<?php

namespace App\Http\Resources\Internal\Account;

use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\JsonResource;

class LoginError extends JsonResource
{
    private int $result;

    public function __construct($request, $result)
    {
        $this->result = $result;
        parent::__construct($request);
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
