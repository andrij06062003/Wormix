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
maps after id 79 out of the list it hands out. Compile it like the `no-vk-125/` patches:

    FFDEC=ffdec-cli.jar python3 client-patches/no-vk-125/fiximports.py src125/scripts \
        client-patches/maps-125/MapList.as Wormix.swf Wormix.new.swf 'config.§##r§' && mv Wormix.new.swf Wormix.swf

## `crosshair-125/`: the aiming arrow stuck on the map and the mouse cursor hidden, in the 1.25.0 client

`PointCrosshair.as` is
`§!!=§.§'2§`: its ENTER_FRAME handler (`§#"l§`) puts the crosshair back after deactivation unless it stops
when the crosshair is off (`§8"F§`), and `deactivate()` now removes the listeners and the crosshair (and
calls `Mouse.show()`) even when the turn is already over. Hiding the crosshair removes it from its parent,
not from the current scene, which threw when the arrow wasn't there. Compile it like the `no-vk-125/` patches:

    FFDEC=ffdec-cli.jar python3 client-patches/no-vk-125/fiximports.py src125/scripts \
        client-patches/crosshair-125/PointCrosshair.as Wormix.swf Wormix.new.swf "§!!=§.§'2§" && mv Wormix.new.swf Wormix.swf

## `hint-125/`: the weapon description stays on screen after closing the arsenal

The description (hint) of a weapon or item is added to the scene, not inside the cell it belongs to, and
is removed when the cell sends ROLL_OUT. Choosing a weapon closes the arsenal under the mouse; Ruffle sends
no ROLL_OUT then, so the description stayed. `Hintable.as` (`§7L§.§#$N§`) also listens for
REMOVED_FROM_STAGE on the cell and then removes the hint (and cancels one still waiting to appear).
Compile it like the `no-vk-125/` patches:

    FFDEC=ffdec-cli.jar python3 client-patches/no-vk-125/fiximports.py src125/scripts \
        client-patches/hint-125/Hintable.as Wormix.swf Wormix.new.swf '§7L§.§#$N§' && mv Wormix.new.swf Wormix.swf
