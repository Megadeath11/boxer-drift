// Package levels defines the power-puzzle levels: get every beacon powered.
package levels

import "elementsandbox/sim"

// Tools the player may be given (materials are given by Mat).
const (
	ToolErase = "erase"
	ToolFire  = "fire"
	ToolHeat  = "heat"
	ToolCool  = "cool"
	ToolWind  = "wind"
	ToolBomb  = "bomb"
)

// Allow is one palette entry for a level. Budget is how many cells of the
// material may be placed (-1 = unlimited; ignored for tools).
type Allow struct {
	Tool   string
	Mat    sim.Mat
	Budget int
}

// Level is one puzzle.
type Level struct {
	Name    string
	Brief   []string
	Wind    float32
	OpenTop bool
	Allow   []Allow
	Build   func(s *sim.Sim)
	// MaxPPM, if set, is the CO2 level the air must be at or below (while
	// every beacon is powered) to win.
	MaxPPM float32
	// Hold, if set, is how many frames the goal must be held (default
	// HoldFrames). Longer holds make the player ride out timed events.
	Hold int
	// Tick, if set, runs scripted events (gusts, eruptions...) before each
	// simulation step; frame counts from 0 when the level starts.
	Tick func(s *sim.Sim, frame int)
}

// HoldTime is how many frames the goal must be held to win.
func (l *Level) HoldTime() int {
	if l.Hold > 0 {
		return l.Hold
	}
	return HoldFrames
}

// Step runs the level's scripted events for this frame, then the simulation.
func (l *Level) Step(s *sim.Sim, frame int) {
	if l.Tick != nil {
		l.Tick(s, frame)
	}
	s.Step()
}

// Won reports whether the level's goal is currently met.
func (l *Level) Won(s *sim.Sim) bool {
	st := s.Stats
	if st.Beacons == 0 || st.BeaconOn != st.Beacons {
		return false
	}
	return l.MaxPPM <= 0 || s.Climate.PPM <= l.MaxPPM
}

// summer locks the level in a calm summer under a roof.
func summer(s *sim.Sim) {
	s.Climate = sim.Climate{Enabled: true, Locked: true, YearFrames: 60 * 60 * 4}
	s.Climate.SetSeason(sim.Summer)
	s.Climate.Phase += 0.125
	s.AddRoof(Sc(3))
}

// HoldFrames is how long every beacon must stay powered to win (3 s).
const HoldFrames = 180

// Levels are laid out on a 360x200 design grid and scaled up to the
// simulation's resolution, so they keep their shape if the resolution changes.

// Sc converts a design-grid coordinate to simulation cells.
func Sc(v int) int { return int(float64(v) * sim.Res) }

func tool(name string) Allow { return Allow{Tool: name, Budget: -1} }

// mat offers material m; budget is in design-grid cells and scales with area.
func mat(m sim.Mat, budget int) Allow {
	return Allow{Mat: m, Budget: int(float64(budget) * sim.Res * sim.Res)}
}

func fill(s *sim.Sim, x0, y0, x1, y1 int, m sim.Mat, fixed bool) {
	s.Fill(Sc(x0), Sc(y0), Sc(x1), Sc(y1), m, fixed)
}
func wall(s *sim.Sim, x0, y0, x1, y1 int)             { fill(s, x0, y0, x1, y1, sim.Wall, true) }
func fixed(s *sim.Sim, x0, y0, x1, y1 int, m sim.Mat) { fill(s, x0, y0, x1, y1, m, true) }

func floor(s *sim.Sim) { wall(s, 0, 192, 360, 200) }

// boiler builds an iron pot of water at x0..x0+70 with a turbine lid and a
// copper lead rising from the lid to y=wireY. Returns the lead's x.
func boiler(s *sim.Sim, x0, wireY int, vent bool) int {
	return wideBoiler(s, x0, 70, wireY, vent)
}

// wideBoiler is boiler with a chosen width: wider pots boil more water.
func wideBoiler(s *sim.Sim, x0, width, wireY int, vent bool) int {
	x1 := x0 + width
	// the pot hangs over an open firebox so a fire under it can draw air
	fixed(s, x0, 130, x0+3, 173, sim.Metal)
	fixed(s, x1-3, 130, x1, 173, sim.Metal)
	fixed(s, x0, 170, x1, 173, sim.Metal) // bottom
	fill(s, x0+3, 150, x1-3, 170, sim.Water, false)
	fixed(s, x0+3, 126, x1-3, 130, sim.Turbine) // lid: steam spins it
	lead := x0 + 33
	fixed(s, lead, wireY, lead+3, 126, sim.Copper)
	if vent {
		// volcanic vent under the pot keeps it boiling
		wall(s, x0-3, 173, x0, 192)
		wall(s, x1, 173, x1+3, 192)
		fixed(s, x0+3, 173, x1-3, 192, sim.Lava)
	}
	return lead
}

// All levels in order.
var All = []Level{
	{
		Name: "1. First Spark",
		Brief: []string{
			"The pot is full of water and its lid is a turbine.",
			"Pile Coal under the pot, leaving a little air gap so",
			"it can breathe, and hold the Fire on it until it catches.",
			"Boiling water makes steam, the steam spins the",
			"turbine, and the turbine powers the beacon.",
			"There is a gap in the wire: bridge it with Copper.",
		},
		Allow: []Allow{mat(sim.Coal, 1100), tool(ToolFire), mat(sim.Copper, 300), tool(ToolErase)},
		Build: func(s *sim.Sim) {
			floor(s)
			lead := boiler(s, 40, 100, false)
			fixed(s, lead, 100, 160, 103, sim.Copper)
			fixed(s, 190, 100, 300, 103, sim.Copper)
			fixed(s, 297, 100, 300, 176, sim.Copper)
			fixed(s, 290, 176, 306, 192, sim.Beacon)
		},
	},
	{
		Name: "2. Long Haul",
		Brief: []string{
			"A volcanic vent keeps this boiler running.",
			"Two beacons need 24 power between them.",
			"Run Copper wire from the turbine's lead to both.",
			"Copper is limited, so plan a short route.",
			"Tip: a brush size of 0 draws a 1-pixel wire.",
		},
		Allow: []Allow{mat(sim.Copper, 700), tool(ToolErase)},
		Build: func(s *sim.Sim) {
			floor(s)
			wideBoiler(s, 20, 110, 90, true)
			wall(s, 150, 120, 170, 192) // a ridge to route around
			fixed(s, 220, 176, 236, 192, sim.Beacon)
			wall(s, 290, 60, 360, 66) // high ledge
			fixed(s, 320, 44, 336, 60, sim.Beacon)
		},
	},
	{
		Name: "3. Hot and Cold",
		Brief: []string{
			"No water here, just a volcanic vent.",
			"A Thermo Gen makes power from a temperature",
			"difference: put it on the lava with something",
			"cold (Ice, Snow or Liquid N2) on its other side.",
			"Then wire it to the beacon with Copper.",
		},
		Allow: []Allow{mat(sim.ThermoGen, 60), mat(sim.Ice, 600), mat(sim.Snow, 400), mat(sim.LiquidN2, 200),
			mat(sim.Copper, 500), tool(ToolErase)},
		Build: func(s *sim.Sim) {
			floor(s)
			wall(s, 20, 140, 24, 192)
			wall(s, 136, 140, 140, 192)
			fixed(s, 24, 150, 136, 192, sim.Lava)
			wall(s, 200, 150, 260, 192)
			fixed(s, 300, 176, 316, 192, sim.Beacon)
		},
	},
	{
		Name: "4. Gold Standard",
		Brief: []string{
			"The beacon is past a wall that dips into acid.",
			"Acid dissolves copper and breaks the circuit,",
			"but gold can't corrode. Use precious Gold for",
			"the part of the wire that runs through the acid:",
			"you can draw it straight into the acid.",
		},
		Allow: []Allow{mat(sim.Copper, 600), mat(sim.Gold, 160), tool(ToolErase)},
		Build: func(s *sim.Sim) {
			floor(s)
			lead := boiler(s, 20, 100, true)
			fixed(s, lead, 100, 150, 103, sim.Copper)
			wall(s, 140, 160, 144, 192) // acid trench
			wall(s, 226, 160, 230, 192)
			fixed(s, 144, 168, 226, 192, sim.Acid) // the acid is layout: it can't be erased
			wall(s, 182, 0, 188, 178)              // wall down into the acid
			fixed(s, 300, 176, 316, 192, sim.Beacon)
		},
	},
	{
		Name:    "5. Wind Farm",
		Wind:    3.5,
		OpenTop: true,
		Brief: []string{
			"A strong wind blows across the hills.",
			"Turbines spin in the wind too, but the valleys",
			"are sheltered. Put Turbines up high and wire",
			"them to both beacons. Turbines are limited!",
		},
		Allow: []Allow{mat(sim.Turbine, 24), mat(sim.Copper, 700), mat(sim.Stone, 400), tool(ToolErase)},
		Build: func(s *sim.Sim) {
			floor(s)
			for k, x := range []int{40, 160, 280} {
				h := 90 + k*15
				for y := h; y < 192; y++ {
					w := (y - h) / 2
					wall(s, x-w, y, x+w+1, y+1)
				}
			}
			fixed(s, 92, 176, 108, 192, sim.Beacon)
			fixed(s, 212, 176, 228, 192, sim.Beacon)
		},
	},
	{
		Name: "6. Clean Air",
		Brief: []string{
			"The power is on, but the valley is choking on CO2.",
			"Plant Tree Seeds on the dirt. Trees grow in the warm",
			"summer, and their leaves breathe CO2 in and oxygen out.",
			"Bring CO2 down to 500 ppm and keep the beacon lit.",
		},
		MaxPPM: 500,
		Allow:  []Allow{{Mat: sim.Seed, Budget: 14}, mat(sim.Dirt, 400), tool(ToolErase)},
		Build: func(s *sim.Sim) {
			summer(s)
			floor(s)
			lead := boiler(s, 20, 120, true)
			fixed(s, lead, 120, 312, 123, sim.Copper)
			fixed(s, 309, 123, 312, 168, sim.Copper)
			fixed(s, 302, 168, 318, 184, sim.Beacon)
			fill(s, 110, 184, 360, 192, sim.Dirt, true)
			fill(s, 110, 178, 300, 181, sim.CO2, false) // the smog
		},
	},
	{
		Name: "7. Coal Country",
		Brief: []string{
			"Only coal can boil this pot, and coal makes CO2.",
			"Keep the beacon powered AND the air at or under",
			"500 ppm. Trees take time to grow: plant them early,",
			"and don't burn more coal than you need.",
		},
		MaxPPM: 500,
		Allow: []Allow{{Mat: sim.Seed, Budget: 12}, mat(sim.Coal, 600), tool(ToolFire),
			tool(ToolErase)},
		Build: func(s *sim.Sim) {
			summer(s)
			floor(s)
			lead := boiler(s, 20, 120, false)
			fixed(s, lead, 120, 312, 123, sim.Copper)
			fixed(s, 309, 123, 312, 168, sim.Copper)
			fixed(s, 302, 168, 318, 184, sim.Beacon)
			fill(s, 110, 184, 300, 192, sim.Dirt, true)
		},
	},
}
