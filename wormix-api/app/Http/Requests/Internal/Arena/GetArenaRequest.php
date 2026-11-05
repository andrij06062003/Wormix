<?php

namespace App\Http\Requests\Internal\Arena;

use Illuminate\Foundation\Http\FormRequest;

class GetArenaRequest extends FormRequest
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
            'ReturnUsersProfiles' => 'required|boolean',
        ];
    }
}
