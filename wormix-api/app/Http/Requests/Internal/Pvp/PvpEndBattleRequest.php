<?php

namespace App\Http\Requests\Internal\Pvp;

use Illuminate\Foundation\Http\FormRequest;

class PvpEndBattleRequest extends FormRequest
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
            'Result' => 'required|integer|min:-1|max:1',
        ];
    }
}
