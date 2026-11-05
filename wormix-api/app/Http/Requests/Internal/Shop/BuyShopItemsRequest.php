<?php

namespace App\Http\Requests\Internal\Shop;

use Illuminate\Foundation\Http\FormRequest;

class BuyShopItemsRequest extends FormRequest
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
            'ShopItems' => 'required|array',
            'ShopItems.*.Id' => 'required|exists:wormix_weapons,id',
            'ShopItems.*.Count' => 'required|integer|min:-1',
            'ShopItems.*.MoneyType' => 'required|integer|max:1|min:0',
        ];
    }

    public function messages()
    {
        return [
            'ShopItems.required' => 'The ShopItems is required.',
            'ShopItems.array' => 'The ShopItems must be an array.',

            'ShopItems.*.Id.required' => 'The ShopItems field Id is required.',
            'ShopItems.*.Id.exists' => 'The ShopItems not exists required.',

            'ShopItems.*.Count.required' => 'The ShopItems Count field is required.',
            'ShopItems.*.MoneyType.required' => 'The ShopItems MoneyType field is required.',
        ];
    }
}
