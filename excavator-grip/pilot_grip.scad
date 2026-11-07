// =====================================================================
//  Mini Excavator Pilot Control Grip  (parametric, 3D-printable)
//  Ergonomic joystick handle with finger grooves, palm head, optional
//  horn/thumb button hole and wire channel.
//
//  MEASURE YOUR LEVER FIRST and set the "Mount" parameters below.
//  Units: mm.  +Y = front of grip (finger side), -Y = back (palm/thumb).
// =====================================================================

/* [Mount] */
// "bore"  = smooth shaft, held with set screws
// "thread" = threaded stud end, grip screws on and is held by a nut in the base
mount_type     = "bore";   // ["bore", "thread"]
// Lever shaft outside diameter (bore mode)
shaft_d        = 19.0;
// Clearance added to shaft_d (0.3–0.6 for most FDM printers)
shaft_clear    = 0.4;
// How far the shaft goes into the grip
bore_depth     = 55;
// Set screw tap-drill size (5.0 = tap M6; 8.0 = M6 heat-set insert)
setscrew_d     = 5.0;
// Heights of the set screws above the bottom of the grip
setscrew_z     = [12, 32];
// Thread mode: stud diameter (M12 -> 12)
stud_d         = 12;
// Thread mode: nut across flats (M12 = 18, M10 = 16/17)
nut_af         = 18;
// Thread mode: nut thickness (M12 = 10.8)
nut_h          = 10.8;

/* [Button & wiring] */
add_button     = true;
// Panel hole for a momentary push button (16 mm is common)
button_d       = 16.2;
// Pocket depth behind the button for the switch body/terminals
button_pocket_d = 20;
button_pocket_h = 22;
// Wire channel from the shaft bore up to the button
wire_d         = 8;
// Tilt of the button toward the operator's thumb (degrees)
button_tilt    = 15;

/* [Ergonomics] */
// Overall scale of the handle body (1.0 = medium glove, 1.08 = large hand)
size           = 1.0;
// Number of finger grooves on the front
n_fingers      = 4;
finger_pitch   = 20;
finger_r       = 9;
finger_depth   = 3;
first_finger_z = 28;

/* [Quality] */
$fn = 72;

// ---------------------------------------------------------------------
// Body profile: [z, width (X), depth (Y), y offset]
profile = [
    [  0, 42, 42,  0  ],   // base flare (sits over rubber boot)
    [  5, 35, 37,  0  ],
    [ 20, 33, 36,  0  ],
    [ 50, 35, 39,  1  ],
    [ 80, 37, 41,  2  ],
    [ 96, 42, 47,  3  ],
    [108, 47, 53,  2.5],   // palm head
    [118, 47, 52,  1  ],
    [125, 42, 46,  0  ],
    [130, 32, 35, -1  ],   // top crown
];

top_z = profile[len(profile)-1][0] * size;

module slice(p) {
    translate([0, p[3]*size, p[0]*size])
        linear_extrude(0.01)
            scale([p[1]*size/2, p[2]*size/2]) circle(1);
}

module body() {
    for (i = [0 : len(profile)-2])
        hull() { slice(profile[i]); slice(profile[i+1]); }
    // rounded crown
    p = profile[len(profile)-1];
    translate([0, p[3]*size, p[0]*size])
        scale([p[1]*size/2, p[2]*size/2, 4*size]) sphere(1);
}

// depth/2 + y offset at height z (front surface), linear interpolation
function front_y(z, i=0) =
    (i >= len(profile)-1) ? 0 :
    (z/size <= profile[i+1][0])
        ? let(a = profile[i], b = profile[i+1],
              t = (z/size - a[0]) / (b[0] - a[0]))
          size * ((a[2]/2 + a[3]) * (1-t) + (b[2]/2 + b[3]) * t)
        : front_y(z, i+1);

module finger_grooves() {
    for (k = [0 : n_fingers-1]) {
        z = (first_finger_z + k*finger_pitch) * size;
        translate([0, front_y(z) + finger_r - finger_depth, z])
            rotate([0, 90, 0])
                cylinder(r = finger_r, h = 60, center = true);
    }
}

module mount_cut() {
    if (mount_type == "bore") {
        translate([0, 0, -1])
            cylinder(d = shaft_d + shaft_clear, h = bore_depth + 1);
        // chamfer entry
        translate([0, 0, -0.01])
            cylinder(d1 = shaft_d + shaft_clear + 3, d2 = shaft_d + shaft_clear, h = 1.5);
        // set screws from the back side
        for (z = setscrew_z)
            translate([0, 0, z]) rotate([90, 0, 0])
                cylinder(d = setscrew_d, h = 40);
    } else {
        // nut pocket in base + stud bore
        translate([0, 0, -1])
            cylinder(d = nut_af / cos(30) + 0.4, h = nut_h + 1, $fn = 6);
        cylinder(d = stud_d + 0.6, h = bore_depth);
    }
}

module button_cut() {
    // Wire channel from bore to button pocket
    translate([0, 0, bore_depth - 1])
        cylinder(d = wire_d, h = top_z - bore_depth - 10);
    if (add_button) {
        translate([0, 0, top_z + 1])
            rotate([button_tilt, 0, 0]) {
                // through hole for button
                translate([0, 0, -12]) cylinder(d = button_d, h = 30);
                // switch body pocket
                translate([0, 0, -12 - button_pocket_h])
                    cylinder(d = button_pocket_d, h = button_pocket_h + 0.01);
                // flat spot so the button bezel seats
                translate([0, 0, -3]) cylinder(d = button_d + 8, h = 20);
            }
    } else {
        // side exit for wires near the base, back side
        translate([0, 0, bore_depth - wire_d/2]) rotate([90, 0, 0])
            cylinder(d = wire_d, h = 40);
    }
}

difference() {
    body();
    finger_grooves();
    mount_cut();
    button_cut();
}
