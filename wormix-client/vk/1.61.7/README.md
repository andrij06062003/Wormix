# Wormix 1.61.7 client (being ported)

- Source: github.com/arbitron-adm2/arena-assets, `swf/1.61.7` (the original game's files: `Wormix.swf`
  and `data/`). `Wormix.swf` has three patches for running without the preloader (`client-patches/standalone-161`).
- The default client since 2026-11-17 (`play.html`); `play.html?125` runs the 1.25.0 one, which still has PvP
  and co-op. The page loads `Wormix.swf` without its preloader (`preloader_251.swf`, built for
  VK and the original servers) and starts it with the game's own "none" social network
  (`_social_type=none`; its files come from `data/social/vkontakte`, the page rewrites `social/none`): no VK, the player's id and auth key come from the flashvars `user_id` and
  `_masterKey`.
- The client has the original game's servers built in (main `my.rmart.ru:6020`, PvP `my.rmart.ru:6002`, files
  from `tesla.rmart.ru/engine/`); the page's Ruffle `socketProxy` and `urlRewriteRules` send them to ours.
- Maps and sounds come from `../storage/` (`wormix-client/vk/storage`): all 152 maps of the original game's
  CDN, `sounds_1.53.swf` and `music_theme_1.swf`.
- wormix-core recognizes the client by its Login signature (salt `s0PaWHUokZZMlhtKVyj3S1n5BruorNmk`) and
  answers in the 1.61.7 format where it differs from 1.25.0 (services/core/internal/proto/v161.go).
