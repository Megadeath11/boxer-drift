package sim

import "testing"

func seasonal(season int, locked bool) *Sim {
	s := New()
	s.Climate = Climate{Enabled: true, Locked: locked, YearFrames: 60 * 60 * 4}
	s.Climate.SetSeason(season)
	s.Climate.Phase += 0.125 // middle of the season
	return s
}

func TestSeasonsChangeTemperature(t *testing.T) {
	s := seasonal(Summer, true)
	s.Step()
	summer := s.Ambient
	s.Climate.SetSeason(Winter)
	s.Climate.Phase += 0.125
	s.Step()
	if winter := s.Ambient; summer-winter < 20 || winter > 0 {
		t.Fatalf("summer %.1f°C vs winter %.1f°C", summer, winter)
	}
}

func TestWinterFreezesPond(t *testing.T) {
	s := seasonal(Summer, true)
	fill(s, 200, H-20, 340, H, Water)
	run(s, 10)
	s.Climate.SetSeason(Winter)
	s.Climate.Phase += 0.125
	run(s, 2500)
	if count(s, Ice) == 0 {
		t.Fatal("pond never froze in winter")
	}
}

func growForest(s *Sim) {
	fill(s, 0, H-10, W, H, Dirt)
	for x := 60; x < W-60; x += 40 {
		s.put((H-11)*W+x, Seed)
	}
}

func TestSeedsGrowIntoTrees(t *testing.T) {
	s := seasonal(Summer, true)
	growForest(s)
	run(s, 2000)
	trunks, leaves := count(s, Trunk), count(s, Leaf)
	if trunks < 50 || leaves < 200 {
		t.Fatalf("trees did not grow: %d trunk, %d leaf cells", trunks, leaves)
	}
}

func TestTreesAbsorbCO2(t *testing.T) {
	withTrees := func(trees bool) int {
		s := seasonal(Summer, true)
		s.AddRoof(4)
		fill(s, 0, H-10, W, H, Dirt)
		if trees {
			growForest(s)
			run(s, 2000)
		}
		fill(s, 0, 30, W, 33, CO2) // a CO2 layer that sinks through the canopy
		run(s, 1500)
		return count(s, CO2)
	}
	bare, forest := withTrees(false), withTrees(true)
	if forest > bare*7/10 {
		t.Fatalf("trees barely absorbed CO2: %d left vs %d without trees", forest, bare)
	}
}

func TestRoofMakesRain(t *testing.T) {
	s := New()
	s.AddRoof(4)
	fill(s, 200, 120, 340, 140, Steam)
	steam := count(s, Steam)
	run(s, 1200)
	if count(s, Steam) > steam/3 {
		t.Fatalf("steam did not condense under the roof: %d of %d left", count(s, Steam), steam)
	}
	if count(s, Water) == 0 {
		t.Fatal("no rain fell")
	}
}

func TestBurningCoalRaisesCO2(t *testing.T) {
	s := New()
	s.AddRoof(4)
	fill(s, 200, H-10, 340, H, Coal)
	for k := 0; k < 30; k++ {
		s.Ignite(270, H-11, 30)
		s.Step()
	}
	run(s, 600)
	if s.Climate.PPM < basePPM+100 {
		t.Fatalf("burning coal barely raised CO2: %.0f ppm", s.Climate.PPM)
	}
	if s.Ambient <= DefaultAmbient {
		t.Fatalf("no greenhouse warming at %.0f ppm", s.Climate.PPM)
	}
}

func TestLeavesFallInAutumn(t *testing.T) {
	s := seasonal(Summer, true)
	growForest(s)
	run(s, 2000)
	leaves := count(s, Leaf)
	s.Climate.SetSeason(Autumn)
	s.Climate.Phase += 0.24 // late autumn
	run(s, 1200)
	if count(s, Leaf) > leaves/2 || count(s, FallenLeaf) == 0 {
		t.Fatalf("leaves did not fall: %d -> %d", leaves, count(s, Leaf))
	}
}

// A tree beside lava catches fire, but it must burn out: a scorched tree
// mustn't keep growing fresh leaves into the heat and feeding the fire.
func TestTreeByLavaBurnsOut(t *testing.T) {
	s := New()
	s.Climate = Climate{Enabled: true, Locked: true, YearFrames: 60 * 60 * 4}
	s.Climate.SetSeason(Summer)
	s.AddRoof(4)
	s.Fill(0, H-10, W, H, Wall, true)
	s.Fill(0, H-16, W, H-10, Dirt, true)
	s.Fill(200, H-16, 240, H-10, Lava, true) // a vent
	for _, x := range []int{196, 244, 250} {
		s.Fill(x, H-17, x+1, H-16, Seed, false)
	}
	burning := func() int {
		n := 0
		for i := range s.Cells {
			if s.Cells[i].M == Fire || s.Cells[i].Flags&fBurning != 0 {
				n++
			}
		}
		return n
	}
	peak := 0
	for f := 0; f < 60*90; f++ {
		s.Step()
		peak = max(peak, burning())
	}
	tail := 0
	for f := 0; f < 60*30; f++ {
		s.Step()
		tail += burning()
	}
	t.Logf("peak %d burning cells; %.1f on average over the last 30 s; %.0f ppm", peak, float64(tail)/(60*30), s.Climate.PPM)
	if float64(tail)/(60*30) > 5 {
		t.Fatal("the trees by the lava are still burning after 90 s")
	}
}
