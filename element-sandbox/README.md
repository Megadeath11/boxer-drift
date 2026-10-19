# Element Sandbox

A 2D falling-sand physics and chemistry sandbox with 66 real-world materials. You can heat, burn, freeze, blow up and blow around whatever you build. It also has a **power puzzle mode**: make electricity from steam, heat or wind, and run it through metal wire to light every beacon.

It ships as **one portable `ElementSandbox.exe`**: no installer and no DLLs. Put it anywhere, even on a USB stick, and double-click it.

## Materials

| Group | Materials |
|---|---|
| Solids | Wall, Stone, Wood, Glass, Ice, Plant, TNT, Dry Ice |
| Metals | Iron, Copper, Aluminum, Lead, Tin, Titanium, Silver, Gold, Platinum, and a molten form of each |
| Power | Turbine, Thermo Gen, Battery, Lamp, Heater, Beacon |
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

## Levels

Pick a level from the main menu. Each one gives you a few materials in limited amounts, shown as the numbers on the palette buttons. Erasing your own material refunds it. To win, keep every beacon powered for 3 seconds. Your progress is saved in `ElementSandbox-progress.txt` next to the exe.

1. **First Spark**: light coal under a pot of water. The steam spins the turbine lid, and you bridge the gap in the wire.
2. **Long Haul**: a boiler heated by a volcanic vent. Run a limited amount of copper to two beacons.
3. **Hot and Cold**: no water. Put thermoelectric generators between the lava and ice or liquid nitrogen.
4. **Gold Standard**: the wire has to pass through acid. Copper dissolves; gold doesn't.
5. **Wind Farm**: strong wind but sheltered valleys. Put a limited number of turbines up high.

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
| Mouse wheel, `[` `]` | Brush size (wheel over the palette scrolls it) |
| `←` `→` / `↓` | Global wind / calm |
| `Space` / `N` | Pause / single step |
| `T` | Heat (temperature) view |
| `V` | Show wind / air flow |
| `O` | Open or close the sky (whether gases escape at the top) |
| `C` | Clear (sandbox) |
| `R` | Restart level |
| `Esc` | Main menu |
| `F11` | Fullscreen |
| `H` / `F1` | Help |

## Building

You need [Go](https://go.dev/dl/) 1.22 or newer. You can cross-compile the Windows exe from Windows, macOS or Linux, and no C compiler is needed.

```sh
./build.sh        # macOS / Linux  -> dist/ElementSandbox.exe
build.bat         # Windows        -> dist\ElementSandbox.exe
go run .          # run directly (on Linux this needs X11/GL dev packages)
go test ./...     # headless physics tests + proof that every level is winnable
```

The exe's icon comes from `rsrc_windows_amd64.syso`, which Go links in automatically. To regenerate it from `assets/icon.ico`, run `go run github.com/akavel/rsrc@v0.10.2 -ico assets/icon.ico -arch amd64 -o rsrc_windows_amd64.syso`.

GitHub Actions builds the exe on every push that changes this folder; download it from the run's artifacts. Pushing a tag like `sandbox-v1.0.0` publishes it to **Releases**.

Windows SmartScreen may warn you because the exe is unsigned. Click *More info → Run anyway*.
