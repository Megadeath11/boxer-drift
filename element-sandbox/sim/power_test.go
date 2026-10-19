package sim

import "testing"

func TestGoldResistsAcidCopperDoesNot(t *testing.T) {
	s := New()
	fill(s, 100, H-30, 200, H, Acid)
	fill(s, 120, H-20, 130, H-10, Gold)
	fill(s, 160, H-20, 170, H-10, Copper)
	gold, cu := count(s, Gold), count(s, Copper)
	run(s, 600)
	if count(s, Gold) != gold {
		t.Fatalf("acid ate gold: %d -> %d", gold, count(s, Gold))
	}
	if count(s, Copper) >= cu {
		t.Fatal("acid did not dissolve copper")
	}
}

func TestMetalsMeltAndResolidify(t *testing.T) {
	s := New()
	s.Fill(150, H-5, 160, H, Tin, false)
	for k := 0; k < 200; k++ {
		s.AddHeat(155, H-3, 8, 20)
		s.Step()
	}
	if count(s, MoltenTin) == 0 {
		t.Fatal("tin did not melt")
	}
	run(s, 3000)
	if count(s, MoltenTin) != 0 || count(s, Tin) == 0 {
		t.Fatalf("molten tin did not resolidify: molten %d solid %d", count(s, MoltenTin), count(s, Tin))
	}
}

func TestThermoGenPowersLamp(t *testing.T) {
	s := New()
	s.Fill(100, H-20, 140, H, Lava, false)
	s.Fill(140, H-20, 142, H, ThermoGen, false)
	s.Fill(142, H-20, 160, H, Ice, false)
	s.Fill(140, H-24, 142, H-20, ThermoGen, false)
	s.Fill(141, H-40, 142, H-24, Copper, false)
	s.Fill(141, H-41, 200, H-40, Copper, false)
	s.Fill(200, H-44, 203, H-40, Lamp, false)
	run(s, 120)
	if s.Stats.Generated <= 0 {
		t.Fatal("thermo gen made no power")
	}
	if !s.At(201, H-42).IsPowered() {
		t.Fatalf("lamp not powered, generated %.1f", s.Stats.Generated)
	}
}

func TestBatteryBuffersPower(t *testing.T) {
	s := New()
	s.Wind.Global = 4
	s.Fill(100, 50, 110, 60, Turbine, false)
	s.Fill(110, 55, 150, 56, Copper, false)
	s.Fill(150, 54, 154, 58, Battery, false)
	run(s, 400)
	if s.Stats.Stored <= 0 {
		t.Fatal("battery did not charge from wind turbine")
	}
	s.Wind.Global = 0
	s.Fill(100, 50, 110, 60, Empty, false) // remove the turbine
	s.Fill(154, 55, 170, 56, Copper, false)
	s.Fill(170, 54, 172, 56, Lamp, false)
	run(s, 30)
	if !s.At(170, 55).IsPowered() {
		t.Fatal("battery did not power the lamp")
	}
}
