<?php

namespace App\Http\Requests\Account;

use Illuminate\Foundation\Http\FormRequest;

class UpdatePhotoRequest extends FormRequest
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
            'photo' => 'required|image|mimes:jpeg,png,jpg|max:4096|dimensions:min_width=100,min_height=100',
        ];
    }

    public function messages(): array {
        return [
            'photo.required' => 'Photo is required.',
            'photo.image' => 'Photo is invalid.',
            'photo.mimes' => 'Photo format is invalid.',
            'photo.max' => 'Photo is too large.',
            'photo.dimensions' => 'Photo dimensions invalid.(min: 100x100)',
        ];
    }
}
