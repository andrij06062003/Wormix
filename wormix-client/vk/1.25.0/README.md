# Wormix 1.25.0 client (work in progress)

Started with `play.html?v=125`; 1.05.0 stays the default.

- Source: github.com/lightcool697/wormix.github.io, `vk/1.25.0`, unchanged.
- It loads maps and sounds from `../storage/`, which it doesn't ship with. play.html sends
  those requests to the 1.05.0 files (`vk/1.05.0/data/levels`, `vk/1.05.0/data/common/sound`).
  Missing for now: `music_theme_1.swf`, the zombie voice and maps that 1.05.0 doesn't have.
- wormix-core recognizes the client by its Login signature (salt `Uzg4IbZnpskiY73hqnjkPsnRY03H`) and answers
  in the 1.25.0 format (services/core/internal/proto/v125.go).
