<?php

namespace App\Http\Requests\Internal\Pvp;

use Illuminate\Foundation\Http\FormRequest;

class PvpStartBattleRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    public function rules(): array
    {
        return [
            'internal_user_id' => 'required|exists:users,id',
            'BattleId' => 'required|integer|min:1',
            'Wager' => 'required|integer|min:0',
            'Reagents' => 'nullable|array',
            'Reagents.*' => 'integer',
        ];
    }
}
