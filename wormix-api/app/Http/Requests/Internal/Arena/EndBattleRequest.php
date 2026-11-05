<?php

namespace App\Http\Requests\Internal\Arena;

use Illuminate\Foundation\Http\FormRequest;

class EndBattleRequest extends FormRequest
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
            'internal_user_id' => 'required|exists:users,id',
            'Result' => 'required|integer',
            'Type' => 'required|integer|min:0|max:3',

            'ExpBonus' => 'required|integer|min:0|max:10',

            'BattleId' => 'required|integer',

            'Items' => 'array',
            'CollectedReagents' => 'array',
        ];
    }
}
