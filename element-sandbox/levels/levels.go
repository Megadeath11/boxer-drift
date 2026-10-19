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
}

// HoldFrames is how long every beacon must stay powered to win (3 s).
const HoldFrames = 180

func tool(name string) Allow          { return Allow{Tool: name, Budget: -1} }
func mat(m sim.Mat, budget int) Allow { return Allow{Mat: m, Budget: budget} }

func wall(s *sim.Sim, x0, y0, x1, y1 int)             { s.Fill(x0, y0, x1, y1, sim.Wall, true) }
func fixed(s *sim.Sim, x0, y0, x1, y1 int, m sim.Mat) { s.Fill(x0, y0, x1, y1, m, true) }

func floor(s *sim.Sim) { wall(s, 0, sim.H-8, sim.W, sim.H) }

// boiler builds an iron pot of water at x0..x0+70 with a turbine lid and a
// copper lead rising from the lid to y=wireY. Returns the lead's x.
func boiler(s *sim.Sim, x0, wireY int, vent bool) int {
	x1 := x0 + 70
	fixed(s, x0, 130, x0+3, 192, sim.Metal) // left side + leg
	fixed(s, x1-3, 130, x1, 192, sim.Metal) // right side + leg
	fixed(s, x0, 170, x1, 173, sim.Metal)   // bottom
	s.Fill(x0+3, 150, x1-3, 170, sim.Water, false)
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
			"Pile Coal under the pot and light it with Fire.",
			"Boiling water makes steam, the steam spins the",
			"turbine, and the turbine powers the beacon.",
			"There is a gap in the wire: bridge it with Copper.",
		},
		Allow: []Allow{mat(sim.Coal, 600), tool(ToolFire), mat(sim.Copper, 200), tool(ToolErase)},
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
			boiler(s, 30, 90, true)
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
			"the part of the wire that runs through the acid.",
		},
		Allow: []Allow{mat(sim.Copper, 600), mat(sim.Gold, 160), tool(ToolErase)},
		Build: func(s *sim.Sim) {
			floor(s)
			lead := boiler(s, 20, 100, true)
			fixed(s, lead, 100, 150, 103, sim.Copper)
			wall(s, 140, 160, 144, 192) // acid trench
			wall(s, 226, 160, 230, 192)
			s.Fill(144, 168, 226, 192, sim.Acid, false)
			wall(s, 182, 0, 188, 178) // wall down into the acid
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
}
