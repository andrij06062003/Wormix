<?php

namespace App\Http\Requests\Internal\Craft;

use Illuminate\Foundation\Http\FormRequest;

class DowngradeWeaponRequest extends FormRequest
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
            'RecipeId' => 'required|integer|exists:wormix_craft,id',
            'internal_user_id' => 'required|integer|exists:users,id'
        ];
    }
}
