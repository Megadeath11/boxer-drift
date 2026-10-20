package campaign

import (
	"elementsandbox/levels"
	"elementsandbox/sim"
)

// Item is one thing in the shop: a kit, a material sold by the cell, or a
// free tool.
type Item struct {
	Kit  *Kit
	Mat  sim.Mat // when Kit is nil and Tool is ""
	Tool string  // levels.ToolFire, or ToolDemolish
}

// ToolDemolish pulls down the player's own buildings for a partial refund.
const ToolDemolish = "demolish"

// Name is what the shop calls the item.
func (it Item) Name() string {
	switch {
	case it.Kit != nil:
		return it.Kit.Name
	case it.Tool == ToolDemolish:
		return "Demolish"
	case it.Tool == levels.ToolFire:
		return "Fire"
	case it.Mat == sim.Copper:
		return "Copper Wire"
	}
	return sim.Info(it.Mat).Name
}

// Ctx is what a tutorial step can look at.
type Ctx struct {
	S        *sim.Sim
	Selected string         // name of the selected shop item
	Built    map[string]int // kits built so far, by name
}

// Step is one tutorial instruction; it ticks off when Done reports true.
type Step struct {
	Text string
	Done func(c *Ctx) bool
}

// Stage is one chapter of the campaign. The world carries over from the
// stage before; Setup adds what's new (more houses, a change of weather).
type Stage struct {
	Name  string
	Brief []string
	Tips  []string // shown in the objectives panel
	Steps []Step   // tutorial checklist, ticked off in order
	Grant float64  // money handed over when the stage begins
	Shop  []Item   // everything on sale from this stage on (cumulative)
	Setup func(s *sim.Sim)
	// Tick runs scripted weather each frame (frame counts from the stage's start).
	Tick func(s *sim.Sim, frame int)
	// Clear, if set, is a design-grid rectangle (x0, y0, x1, y1) cleared
	// before Setup runs (new building plots, a quake); the player is paid
	// in full for anything of theirs that was there.
	Clear  *[4]int
	Cover  float64 // share of their value paid for cleared buildings (0 means all of it)
	Hold   int     // frames every house must stay lit
	MaxPPM float32 // if set, CO2 must also be at or below this
}

// Won reports whether the stage's goal is met this frame.
func (st *Stage) Won(s *sim.Sim) bool {
	if s.Stats.Beacons == 0 || s.Stats.BeaconOn != s.Stats.Beacons {
		return false
	}
	return st.MaxPPM <= 0 || s.Climate.PPM <= st.MaxPPM
}

// sc converts the 360x200 design grid to simulation cells.
func sc(v int) int { return levels.Sc(v) }

func fill(s *sim.Sim, x0, y0, x1, y1 int, m sim.Mat) {
	s.Fill(sc(x0), sc(y0), sc(x1), sc(y1), m, true)
}

// house builds a house with its left wall at design x, standing on the ground.
func house(s *sim.Sim, x int) {
	fill(s, x, 172, x+16, 186, sim.House)
	for k := 0; k < 6; k++ { // a pitched tin roof: wire can connect to it too
		fill(s, x-1+k, 166+k, x+17-k, 167+k, sim.Metal)
	}
}

// Ground level (design y) the town and valley stand on.
const ground = 186

// World builds the campaign's valley: a windy hill on the left, open
// ground in the middle and the town on the right, under a calm summer sky.
func World(s *sim.Sim) {
	s.Climate = sim.Climate{Enabled: true, Locked: true, YearFrames: 60 * 60 * 4}
	s.Climate.SetSeason(sim.Summer)
	s.Climate.Phase += 0.125
	s.Step()
	s.AddRoof(sc(3))
	fill(s, 0, 192, 360, 200, sim.Wall)
	fill(s, 0, ground, 360, 192, sim.Dirt)
	for y := 116; y < ground; y++ { // the hill
		w := (y - 116) * 3 / 4
		fill(s, 62-w, y, 63+w, y+1, sim.Dirt)
	}
}

var (
	itTurbine = Item{Kit: WindTurbine}
	itWire    = Item{Mat: sim.Copper}
	itDemol   = Item{Tool: ToolDemolish}
	itBoiler  = Item{Kit: SteamBoiler}
	itCoal    = Item{Mat: sim.Coal}
	itFire    = Item{Tool: levels.ToolFire}
	itBattery = Item{Kit: BatteryBank}
	itSeed    = Item{Mat: sim.Seed}
	itDirt    = Item{Mat: sim.Dirt}
)

func has(c *Ctx, kit string) bool { return c.Built[kit] > 0 }

// Stages are the campaign, in order.
var Stages = []Stage{
	{
		Name: "1. First Light",
		Brief: []string{
			"Welcome to the valley! You run its new power company.",
			"The town on the right has one house and no power.",
			"You have some money to spend in the shop on the right.",
			"Follow the checklist at the top left to get started.",
		},
		Grant: 260,
		Shop:  []Item{itTurbine, itWire, itDemol},
		Setup: func(s *sim.Sim) {
			s.Wind.Global = 1.3
			house(s, 330)
		},
		Steps: []Step{
			{"Pick the Wind Turbine in the shop on the right.",
				func(c *Ctx) bool { return c.Selected == WindTurbine.Name }},
			{"Click on top of the hill (left) to build it there.",
				func(c *Ctx) bool { return has(c, WindTurbine.Name) }},
			{"Wait for it to spin up: watch Power in the panel.",
				func(c *Ctx) bool { return c.S.Stats.Generated > 3 }},
			{"Pick Copper Wire. Drag a line from the mast to the house.",
				func(c *Ctx) bool { return c.S.Stats.BeaconOn > 0 }},
		},
		Tips: []string{
			"Wire costs money per pixel: keep it thin (brush 0).",
			"Demolish (or right-click) refunds half the price.",
		},
		Hold: 60 * 4,
	},
	{
		Name: "2. Growing Town",
		Brief: []string{
			"Word has spread: three more families moved in.",
			"Four houses need 48 power, but your money is tight.",
			"You can keep buying wind turbines while the money",
			"lasts, or demolish (half the money back) and switch",
			"to steam: a Steam Boiler makes far more power, but",
			"it needs Coal burning under it, and coal costs too.",
		},
		Grant: 250,
		Shop:  []Item{itTurbine, itBoiler, itWire, itCoal, itFire, itDemol},
		Setup: func(s *sim.Sim) {
			for _, x := range []int{258, 281, 304} {
				house(s, x)
			}
		},
		Tips: []string{
			"Turbines downwind of others get less wind:",
			"press V to see how the air flows.",
			"Boiler: build it, pile Coal in its firebox,",
			"light it with Fire, then wire its copper lead.",
			"Wire everything into one connected grid.",
		},
		Hold: 60 * 5,
	},
	{
		Name: "3. Still Nights",
		Brief: []string{
			"The weather has turned: now the wind blows in",
			"gusts, with long calm spells in between, and two",
			"more houses have gone up: six houses need 72 power.",
			"Coal burns away and has to be bought again, so",
			"store the wind's spare power in a Battery Bank to",
			"carry the town through the calm. Keep every house",
			"lit for 20 seconds.",
		},
		Grant: 210,
		Shop:  []Item{itTurbine, itBoiler, itBattery, itWire, itCoal, itFire, itDemol},
		Clear: &[4]int{212, 150, 256, ground},
		Setup: func(s *sim.Sim) {
			for _, x := range []int{215, 238} {
				house(s, x)
			}
		},
		Tick: gusts(1.6, 60*8, 60*6),
		Tips: []string{
			"Batteries charge slowly: a few banks charge faster.",
			"Wire batteries into the same grid as the houses.",
		},
		Hold: 60 * 20,
	},
	{
		Name: "4. Clean Air",
		Brief: []string{
			"The townsfolk are complaining about the smoke.",
			"The council wants CO2 down to 450 ppm while the",
			"lights stay on. Plant Tree Seeds: their leaves",
			"breathe the CO2 in. Fewer coal fires help too.",
		},
		Grant: 120,
		Shop:  []Item{itTurbine, itBoiler, itBattery, itWire, itCoal, itFire, itSeed, itDirt, itDemol},
		Setup: func(s *sim.Sim) {
			// smog from the town's chimneys hangs over the valley
			n := 0
			for y := sc(150); y < sc(ground) && n < 800; y++ {
				for x := sc(90); x < sc(250) && n < 800; x += 3 {
					if s.Open(x, y) && s.At(x, y).M == sim.Empty {
						s.Fill(x, y, x+1, y+1, sim.CO2, false)
						n++
					}
				}
			}
		},
		Tick:   gusts(1.6, 60*8, 60*6),
		MaxPPM: 450,
		Tips: []string{
			"Seeds sprout on dirt and grow in summer.",
			"Trees take a while to grow: plant plenty.",
			"Keep them clear of fires: a forest fire is all CO2.",
		},
		Hold: 60 * 5,
	},
	{
		Name: "5. Hot Springs",
		Brief: []string{
			"An earthquake split the valley floor and opened a",
			"volcanic vent. It flattened everything nearby;",
			"insurance paid half of what your losses were worth.",
			"Two more houses went up beside the vent: eight",
			"houses now need 96 power. A Steam Boiler standing",
			"over the vent boils for free, with no coal and no",
			"smoke. Keep every house lit, with CO2 at or under",
			"500 ppm, for 10 seconds.",
		},
		Grant: 210,
		Shop:  []Item{itTurbine, itBoiler, itBattery, itWire, itCoal, itFire, itSeed, itDirt, itDemol},
		Clear: &[4]int{116, 20, 212, ground},
		Cover: 0.5,
		Setup: func(s *sim.Sim) {
			fill(s, 118, ground, 166, 192, sim.Lava) // the vent
			for _, x := range []int{170, 192} {
				house(s, x)
			}
		},
		Tick:   gusts(1.6, 60*8, 60*6),
		MaxPPM: 500,
		Hold:   60 * 10,
	},
}

func gusts(speed float32, on, off int) func(s *sim.Sim, frame int) {
	return func(s *sim.Sim, frame int) {
		if frame%(on+off) < on {
			s.Wind.Global = speed
		} else {
			s.Wind.Global = 0
		}
	}
}
