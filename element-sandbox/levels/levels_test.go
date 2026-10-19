package levels

import (
	"testing"

	"elementsandbox/sim"
)

// place fills a rectangle with a player material and returns the cell count.
func place(s *sim.Sim, x0, y0, x1, y1 int, m sim.Mat) int {
	s.Fill(Sc(x0), Sc(y0), Sc(x1), Sc(y1), m, false) // design-grid coordinates
	return (Sc(x1) - Sc(x0)) * (Sc(y1) - Sc(y0))
}

// solutions apply each level's intended answer and return cells used per material.
var solutions = []func(s *sim.Sim) map[sim.Mat]int{
	func(s *sim.Sim) map[sim.Mat]int {
		used := map[sim.Mat]int{}
		used[sim.Coal] = place(s, 45, 178, 105, 192, sim.Coal)
		used[sim.Copper] = place(s, 158, 100, 192, 103, sim.Copper)
		for x := 46; x < 105; x += 3 { // hold a small flame along the coal
			for k := 0; k < 8; k++ {
				s.Ignite(Sc(x), Sc(177), 3)
				s.Step()
			}
		}
		return used
	},
	func(s *sim.Sim) map[sim.Mat]int {
		n := place(s, 53, 40, 54, 90, sim.Copper)
		n += place(s, 53, 40, 329, 41, sim.Copper)
		n += place(s, 328, 40, 329, 44, sim.Copper)
		n += place(s, 228, 41, 229, 176, sim.Copper)
		return map[sim.Mat]int{sim.Copper: n}
	},
	func(s *sim.Sim) map[sim.Mat]int {
		used := map[sim.Mat]int{}
		used[sim.ThermoGen] = place(s, 50, 148, 80, 150, sim.ThermoGen)
		used[sim.Ice] = place(s, 50, 140, 78, 148, sim.Ice)
		n := place(s, 79, 138, 80, 148, sim.Copper)
		n += place(s, 79, 138, 309, 139, sim.Copper)
		n += place(s, 308, 139, 309, 176, sim.Copper)
		used[sim.Copper] = n
		return used
	},
	func(s *sim.Sim) map[sim.Mat]int {
		cu := place(s, 148, 103, 149, 160, sim.Copper)
		au := place(s, 148, 160, 149, 188, sim.Gold)
		au += place(s, 148, 186, 223, 187, sim.Gold)
		au += place(s, 222, 160, 223, 186, sim.Gold)
		cu += place(s, 222, 150, 223, 160, sim.Copper)
		cu += place(s, 222, 150, 309, 151, sim.Copper)
		cu += place(s, 308, 151, 309, 176, sim.Copper)
		return map[sim.Mat]int{sim.Copper: cu, sim.Gold: au}
	},
	func(s *sim.Sim) map[sim.Mat]int {
		t := place(s, 34, 86, 46, 88, sim.Turbine)
		cu := place(s, 34, 85, 221, 86, sim.Copper)
		cu += place(s, 100, 86, 101, 176, sim.Copper)
		cu += place(s, 220, 86, 221, 176, sim.Copper)
		return map[sim.Mat]int{sim.Turbine: t, sim.Copper: cu}
	},
	func(s *sim.Sim) map[sim.Mat]int {
		return map[sim.Mat]int{sim.Seed: plant(s, 120, 350, 12)}
	},
	func(s *sim.Sim) map[sim.Mat]int {
		n := plant(s, 120, 290, 10)
		for k := 0; k < 25*60; k++ { // let the forest grow before burning anything
			s.Step()
		}
		coal := place(s, 45, 184, 105, 192, sim.Coal)
		for x := 46; x < 105; x += 3 {
			for k := 0; k < 8; k++ {
				s.Ignite(Sc(x), Sc(183), 3)
				s.Step()
			}
		}
		return map[sim.Mat]int{sim.Seed: n, sim.Coal: coal}
	},
	// 8. Second Wind: turbines on the left hill, a battery bank, one long wire
	func(s *sim.Sim) map[sim.Mat]int {
		tu := place(s, 64, 96, 72, 97, sim.Turbine)
		cu := place(s, 66, 97, 67, 100, sim.Copper)
		cu += place(s, 64, 95, 329, 96, sim.Copper)
		cu += place(s, 148, 96, 149, 176, sim.Copper)
		cu += place(s, 229, 96, 230, 104, sim.Copper)
		cu += place(s, 328, 96, 329, 176, sim.Copper)
		b := place(s, 238, 80, 253, 81, sim.Battery)
		cu += place(s, 244, 81, 245, 96, sim.Copper)
		return map[sim.Mat]int{sim.Turbine: tu, sim.Copper: cu, sim.Battery: b}
	},
	// 9. Deep Freeze: clear the snow, burn coal, Thermo Gen against the drift
	func(s *sim.Sim) map[sim.Mat]int {
		s.Fill(Sc(26), Sc(173), Sc(117), Sc(192), sim.Empty, false) // erase the snow
		coal := place(s, 33, 178, 110, 192, sim.Coal)
		tg := place(s, 115, 173, 116, 192, sim.ThermoGen)
		cu := place(s, 120, 103, 121, 131, sim.Copper) // ties the pot, and the Thermo Gen under it, in
		cu += place(s, 168, 100, 202, 103, sim.Copper)
		cu += place(s, 248, 100, 282, 103, sim.Copper)
		light(s, 34, 117, 177)
		return map[sim.Mat]int{sim.Coal: coal, sim.ThermoGen: tg, sim.Copper: cu}
	},
	// 10. Flood Gate: two blasts through the dam, then bridge the wire
	func(s *sim.Sim) map[sim.Mat]int {
		s.Explode(Sc(104), Sc(81), 4) // a small bomb on top of the TNT gate
		return map[sim.Mat]int{sim.Copper: place(s, 258, 90, 292, 93, sim.Copper)}
	},
	// 11. Eruption: copper overhead, platinum down into the lava pit
	func(s *sim.Sim) map[sim.Mat]int {
		cu := place(s, 98, 70, 233, 71, sim.Copper)
		cu += place(s, 232, 71, 233, 140, sim.Copper)
		pt := place(s, 232, 140, 233, 184, sim.Platinum)
		return map[sim.Mat]int{sim.Copper: cu, sim.Platinum: pt}
	},
	// 12. The Grid: boiler + gold through the acid + hilltop turbines + batteries
	func(s *sim.Sim) map[sim.Mat]int {
		lead := 10 + 33
		cu := place(s, lead, 60, lead+1, 80, sim.Copper)
		cu += place(s, lead, 60, 160, 61, sim.Copper)
		cu += place(s, 120, 61, 121, 176, sim.Copper)
		cu += place(s, 159, 61, 160, 160, sim.Copper)
		au := place(s, 159, 160, 160, 187, sim.Gold)
		au += place(s, 159, 186, 190, 187, sim.Gold)
		au += place(s, 189, 160, 190, 186, sim.Gold)
		cu += place(s, 189, 150, 190, 160, sim.Copper)
		cu += place(s, 189, 150, 260, 151, sim.Copper)
		cu += place(s, 228, 151, 229, 176, sim.Copper)
		cu += place(s, 259, 90, 260, 150, sim.Copper)
		cu += place(s, 259, 90, 301, 91, sim.Copper)
		tu := place(s, 298, 84, 302, 85, sim.Turbine)
		cu += place(s, 300, 85, 301, 96, sim.Copper)
		cu += place(s, 308, 100, 349, 101, sim.Copper)
		cu += place(s, 348, 101, 349, 176, sim.Copper)
		b := place(s, 266, 120, 274, 121, sim.Battery)
		cu += place(s, 260, 120, 266, 121, sim.Copper)
		return map[sim.Mat]int{sim.Copper: cu, sim.Gold: au, sim.Turbine: tu, sim.Battery: b}
	},
}

// light holds a small flame along y from design x0 to x1, like a player would.
func light(s *sim.Sim, x0, x1, y int) {
	for x := x0; x < x1; x += 3 {
		for k := 0; k < 8; k++ {
			s.Ignite(Sc(x), Sc(y), 3)
			s.Step()
		}
	}
}

// plant drops n seeds evenly between design x0 and x1, just above the dirt.
func plant(s *sim.Sim, x0, x1, n int) int {
	for k := 0; k < n; k++ {
		x := Sc(x0 + (x1-x0)*k/max(n-1, 1))
		s.Fill(x, Sc(183), x+1, Sc(183)+1, sim.Seed, false)
	}
	return n
}

func setup(l Level) *sim.Sim {
	s := sim.New()
	s.Wind.Global = l.Wind
	s.OpenTop = l.OpenTop
	l.Build(s)
	return s
}

func TestEveryLevelHasASolution(t *testing.T) {
	if len(solutions) != len(All) {
		t.Fatalf("%d levels but %d solutions", len(All), len(solutions))
	}
	for li, l := range All {
		t.Run(l.Name, func(t *testing.T) {
			s := setup(l)
			used := solutions[li](s)
			for m, n := range used {
				budget := -2
				for _, a := range l.Allow {
					if a.Tool == "" && a.Mat == m {
						budget = a.Budget
					}
				}
				if budget == -2 {
					t.Fatalf("solution uses %s, which the level doesn't offer", sim.Info(m).Name)
				}
				if budget >= 0 && n > budget {
					t.Fatalf("solution uses %d %s, budget is %d", n, sim.Info(m).Name, budget)
				}
			}
			hold := 0
			for f := 0; f < 60*60; f++ {
				l.Step(s, f)
				if l.Won(s) {
					hold++
					if hold >= l.HoldTime() {
						t.Logf("won after %.1f s (generating %.1f)", float64(f)/60, s.Stats.Generated)
						return
					}
				} else {
					hold = 0
				}
			}
			t.Fatalf("not won after 60 s: beacons %d/%d, generated %.1f, CO2 %.0f ppm", s.Stats.BeaconOn, s.Stats.Beacons, s.Stats.Generated, s.Climate.PPM)
		})
	}
}

func TestNoLevelWinsByItself(t *testing.T) {
	for _, l := range All {
		s := setup(l)
		for f := 0; f < 900; f++ {
			l.Step(s, f)
			if l.Won(s) {
				t.Fatalf("%s: beacons powered without the player doing anything", l.Name)
			}
		}
	}
}

func TestGoldStandardNeedsGold(t *testing.T) {
	s := setup(All[3])
	place(s, 148, 103, 149, 188, sim.Copper)
	place(s, 148, 186, 223, 187, sim.Copper)
	place(s, 222, 150, 223, 186, sim.Copper)
	place(s, 222, 150, 309, 151, sim.Copper)
	place(s, 308, 151, 309, 176, sim.Copper)
	won := 0
	for f := 0; f < 60*40; f++ {
		s.Step()
		if s.Stats.BeaconOn == s.Stats.Beacons {
			won = f
		}
	}
	if won > 60*20 {
		t.Fatalf("an all-copper wire through acid still powers the beacon at %.0f s", float64(won)/60)
	}
}

// held runs a level for up to secs seconds and reports whether its goal was
// held long enough to win.
func held(l Level, s *sim.Sim, secs int) bool {
	hold := 0
	for f := 0; f < 60*secs; f++ {
		l.Step(s, f)
		if l.Won(s) {
			hold++
			if hold >= l.HoldTime() {
				return true
			}
		} else {
			hold = 0
		}
	}
	return false
}

func TestSecondWindNeedsBatteries(t *testing.T) {
	l := All[7]
	s := setup(l)
	place(s, 64, 96, 72, 97, sim.Turbine)
	place(s, 66, 97, 67, 100, sim.Copper)
	place(s, 64, 95, 329, 96, sim.Copper)
	place(s, 148, 96, 149, 176, sim.Copper)
	place(s, 229, 96, 230, 104, sim.Copper)
	place(s, 328, 96, 329, 176, sim.Copper)
	if held(l, s, 40) {
		t.Fatal("won without any batteries")
	}
}

func TestEruptionMeltsCopper(t *testing.T) {
	l := All[10]
	s := setup(l)
	place(s, 98, 70, 233, 71, sim.Copper)
	place(s, 232, 71, 233, 184, sim.Copper)
	if held(l, s, 60) {
		t.Fatal("an all-copper wire survived the eruptions")
	}
}

func TestFloodGateNeedsTheDamBlown(t *testing.T) {
	l := All[9]
	s := setup(l)
	place(s, 258, 90, 292, 93, sim.Copper)
	if held(l, s, 30) {
		t.Fatal("won without letting the water in")
	}
}

func TestTheGridNeedsBatteries(t *testing.T) {
	l := All[11]
	s := setup(l)
	solutions[11](s)
	for i := 0; i < sim.W*sim.H; i++ { // take the batteries back out
		if c := &s.Cells[i]; c.M == sim.Battery {
			s.Fill(i%sim.W, i/sim.W, i%sim.W+1, i/sim.W+1, sim.Empty, false)
		}
	}
	if held(l, s, 40) {
		t.Fatal("won without batteries")
	}
}
