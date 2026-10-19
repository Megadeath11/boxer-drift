// Package campaign is the linear story mode: one persistent world, a
// budget, a shop of ready-made kits, and a town whose demand keeps growing.
package campaign

import "elementsandbox/sim"

// Piece is one cell of a kit, relative to the kit's anchor (bottom centre;
// y grows downward, so pieces have y <= 0).
type Piece struct {
	X, Y int
	M    sim.Mat
}

// Kit is a ready-made structure the player buys and drops into the world.
type Kit struct {
	Name, Desc string
	Pieces     []Piece
}

// cellPrice is what one cell of each material costs, in dollars. Kits cost
// the sum of their cells; demolishing refunds half.
var cellPrice = map[sim.Mat]float64{
	sim.Metal:     0.8,
	sim.Turbine:   2.5,
	sim.Water:     0.05,
	sim.Copper:    0.05,
	sim.Battery:   2,
	sim.Coal:      0.6,
	sim.Seed:      8,
	sim.Dirt:      0.2,
	sim.ThermoGen: 3,
}

// CellPrice is the price of one cell of m (0 if it isn't sold).
func CellPrice(m sim.Mat) float64 { return cellPrice[m] }

// Refund is the share of the price paid back when the player demolishes.
const Refund = 0.5

// Cost is the kit's price.
func (k *Kit) Cost() float64 {
	c := 0.0
	for _, p := range k.Pieces {
		c += cellPrice[p.M]
	}
	return float64(int(c + 0.5))
}

// Fits reports whether the kit can stand with its anchor at (x, y).
func (k *Kit) Fits(s *sim.Sim, x, y int) bool {
	for _, p := range k.Pieces {
		if !s.Open(x+p.X, y+p.Y) {
			return false
		}
	}
	return true
}

// Drop finds where the kit lands if let go at (x, y): it falls until it
// rests on something. ok is false if it doesn't fit there at all, or if it
// came to rest on the player's own building (a wire, another kit) rather
// than on the ground.
func (k *Kit) Drop(s *sim.Sim, x, y int) (int, bool) {
	if !k.Fits(s, x, y) {
		return y, false
	}
	for y+1 < sim.H && k.Fits(s, x, y+1) {
		y++
	}
	for _, p := range k.Pieces {
		if p.Y == 0 && y+1 < sim.H {
			if c := s.At(x+p.X, y+1); !s.Open(x+p.X, y+1) && !c.IsBuilt() {
				return y, true // standing on the ground
			}
		}
	}
	return y, false
}

// Place builds the kit with its anchor at (x, y).
func (k *Kit) Place(s *sim.Sim, x, y int) {
	for _, p := range k.Pieces {
		s.Place(x+p.X, y+p.Y, p.M)
	}
}

func rect(ps []Piece, x0, y0, x1, y1 int, m sim.Mat) []Piece {
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			ps = append(ps, Piece{x, y, m})
		}
	}
	return ps
}

// WindTurbine is a mast with a rotor at the top. The iron mast carries the
// power down to the ground, so wire it at the foot.
var WindTurbine = func() *Kit {
	var ps []Piece
	ps = rect(ps, 0, -33, 1, 0, sim.Metal)       // mast
	ps = rect(ps, -1, -35, 2, -34, sim.Metal)    // nacelle
	ps = rect(ps, -3, -42, -2, -28, sim.Turbine) // rotor blades, facing the wind
	return &Kit{Name: "Wind Turbine", Pieces: ps,
		Desc: "Spins in the wind. Higher is windier. Wire it at the foot of its iron mast."}
}()

// SteamBoiler is a pot of water on legs with a turbine lid under a hood
// (so only steam turns it). Light a fire under it; wire it at the copper
// lead on its right.
var SteamBoiler = func() *Kit {
	var ps []Piece
	ps = rect(ps, -14, -11, -13, 0, sim.Metal)   // legs around an open firebox
	ps = rect(ps, 12, -11, 13, 0, sim.Metal)     //
	ps = rect(ps, -14, -13, 13, -12, sim.Metal)  // pot bottom
	ps = rect(ps, -14, -36, -13, -14, sim.Metal) // walls
	ps = rect(ps, 12, -36, 13, -14, sim.Metal)   //
	ps = rect(ps, -12, -26, 11, -14, sim.Water)  // water
	ps = rect(ps, -14, -37, 13, -37, sim.Turbine)
	ps = rect(ps, -14, -41, 13, -41, sim.Metal) // hood keeps the wind off the lid
	ps = rect(ps, -14, -40, -14, -38, sim.Metal)
	ps = rect(ps, 13, -40, 13, -38, sim.Metal)
	ps = rect(ps, 14, -41, 15, 0, sim.Copper) // lead down to the ground
	return &Kit{Name: "Steam Boiler", Pieces: ps,
		Desc: "A pot of water with a turbine lid. Burn coal underneath. Wire it at the copper lead."}
}()

// BatteryBank stores spare power for calm spells.
var BatteryBank = func() *Kit {
	var ps []Piece
	ps = rect(ps, -8, -5, 7, 0, sim.Battery)
	ps = rect(ps, 8, -5, 9, 0, sim.Copper) // terminal
	return &Kit{Name: "Battery Bank", Pieces: ps,
		Desc: "Stores spare power and gives it back when the wind drops. Wire it at the copper terminal."}
}()
