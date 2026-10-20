# Element Sandbox

A 2D falling-sand physics and chemistry sandbox with 70 real-world materials. You can heat, burn, freeze, blow up and blow around whatever you build. The world has **seasons, weather and a roof of sky** that steam and smoke gather under. Every fire adds CO2, and **trees** are how you clean the air. It also has a **power puzzle mode**: make electricity from steam, heat or wind, and run it through metal wire to light every beacon without choking the valley.

It ships as **one portable `ElementSandbox.exe`**: no installer and no DLLs. Put it anywhere, even on a USB stick, and double-click it.

## Materials

| Group | Materials |
|---|---|
| Solids | Wall, Stone, Wood, Glass, Ice, Plant, TNT, Dry Ice |
| Metals | Iron, Copper, Aluminum, Lead, Tin, Titanium, Silver, Gold, Platinum, and a molten form of each |
| Power | Turbine, Thermo Gen, Battery, Lamp, Heater, Beacon |
| Nature | Tree Seed, Tree Trunk, Leaves, Fallen Leaves |
| Powders | Sand, Dirt, Gravel, Salt, Snow, Gunpowder, Coal, Sawdust, Thermite, Ash, Rust |
| Liquids | Water, Salt Water, Oil, Gasoline, Diesel, Ethanol, Lava, Acid, Mercury, Liquid N2, Molten Glass, Mud, Nitroglycerin |
| Gases | Steam, Smoke, Hydrogen, Oxygen, Methane, Propane, CO2, Nitrogen, Chlorine, Fuel Vapor |
| Tools | Erase, Fire, Heat, Cool, Wind, Bomb |

## Graphics

- The world is 540 × 300 simulated pixels, each drawn as a 2 × 2 block.
- Flames, red-hot metal, lava, lamps and beacons give off a soft **glow**: a blurred light map added on top of the scene.
- Solids get lit top edges and shadowed undersides, liquids have bright surfaces, and water shimmers.
- The main menu runs a live scene behind its buttons.
- Rendering, glow and heat conduction run in parallel on all CPU cores.

## Seasons, weather and the roof

- **Seasons**: a sandbox year lasts 4 minutes and cycles through spring, summer, autumn and winter.
  - The outdoor temperature runs from about −6 °C in midwinter to 26 °C in midsummer, and the air drifts toward it.
  - Ponds freeze in winter and snow melts in spring.
  - The sky colour changes with the season.
- **Weather**: it rains in spring and autumn and snows in winter.
- **Roof**: a cold roof caps the sky.
  - Hot gases rise and gather under it instead of escaping.
  - Steam condenses on its cold underside and falls back as rain.
  - Smoke eventually settles out of the air as ash.
  - Press `O` to remove the roof and let gases escape into space.

## Trees and pollution

- **Trees**: plant **Tree Seeds** on dirt or mud. In the warm seasons they grow a trunk, side branches and a crown of leaves.
  - In autumn the leaves turn red, orange and yellow, then drop. The trees are bare in winter and grow new leaves in spring.
  - Fallen leaves blow around, burn easily and rot into soil.
- **Pollution**: burning coal, wood, oil, fuel or anything else carbon-based releases **CO2**, and burning gases do too.
  - The climate panel shows the concentration in **ppm**: green when clean, yellow when polluted, red when toxic.
  - CO2 traps heat. Each 150 ppm above the pre-industrial 280 ppm warms the world by about 1 °C.
  - Above 1200 ppm, leaves start to wilt and fall.
- **Photosynthesis**: leaves take CO2 out of the air and give off oxygen. It's fastest in spring and summer, slow in autumn and stops in winter, so a forest has to be big enough to keep up with your fires.

## Camera

- **Zoom**: the mouse wheel zooms toward the cursor, up to 8×. `Z` and `X` also zoom.
- **Pan**: `W` `A` `S` `D` or drag with the middle mouse button. When zoomed in, a minimap shows where you are.
- **Reset**: `Home` or `0` zooms back out to the whole map.

## Campaign

The place to start. You run the power company for a small valley town, in one world that carries on from stage to stage. Stage 1 is a step-by-step tutorial with a checklist.

- **Money**: each stage hands you a budget. The shop (the right-hand sidebar) sells ready-made **kits**, and materials by the pixel:
  - **Wind Turbine**: a mast with a rotor. Hilltops and open ground catch the most wind; turbines downwind of the hill or of each other get less. Press `V` to see the air flow.
  - **Steam Boiler**: a pot of water on legs with a turbine lid, under a hood so only steam turns it. Pile **Coal** in the firebox and light it with **Fire**.
  - **Battery Bank**: stores spare power for when the wind drops.
  - **Copper Wire**, **Coal**, **Tree Seeds** and **Dirt** cost money per pixel.
- **Placing kits**: pick one and click. A ghost shows where it will land (red means it doesn't fit, it isn't standing on the ground, or you can't afford it).
- **Demolish** (or right-click) pulls down your own buildings and wire for half their price back.
- **The town grows**: each stage adds houses or changes the weather, and everything you built stays. When the wind can't keep up you can keep buying turbines until the money runs out, or demolish and rebuild with steam.

| Stage | What changes |
|---|---|
| 1. First Light | Tutorial: build a turbine on the hill and wire it to the first house. |
| 2. Growing Town | Four houses need 48 power. Wind alone won't stretch that far on your budget: steam will. |
| 3. Still Nights | Two more houses (72 power) and gusty wind with calm spells. Keep the lights on for 20 s: more coal alone won't bridge the calm, so you need batteries. |
| 4. Clean Air | Smog: CO2 must come down to 450 ppm. Plant about ten trees, well away from your fires (a forest fire is all CO2). |
| 5. Hot Springs | An earthquake opens a volcanic vent and two more houses appear (96 power). A boiler over the vent boils for free. Insurance pays half the value of what the quake destroys. |

Each stage leaves you a little less room than the one before. Where new houses go up, the council pays you in full for anything of yours that was in the way.

The campaign saves itself next to the exe (`ElementSandbox-campaign.sav`) when you leave it, when you close the game, and at the start of each stage. `R` twice restarts the current stage.

## Levels

Pick a level from the main menu. Each one gives you a few materials in limited amounts, shown as the numbers on the palette buttons. Erasing your own material refunds it. To win, keep every beacon powered for 3 seconds (the later levels ask for longer, so you have to ride out what they throw at you). Your progress is saved in `ElementSandbox-progress.txt` next to the exe.

1. **First Spark**: light coal under a pot of water. The steam spins the turbine lid, and you bridge the gap in the wire.
2. **Long Haul**: a boiler heated by a volcanic vent. Run a limited amount of copper to two beacons.
3. **Hot and Cold**: no water. Put thermoelectric generators between the lava and ice or liquid nitrogen.
4. **Gold Standard**: the wire has to pass through acid. Copper dissolves; gold doesn't.
5. **Wind Farm**: strong wind but sheltered valleys. Put a limited number of turbines up high.
6. **Clean Air**: the power is on, but the valley is full of smog. Plant a forest to bring CO2 under 500 ppm.
7. **Coal Country**: only coal can boil this pot. Keep the beacon lit and the air under 500 ppm, so plant trees before you light the fire.
8. **Second Wind**: the wind blows in gusts with dead calm in between. Charge a battery bank while it blows so three beacons stay lit for 20 seconds.
9. **Deep Freeze**: midwinter. The boiler is frozen solid and its firebox is full of snow. Coal alone struggles in the cold, so put the cold to work too: a Thermo Gen between the fire and a snowdrift.
10. **Flood Gate**: a dry boiler on a volcanic vent and a reservoir behind a stone dam. No eraser: blow the dam with the Bomb so the water runs down the chute into the pot.
11. **Eruption**: the beacon sits in a pit at the foot of an erupting volcano. Lava melts copper, so the wire into the pit has to be platinum or titanium. Hold for 25 seconds.
12. **The Grid**: four beacons, a boiler that can't run them all, an acid trench, a windy hill and gusts. Everything you've learned at once.

## Electricity

- **Generators**:
  - **Turbines** spin when steam passes through them, and the steam condenses back into water, so a sealed boiler runs as a closed steam cycle. Turbines also spin in wind and in rising hot air.
  - **Thermo Gens** use the Seebeck effect: they make power from the temperature difference across them.
- **Circuits**: any touching conductive material forms one circuit. That includes every metal, molten metal, mercury and salt water.
  - Generators share their output with the consumers on the same circuit: Lamps, Heaters and Beacons.
  - **Batteries** store spare power and release it when the generators fall short.
- **Resistance**: high-resistance conductors heat up when they carry power. Iron, lead, titanium and especially salt water warm up; copper, silver and gold barely do.
- **Metals have their real properties**:
  - Melting points: tin 232 °C, lead 327 °C, aluminum 660 °C, silver 962 °C, gold 1064 °C, copper 1085 °C, iron 1538 °C, titanium 1668 °C, platinum 1768 °C.
  - Densities: platinum and gold sink through everything.
  - Corrosion: gold, platinum and titanium are acid-proof. Acid makes iron, aluminum, tin and lead fizz out hydrogen.

## Physics

- **Density**: everything floats or sinks by real density. Oil floats on water, mercury sinks through everything, sawdust floats, and sand sinks.
- **Heat**: every cell has a temperature, and heat conducts between neighbours at each material's own rate (iron is fast, wood and air are slow). Hot things glow. Press `T` for a thermal camera view.
- **Thermal radiation**: anything glowing above about 350 °C beams heat at whatever it can see, following the T⁴ Stefan–Boltzmann law. That's how a fire heats a pot across an air gap. Heat conduction runs twice per frame, so heating times stay realistic at the finer resolution.
- **Phase changes at real temperatures**: water freezes at 0 °C and boils at 100 °C; salt water freezes at −21 °C; steam condenses below 100 °C and falls back as rain; lava sets into stone below 700 °C and stone melts at about 1200 °C; sand melts into glass at about 1700 °C; iron melts at 1538 °C; liquid nitrogen boils at −196 °C; dry ice sublimates at −78 °C.
- **Combustion**: each fuel has its own flash point, auto-ignition temperature, flame temperature and burn time, and needs oxygen.
  - Gasoline has a flash point of −43 °C, so it lights instantly and gives off heavy fuel vapor.
  - Diesel has a flash point of 52 °C, so it is hard to light cold, and it burns slowly with a lot of soot.
  - Hydrogen, methane and propane burn with a blue flame and explode in air. Hydrogen burns into steam.
  - Fire burns hotter in pure oxygen. CO2 and water smother it.
  - Thermite needs about 900 °C to light, burns at 2700 °C, and leaves molten iron.
- **Explosions**: gunpowder, TNT and nitroglycerin set each other off in chain reactions. Nitroglycerin also goes off on impact. Blasts break solids into debris, throw particles and send a pressure wave through the air.
- **Wind and air flow**: an incompressible air-flow solver runs underneath the particles.
  - Hot air rises, so fires make their own updrafts and cold dry-ice fog sinks.
  - Air flows around walls.
  - Gases, flames, smoke, snow, ash and sawdust are carried by the wind.
  - Set a steady wind with the arrow keys, or drag the Wind tool to make gusts. Press `V` to see the flow.
- **Gas buoyancy**: depends on molar mass and temperature. Hydrogen and methane rise, propane, CO2 and chlorine pool on the floor, and hot gas rises while cold gas sinks.
- **Chemistry**:
  - Salt dissolves into brine and melts ice. Boiling brine leaves its salt behind.
  - Acid eats most things but not glass, and makes metal fizz out hydrogen.
  - Iron rusts in water, and faster in salt water.
  - Wet dirt turns to mud.
  - Plants grow by drinking water and turn CO2 into oxygen. Chlorine kills them.

## Controls

| Input | Action |
|---|---|
| Left mouse | Paint the selected material or use the selected tool |
| Right mouse | Erase |
| Shift + paint | Replace existing material |
| Mouse wheel | Zoom in / out (over the palette it scrolls) |
| `[` `]`, Shift + wheel | Brush size |
| `W` `A` `S` `D`, middle-drag | Move the camera |
| `Z` / `X`, `Home` | Zoom in / out, show the whole map |
| `←` `→` / `↓` | Global wind / calm |
| `Space` / `N` | Pause / single step |
| `T` | Heat (temperature) view |
| `V` | Show wind / air flow |
| `O` | Roof on / off |
| `K` / `L` | Next season / lock the season |
| `J` / `P` | Seasons on / off, weather on / off |
| `C` twice | Clear everything (sandbox) |
| `R` | Restart level |
| `Ctrl+Z` | Undo your last action: a stroke, a kit, a bomb, a clear (up to 15 steps; the world rewinds to just before it) |
| `Esc` | Main menu (Esc again, or Resume, goes back to your game) |
| `F11` | Fullscreen |
| `H` / `F1` | Help |

## Building

You need [Go](https://go.dev/dl/) 1.22 or newer. You can cross-compile the Windows exe from Windows, macOS or Linux, and no C compiler is needed.

```sh
./build.sh        # macOS / Linux  -> dist/ElementSandbox.exe
build.bat         # Windows        -> dist\ElementSandbox.exe
go run .          # run directly (on Linux this needs X11/GL dev packages)
go test ./...     # headless physics tests + proof that every level and the campaign are winnable
BALANCE=1 go test ./campaign/ -run Balance -v -timeout 60m   # campaign balance report
```

The exe's icon comes from `rsrc_windows_amd64.syso`, which Go links in automatically. To regenerate it from `assets/icon.ico`, run `go run github.com/akavel/rsrc@v0.10.2 -ico assets/icon.ico -arch amd64 -o rsrc_windows_amd64.syso`.

GitHub Actions builds the exe on every push that changes this folder; download it from the run's artifacts. Pushing a tag like `sandbox-v1.0.0` publishes it to **Releases**.

Windows SmartScreen may warn you because the exe is unsigned. Click *More info → Run anyway*.
