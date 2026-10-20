package sim

// Brush tools used by the UI.

func (s *Sim) eachInBrush(cx, cy, r int, f func(x, y, i int)) {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy > r*r+r/2 {
				continue
			}
			x, y := cx+dx, cy+dy
			if in(x, y) {
				f(x, y, y*W+x)
			}
		}
	}
}

// Paint places material m in a circle. Without replace it only fills air.
// At most limit cells are placed (limit < 0: no limit); it returns how many.
func (s *Sim) Paint(cx, cy, r int, m Mat, replace bool, limit int) int {
	return s.paint(cx, cy, r, m, replace, limit, 0)
}

// Build is Paint for the campaign: it only fills air, and marks what it
// places as the player's, so it can be demolished for a refund.
func (s *Sim) Build(cx, cy, r int, m Mat, limit int) int {
	return s.paint(cx, cy, r, m, false, limit, fBuilt)
}

func (s *Sim) paint(cx, cy, r int, m Mat, replace bool, limit int, flags uint8) int {
	st := mats[m].State
	density := float32(1)
	if st == SPowder || st == SLiquid || st == SGas || st == SFire {
		density = 0.45
	}
	placed := 0
	s.eachInBrush(cx, cy, r, func(x, y, i int) {
		c := &s.Cells[i]
		if limit >= 0 && placed >= limit {
			return
		}
		// a solid (a wire, a wall) can be drawn straight into a liquid or a
		// gas, pushing it aside: even a level's own acid or water
		into := mats[c.M].State
		sinks := st == SSolid && (into == SLiquid || into == SGas)
		if c.M == m || (c.Flags&fFixed != 0 && !sinks) || (!replace && c.M != Empty && !sinks) || !s.chance(density) {
			return
		}
		s.put(i, m)
		c.Flags |= flags
		placed++
	})
	return placed
}

// Place puts one player-built cell of m at (x, y) (a piece of a kit).
func (s *Sim) Place(x, y int, m Mat) {
	if in(x, y) {
		i := y*W + x
		s.put(i, m)
		s.Cells[i].Flags |= fBuilt
	}
}

// Open reports whether (x, y) is in the world and holds only air or gas,
// so something can be built there.
func (s *Sim) Open(x, y int) bool {
	if !in(x, y) {
		return false
	}
	st := mats[s.Cells[y*W+x].M].State
	return st == SEmpty || st == SGas
}

// Demolish removes the player's own buildings in a circle and reports how
// many cells of each material it removed.
func (s *Sim) Demolish(cx, cy, r int) (removed [NumMats]int) {
	s.eachInBrush(cx, cy, r, func(x, y, i int) {
		c := &s.Cells[i]
		if c.Flags&fBuilt == 0 || c.Flags&fFixed != 0 {
			return
		}
		removed[c.M]++
		*c = Cell{T: s.Ambient, Flags: s.clock}
	})
	return
}

// Erase removes everything in a circle except fixed level layout, and
// reports how many cells of each material it removed.
func (s *Sim) Erase(cx, cy, r int) (removed [NumMats]int) {
	s.eachInBrush(cx, cy, r, func(x, y, i int) {
		if s.Cells[i].Flags&fFixed != 0 {
			return
		}
		removed[s.Cells[i].M]++
		s.Cells[i] = Cell{T: s.Ambient, Flags: s.clock}
	})
	return
}

// Fill sets a rectangle [x0,x1)x[y0,y1) to material m. Fixed cells can't be
// erased or painted over by the player.
func (s *Sim) Fill(x0, y0, x1, y1 int, m Mat, fixed bool) {
	for y := max(y0, 0); y < min(y1, H); y++ {
		for x := max(x0, 0); x < min(x1, W); x++ {
			i := y*W + x
			s.put(i, m)
			if fixed {
				s.Cells[i].Flags |= fFixed
			}
		}
	}
}

// IsFixed reports whether the cell at index i is protected level layout.
func (c *Cell) IsFixed() bool { return c.Flags&fFixed != 0 }

// IsBuilt reports whether the player built this cell (in the campaign).
func (c *Cell) IsBuilt() bool { return c.Flags&fBuilt != 0 }

// IsPowered reports whether the cell is a running consumer or a live wire.
func (c *Cell) IsPowered() bool { return c.Flags&fPowered != 0 }

// AddHeat heats (dT>0) or cools (dT<0) everything in a circle.
func (s *Sim) AddHeat(cx, cy, r int, dT float32) {
	s.eachInBrush(cx, cy, r, func(x, y, i int) {
		c := &s.Cells[i]
		c.T += dT / mats[c.M].HeatCap
		if c.T < -273 {
			c.T = -273
		}
		if c.T > maxT {
			c.T = maxT
		}
	})
}

// Ignite sets fire to the air and lights anything flammable in a circle.
func (s *Sim) Ignite(cx, cy, r int) {
	s.eachInBrush(cx, cy, r, func(x, y, i int) {
		c := &s.Cells[i]
		switch {
		case c.M == Empty:
			if s.chance(0.3) {
				s.put(i, Fire)
			}
		case mats[c.M].Flammable:
			// a lighter flame heats things up until they reach their flash point
			c.T += 25 / mats[c.M].HeatCap
			if c.T >= mats[c.M].FlashT && s.chance(0.3) {
				s.ignite(x, y, i)
			}
		default:
			c.T += 15 / mats[c.M].HeatCap
		}
	})
}
