// Quick fit test: bottom 22 mm of the M14 grip (prints in ~30 min)
intersection() {
    import("pilot_grip_threaded_M14.stl");
    translate([-50, -50, 0]) cube([100, 100, 22]);
}
