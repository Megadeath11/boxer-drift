# Element Sandbox

A 2D falling-sand physics and chemistry sandbox. Spawn 44 real-world materials, then heat, burn, freeze, blow up and blow around whatever you build.

It ships as **one portable `ElementSandbox.exe`**: no installer and no DLLs. Put it anywhere, even on a USB stick, and double-click it.

## Materials

| Group | Materials |
|---|---|
| Solids | Wall, Stone, Wood, Iron, Glass, Ice, Plant, TNT, Dry Ice |
| Powders | Sand, Dirt, Gravel, Salt, Snow, Gunpowder, Coal, Sawdust, Thermite, Ash, Rust |
| Liquids | Water, Salt Water, Oil, Gasoline, Diesel, Ethanol, Lava, Acid, Mercury, Liquid N2, Molten Glass, Molten Iron, Mud, Nitroglycerin |
| Gases | Steam, Smoke, Hydrogen, Oxygen, Methane, Propane, CO2, Nitrogen, Chlorine, Fuel Vapor |
| Tools | Erase, Fire, Heat, Cool, Wind, Bomb |

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
| Mouse wheel, `[` `]` | Brush size |
| `←` `→` / `↓` | Global wind / calm |
| `Space` / `N` | Pause / single step |
| `T` | Heat (temperature) view |
| `V` | Show wind / air flow |
| `O` | Open or close the sky (whether gases escape at the top) |
| `C` | Clear |
| `F11` | Fullscreen |
| `H` / `F1` | Help |

## Building

You need [Go](https://go.dev/dl/) 1.22 or newer. You can cross-compile the Windows exe from Windows, macOS or Linux, and no C compiler is needed.

```sh
./build.sh        # macOS / Linux  -> dist/ElementSandbox.exe
build.bat         # Windows        -> dist\ElementSandbox.exe
go run .          # run directly (on Linux this needs X11/GL dev packages)
go test ./sim     # headless physics tests
```

GitHub Actions builds the exe on every push that changes this folder; download it from the run's artifacts. Pushing a tag like `sandbox-v1.0.0` publishes it to **Releases**.

Windows SmartScreen may warn you because the exe is unsigned. Click *More info → Run anyway*.
