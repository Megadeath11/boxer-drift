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
				s.Step()
				if l.Won(s) {
					hold++
					if hold >= HoldFrames {
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
			s.Step()
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
