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
func (s *Sim) Paint(cx, cy, r int, m Mat, replace bool) {
	st := mats[m].State
	density := float32(1)
	if st == SPowder || st == SLiquid || st == SGas || st == SFire {
		density = 0.45
	}
	s.eachInBrush(cx, cy, r, func(x, y, i int) {
		c := &s.Cells[i]
		if c.M == m || (!replace && c.M != Empty) || !s.chance(density) {
			return
		}
		s.put(i, m)
	})
}

// Erase removes everything in a circle.
func (s *Sim) Erase(cx, cy, r int) {
	s.eachInBrush(cx, cy, r, func(x, y, i int) {
		s.Cells[i] = Cell{T: Ambient, Flags: s.clock}
	})
}

// AddHeat heats (dT>0) or cools (dT<0) everything in a circle.
func (s *Sim) AddHeat(cx, cy, r int, dT float32) {
	s.eachInBrush(cx, cy, r, func(x, y, i int) {
		c := &s.Cells[i]
		c.T += dT / mats[c.M].HeatCap
		if c.T < -273 {
			c.T = -273
		}
		if c.T > 6000 {
			c.T = 6000
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
