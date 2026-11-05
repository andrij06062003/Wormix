<?php

use App\Http\Controllers\Account\AccountController;
use App\Http\Controllers\Account\AuthController;
use Illuminate\Support\Facades\Route;

Route::post('sign_in', [AuthController::class, 'signIn']);
Route::post('sign_up', [AuthController::class, 'signUp']);

Route::middleware(['auth:sanctum'])->group(function () {
    Route::post('sign_out', [AuthController::class, 'signOut']);

    Route::get('', [AccountController::class, 'getAccount']);
    Route::post('', [AccountController::class, 'updateAccount']);
    Route::post('update_photo', [AccountController::class, 'updateAccountPhoto']);
    Route::get('info/{user}', [AccountController::class, 'getAccountInfo']);
    Route::post('game', [AccountController::class, 'startGame']);
});
