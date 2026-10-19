package sim

import "image/color"

// Render view modes.
const (
	ViewNormal = iota
	ViewHeat
)

// Render writes the world as RGBA pixels (W*H*4 bytes).
func (s *Sim) Render(pix []byte, view int) {
	bg := mats[Empty].Color
	for i := range s.Cells {
		c := &s.Cells[i]
		var r, g, b float32
		if view == ViewHeat {
			r, g, b = heatColor(c.T)
			if c.M == Empty {
				r, g, b = r*0.45, g*0.45, b*0.45
			}
		} else {
			r, g, b = s.cellColor(c, bg)
		}
		o := i * 4
		pix[o] = u8(r)
		pix[o+1] = u8(g)
		pix[o+2] = u8(b)
		pix[o+3] = 255
	}
}

func (s *Sim) cellColor(c *Cell, bg color.RGBA) (r, g, b float32) {
	info := &mats[c.M]
	br, bgc, bb := float32(bg.R), float32(bg.G), float32(bg.B)
	sh := float32(c.Shade)
	switch info.State {
	case SEmpty:
		r, g, b = br, bgc, bb
		if c.T > 60 {
			t := clamp((c.T-60)/700, 0, 1)
			r += 110 * t
			g += 35 * t
		} else if c.T < -10 {
			t := clamp((-10-c.T)/150, 0, 1)
			g += 25 * t
			b += 60 * t
		}
		return
	case SFire:
		t := clamp(float32(c.Life)/25, 0, 1)
		if c.Aux == Steam || c.Aux == CO2 {
			// clean-burning gases: blue flame
			return 90 + 120*t, 140 + 90*t, 255
		}
		return 255, 70 + 170*t + sh*2, 15 + 90*t*t
	}
	col := info.Color
	r, g, b = float32(col.R)+sh, float32(col.G)+sh, float32(col.B)+sh
	if info.State == SGas {
		a := info.Alpha
		if c.M == Smoke {
			a *= clamp(float32(c.Life)/150, 0.15, 1)
		}
		r = br + (r-br)*a
		g = bgc + (g-bgc)*a
		b = bb + (b-bb)*a
	}
	// glow when hot (incandescence)
	if c.T > 450 {
		t := clamp((c.T-450)/1300, 0, 1)
		gr, gg, gb := float32(255), 60+190*t, 20+200*t*t
		k := clamp(t*1.6, 0, 0.9)
		r += (gr - r) * k
		g += (gg - g) * k
		b += (gb - b) * k
	}
	if c.Flags&fBurning != 0 {
		k := 0.35 + s.rf()*0.4
		r += (255 - r) * k
		g += (140 - g) * k
		b += (30 - b) * k
	}
	return
}

type stop struct{ t, r, g, b float32 }

var heatStops = []stop{
	{-273, 120, 0, 160},
	{-100, 40, 40, 255},
	{0, 0, 170, 255},
	{20, 0, 120, 60},
	{100, 230, 230, 0},
	{500, 255, 120, 0},
	{1000, 255, 0, 0},
	{2500, 255, 255, 255},
}

func heatColor(t float32) (float32, float32, float32) {
	if t <= heatStops[0].t {
		s := heatStops[0]
		return s.r, s.g, s.b
	}
	for k := 1; k < len(heatStops); k++ {
		a, b := heatStops[k-1], heatStops[k]
		if t <= b.t {
			f := (t - a.t) / (b.t - a.t)
			return a.r + (b.r-a.r)*f, a.g + (b.g-a.g)*f, a.b + (b.b-a.b)*f
		}
	}
	s := heatStops[len(heatStops)-1]
	return s.r, s.g, s.b
}

func u8(v float32) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
