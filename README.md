# Bolt Barrage

An original turn-based artillery game with tiny robots, playable in the browser.
Two teams of three robots take turns on destructible terrain. The last team standing wins.

## Play

Open `index.html` in any modern browser. There's no build step and nothing to install.

Or serve it locally:

```bash
python3 -m http.server 8000
```

Then open http://localhost:8000.

## Modes

- **Vs CPU:** you control the Blue Bolts and the CPU controls the Red Rivets.
- **Two players (hot-seat):** both teams are played on the same keyboard.

## Controls

| Key | Action |
| --- | --- |
| ← / → | Move |
| ↑ / ↓ | Aim |
| Enter | Jump |
| Space (hold, then release) | Charge and fire |
| 1 / 2 / 3 | Rocket / Grenade / Air Strike |
| Mouse click | Pick the air-strike target |
| R / M | Rematch / main menu (after a game) |

## Rules

- Each turn lasts 30 seconds. After you fire, you get 3 seconds to retreat.
- Wind changes every turn. It pushes rockets and air-strike bombs, but not grenades.
- Explosions dig craters, do damage and knock robots back.
- Long falls hurt. Falling into the water knocks a robot out instantly.
- A robot at 0 HP blows up when everything settles, which can damage robots nearby.

## Weapons

- **Rocket:** explodes on impact and drifts with the wind.
- **Grenade:** bounces around and goes off after 3 seconds.
- **Air Strike:** five bombs fall on the spot you click. Each team gets two per game.

## Code layout

- `index.html`: the page and canvas
- `src/game.js`: all of the game (terrain, physics, weapons, CPU, rendering, input)
