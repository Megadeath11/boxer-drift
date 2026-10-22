package levels

import "elementsandbox/sim"

// The expert levels come after the rest. Nothing is built for you, the
// briefings only say what's wanted, budgets are tight and holds are long.

func init() { All = append(All, expert...) }

// acidRain drizzles acid between design x0 and x1, with a heavy downpour
// for burst frames out of every period. Acid that reaches the ground soaks
// away, so the valley floor doesn't flood.
func acidRain(x0, x1, period, burst int) func(s *sim.Sim, frame int) {
	return func(s *sim.Sim, frame int) {
		rate := uint32(3) // drops per 1000 columns per frame
		if frame%period < burst {
			rate = 40
		}
		for x := Sc(x0); x < Sc(x1); x++ {
			h := uint32(frame*7919+x*104729) * 2654435761
			if h%1000 < rate {
				if c := s.At(x, Sc(8)); c.M == sim.Empty {
					s.Fill(x, Sc(8), x+1, Sc(8)+1, sim.Acid, false)
				}
			}
		}
		for x := 0; x < sim.W; x++ {
			for y := Sc(189); y < Sc(192); y++ {
				if c := s.At(x, y); c.M == sim.Acid && !c.IsFixed() {
					s.Fill(x, y, x+1, y+1, sim.Empty, false)
				}
			}
		}
	}
}

var expert = []Level{
	{
		Name: "13. From Scratch",
		Brief: []string{
			"A beacon, an empty field and a pile of parts.",
			"Light it.",
		},
		Allow: []Allow{mat(sim.Metal, 520), mat(sim.Water, 1250), mat(sim.Turbine, 130), mat(sim.Coal, 1000),
			tool(ToolFire), mat(sim.Copper, 300), tool(ToolErase)},
		Build: func(s *sim.Sim) {
			floor(s)
			fixed(s, 320, 176, 336, 192, sim.Beacon)
		},
	},
	{
		Name: "14. Hot Ground",
		Brief: []string{
			"The ground is hot here. Two beacons, 24 power.",
			"No fuel.",
		},
		Allow: []Allow{mat(sim.Metal, 560), mat(sim.Water, 1500), mat(sim.Turbine, 140), mat(sim.Copper, 400),
			tool(ToolErase)},
		Build: func(s *sim.Sim) {
			floor(s)
			wall(s, 130, 180, 134, 192)
			wall(s, 226, 180, 230, 192)
			fixed(s, 134, 184, 226, 192, sim.Lava)
			fixed(s, 20, 176, 36, 192, sim.Beacon)
			wall(s, 290, 80, 360, 86) // a high ledge
			fixed(s, 324, 64, 340, 80, sim.Beacon)
		},
	},
	{
		Name: "15. Acid Rain",
		Tick: acidRain(110, 300, 60*10, 60*4),
		Hold: 60 * 20,
		Brief: []string{
			"It's raining acid. Keep the beacon lit for 20 s.",
		},
		Allow: []Allow{mat(sim.Copper, 300), mat(sim.Gold, 40), mat(sim.Glass, 180), tool(ToolErase)},
		Build: func(s *sim.Sim) {
			floor(s)
			// a power station in a shelter on the left
			wall(s, 0, 60, 104, 64)
			wall(s, 100, 64, 104, 120)
			lead := boiler(s, 10, 90, true)
			fixed(s, lead, 90, 100, 93, sim.Copper)
			fixed(s, 320, 176, 336, 192, sim.Beacon)
		},
	},
	{
		Name:    "16. Long Night",
		Wind:    2,
		OpenTop: true,
		Hold:    60 * 30,
		Tick:    gusts(2, 60*6, 60*7),
		Brief: []string{
			"Short gusts, long calms, four beacons.",
			"Keep them all lit for 30 s.",
		},
		Allow: []Allow{mat(sim.Turbine, 13), mat(sim.Battery, 22), mat(sim.Copper, 450), tool(ToolErase)},
		Build: func(s *sim.Sim) {
			floor(s)
			hill(s, 60, 100)
			hill(s, 300, 110)
			fixed(s, 120, 176, 136, 192, sim.Beacon)
			fixed(s, 180, 176, 196, 192, sim.Beacon)
			fixed(s, 240, 176, 256, 192, sim.Beacon)
			fixed(s, 292, 94, 308, 110, sim.Beacon) // on the second hill
		},
	},
	{
		Name:    "17. Final Exam",
		Wind:    2,
		OpenTop: true,
		Hold:    60 * 30,
		Tick:    gusts(2, 60*6, 60*6),
		Brief: []string{
			"Five beacons. Hot ground, acid, wind that comes and",
			"goes. Everything you've learned. 30 s.",
		},
		Allow: []Allow{mat(sim.Metal, 520), mat(sim.Water, 1400), mat(sim.Turbine, 145), mat(sim.Battery, 18),
			mat(sim.Copper, 680), mat(sim.Gold, 60), tool(ToolErase)},
		Build: func(s *sim.Sim) {
			floor(s)
			// a vent on the left, an acid moat in the middle, a hill on the right
			wall(s, 10, 180, 14, 192)
			wall(s, 106, 180, 110, 192)
			fixed(s, 14, 184, 106, 192, sim.Lava)
			wall(s, 150, 160, 154, 192)
			wall(s, 214, 160, 218, 192)
			fixed(s, 154, 168, 214, 192, sim.Acid)
			wall(s, 182, 40, 186, 176) // a wall dipping into the acid
			hill(s, 290, 110)
			fixed(s, 120, 176, 136, 192, sim.Beacon)
			fixed(s, 226, 176, 242, 192, sim.Beacon)
			fixed(s, 282, 94, 298, 110, sim.Beacon)
			fixed(s, 330, 176, 346, 192, sim.Beacon)
			fixed(s, 30, 40, 46, 56, sim.Beacon) // up in the air: wire up to it
		},
	},
}
