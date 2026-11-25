# Groundbreak — earthmoving simulator

A realistic digging & construction sim running in the browser (Three.js / WebGL,
no build step). Operate a 20 t hydraulic excavator, a tandem dump truck and a
medium dozer on a deformable job site with a physically based soil model.

## Run as a desktop game

**Windows:** download `Groundbreak-win64.zip`, unzip it anywhere, and double-click
`Groundbreak.exe`. **F11** or **Alt+Enter** toggles fullscreen, **Esc** leaves fullscreen.

To run from source, or to build the desktop app yourself (Node.js 18+):

```bash
cd groundbreak
npm install
npm start                 # play in a desktop window
npm run package:win       # -> dist/Groundbreak-win32-x64/Groundbreak.exe
npm run package:mac       # macOS (run on a Mac)
npm run package:linux
```

## Run in a browser instead

```bash
cd groundbreak
node serve.js          # or: python3 -m http.server 8080
# open http://localhost:8080
```

ES modules need to be served over http (opening `index.html` from disk won't work).
A gamepad is supported; the excavator uses the standard **ISO joystick pattern**.

## The job

Work order on the right of the screen:

1. **Excavate the basement**: 12 × 9 m to 2.5 m below grade (orange paint lines, offset stakes "C2.5").
2. **Haul the spoil** with the dump truck to the fill pad (blue paint lines, stakes "F0.5").
3. **Spread and grade the pad** with the dozer to +0.5 m, ±8 cm.

Press **O** for the grade-control overlay (red = cut, blue = fill, green = on grade). When the
bucket is over the excavation, the grade indicator shows tooth-tip height relative to the design floor,
like a 2D machine-control system.

## Controls

| Excavator | | Dump truck | | Dozer | |
|---|---|---|---|---|---|
| W / S | stick out / in | W | throttle | W / S | forward / reverse |
| A / D | swing left / right | S | brake (hold when stopped: reverse) | A / D | steer |
| I / K | boom down / up | A / D | steer | R / F | blade raise / lower |
| J / L | bucket curl / dump | R / F | raise / lower body | Q / E | blade tilt |
| ↑ ↓ ← → | travel / pivot | Space | park brake | G / T | ripper down / up |
| T | travel speed Hi/Lo | | | Shift | inching |

Global: **1 2 3 / Tab** switch machine · **C** cab view · mouse drag / wheel to look and zoom ·
**O** grade overlay · **M** mute · **H** help · **P** pause.

## How the ground works

The site is a 96 × 96 m grid at 25 cm resolution (147,000 soil columns). Each column stores:

* **bedrock** top. It can't be dug, only ripped with the dozer's ripper.
* **bank** (undisturbed or compacted) soil, layered by depth: topsoil (0.35 m), clay, then gravelly till near rock.
* **loose** soil thickness, with its material type.

Rules, all volume-conserving:

* **Swell.** Excavated bank material bulks by 25 % when it turns loose (typical common earth).
* **Angle of repose.** Loose soil flows between neighbours whenever the slope is steeper than 36°.
  Spoil piles, windrows and trench walls form and slump on their own.
* **Cut faces.** Undisturbed bank can stand up to about 74°. Steeper faces break off and collapse
  into loose material at the toe.
* **Compaction.** Track and tyre pressure (Fn / contact area) compacts loose fill back to bank
  (84 % of loose volume). Grass gets worn into dirt and wheels sink and rut in fresh spoil.
* **Digging energy.** Every cut costs specific cutting energy, J/m³ by material
  (topsoil 120 kJ, clay 260 kJ, till 430 kJ). The excavator's hydraulic power (85 kW) and
  breakout-force limit cap how fast the bucket can move through the ground. Dig too deep and
  the relief valve stalls the function. Bedrock stops the bucket dead. Hard digging pushes back
  on the machine, so a stalled boom-down can lift the tracks.
* **Bucket contents** depend on orientation. Lift the bucket out with the mouth facing sideways
  and it spills, so curl before you lift.
* **Truck body load** is its own small granular grid. It shifts on slopes, heaps above the rails
  (and spills over them), and slides out past the open tailgate once the body is steeper than the
  soil-on-steel friction angle. Payload mass moves the truck's centre of gravity and changes
  how it accelerates, brakes and rides.
* **Dozer blade.** It cuts everything above the cutting edge and carries it as a real pile, which
  spills around the wings and over the top. Resistance is soil-on-soil friction of the carried
  load plus specific cutting resistance. When that exceeds track traction (μ·W), the tracks slip.

Falling soil from the bucket, the truck tailgate or over the blade is simulated as particles.
Each particle carries its real volume and is deposited where it lands (on terrain or in the truck body).

## Machines and reference specs

All models are built procedurally from published spec-sheet dimensions of the class. They are
generic and unbranded.

| | Reference class | Key figures used |
|---|---|---|
| **EX-20 excavator** | 20 t crawler excavator (CAT 320 / PC210 class) | 22.5 t, 129 kW, 5.7 m boom, 2.9 m stick, 1.19 m³ bucket, 2.38 m gauge, 600 mm shoes, 4.45 m track, 2.83 m tail swing, 11.25 rpm swing, 3.4 / 5.7 km/h travel, ~6.7 m dig depth |
| **DT-16 dump truck** | 6×4 tandem vocational truck, 16 yd³ body | 13.6 t empty / 33.1 t GVWR, 336 kW, 2,237 N·m, 5.3 m wheelbase, 315/80R22.5 tyres, 4.9 × 2.3 × 1.15 m body, 50° tip, 6-speed automatic with torque converter |
| **DZ-6 dozer** | Medium track-type tractor (CAT D6 class) | 20.5 t, 160 kW, 1.98 m gauge, 560 mm shoes, 2.67 m track on ground, elevated sprocket, 3.26 m semi-U blade (1.1 m lift / 0.5 m dig), 3-shank ripper |

Physics: rigid bodies at 240 Hz with penalty ground contacts and an elasto-plastic ("bristle")
friction model. Machines park on slopes without creeping but slide or spin when traction is
exceeded. Gravity is applied per component (boom, stick, bucket + load, payload), so the centre of
gravity really moves and a heavy load at full reach can tip the excavator.
Sound is synthesised live with WebAudio: diesel firing frequency from RPM, turbo, hydraulic whine,
relief-valve squeal, track clatter, soil pour and the reversing alarm.

## Code map

```
src/config.js      soil properties + machine specs
src/terrain.js     layered soil grid, repose flow, compaction, terrain shader
src/physics.js     rigid body, contacts, friction, machine collisions
src/excavator.js   excavator model, kinematics, hydraulics, digging
src/dozer.js       dozer model, blade & ripper soil interaction
src/truck.js       truck model, suspension, drivetrain, body load simulation
src/particles.js   falling soil + dust/exhaust
src/env.js         sky, lighting, surroundings, site props
src/audio.js       procedural engine/hydraulic/soil sounds
src/main.js        game loop, camera, HUD, job tracking
```
