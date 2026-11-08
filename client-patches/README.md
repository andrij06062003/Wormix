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
- Data: `ConfigsConverter.ARTIFACTS` is generated from `services/api/internal/assets/game/artifacts.json`
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

## `no-vk/`: no VK posts, notes or notifications

The game was made for VK: after a battle it offered "Create note" (a VK note) and "Share" (a VK status),
and buying a hat, an expensive weapon, an award, a new race etc. asked "Share on the wall?". None of that
exists here; the note waited on "Збереження..." for an answer that never came.

- `PostController`: the wall poster, activity poster and notifier taken from the VK controller are set
  to null, so every share/notify prompt and post is skipped (the code already checks each for null).
  `askAndNotify` now calls its callback when there is nothing to ask, so the level-up dialog after a
  won friend battle still appears.
- `AbstractGameOverDialog`: the "Create note" and "Share" buttons are hidden; only OK is left, centred
  (as in hot seat).

The API also refuses every VK method it doesn't implement with VK's "denied by user" error (10007)
instead of an empty answer, so anything left (photo studio upload, video upload) gives up at once.

Rebuild on top of the patches above:

    for c in social.common.post.PostController gui.window.gameover.AbstractGameOverDialog; do
        java -jar ffdec-cli.jar -replace Wormix.swf Wormix.new.swf ru.pragmatix.wormix.$c \
            client-patches/no-vk/${c##*.}.as && mv Wormix.new.swf Wormix.swf
    done

## `pvp-2v2/`: 2v2 wager battles (four real players in one battle)

Arena > PvP: pressing a wager cell's button asks for the mode, "1 на 1" (OK) or "2 на 2" (the cancel
button; closing the window with X asks for nothing: only a full click on the cancel button counts). For 2v2 the server waits until four players asked
for the same wager, splits them into two teams at random and starts one battle for all four.

The game engine already handled any number of squads and teams (missions use them): the `GameMaster` gives
turns to the squads in list order, drops a squad whose worms are all dead, and the PvP result
determinator waits for the server once a single team is left. Only the PvP set-up knew just two players.

- Protocol (both additions are invisible to 1v1 battles and older clients):
  - `WagerBattleRequest` (2005): a mode byte after `mapId` (1 = 2v2), sent only for 2v2.
  - `WagerBattleRequestResult` (2006): for 2v2, after the reagents, `u8 count` and per player
    `u8 team` (1 or 2) + `UserProfileStructure` (with its u16 length), in turn order. Teams alternate
    (A, B, A, B). `enemyProfile` is the next player in turn order (always an opponent).
- `BattleStartInfo`: `teamProfiles` / `teamIds` (turn order; this player is `Application.userProfile`),
  `isTeamBattle()`.
- `BattleRequestController`: `wagerRequest(wager, team)`; sends the mode, builds `teamProfiles`.
- `PvpGameRules`: a squad per player in turn order with its team id.
- `GameRulesUtil`: places the worms of all four players (per-team spawn areas on maps that have them,
  otherwise the shared positions; positions are reused a bit higher up if a map runs out).
- `BattleStartDataLoader`: loads names/avatars of all four teams.
- `BattleController`: chat messages show who wrote them.
- `GameInterface`: during a teammate's turn the bar says "очікування ходу напарника" (`TEAMMATE_TURN_MSG`)
  instead of the opponent's turn.
- `PvpCell` + `WagerBattleRequest*`: the mode choice and the mode byte. Texts: `PVP_MODE_*` in
  `data/lang/messages_{ru,en}.xml` (with fallbacks in the code).

Server side (services/core/internal/pvp): the battle relays each move to all other players, waits for
every player who still has worms to confirm the turn, and follows the client's turn order (an eliminated
squad is removed and the index kept, as `GameMaster` does). A player who leaves (surrender, timeout) before
all their worms are dead loses the battle for the whole team; one who is already out may leave freely.
Winners and losers are paid per player as in 1v1.

Rebuild on top of the patches above:

    for c in pvp.messages.wager.WagerBattleRequest serialization.wager.WagerBattleRequestBinarySerializer \
             pvp.messages.wager.WagerBattleRequestResult serialization.wager.WagerBattleRequestResultBinarySerializer \
             model.battle.BattleStartInfo game.rules.GameRulesUtil game.rules.PvpGameRules \
             loader.BattleStartDataLoader controller.BattleController pvp.controllers.BattleRequestController \
             gui.menu.arena.pvp.PvpCell gui.GameInterface; do
        java -jar ffdec-cli.jar -replace Wormix.swf Wormix.new.swf ru.pragmatix.wormix.$c \
            client-patches/pvp-2v2/${c##*.}.as && mv Wormix.new.swf Wormix.swf
    done

`pvp-2v2/BattleRequestController.as` already contains the `BattleRequestController.as` fix above.

## `pvp-party/`: teams for 2v2 (play with a friend on your side)

Calling a friend to battle (friends list, avatar menu or "call by id") first asks: **1 на 1** (the old
battle challenge) or **У команду** (invite them into your 2v2 team). The friend gets "X invites you into
their team" and accepts or declines. A team is two players. It lasts until one of you logs out or joins
another team; the other is then told. The Arena 2v2 window shows your teammate.

When you pick 2v2 and your teammate isn't searching yet, they get "your teammate is looking for a 2v2 battle
with wager N, join?". Joining puts them in the queue too. The pair is matched only once both are queued for
the same wager, and always on the same side, against another team or two single players. If the teammate
declines, doesn't answer within 60 s, can't afford the wager or leaves, the search stops with "your
teammate didn't join".

- Protocol (nothing changes for older clients or plain challenges):
  - `BattleRequest` (1002): a mode byte at the end, 1 = team invite.
  - `BattleOffer` (2002): a kind byte at the end: 1 team invite, 2 teammate call (the wager in `seed`),
    3 the teammate left. Answers are the usual `BattleOfferResult`.
  - `WagerBattleRequestError` (2008) code 10: the teammate didn't join.
- `BattleRequestController`: the 1v1/team choice, team invites, `teammate`, `findWager`, error 10 text.
- `BattleOfferController`: the invite/call/left windows.
- `PvpCell`: the 2v2 window names the teammate.
- `BattleRequest*`, `BattleOffer*`: the mode/kind bytes. Texts: `TEAM_*`, `PVP_MODE_TEAMMATE`.

Server side: `services/core/internal/pvp/party.go`.

Rebuild on top of the patches above:

    for c in pvp.messages.client.BattleRequest serialization.client.BattleRequestBinarySerializer \
             pvp.messages.server.BattleOffer serialization.server.BattleOfferBinarySerializer \
             pvp.controllers.BattleRequestController pvp.controllers.BattleOfferController \
             gui.menu.arena.pvp.PvpCell; do
        java -jar ffdec-cli.jar -replace Wormix.swf Wormix.new.swf ru.pragmatix.wormix.$c \
            client-patches/pvp-party/${c##*.}.as && mv Wormix.new.swf Wormix.swf
    done

`pvp-party/BattleRequestController.as` and `pvp-party/PvpCell.as` include the `pvp-2v2/` changes.

## `pvp-party/` fix: the 2v2 and team buttons did nothing

Pressing "2 на 2" (Arena) or the team button (friend challenge) only closed the window: no request reached
the server, so nobody could enter a 2v2 battle or form a team.

Cause: both windows tell that button from X by a click listener on the cancel clip, found with
`box.getChildByName("cancel")`. `MessageBox` moves its buttons into a container of their own
(`putInContainer`), so the clip isn't a direct child of the window and the lookup returned null. No listener
was added, and the window just closed.

Fix: `PvpCell.findCancelClip` searches the whole window; `BattleRequestController` uses it too. The labels
"У команду", "Приєднатися" and "Відмовитись" didn't fit the button and were cut off ("У", "Відмови"), so
they are shorter now: "Разом", "Прийняти", "Відмова".

These changes are in the `pvp-party/` sources, so the rebuild above already contains them.

When testing with changed stats: the client treats a hit above 800 damage as a cheat, drops the player and
keeps a "delayed ban" in the browser's stored Flash data, which closes the main connection on the next logins.

## `missions/BattleRequestController.as`: a refused mission no longer hangs or disconnects

The client counts the battles left itself. After a page reload its battle timer is unset, so it can think a
battle is available when the server has none left. The server refused StartBattle, and the connection dropped
("Гра втратила з'єднання із сервером"). Now the game server answers a refused StartBattle with ArenaLocked (the
arena's delay). This patch waits for either StartBattleResult or ArenaLocked. On ArenaLocked it leaves the
battle loading screen and opens the arena's Missions tab, which shows the "battles used up" timer.

Built from `pvp-party/BattleRequestController.as`, on top of every patch above:

    java -jar ffdec-cli.jar -replace Wormix.swf Wormix.new.swf \
        ru.pragmatix.wormix.pvp.controllers.BattleRequestController client-patches/missions/BattleRequestController.as

## `no-vk-125/`: no VK posts, notifications or share buttons in the 1.25.0 client

The same as `no-vk/`, for `wormix-client/vk/1.25.0/Wormix.swf`. That client is obfuscated, so the
classes have names like `§1Q§.§##5§`; the file names below say what they are.

- `Poster.as` (`§1Q§.§##5§`, the base of the post controller): the activity poster, wall poster and
  notifier taken from the VK controller are set to null, so every post and notification is skipped.
- `PostController.as` (`ru.pragmatix.wormix.social.common.post.§'i§`): "ask and notify" calls its
  callback when there is nothing to ask, so what waits for it (the level-up dialog after a friend battle) goes on.
- The "Share" buttons are hidden: `LevelUpDialog.as` (`§+!U§.LevelUpDialog`), `CraftEquipCompleteDialog.as`
  (`§=#A§.CraftEquipCompleteDialog`), `AchievShare.as` (an achievement, `§%9§.§1!S§`) and `BonusDialog.as`
  (a bonus, `§"!+§.§>#x§`).

FFDec's decompiler leaves out some imports of the obfuscated classes, so compile with `fiximports.py`,
which adds them (the patch files already have them). With the 1.25.0 classes decompiled to `src125/scripts`
(`java -jar ffdec-cli.jar -export script src125 Wormix.orig.swf`):

    cp Wormix.orig.swf Wormix.swf
    for p in '§1Q§.§##5§:Poster' "ru.pragmatix.wormix.social.common.post.§'i§:PostController" \
             '§+!U§.LevelUpDialog:LevelUpDialog' '§=#A§.CraftEquipCompleteDialog:CraftEquipCompleteDialog' \
             '§%9§.§1!S§:AchievShare' '§"!+§.§>#x§:BonusDialog'; do
        FFDEC=ffdec-cli.jar python3 client-patches/no-vk-125/fiximports.py src125/scripts \
            client-patches/no-vk-125/${p##*:}.as Wormix.swf Wormix.new.swf "${p%%:*}" && mv Wormix.new.swf Wormix.swf
    done

`Wormix.orig.swf` is the unpatched file (commit 5fe8ede).

## `maps-125/`: only the maps we have, in the 1.25.0 client

We only have the 1.05.0 map files (`play.html` sends the 1.25.0 map requests to them). The 1.25.0 client
knows 34 newer maps (ids 80–114), and battles, missions and bosses pick a random map from its list,
so they could stop with "error loading map". `MapList.as` (`config.§##r§`, the map list) leaves the
maps after id 79 out of the list it hands out. Compile it like the `no-vk-125/` patches:

    FFDEC=ffdec-cli.jar python3 client-patches/no-vk-125/fiximports.py src125/scripts \
        client-patches/maps-125/MapList.as Wormix.swf Wormix.new.swf 'config.§##r§' && mv Wormix.new.swf Wormix.swf

## `crosshair-125/`: the aiming arrow stuck on the map and the mouse cursor hidden, in the 1.25.0 client

The `PointCrosshair.as` fix above, for `wormix-client/vk/1.25.0/Wormix.swf`. `PointCrosshair.as` is
`§!!=§.§'2§`: its ENTER_FRAME handler (`§#"l§`) puts the crosshair back after deactivation unless it stops
when the crosshair is off (`§8"F§`), and `deactivate()` now removes the listeners and the crosshair (and
calls `Mouse.show()`) even when the turn is already over. Hiding the crosshair removes it from its parent
(as the 1.05.0 patch does), not from the current scene, which threw when the arrow wasn't there. Compile it like the `no-vk-125/` patches:

    FFDEC=ffdec-cli.jar python3 client-patches/no-vk-125/fiximports.py src125/scripts \
        client-patches/crosshair-125/PointCrosshair.as Wormix.swf Wormix.new.swf "§!!=§.§'2§" && mv Wormix.new.swf Wormix.swf
