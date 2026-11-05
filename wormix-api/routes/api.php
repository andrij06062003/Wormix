<?php

use App\Http\Controllers\FileController;
use App\Http\Controllers\VkApiController;
use Illuminate\Support\Facades\Route;

//Vk api
Route::post('', [VkApiController::class, 'handleRequest']);

//Internal api for local server
Route::prefix('internal')->middleware(['internal-request'])->group(base_path('routes/internal/internal_api.php'));

//Public api for web view
Route::prefix('account')->group(base_path('routes/public/account_api.php'));

Route::get('photos/{photo}/{name}', [FileController::class, 'getPhoto']);

