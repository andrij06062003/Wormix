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

## `artifacts/`: artifacts from 1.25.0 (stage 1: shop, wardrobe, bonuses)

1.25.0 added artifacts: an item held in the hand, in its own slot next to the hat. They are ported
on top of the hat code. An artifact is a `HatRecord` with `isArtifact`, kept with the hats of every race,
so the shop's stuff tab, buying and the wardrobe list them with no new screens.

- Protocol: `WormStructure` has a new `artifact` short (item id 2001+, 0 = none) after `socialOwnerId`.
  Server (wormix-core `WormStructure.Artifact`) and client must ship together.
- `HatRecord`: the 1.25.0 names of the bonuses 1.05.0 already has are mapped:
  `fireDamage`/`fireReduction` → `fireDamageBonus`/`fireDamageReduction`, and `poisonDamage` (%) →
  `poison` levels (20% each). Bonuses with no effect in 1.05.0 yet (shock/air/frost, penetration,
  squad armor/attack, fortitude, heal bonus, poison protection %, ...) are kept in `extraBonuses` for a
  later stage. An artifact whose bonuses all have no effect yet is `hideInShop` in artifacts.json.
- `UserProfile`: the artifact slot (`getArtifact`/`setArtifact`). `setHat(artifact)` puts it in that slot,
  so shop purchases and rewards work unchanged. `BonusStats` adds the artifact's bonuses to the hat's.
- `WardrobeRoom`: the hat and the artifact are both highlighted. A click on an artifact puts it on, a
  click on the one you wear takes it off. Saving sends `SelectStuff` for the hat, then for the artifact
  (`2000` = take it off).
- `CharGraphics.setHat` ignores artifacts (they are not drawn on the head; drawing them in the hand is a later stage).
- Data: `ConfigsConverter.ARTIFACTS` is generated from `wormix-api/resources/game/artifacts.json`
  (the server's copy) by `gen_artifacts_config.py`. Graphics: `data/common/objects/artifacts.swf` from a
  1.25.0 client (github.com/lightcool697/wormix.github.io, `vk/1.25.0/data/common/objects/artifacts.swf`),
  listed in resconfig.xml. Names: `ARTIFACT.*` in `items.messages_*.xml`.

Rebuild on top of the patches above (one `-replace` per class, in this order):

    python3 client-patches/artifacts/gen_artifacts_config.py
    for c in model.items.HatRecord model.items.ItemsStore messages.structures.WormStructure \
             serialization.structures.WormStructureBinarySerializer model.user.UserProfile \
             model.worm.BonusStats objects.graphics.CharGraphics loader.ConfigsConverter \
             gui.house.wardrobe.WardrobeRoom; do
        java -jar ffdec-cli.jar -replace Wormix.swf Wormix.new.swf ru.pragmatix.wormix.$c \
            client-patches/artifacts/${c##*.}.as && mv Wormix.new.swf Wormix.swf
    done
