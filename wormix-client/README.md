# Wormix Client Files

The Wormix 1.25.0 Flash client files (Pragmatix's), served by the relay (mounted read-only).

    wormix-client/
      └── vk/
          ├── 1.25.0/            the client: preloader.swf, Wormix.swf (patched, see client-patches/), data/
          └── storage/           maps and sounds, which the client loads from ../storage/
              ├── levels/<map>/map.swf, preview.png
              └── sound/sounds.swf, units/sound_*.swf

The maps and sounds come from the 1.05.0 client archive (`data/levels`, `data/common/sound`): the 1.25.0
client doesn't ship with them. The client asks for `map<N>.swf`; play.html sends that to `map.swf`.
The files in the root are from the client archive's `public/` folder and aren't used by play.html.
