<?php

namespace App\Http\Requests\Internal\Team;

use Illuminate\Foundation\Http\FormRequest;

class AddTeammateRequest extends FormRequest
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
            'ProfileId' => 'required|integer|exists:users,id',
            'MoneyType' => 'required|integer|between:0,1'
        ];
    }
}
