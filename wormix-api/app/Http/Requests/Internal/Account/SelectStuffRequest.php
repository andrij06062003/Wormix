<?php

namespace App\Http\Requests\Internal\Account;

use App\Helpers\Wormix\WormixTrashHelper;
use App\Models\Wormix\Weapon;
use Illuminate\Foundation\Http\FormRequest;

class SelectStuffRequest extends FormRequest
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
            'StuffId' => [
                'required', 'integer', 'min:1000',
                // An item id, or ARTIFACT_START_INDEX (no item) to take the artifact off.
                function (string $attribute, mixed $value, \Closure $fail) {
                    if ((int)$value !== WormixTrashHelper::ARTIFACT_START_INDEX && !Weapon::query()->whereKey($value)->exists())
                        $fail('Unknown item.');
                },
            ],
        ];
    }
}
