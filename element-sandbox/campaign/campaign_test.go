package campaign

import (
	"bytes"
	"testing"

	"elementsandbox/sim"
)

// wire draws a one-cell copper line between two cells, like a player would.
func wire(t *testing.T, c *State, s *sim.Sim, x0, y0, x1, y1 int) {
	t.Helper()
	n := max(abs(x1-x0), abs(y1-y0))
	for k := 0; k <= n; k++ {
		x := x0 + (x1-x0)*k/max(n, 1)
		y := y0 + (y1-y0)*k/max(n, 1)
		c.Paint(s, x, y, 0, sim.Copper)
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func buy(t *testing.T, c *State, s *sim.Sim, k *Kit, x, y int) int {
	t.Helper()
	y, ok := k.Drop(s, x, y)
	if !ok || !c.BuyKit(s, k, x, y) {
		t.Fatalf("couldn't buy %s at %d,%d with $%.0f", k.Name, x, y, c.Money)
	}
	return y
}

// coal fills a boiler's firebox (anchor x, y) with rows of coal and lights it.
func coal(t *testing.T, c *State, s *sim.Sim, x, y, rows int) {
	for yy := y - rows + 1; yy <= y; yy++ {
		for xx := x - 11; xx <= x+10; xx++ {
			c.Paint(s, xx, yy, 0, sim.Coal)
		}
	}
	for k := 0; k < 40; k++ {
		s.Ignite(x-11+k%22, y-rows-1, 3)
		c.Update(s, "Fire")
	}
}

// play runs the stage until it's won, failing after secs seconds.
func play(t *testing.T, c *State, s *sim.Sim, selected string, secs int) {
	t.Helper()
	for f := 0; f < 60*secs; f++ {
		c.Update(s, selected)
		if c.Money < -1e-6 {
			t.Fatalf("money went negative: %.2f", c.Money)
		}
		if f%300 == 0 {
			t.Logf("  %s t=%ds houses %d/%d power %.0f/%.0f stored %.0f ppm %.0f $%.0f step %d",
				c.Cur().Name, f/60, s.Stats.BeaconOn, s.Stats.Beacons, s.Stats.Used, s.Stats.Generated, s.Stats.Stored, s.Climate.PPM, c.Money, c.Step)
		}
		if c.Won {
			t.Logf("%s won after %.1f s with $%.0f left", c.Cur().Name, float64(f)/60, c.Money)
			return
		}
	}
	t.Fatalf("%s not won after %d s: houses %d/%d, power %.1f, ppm %.0f, step %d", c.Cur().Name, secs,
		s.Stats.BeaconOn, s.Stats.Beacons, s.Stats.Generated, s.Climate.PPM, c.Step)
}

// toCleanAir plays stages 1 to 3 the intended way and begins stage 4.
func toCleanAir(t *testing.T) (*State, *sim.Sim) {
	s := sim.New()
	c := Start(s)

	// 1. First Light: a turbine on the hill, wired to the house
	for k := 0; k < 30; k++ {
		c.Update(s, WindTurbine.Name)
	}
	mast := sc(62)
	buy(t, c, s, WindTurbine, mast, sc(60))
	wire(t, c, s, mast+2, 150, 492, 150)
	wire(t, c, s, 492, 150, 492, 277)
	wire(t, c, s, 492, 277, 494, 277)
	play(t, c, s, "Copper Wire", 60)

	// 2. Growing Town: wire the new houses together and add a coal boiler
	c.Next(s)
	wire(t, c, s, 380, 277, 494, 277)
	bx := sc(150)
	by := buy(t, c, s, SteamBoiler, bx, 230)
	wire(t, c, s, bx+15, by-42, bx+15, 151)
	t.Logf("before coal $%.0f", c.Money)
	coal(t, c, s, bx, by, 5)
	play(t, c, s, "Fire", 60)

	// 3. Still Nights: a battery bank, and more coal on the fire
	c.Next(s)
	kx := sc(240)
	ky := buy(t, c, s, BatteryBank, kx, 230)
	wire(t, c, s, kx+9, ky-6, kx+9, 151)
	coal(t, c, s, bx, by, 5)
	play(t, c, s, "Fire", 90)
	c.Next(s)
	return c, s
}

func TestCampaignPlaysThrough(t *testing.T) {
	c, s := toCleanAir(t)

	// 4. Clean Air: plant trees, keep the wind and batteries running
	for k := 0; k < 9; k++ {
		x := sc(100 + k*16)
		c.Paint(s, x, sc(ground)-1, 0, sim.Seed)
	}
	play(t, c, s, "Tree Seed", 180)

	// 5. Hot Springs: a boiler over the vent, wired to the new houses
	c.Next(s)
	vy := buy(t, c, s, SteamBoiler, sc(168), 230)
	wire(t, c, s, sc(168)+15, vy-42, sc(168)+15, 151)
	t.Logf("insurance paid $%.0f", c.Paid)
	wire(t, c, s, sc(138), 150, sc(258), 150) // mend the main line
	wire(t, c, s, sc(194)-2, 277, sc(258), 277)
	ky := buy(t, c, s, BatteryBank, sc(270), 200) // between the houses and the hill line
	wire(t, c, s, sc(270)+9, ky-6, sc(270)+9, 151)
	play(t, c, s, "Copper Wire", 120)
	if !c.Final() {
		t.Fatal("campaign not finished")
	}

	// the save round-trips
	var buf bytes.Buffer
	if err := c.Save(&buf, s); err != nil {
		t.Fatal(err)
	}
	t.Logf("save is %d KB", buf.Len()/1024)
	s2 := sim.New()
	c2, err := Load(&buf, s2)
	if err != nil {
		t.Fatal(err)
	}
	if c2.Stage != c.Stage || c2.Money != c.Money || s2.Cells[150*sim.W+300] != s.Cells[150*sim.W+300] {
		t.Fatal("save didn't round-trip")
	}
}

// firstLight plays stage 1 and moves on to stage 2.
func firstLight(t *testing.T) (*State, *sim.Sim) {
	s := sim.New()
	c := Start(s)
	c.Update(s, WindTurbine.Name)
	buy(t, c, s, WindTurbine, sc(62), sc(60))
	wire(t, c, s, sc(62)+2, 150, 492, 150)
	wire(t, c, s, 492, 150, 492, 277)
	wire(t, c, s, 492, 277, 494, 277)
	play(t, c, s, "Copper Wire", 60)
	c.Next(s)
	wire(t, c, s, 380, 277, 494, 277)
	return c, s
}

// Stage 2 can't be won with what stage 1 left behind.
func TestGrowingTownNeedsMorePower(t *testing.T) {
	c, s := firstLight(t)
	for f := 0; f < 60*30; f++ {
		c.Update(s, "")
		if c.Won {
			t.Fatal("won stage 2 with one turbine")
		}
	}
}

// Stage 4's smog doesn't clear by itself: it takes trees.
func TestCleanAirNeedsTrees(t *testing.T) {
	c, s := toCleanAir(t)
	for f := 0; f < 60*90; f++ {
		c.Update(s, "")
		if c.Won {
			t.Fatalf("won Clean Air without trees (ppm %.0f)", s.Climate.PPM)
		}
	}
	t.Logf("after 90 s without trees: %.0f ppm", s.Climate.PPM)
}
