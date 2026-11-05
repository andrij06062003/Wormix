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

## Missing file: `data/common/rooms/achiev_room.swf`

The 1.05.0 client archive had no achievements room (the house's achievements screen failed to open
with a 404). `vk/1.05.0/data/common/rooms/achiev_room.swf` is taken from a 1.25.0 client
(github.com/mazmax228228/wormix, `vk/1.25.0/data/common/rooms/achiev_room.swf`). It is graphics only,
with the same classes (AchievRoom, AwardCell, AchievTabPage, AchievTabAwards) and the named parts
the 1.05.0 code uses. It also has a "bosses" tab button from 1.25.0 that does nothing here.
