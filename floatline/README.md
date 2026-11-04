# Floatline

A 1v1 asymmetric tower-defense game in 3D. It's a single HTML file, so there's nothing to install. Open `index.html` in a browser (Three.js loads from a CDN).

- **Balloon player (keyboard):** spend stat points on Health, Speed, Regrow, Jump and Camo. Move with WASD (you're locked to the track but can weave side to side), jump with Space, dash with Shift. Reach the gate and every layer you have left is taken off its HP. You get +1 stat point after every run.
- **Defender (mouse):** build Dart Posts, Tack Rings, Spike Piles (placed on the track), Mortars and Frost Spires. Click a tower to upgrade or sell it. Survive the 4-minute clock to win.

Maps: Clover Meadow, Ridgeback Canyon, Lantern Lake. Press C for the chase camera and P to pause.

## Windows .exe

`desktop/` packages the game with Electron. Running `npm install && npm run dist` there produces `desktop/dist/Floatline.exe`, a portable exe that runs offline because three.js is bundled. F11 toggles fullscreen.
