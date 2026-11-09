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

## `maps-125/`: only the maps we have, in the 1.25.0 client

We only have the 1.05.0 map files (`wormix-client/vk/storage/levels`). The 1.25.0 client
knows 34 newer maps (ids 80–114), and battles, missions and bosses pick a random map from its list,
so they could stop with "error loading map". `MapList.as` (`config.§##r§`, the map list) leaves the
maps after id 79 out of the list it hands out. The bosses still ask for some of those maps by id ("error loading map (id = 96)"),
so `§5"W§` (get a map by id) hands out one of our maps instead (`STAND_IN`: one with as many worm places as the
boss level uses; all our maps and the boss maps are 2800x1400). The real files can't be found (not in the
1.25.0 repo or other public copies); once one is added as `vk/storage/levels/<name>/map.swf`, take its id
out of `STAND_IN`. Compile it like the `no-vk-125/` patches:

    FFDEC=ffdec-cli.jar python3 client-patches/no-vk-125/fiximports.py src125/scripts \
        client-patches/maps-125/MapList.as Wormix.swf Wormix.new.swf 'config.§##r§' && mv Wormix.new.swf Wormix.swf

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
