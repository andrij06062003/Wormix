<?php

namespace App\Http\Requests\Internal\House;

use Illuminate\Foundation\Http\FormRequest;

class SearchTheHouseRequest extends FormRequest
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
            'FriendId' => 'required|exists:users,id',
        ];
    }
}
