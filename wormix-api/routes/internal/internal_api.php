<?php

use App\Http\Controllers\Internal\ArenaController;
use App\Http\Controllers\Internal\HouseController;
use App\Http\Controllers\Internal\InfoController;
use App\Http\Controllers\Internal\InternalAccountController;
use App\Http\Controllers\Internal\InternalLoginController;
use App\Http\Controllers\Internal\ResetController;
use App\Http\Controllers\Internal\ShopController;
use App\Http\Controllers\Internal\TeamController;
use App\Http\Controllers\Internal\UpgradeController;
use Illuminate\Support\Facades\Route;

Route::post('login', [InternalLoginController::class, 'login']);

Route::middleware(['internal-auth'])->group(function () {

    Route::prefix('account')->group(function () {
        Route::prefix('buy')->group(function () {
            Route::post('items', [ShopController::class, 'buyItems']);
            Route::post('race', [ShopController::class, 'changeRace']);
            Route::post('battles', [ShopController::class, 'buyBattle']);
            Route::post('mission', [ShopController::class, 'unlockMission']);
            Route::post('reaction', [ShopController::class, 'buyReaction']);
        });

        Route::prefix('reset')->group(function () {
            Route::post('parameters', [ResetController::class, 'resetParameters']);
            Route::post('account', [ResetController::class, 'resetAccount']);
        });

        Route::post('distribute_points', [InternalAccountController::class, 'distributePoints']);
        Route::post('select_stuff', [InternalAccountController::class, 'selectStuff']);
    });

    Route::prefix('game')->group(function (){
        Route::post('get_arena', [ArenaController::class, 'getArena']);
        Route::post('start_battle', [ArenaController::class, 'startBattle']);
        Route::post('end_battle', [ArenaController::class, 'endBattle']);
    });

    Route::prefix('info')->group(function () {
        Route::post('rating', [InfoController::class, 'getRating']);
        Route::post('pumped_reaction', [InfoController::class, 'getPumpedReaction']);
    });

    Route::prefix('team')->group(function () {
        Route::post('add', [TeamController::class, 'addTeammate']);
        Route::post('remove', [TeamController::class, 'deleteTeammate']);
        Route::post('reorder', [TeamController::class, 'reorderTeam']);
    });

    Route::prefix('house')->group(function (){
        Route::post('search', [HouseController::class, 'searchTheHouse']);
        Route::post('pump_reaction', [HouseController::class, 'pumpReaction']);
        Route::post('pump_reactions', [HouseController::class, 'pumpReactions']);
    });

    Route::prefix('craft')->group(function () {
        Route::post('upgrade_weapon', [UpgradeController::class, 'upgradeWeapon']);
        Route::post('downgrade_weapon', [UpgradeController::class, 'downgradeWeapon']);
    });

    Route::prefix('achievements')->group(base_path('routes/internal/achievements_api.php'));

    Route::prefix('pvp')->group(base_path('routes/internal/pvp_api.php'));
});
