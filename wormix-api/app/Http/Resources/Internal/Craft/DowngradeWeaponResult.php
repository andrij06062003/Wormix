<?php

namespace App\Http\Resources\Internal\Craft;

use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\JsonResource;

class DowngradeWeaponResult extends JsonResource
{
    public const Success = 0;
    public const Error = 1;
    public const NotEnoughMoney = 3;

    private int $result;

    private int $recipe_id;

    public function __construct($resource, int $result, int $recipe_id)
    {
        $this->result = $result;
        $this->recipe_id = $recipe_id;
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
            'RecipeId' => $this->recipe_id,
        ];
    }
}
