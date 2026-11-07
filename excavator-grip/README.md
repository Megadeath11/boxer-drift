# Mini Excavator Pilot Control Grip

A 3D-printable replacement grip for a mini excavator pilot joystick. It has
four finger grooves, a palm head, and an optional thumb/horn button hole with
a wire channel. All dimensions can be changed in `pilot_grip.scad`.

| File | What it is |
|---|---|
| `pilot_grip.scad` | Parametric source (open in [OpenSCAD](https://openscad.org), use the Customizer panel) |
| `pilot_grip.stl` | Default: 19 mm smooth shaft, 2 set screws, 16 mm button hole |
| `pilot_grip_no_button.stl` | Same, no button. Wires exit at the back |
| `pilot_grip_threaded_M14.stl` | **Most likely fit for 1–1.5 t Chinese minis:** M14 threaded lever, M14 nut (22 mm AF) pressed into the base |
| `fit_test_M14.stl` | Bottom 22 mm of the M14 grip. Print it first (~30 min) to check the fit |
| `pilot_grip_threaded_M12.stl` | For an M12 threaded lever end, with an M12 nut pressed into the base |

## Before printing: measure your lever
Lever sizes vary by make and model (Kubota, Bobcat, Takeuchi, CAT, Yanmar, etc.).
Pull the old grip and check:
1. **Smooth shaft:** measure the OD with calipers and set `shaft_d`.
2. **Threaded end:** set `mount_type = "thread"`, `stud_d`, `nut_af` and `nut_h`.
3. Set `bore_depth` so it is a little less than how much bare shaft sticks
   out above the rubber boot.
4. If the old grip had a button, measure its panel hole and set `button_d`.

Overall size: 130 mm tall, about 47 × 53 mm at the palm head. Use `size = 1.08`
for large or gloved hands.

## Printing
- **Material:** PETG, ASA or Nylon. Don't use PLA, because it creeps and goes soft in a hot cab.
- Print upright on the base flare. Use 4+ perimeters, 30–40% gyroid infill, and 0.2 mm layers.
  Supports are only needed under the button pocket overhang (touching buildplate is fine
  if you disable the button).
- **Set screws:** tap the 5 mm holes to M6, or set `setscrew_d = 8` and use M6
  heat-set inserts. Add a drop of medium threadlocker.
- **Threaded version:** press or heat-set the nut into the hex pocket in the base. Screw the
  grip down onto the stud and tighten a jam nut under it.

## Safety
This is a control on heavy equipment. Before you operate the machine, check that the grip
is tight and can't rotate or pull off. Re-check it after the first few hours. Keep any
dead-man or pilot-lockout functions working.
