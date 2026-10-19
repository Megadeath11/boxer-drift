package sim

import (
	"image/color"
	"math"
)

// Render view modes.
const (
	ViewNormal = iota
	ViewHeat
)

// Render writes the world as RGBA pixels (W*H*4 bytes).
func (s *Sim) Render(pix []byte, view int) {
	base := s.skyColor()
	parallelRows(H, func(y0, y1 int) {
		for y := y0; y < y1; y++ {
			// sky gradient: a little lighter and bluer toward the top
			t := 1 - float32(y)/H
			bg := color.RGBA{u8(float32(base.R) + 6*t), u8(float32(base.G) + 9*t), u8(float32(base.B) + 18*t), 255}
			for i := y * W; i < (y+1)*W; i++ {
				s.renderCell(pix, view, bg, i)
			}
		}
	})
}

func (s *Sim) renderCell(pix []byte, view int, bg color.RGBA, i int) {
	{
		c := &s.Cells[i]
		var r, g, b float32
		if view == ViewHeat {
			r, g, b = heatColor(c.T)
			if c.M == Empty {
				r, g, b = r*0.45, g*0.45, b*0.45
			}
		} else {
			r, g, b = s.cellColor(c, bg, i)
		}
		o := i * 4
		pix[o] = u8(r)
		pix[o+1] = u8(g)
		pix[o+2] = u8(b)
		pix[o+3] = 255
	}
}

func (s *Sim) cellColor(c *Cell, bg color.RGBA, i int) (r, g, b float32) {
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
	switch c.M {
	case Leaf:
		col = s.leafColor(c)
	case Roof:
		if (i/W+i%W/3)%5 == 0 {
			col.R, col.G, col.B = col.R+12, col.G+12, col.B+14
		}
	}
	r, g, b = float32(col.R)+sh, float32(col.G)+sh, float32(col.B)+sh
	if info.State != SGas {
		r, g, b = s.shade(c, info, i, r, g, b)
	}
	if info.Device != DevNone {
		r, g, b = s.deviceColor(c, info, i, r, g, b)
	} else if info.Conductive && c.Flags&fPowered != 0 && info.State == SSolid {
		// live wire: faint electric shimmer
		k := 0.12 + noise(i, s.Frame)*0.1
		r += (255 - r) * k
		g += (240 - g) * k
		b += (150 - b) * k
	}
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
		k := 0.35 + noise(i, s.Frame+7)*0.4
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

func (s *Sim) deviceColor(c *Cell, info *MatInfo, i int, r, g, b float32) (float32, float32, float32) {
	x, y := i%W, i/W
	on := c.Flags&fPowered != 0
	switch info.Device {
	case DevTurbine:
		// diagonal blades that sweep faster the more power it makes
		if (x+y+int(c.VY))%4 < 2 {
			r, g, b = r+40, g+40, b+40
		}
		k := clamp(c.VX/3, 0, 1)
		b += 50 * k
	case DevThermoGen:
		if (x+y)%3 == 0 {
			r, g, b = r+40, g+20, b-20
		}
		k := clamp(c.VX/8, 0, 1)
		r += 120 * k
		g += 50 * k
	case DevBattery:
		f := float32(c.Life) / batteryCap
		r, g, b = 30+20*f, 60+180*f, 40+40*f
		if (x+y)%5 == 0 {
			r, g, b = r*0.7, g*0.7, b*0.7
		}
	case DevLamp:
		if on {
			return 255, 248, 190
		}
	case DevBeacon:
		if on {
			p := float32(math.Sin(float64(s.Frame)*0.15))*0.5 + 0.5
			return 80 + 100*p, 255, 120 + 80*p
		}
		if (s.Frame/20)%2 == 0 {
			return 120, 40, 40
		}
	case DevHeater:
		if (x+y)%2 == 0 {
			r, g, b = r*0.7, g*0.7, b*0.7
		}
	}
	return r, g, b
}

// shade adds edge lighting: lit top edges and shadowed bottoms on solids and
// powders, bright surfaces and a gentle shimmer on liquids.
func (s *Sim) shade(c *Cell, info *MatInfo, i int, r, g, b float32) (float32, float32, float32) {
	x, y := i%W, i/W
	open := func(j int) bool {
		st := mats[s.Cells[j].M].State
		return st == SEmpty || st == SGas || st == SFire
	}
	k := float32(1)
	switch info.State {
	case SLiquid:
		if y > 0 && open(i-W) {
			k = 1.3 // surface
		} else if y > 1 && s.Cells[i-W].M == c.M && open(i-2*W) {
			k = 1.12
		}
		if c.M == Water || c.M == SaltWater {
			k += 0.05 * float32(math.Sin(float64(x)*0.35+float64(y)*0.2+float64(s.Frame)*0.07))
		}
	case SSolid, SPowder:
		if y > 0 && open(i-W) {
			k = 1.18
		} else if y+1 < H && open(i+W) {
			k = 0.78
		} else if (x > 0 && open(i-1)) || (x+1 < W && open(i+1)) {
			k = 0.9
		}
	}
	return r * k, g * k, b * k
}

// Glow writes a light map at 1/ds resolution (gw = W/ds, gh = H/ds, RGBA)
// of everything that emits light: flames, red-hot matter, lamps and
// beacons, blurred so it can be added over the scene as bloom.
func (s *Sim) Glow(dst []byte, ds int) {
	gw, gh := W/ds, H/ds
	if len(s.glow) != gw*gh*3 {
		s.glow = make([]float32, gw*gh*3)
		s.glowTmp = make([]float32, gw*gh*3)
	}
	acc := s.glow
	for k := range acc {
		acc[k] = 0
	}
	parallelRows(gh, func(g0, g1 int) {
		s.glowRows(acc, gw, ds, g0, g1)
	})
	norm := float32(1) / float32(ds*ds)
	for k := range acc {
		acc[k] *= norm
	}
	for pass := 0; pass < 2; pass++ {
		boxBlur(acc, s.glowTmp, gw, gh, 3)
	}
	for k := 0; k < gw*gh; k++ {
		dst[k*4] = u8(acc[k*3] * 255 * 1.6)
		dst[k*4+1] = u8(acc[k*3+1] * 255 * 1.6)
		dst[k*4+2] = u8(acc[k*3+2] * 255 * 1.6)
		dst[k*4+3] = 255
	}
}

func (s *Sim) glowRows(acc []float32, gw, ds, g0, g1 int) {
	for y := g0 * ds; y < g1*ds; y++ {
		row := (y / ds) * gw
		for x := 0; x < gw*ds; x++ {
			c := &s.Cells[y*W+x]
			info := &mats[c.M]
			var r, g, b float32
			switch {
			case c.M == Fire:
				t := clamp(float32(c.Life)/25, 0.3, 1)
				if c.Aux == Steam || c.Aux == CO2 {
					r, g, b = 0.25*t, 0.45*t, 1.0*t
				} else {
					r, g, b = 1.0*t, 0.5*t, 0.12*t
				}
			case c.T > 520 && info.State != SEmpty && info.State != SGas:
				t := clamp((c.T-520)/1400, 0, 1)
				r, g, b = 1.0*t, (0.25+0.55*t)*t, (0.05+0.5*t*t)*t
			case c.Flags&fBurning != 0:
				r, g, b = 0.8, 0.4, 0.1
			case info.Device == DevLamp && c.Flags&fPowered != 0:
				r, g, b = 1.0, 0.95, 0.7
			case info.Device == DevBeacon:
				if c.Flags&fPowered != 0 {
					r, g, b = 0.3, 1.0, 0.5
				} else if (s.Frame/20)%2 == 0 {
					r, g, b = 0.5, 0.05, 0.05
				}
			default:
				continue
			}
			o := (row + x/ds) * 3
			acc[o] += r
			acc[o+1] += g
			acc[o+2] += b
		}
	}
}

// boxBlur blurs an RGB float image in place (horizontal then vertical).
func boxBlur(a, tmp []float32, w, h, r int) {
	inv := 1 / float32(2*r+1)
	for y := 0; y < h; y++ {
		for c := 0; c < 3; c++ {
			var sum float32
			for x := -r; x <= r; x++ {
				sum += a[(y*w+max(0, min(w-1, x)))*3+c]
			}
			for x := 0; x < w; x++ {
				tmp[(y*w+x)*3+c] = sum * inv
				sum += a[(y*w+min(w-1, x+r+1))*3+c] - a[(y*w+max(0, x-r))*3+c]
			}
		}
	}
	for x := 0; x < w; x++ {
		for c := 0; c < 3; c++ {
			var sum float32
			for y := -r; y <= r; y++ {
				sum += tmp[(max(0, min(h-1, y))*w+x)*3+c]
			}
			for y := 0; y < h; y++ {
				a[(y*w+x)*3+c] = sum * inv
				sum += tmp[(min(h-1, y+r+1)*w+x)*3+c] - tmp[(max(0, y-r)*w+x)*3+c]
			}
		}
	}
}

type rgbf struct{ r, g, b float32 }

func lerpc(a, b rgbf, t float32) rgbf {
	return rgbf{a.r + (b.r-a.r)*t, a.g + (b.g-a.g)*t, a.b + (b.b-a.b)*t}
}

// seasonBlend returns the two seasons around now and how far we are from
// the middle of the first toward the middle of the second.
func (s *Sim) seasonBlend() (int, int, float32) {
	p := s.Climate.Phase*4 - 0.5 // 0 = mid-spring
	if p < 0 {
		p += 4
	}
	a := int(p) % 4
	return a, (a + 1) % 4, float32(p - float64(int(p)))
}

var skyBySeason = [4]rgbf{{13, 19, 26}, {11, 18, 32}, {20, 16, 18}, {22, 25, 32}}

// skyColor is the air colour: it follows the seasons and turns a dirty
// brown as CO2 and smoke build up.
func (s *Sim) skyColor() color.RGBA {
	base := mats[Empty].Color
	sky := rgbf{float32(base.R), float32(base.G), float32(base.B)}
	if s.Climate.Enabled {
		a, b, t := s.seasonBlend()
		sky = lerpc(skyBySeason[a], skyBySeason[b], t)
	}
	haze := clamp((s.Climate.PPM-500)/2500, 0, 0.6)
	sky = lerpc(sky, rgbf{48, 38, 22}, haze)
	return color.RGBA{u8(sky.r), u8(sky.g), u8(sky.b), 255}
}

var leafBySeason = [4]rgbf{{110, 190, 70}, {50, 140, 45}, {200, 110, 30}, {120, 80, 40}}
var autumnLeaf = [3]rgbf{{215, 80, 30}, {230, 150, 30}, {190, 50, 40}}

func (s *Sim) leafColor(c *Cell) color.RGBA {
	col := leafBySeason[Summer]
	if s.Climate.Enabled {
		a, b, t := s.seasonBlend()
		ca, cb := leafBySeason[a], leafBySeason[b]
		// each leaf turns its own shade of red, orange or yellow in autumn
		v := autumnLeaf[(int(c.Shade)+12)%3]
		if a == Autumn {
			ca = v
		}
		if b == Autumn {
			cb = v
		}
		col = lerpc(ca, cb, t)
	}
	if s.Climate.PPM > PPMUnhealthy {
		col = lerpc(col, rgbf{120, 100, 50}, clamp((s.Climate.PPM-PPMUnhealthy)/2000, 0, 0.7))
	}
	return color.RGBA{u8(col.r), u8(col.g), u8(col.b), 255}
}
