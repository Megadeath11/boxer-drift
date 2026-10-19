package sim

import "math"

// CS is the size, in cells, of one air-flow grid square.
const (
	CS = 5
	FW = W / CS
	FH = H / CS
)

const maxWind = 6

// Wind is a coarse incompressible air-flow field (a small "stable fluids"
// solver). Hot air creates buoyant updrafts, solids block and deflect the
// flow, explosions push air outward, and gases/fire/light powders are carried
// along by it. Velocities are in cells per frame.
type Wind struct {
	U, V   []float32
	u0, v0 []float32
	p, div []float32
	solid  []bool
	tsum   []float32
	tcnt   []int16
	scnt   []int16
	Global float32 // steady wind blowing across the map (+ = right)
}

func newWind() *Wind {
	n := FW * FH
	return &Wind{
		U: make([]float32, n), V: make([]float32, n),
		u0: make([]float32, n), v0: make([]float32, n),
		p: make([]float32, n), div: make([]float32, n),
		solid: make([]bool, n), tsum: make([]float32, n),
		tcnt: make([]int16, n), scnt: make([]int16, n),
	}
}

func (w *Wind) reset() {
	for i := range w.U {
		w.U[i], w.V[i], w.p[i] = 0, 0, 0
	}
}

// Solid reports whether air-grid square (i,j) is blocked.
func (w *Wind) Solid(i, j int) bool { return w.solid[j*FW+i] }

func (w *Wind) step(s *Sim) {
	for i := range w.tsum {
		w.tsum[i], w.tcnt[i], w.scnt[i] = 0, 0, 0
	}
	for y := 0; y < H; y++ {
		frow := (y / CS) * FW
		for x := 0; x < W; x++ {
			c := &s.Cells[y*W+x]
			fi := frow + x/CS
			switch mats[c.M].State {
			case SSolid, SPowder, SLiquid:
				w.scnt[fi]++
			default:
				w.tsum[fi] += c.T
				w.tcnt[fi]++
			}
		}
	}
	for fi := range w.solid {
		w.solid[fi] = w.scnt[fi] > CS*CS*6/10
		if w.solid[fi] {
			w.U[fi], w.V[fi] = 0, 0
			continue
		}
		if w.tcnt[fi] > 0 {
			dT := w.tsum[fi]/float32(w.tcnt[fi]) - Ambient
			dT = clamp(dT, -300, 600)
			w.V[fi] -= dT * 0.0003 // buoyancy: hot air rises, cold sinks
		}
		w.U[fi] += (w.Global - w.U[fi]) * 0.02
	}

	// advect velocity along itself (semi-Lagrangian)
	copy(w.u0, w.U)
	copy(w.v0, w.V)
	for j := 0; j < FH; j++ {
		for i := 0; i < FW; i++ {
			fi := j*FW + i
			if w.solid[fi] {
				continue
			}
			px := float32(i) - w.u0[fi]/CS
			py := float32(j) - w.v0[fi]/CS
			w.U[fi] = bilerp(w.u0, px, py) * 0.985
			w.V[fi] = bilerp(w.v0, px, py) * 0.985
		}
	}
	w.project()
	for fi := range w.U {
		w.U[fi] = clamp(w.U[fi], -maxWind, maxWind)
		w.V[fi] = clamp(w.V[fi], -maxWind, maxWind)
	}
}

// project removes divergence so air flows around obstacles instead of
// piling up. Left, right and top edges are open; the floor is closed.
func (w *Wind) project() {
	vel := func(a []float32, i, j int, self float32, vertical bool) float32 {
		if j >= FH {
			return 0
		}
		if i < 0 || i >= FW || j < 0 {
			return self
		}
		if w.solid[j*FW+i] {
			return 0
		}
		return a[j*FW+i]
	}
	pres := func(i, j int, self float32) float32 {
		if j >= FH {
			return self
		}
		if i < 0 || i >= FW || j < 0 {
			return 0
		}
		if w.solid[j*FW+i] {
			return self
		}
		return w.p[j*FW+i]
	}
	for j := 0; j < FH; j++ {
		for i := 0; i < FW; i++ {
			fi := j*FW + i
			if w.solid[fi] {
				w.div[fi], w.p[fi] = 0, 0
				continue
			}
			u, v := w.U[fi], w.V[fi]
			w.div[fi] = -0.5 * (vel(w.U, i+1, j, u, false) - vel(w.U, i-1, j, u, false) +
				vel(w.V, i, j+1, v, true) - vel(w.V, i, j-1, v, true))
			w.p[fi] *= 0.5 // warm start
		}
	}
	for it := 0; it < 20; it++ {
		for j := 0; j < FH; j++ {
			for i := 0; i < FW; i++ {
				fi := j*FW + i
				if w.solid[fi] {
					continue
				}
				p := w.p[fi]
				w.p[fi] = (w.div[fi] + pres(i+1, j, p) + pres(i-1, j, p) + pres(i, j+1, p) + pres(i, j-1, p)) / 4
			}
		}
	}
	for j := 0; j < FH; j++ {
		for i := 0; i < FW; i++ {
			fi := j*FW + i
			if w.solid[fi] {
				continue
			}
			p := w.p[fi]
			w.U[fi] -= 0.5 * (pres(i+1, j, p) - pres(i-1, j, p))
			w.V[fi] -= 0.5 * (pres(i, j+1, p) - pres(i, j-1, p))
		}
	}
}

func bilerp(a []float32, x, y float32) float32 {
	x = clamp(x, 0, FW-1.001)
	y = clamp(y, 0, FH-1.001)
	i, j := int(x), int(y)
	fx, fy := x-float32(i), y-float32(j)
	i1, j1 := i+1, j+1
	if i1 >= FW {
		i1 = FW - 1
	}
	if j1 >= FH {
		j1 = FH - 1
	}
	a00, a10 := a[j*FW+i], a[j*FW+i1]
	a01, a11 := a[j1*FW+i], a[j1*FW+i1]
	return (a00*(1-fx)+a10*fx)*(1-fy) + (a01*(1-fx)+a11*fx)*fy
}

// At samples the air velocity at cell (x,y).
func (w *Wind) At(x, y int) (float32, float32) {
	fx := (float32(x)+0.5)/CS - 0.5
	fy := (float32(y)+0.5)/CS - 0.5
	return bilerp(w.U, fx, fy), bilerp(w.V, fx, fy)
}

// AddForce pushes air within radius r (cells) of (x,y).
func (w *Wind) AddForce(x, y int, r float32, fx, fy float32) {
	rr := int(r/CS) + 1
	ci, cj := x/CS, y/CS
	for j := cj - rr; j <= cj+rr; j++ {
		for i := ci - rr; i <= ci+rr; i++ {
			if i < 0 || i >= FW || j < 0 || j >= FH {
				continue
			}
			d := float32(math.Hypot(float64(i-ci), float64(j-cj)))
			if d > float32(rr) {
				continue
			}
			fi := j*FW + i
			k := 1 - d/float32(rr+1)
			w.U[fi] += fx * k
			w.V[fi] += fy * k
		}
	}
}

// Push makes a small outward puff (gas burning and expanding).
func (w *Wind) Push(x, y int, strength float32) {
	w.radial(x, y, 1.5, strength)
}

// Blast is the pressure wave of an explosion.
func (w *Wind) Blast(x, y int, power float32) {
	w.radial(x, y, power*2/CS+1.5, power*0.8)
}

func (w *Wind) radial(x, y int, rad, strength float32) {
	cx := (float32(x)+0.5)/CS - 0.5
	cy := (float32(y)+0.5)/CS - 0.5
	r := int(rad) + 1
	for j := int(cy) - r; j <= int(cy)+r; j++ {
		for i := int(cx) - r; i <= int(cx)+r; i++ {
			if i < 0 || i >= FW || j < 0 || j >= FH {
				continue
			}
			dx, dy := float32(i)-cx, float32(j)-cy
			d := float32(math.Hypot(float64(dx), float64(dy)))
			if d > rad || d < 0.01 {
				continue
			}
			k := strength * (1 - d/rad) / d
			fi := j*FW + i
			w.U[fi] += dx * k
			w.V[fi] += dy * k
		}
	}
}
