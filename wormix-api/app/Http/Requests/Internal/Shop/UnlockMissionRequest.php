<?php

namespace App\Http\Requests\Internal\Shop;

use Illuminate\Foundation\Http\FormRequest;

class UnlockMissionRequest extends FormRequest
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
            'MissionId' => 'required|min:1|exists:wormix_missions,mission_id'
        ];
    }
}
