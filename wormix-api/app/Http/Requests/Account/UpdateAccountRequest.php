<?php

namespace App\Http\Requests\Account;

use Illuminate\Foundation\Http\FormRequest;

class UpdateAccountRequest extends FormRequest
{
    /**
     * Determine if the user is authorized to make this request.
     */
    public function authorize(): bool
    {
        return true;
    }

    /**
     * Get the validation rules that apply to the request.
     *
     * @return array<string, \Illuminate\Contracts\Validation\ValidationRule|array<mixed>|string>
     */
    public function rules(): array
    {
        return [
            'social_data.first_name' => 'string|max:100',
            'social_data.last_name' => 'string|max:100',

            'user_profile.money' => 'integer|min:0',
            'user_profile.real_money' => 'integer|min:0',
            'user_profile.rating' => 'integer|min:0',
            'user_profile.reaction_rate' => 'integer|min:0',

            'worm_data.level' => 'integer|min:1|max:30',
            'worm_data.armor' => 'integer|min:0|max:60',
            'worm_data.attack' => 'integer|min:0|max:60',
            'worm_data.race' => 'integer|exists:wormix_races,race_id',

            'battle_info.battles_count' => 'integer|min:0|max:999',

            'user.login' => 'string|max:100',
            'user.password' => 'nullable|string|min:8',
            'user.password_confirmation' => 'nullable|string|min:8|required_unless:user.password,null|same:user.password',
        ];
    }
}
