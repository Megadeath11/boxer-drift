package levels

import "elementsandbox/sim"

// The later levels mix several ideas together and add scripted events.

func init() { All = append(All, advanced...) }

// gusts makes the wind blow for on frames, then drop for off frames.
func gusts(speed float32, on, off int) func(s *sim.Sim, frame int) {
	return func(s *sim.Sim, frame int) {
		if frame%(on+off) < on {
			s.Wind.Global = speed
		} else {
			s.Wind.Global = 0
		}
	}
}

// winter locks the level in the depths of winter under a roof.
func winter(s *sim.Sim) {
	s.Climate = sim.Climate{Enabled: true, Locked: true, YearFrames: 60 * 60 * 4}
	s.Climate.SetSeason(sim.Winter)
	s.Climate.Phase += 0.125
	s.Step() // let the air cool to winter before anything is built
	s.AddRoof(Sc(3))
}

// erupt makes a volcano throw lava out of the crater at (x, y) for burst
// frames out of every period frames.
func erupt(x, y, period, burst int) func(s *sim.Sim, frame int) {
	return func(s *sim.Sim, frame int) {
		if frame%period >= burst || frame < 60*3 {
			return
		}
		cx, cy := Sc(x), Sc(y)
		for n := 0; n < 3*Sc(7); n++ {
			k, dy := n%Sc(7)-Sc(7)/2, n/Sc(7)
			h := uint32(frame*131+n*977) * 2654435761 // cheap deterministic noise
			if h>>29 > 4 {
				continue
			}
			c := s.At(cx+k, cy-dy)
			if c.M != sim.Empty {
				continue
			}
			s.Fill(cx+k, cy-dy, cx+k+1, cy-dy+1, sim.Lava, false)
			c.T = 1500 // fresh from deep down
			c.VX = float32(int(h>>8&255)-128) / 128 * 3 * sim.Res
			c.VY = -float32(9+int(h>>16&7)) * sim.Res
		}
	}
}

// hill raises a triangular hill with its peak at (x, top).
func hill(s *sim.Sim, x, top int) {
	for y := top; y < 192; y++ {
		w := (y - top) / 2
		wall(s, x-w, y, x+w+1, y+1)
	}
}

var advanced = []Level{
	{
		Name:    "8. Second Wind",
		Wind:    2.5,
		OpenTop: true,
		Hold:    60 * 20,
		Tick:    gusts(2.5, 60*6, 60*6),
		Brief: []string{
			"The wind here comes in gusts: six seconds of",
			"wind, then six seconds of dead calm.",
			"Wind turbines alone will flicker out in the lulls.",
			"Store spare power in Batteries while it blows and",
			"they'll carry the three beacons through the calm.",
			"A battery cell charges slowly, so you need plenty",
			"of them, and every beacon must stay lit for 20 s.",
		},
		Allow: []Allow{mat(sim.Turbine, 6), mat(sim.Battery, 12), mat(sim.Copper, 700), tool(ToolErase)},
		Build: func(s *sim.Sim) {
			floor(s)
			hill(s, 70, 100)
			hill(s, 230, 120)
			fixed(s, 140, 176, 156, 192, sim.Beacon)
			fixed(s, 222, 104, 238, 120, sim.Beacon) // on the hilltop
			fixed(s, 320, 176, 336, 192, sim.Beacon)
		},
	},
	{
		Name: "9. Deep Freeze",
		Brief: []string{
			"It's the middle of winter. The boiler has frozen",
			"solid and snow has drifted into its firebox.",
			"Clear the snow, then burn Coal under the pot to",
			"melt the ice and bring it to the boil.",
			"Cold air steals heat fast, so the boiler alone",
			"struggles to light both beacons. But cold is useful",
			"too: a Thermo Gen with fire on one side and snow",
			"on the other makes power. Wire it in, then fix",
			"the two gaps in the line to the beacons.",
		},
		Allow: []Allow{mat(sim.Coal, 1300), tool(ToolFire), mat(sim.ThermoGen, 28), mat(sim.Copper, 300),
			tool(ToolErase)},
		Build: func(s *sim.Sim) {
			winter(s)
			floor(s)
			lead := wideBoiler(s, 30, 90, 100, false)
			// the pot is solid ice and the firebox is full of snow
			fill(s, 33, 150, 117, 170, sim.Ice, false)
			fill(s, 26, 173, 124, 192, sim.Snow, false)
			fill(s, 124, 180, 170, 192, sim.Snow, false) // a drift beside it
			fixed(s, lead, 100, 170, 103, sim.Copper)
			fixed(s, 200, 100, 250, 103, sim.Copper)
			fixed(s, 247, 103, 250, 176, sim.Copper)
			fixed(s, 240, 176, 256, 192, sim.Beacon)
			fixed(s, 280, 100, 331, 103, sim.Copper)
			fixed(s, 328, 103, 331, 176, sim.Copper)
			fixed(s, 322, 176, 338, 192, sim.Beacon)
		},
	},
	{
		Name: "10. Flood Gate",
		Brief: []string{
			"This boiler sits on a volcanic vent, but it's dry.",
			"The water is trapped behind a dam up the hill,",
			"and the dam's sluice gate is packed with TNT.",
			"Set it off with the Bomb and let the water run",
			"down the chute into the pot, then bridge the wire",
			"gap. There's no eraser here, so place wire with care.",
		},
		Allow: []Allow{tool(ToolBomb), mat(sim.Copper, 200), mat(sim.Stone, 300)},
		Build: func(s *sim.Sim) {
			floor(s)
			// the reservoir, dammed on its right
			wall(s, 0, 96, 112, 100)
			fill(s, 0, 74, 100, 96, sim.Water, false)
			wall(s, 100, 0, 108, 84)
			fixed(s, 100, 84, 108, 96, sim.TNT) // the sluice gate is packed with TNT
			// a chute down to a gap in the pot's lid
			for x := 110; x < 154; x++ {
				y := 96 + (x-110)/2
				wall(s, x, y, x+1, y+6)
			}
			lead := wideBoiler(s, 150, 80, 90, true)
			fill(s, 153, 150, 227, 170, sim.Empty, false) // dry
			fill(s, 153, 126, 164, 130, sim.Empty, false) // the gap in the lid
			fixed(s, lead, 90, 260, 93, sim.Copper)
			fixed(s, 290, 90, 311, 93, sim.Copper)
			fixed(s, 308, 93, 311, 176, sim.Copper)
			fixed(s, 302, 176, 318, 192, sim.Beacon)
		},
	},
	{
		Name: "11. Eruption",
		Tick: erupt(180, 106, 60*8, 60*3),
		Hold: 60 * 25,
		Brief: []string{
			"The beacon sits in a pit at the foot of a volcano",
			"that erupts every few seconds, and lava pours in.",
			"Lava is 1200 C: Copper melts at 1085 C, so copper",
			"wire in the lava's path won't survive for long.",
			"Platinum (melts 1768 C) and Titanium (1668 C) will.",
			"Both are scarce, so only use them where lava falls.",
			"The beacon has to stay lit for 25 seconds.",
		},
		Allow: []Allow{mat(sim.Copper, 500), mat(sim.Platinum, 70), mat(sim.Titanium, 50), mat(sim.Stone, 400),
			tool(ToolErase)},
		Build: func(s *sim.Sim) {
			floor(s)
			lead := wideBoiler(s, 6, 104, 70, true)
			hill(s, 180, 110)
			// the volcano's throat
			fixed(s, 177, 107, 184, 192, sim.Lava)
			wall(s, 174, 107, 177, 110)
			wall(s, 184, 107, 187, 110)
			wall(s, 0, 40, 360, 46) // the cave roof
			fixed(s, lead, 70, 100, 73, sim.Copper)
			// the beacon sits at the bottom of a pit, right where the lava runs
			wall(s, 238, 178, 300, 192)
			fixed(s, 222, 184, 238, 192, sim.Beacon)
		},
	},
	{
		Name:    "12. The Grid",
		Wind:    2.5,
		OpenTop: true,
		Hold:    60 * 20,
		Tick:    gusts(2.5, 60*6, 60*6),
		Brief: []string{
			"Everything at once. A volcanic boiler on the left,",
			"an acid trench in the middle, a windy hill on the",
			"right, and four beacons spread across them all.",
			"The boiler can't run four beacons alone: add wind",
			"turbines up high and batteries for the calm spells,",
			"and use Gold wherever the wire touches acid.",
			"Every beacon must stay lit for 20 seconds.",
		},
		Allow: []Allow{mat(sim.Copper, 700), mat(sim.Gold, 120), mat(sim.Turbine, 3), mat(sim.Battery, 6),
			tool(ToolErase)},
		Build: func(s *sim.Sim) {
			floor(s)
			// a boiler house keeps the wind off the boiler's turbine
			wall(s, 0, 96, 92, 100)
			wall(s, 88, 100, 92, 140)
			boiler(s, 10, 80, true)
			fixed(s, 112, 176, 128, 192, sim.Beacon)
			// the acid trench, with a wall dipping into it
			wall(s, 136, 160, 140, 192)
			wall(s, 206, 160, 210, 192)
			fill(s, 140, 168, 206, 192, sim.Acid, false)
			wall(s, 170, 40, 176, 178)
			fixed(s, 220, 176, 236, 192, sim.Beacon)
			hill(s, 300, 112)
			fixed(s, 292, 96, 308, 112, sim.Beacon)
			fixed(s, 340, 176, 356, 192, sim.Beacon)
		},
	},
}
