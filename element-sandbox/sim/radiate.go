package sim

// Thermal radiation: anything glowing hot beams heat at whatever it can
// "see". Each hot cell sends one ray per frame in a random direction; the
// first solid or liquid it hits absorbs energy following the Stefan–Boltzmann
// law (proportional to T^4, in kelvin). This is how a fire heats the pot
// above it across an air gap, and why you feel the heat from lava.

const (
	radiateMin = 350 // °C: below this, radiation is negligible here
	radiateK   = 3e-12
)

var radRange = int(18 * Res)

func (s *Sim) radiate() {
	for i := range s.Cells {
		c := &s.Cells[i]
		if c.T < radiateMin || c.M == Empty || mats[c.M].State == SGas {
			continue
		}
		k := s.ri(8)
		dx, dy := dx8[k], dy8[k]
		x, y := i%W, i/W
		for d := 1; d <= radRange; d++ {
			x += dx
			y += dy
			if !in(x, y) {
				break
			}
			j := y*W + x
			t := &s.Cells[j]
			st := mats[t.M].State
			if st == SEmpty || st == SGas || st == SFire {
				continue
			}
			if d == 1 {
				break // touching neighbours already exchange heat by conduction
			}
			ts, tt := c.T+273, t.T+273
			e := radiateK * (ts*ts*ts*ts - tt*tt*tt*tt)
			if e <= 0 {
				break
			}
			e = min(e, (c.T-t.T)*0.25) // never overshoot
			c.T -= e / mats[c.M].HeatCap
			t.T += e / mats[t.M].HeatCap
			break
		}
	}
}
