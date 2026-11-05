<?php

namespace App\Http\Requests;

use Illuminate\Foundation\Http\FormRequest;

class VkApiRequest extends FormRequest
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
            'method' => 'required|string',
        ];
    }

    public function messages(): array
    {
        return [
            'method.required' => 'Method name is required.',
            'method.string' => 'Method name must be a string.',
        ];
    }
}
