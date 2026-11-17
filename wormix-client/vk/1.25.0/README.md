# Wormix 1.25.0 client

- Source: github.com/lightcool697/wormix.github.io, `vk/1.25.0`, patched (see `client-patches/`).
- It loads maps and sounds from `../storage/` (`wormix-client/vk/storage`, taken from the 1.05.0 client;
  the maps 80–114 from the original game's CDN, see `client-patches/README.md`).
  Missing for now: `music_theme_1.swf` and the zombie voice.
- wormix-core recognizes the client by its Login signature (salt `Uzg4IbZnpskiY73hqnjkPsnRY03H`) and answers
  in the 1.25.0 format (services/core/internal/proto/v125.go).
