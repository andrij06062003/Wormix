<?php

namespace App\Http\Requests\Internal\Account;

use Illuminate\Foundation\Http\FormRequest;

class LoginRequest extends FormRequest
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
            'tcp_session' => 'required|uuid',
            'Id' => 'required|integer',
            'AuthKey' => 'required|string'
        ];
    }

    public function messages(): array{
        return [
            'tcp_session.required' => 'TCP session is required.',
            'tcp_session.uuid' => 'TCP session must be is GUID.',

            'Id.required' => 'id is required',
            'Id.integer' => 'id must be an integer',

            'AuthKey.required' => 'auth_key is required',
            'AuthKey.string' => 'auth_key must be a string',
        ];
    }
}
