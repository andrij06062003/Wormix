<?php

namespace App\Http\Resources\Vk;

use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\JsonResource;

class VkProfile extends JsonResource
{
    /**
     * Transform the resource into an array.
     *
     * @return array<string, mixed>
     */
    public function toArray(Request $request): array
    {
        $photo_path = $this->nickname === "bot" ? "bot" : "users";
        return [
            "uid" => (string)$this->user_id,

            "first_name" => $this->first_name,
            "last_name" => $this->last_name === null ? "" : $this->last_name,
            "nickname" => "",

            "sex"=> 1,
            "bdate"=> "23.11.2000",
            "country"=> 1,
            "city"=> 1,
            "timezone"=> 1,

            //Complete xDDD
            "photo"=> $this->photo === null ? "": url("api/photos/{$photo_path}/{$this->photo}"),
            "photo_medium"=> $this->photo === null ? "": url("api/photos/{$photo_path}/{$this->photo}"),
            "photo_big"=> $this->photo === null ? "": url("api/photos/{$photo_path}/{$this->photo}"),

            "has_mobile"=> 1,
        ];
    }
}
