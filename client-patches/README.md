# Client patches

Fixes compiled into `wormix-client/vk/1.05.0/Wormix.swf`.

## PointCrosshair.as

Weapons aimed with a point crosshair (air strikes / fire rockets, homing
rockets, girder "балка", teleport, bunker buster, meteor strike) could leave
the crosshair arrow stuck on the map and the mouse cursor hidden after the shot.

Cause: the shot is processed inside the scene's ENTER_FRAME handler. It
deactivates the crosshair and removes the `moveCrosshair` ENTER_FRAME listener,
but Ruffle still calls that listener later in the same frame. `moveCrosshair`
then shows the crosshair again (`Mouse.hide()` + `addChild`), and nothing ever
removes it. Also, `deactivate()` only cleaned up when it was still the
player's turn.

Fix: `moveCrosshair` stops if the crosshair is no longer active,
`deactivate()` always removes listeners and hides the crosshair, and
`hideCrosshair()` removes the clip from whatever parent it is on.

Rebuild (JPEXS FFDec, https://github.com/jindrapetrik/jpexs-decompiler):

    java -jar ffdec-cli.jar -replace Wormix.orig.swf Wormix.swf \
        ru.pragmatix.wormix.crosshair.PointCrosshair client-patches/PointCrosshair.as

`Wormix.orig.swf` is the unpatched file from git history (commit 1c91837).

## `Hintable.as`: weapon description stays on screen after closing the arsenal

The description (hint) of a weapon or item is added to the scene, not inside the cell it belongs to,
and is removed only when the cell sends ROLL_OUT. When a weapon is chosen the arsenal closes under the
mouse; Ruffle sends no ROLL_OUT then, so the description stayed on screen.

Fix: `Hintable` also listens for REMOVED_FROM_STAGE on the cell and then removes the hint (and cancels
one that is still waiting to appear).

Rebuild on top of the PointCrosshair patch:

    java -jar ffdec-cli.jar -replace Wormix.swf Wormix.new.swf \
        ru.pragmatix.flox.gui.controls.hint.Hintable client-patches/Hintable.as

## `BattleRequestController.as`: friend battle desyncs after adding a teammate

When you add a friend to your team, the client puts the friend in with their own level and stats, but the
server scales a teammate to the owner's level. The player you challenge gets your team from the server,
while your game kept its own version, so the two games started the battle with different worms.
After logging in again the two matched, which is why it only went wrong after adding a teammate.

Fix: when a friend battle is accepted, the challenger loads its own profile from the server's answer
(`BattleRequestResult.myProfile`), as the accepting side and wager battles already do.

Rebuild on top of the patches above:

    java -jar ffdec-cli.jar -replace Wormix.swf Wormix.new.swf \
        ru.pragmatix.wormix.pvp.controllers.BattleRequestController client-patches/BattleRequestController.as

## Missing file: `data/common/rooms/achiev_room.swf`

The 1.05.0 client archive had no achievements room (the house's achievements screen failed to open
with a 404). `vk/1.05.0/data/common/rooms/achiev_room.swf` is taken from a 1.25.0 client
(github.com/mazmax228228/wormix, `vk/1.25.0/data/common/rooms/achiev_room.swf`). It is graphics only,
with the same classes (AchievRoom, AwardCell, AchievTabPage, AchievTabAwards) and the named parts
the 1.05.0 code uses.

The 1.25.0 rooms are laid out 228.4 px further right than in 1.05.0 (compare the race room, which
exists in both), so the room is moved back, and the 1.25.0 "bosses" tab (nothing behind it in 1.05.0)
is removed. To rebuild from the original 1.25.0 file:

    java -jar ffdec-cli.jar -swf2xml achiev_room_125.swf room.xml
    python3 client-patches/achiev_room_layout.py room.xml room_fixed.xml
    java -jar ffdec-cli.jar -xml2swf room_fixed.xml achiev_room.swf
