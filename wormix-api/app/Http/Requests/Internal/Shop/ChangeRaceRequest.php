<?php

namespace App\Http\Requests\Internal\Shop;

use Illuminate\Foundation\Http\FormRequest;

class ChangeRaceRequest extends FormRequest
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
            'internal_user_id' => 'required|integer|exists:users,id',
            'RaceId' => 'required|integer|exists:wormix_races,race_id',
            'MoneyType' => 'required|integer|min:0|max:1'
        ];
    }
}
