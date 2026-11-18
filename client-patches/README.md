# Client patches

Fixes compiled into `wormix-client/vk/1.25.0/Wormix.swf`. The patches of the old 1.05.0 client were
removed with that client; they are in the git history (before the "Remove the 1.05.0 client" commit).

After changing `Wormix.swf`, change `CLIENT_BUILD` in `wwwroot/play.html`. Cloudflare tells browsers to
keep .swf files for 4 hours; the build number makes them fetch the new file at once.

## `no-vk-125/`: no VK posts, notifications or share buttons in the 1.25.0 client

No VK posts or notifications, and no share buttons. The client is obfuscated, so the
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

## Maps: no patch any more

From 2026-11-08 to 2026-11-17 `maps-125/MapList.as` (`config.§##r§`, the map list) left the maps after id 79
out, and later gave the bosses one of our maps instead of theirs, because we only had the 1.05.0 map files.
Since 2026-11-17 `vk/storage/levels` has all the maps of the 1.25.0 client (ids 80–114 downloaded from the
original game's CDN, `https://tesla.rmart.ru/engine/storage/levels/<name>/map<version>.swf`; Mines2 and Castles2
are its `mines2_fix` and `castles2_fix`). `config.§##r§` in `Wormix.swf` is again the original class (recompiled
from its decompiled source), so build from `Wormix.orig.swf` without it.

## `crosshair-125/`: the aiming arrow stuck on the map and the mouse cursor hidden, in the 1.25.0 client

In `Wormix.swf` again since 2026-11-08 (build 2026-11-08.7). The first build (2026-11-08)
broke the air strike, atom bomb and girder: FFDec compiled `§%#6§.§5!W§.activate()` as a public name, but
`activate`/`deactivate` there are IControlSet interface methods (namespace
`ru.pragmatix.wormix.controls:IControlSet`), so the call threw. The patch now imports `§6k§.§%#6§` and names
the interface's namespace (`ics::activate()`). Check the P-code of a rebuilt class
(`-format script:pcode -export script`) against the original for such names.

`PointCrosshair.as` is
`§!!=§.§'2§`: its ENTER_FRAME handler (`§#"l§`) puts the crosshair back after deactivation unless it stops
when the crosshair is off (`§8"F§`), and `deactivate()` now removes the listeners and the crosshair (and
calls `Mouse.show()`) even when the turn is already over. Hiding the crosshair removes it from its parent,
not from the current scene, which threw when the arrow wasn't there. Compile it like the `no-vk-125/` patches:

    FFDEC=ffdec-cli.jar python3 client-patches/no-vk-125/fiximports.py src125/scripts \
        client-patches/crosshair-125/PointCrosshair.as Wormix.swf Wormix.new.swf "§!!=§.§'2§" && mv Wormix.new.swf Wormix.swf

## `standalone-125/`: the last VK buttons, the full screen button and the chat, in the 1.25.0 client

In `Wormix.swf` since build 2026-11-09.1. Each patch calls a function of the game's page (`wwwroot/play.html`)
through `ExternalInterface`:

- `UserProfileHelper.as` (`ru.pragmatix.wormix.model.user.§7X§`): "Профиль игрока" in a friend's menu
  (`§#!#§`) opened the player's VK page; it calls `wormixProfile(id)`, and the page shows the player's profile
  with "write a message".
- `ReturnFriendsDialog.as` (`§9"`§.§?#G§`, "Верни бойцов", VK invitations to friends who left): its button, the
  bird with a letter next to the friends list, opens the chat (`show()` calls `wormixChat()`).
- `FriendsList.as` (`§9"`§.§;!t§`): "+" in an empty cell invited VK friends; it calls `wormixInvite()`, and the
  page copies the game's address and the player's id. The list also hides the main menu's VK group button
  (`groupBtnContainer`, "В") and moves the news button into its place: the main menu (`ru.pragmatix.wormix.§@#8§`)
  can't be recompiled (FFDec stops on its `super.activate()`, a method of an obfuscated namespace), and it
  builds this list right after the button.
- `ScreenController.as` (`ru.pragmatix.wormix.controller.§"$-§`): the full screen button (`§+!h§`) called
  Ruffle's full screen, which Ruffle refuses without `allowFullscreen`, and the game's full screen layout is for the
  screen's size while the page keeps the game at 900x600 (`forceScale`), which cut off the buttons on the right.
  It calls `wormixFullscreen()`, and the page puts the game's box in full screen.

`ReturnFriendsDialog.as` and `ScreenController.as` name the `IUIComponent` namespace (`iuc::`) for the calls
FFDec would otherwise compile as public names (see `crosshair-125/`). The texts of the bird's tip and of the menu
item are in `data/lang/messages_ru.xml` (`RETURN_FRIEND_BTN_QTIP`, `USER_PAGE`). Compile them like the
`no-vk-125/` patches:

    for p in 'ru.pragmatix.wormix.model.user.§7X§:UserProfileHelper' '§9"`§.§?#G§:ReturnFriendsDialog' \
             '§9"`§.§;!t§:FriendsList' 'ru.pragmatix.wormix.controller.§"$-§:ScreenController'; do
        FFDEC=ffdec-cli.jar python3 client-patches/no-vk-125/fiximports.py src125/scripts \
            client-patches/standalone-125/${p##*:}.as Wormix.swf Wormix.new.swf "${p%%:*}" && mv Wormix.new.swf Wormix.swf
    done

## `standalone-161/`: the 1.61.7 client without its preloader

In `wormix-client/vk/1.61.7/Wormix.swf` since build 2026-11-17.2, when 1.61.7 became the default client.
The page starts `Wormix.swf` directly (its preloader is built for VK and the original servers), and some
things the preloader or Flash Player gave it were missing. Each file is a whole method in FFDec's P-code
(from `method` to `end ; method`), put in with `-replace` and the method body's index:

- `Application.onAddedToStage.pcode` (`Application`, body 16): the params object built from the flashvars
  has no `clientVersion` (the client stopped on it) and no `showBanner` function (the VK preloader's; the main
  menu calls it in `activate()`, threw "Error #1006: value is not a function", and stayed blank). It adds
  `clientVersion: "1.61.7"` and `showBanner: Application.§&]§`, an empty function of the client.
- `LoginParams.pcode` (`§&%X§.§5F§`, body 7326): the Login's extra strings read `Capabilities` and `System`
  values that Ruffle returns as null; it sends `"-"` for each instead.
- `BuildUpgrades.pcode` (`§8#o§.§1@§`, `§,&G§`, body 5943, since build 2026-11-17.3): the upgraded weapons are
  built by cloning the previous level (`getWeapons()[prevUpgradeId]`), going through `recipes`, an object keyed
  by recipe id. Flash went through it in id order, so a level 1 upgrade was built before its level 2; Ruffle's
  order differs, so a level 2 met a missing base. Build 2026-11-17.2 skipped those (`WeaponUpgrades.pcode`), and
  a player who owned one got a blank blue screen. It now goes through the recipes again until nothing more can
  be built (at most 20 times).
- `WeaponUpgrades.pcode` (`§8#o§.§1@§`, `§1"U§`, body 5944): building an upgrade whose base is missing returns
  null instead of throwing (no longer happens with `BuildUpgrades.pcode`; kept as a guard).
- `Application.log.pcode` (`Application`, `§1%&§`, body 12, since build 2026-11-17.3): the client's own log
  (`Application.loginfo`/error, including the main menu's caught errors) was an empty function; it goes to the
  browser console as `WLOG <level>: <text>` (`ExternalInterface.call("console.log", ...)`).
- `Crosshair.*.pcode` (`§9%W§.§8#p§`, the aiming arrow, since build 2026-11-18.1), the fix of `crosshair-125/`
  for 1.61.7: `hide.pcode` (`§9!f§`, body 21240) takes the arrow off its parent, if it has one, instead of the
  scene; `enterFrame.pcode` (`§>W§`, body 21245) unhooks itself and hides the arrow when the crosshair is no longer
  active; `deactivate.pcode` (body 21246) removes the listeners and the arrow, and shows the mouse, even when the
  turn is already over (only the control set's `deactivate()` still waits for the player's turn). Before, the arrow
  stayed on the map and the cursor stayed hidden when the turn ended while aiming.
- `ScreenController.fullscreen.pcode` (`ru.pragmatix.wormix.controller.§+$w§`, `§5$q§`, body 2227, since build
  2026-11-18.1): the full screen button calls `wormixFullscreen()` of the page, as in `standalone-125/`.

    cp Wormix.orig.swf a.swf
    java -jar ffdec-cli.jar -replace a.swf b.swf '§8#o§.§1@§' client-patches/standalone-161/WeaponUpgrades.pcode 5944
    java -jar ffdec-cli.jar -replace b.swf c.swf Application client-patches/standalone-161/Application.onAddedToStage.pcode 16
    java -jar ffdec-cli.jar -replace c.swf d.swf '§&%X§.§5F§' client-patches/standalone-161/LoginParams.pcode 7326
    java -jar ffdec-cli.jar -replace d.swf e.swf '§8#o§.§1@§' client-patches/standalone-161/BuildUpgrades.pcode 5943
    java -jar ffdec-cli.jar -replace e.swf f.swf Application client-patches/standalone-161/Application.log.pcode 12
    java -jar ffdec-cli.jar -replace f.swf g.swf '§9%W§.§8#p§' client-patches/standalone-161/Crosshair.hide.pcode 21240
    java -jar ffdec-cli.jar -replace g.swf h.swf '§9%W§.§8#p§' client-patches/standalone-161/Crosshair.enterFrame.pcode 21245
    java -jar ffdec-cli.jar -replace h.swf i.swf '§9%W§.§8#p§' client-patches/standalone-161/Crosshair.deactivate.pcode 21246
    java -jar ffdec-cli.jar -replace i.swf Wormix.swf 'ru.pragmatix.wormix.controller.§+$w§' \
        client-patches/standalone-161/ScreenController.fullscreen.pcode 2227

`Wormix.orig.swf` is the unpatched 1.61.7 file (commit 3d7018b).
