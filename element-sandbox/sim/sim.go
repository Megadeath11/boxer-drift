// Package sim is a cellular-automaton physics and chemistry simulation:
// powders, liquids and gases with densities, heat conduction, phase changes,
// combustion, explosions and a wind (air flow) field.
package sim

import (
	"math"
	"sync"
)

const (
	W              = 540
	H              = 300
	DefaultAmbient = 20 // °C, room temperature when seasons are off
	// Res is cells per unit of the original 360x200 world. Speeds and
	// sizes are multiplied by it so things look the same at any resolution.
	Res     = 1.5
	gravity = 0.25 * Res
	maxFall = 8 * Res
)

// Cell is one grid square.
type Cell struct {
	M      Mat
	Aux    Mat // for fire: what it leaves behind when it dies
	Flags  uint8
	Shade  int8
	Life   int16
	T      float32
	VX, VY float32
}

const (
	fBurning = 1 << 0
	fClock   = 1 << 1
	fFixed   = 1 << 2 // part of a level's layout: the player can't erase it
	fPowered = 1 << 3
	fBuilt   = 1 << 4 // placed by the player in the campaign (can be demolished for a refund)
)

// Sim is the whole world.
type Sim struct {
	Cells   []Cell
	Wind    *Wind
	OpenTop bool // gases may escape through the top edge
	Frame   int
	Stats   PowerStats
	Climate Climate
	Ambient float32 // current outdoor air temperature, °C
	clock   uint8
	rng     uint32

	visit     []uint32
	visitMark uint32
	members   []int32
	co2Sample []int32
	rowBusy   []bool
	breath    float32
	glow      []float32
	glowTmp   []float32
	blob      []int32
	beaconAt  []int32 // every beacon cell, gathered by power()
	blobDone  []uint32
	blobOn    []bool
	blobMark  uint32
}

var dx8 = [8]int{-1, 0, 1, -1, 1, -1, 0, 1}
var dy8 = [8]int{-1, -1, -1, 0, 0, 1, 1, 1}

// New makes an empty world at ambient temperature.
func New() *Sim {
	s := &Sim{Cells: make([]Cell, W*H), Wind: newWind(), OpenTop: true, rng: 0x9e3779b9, Ambient: DefaultAmbient,
		co2Sample: make([]int32, 0, 256)}
	s.Clear()
	return s
}

// Clear empties the world and stills the air.
func (s *Sim) Clear() {
	for i := range s.Cells {
		s.Cells[i] = Cell{T: s.Ambient}
	}
	s.Wind.reset()
}

func in(x, y int) bool { return x >= 0 && x < W && y >= 0 && y < H }

func (s *Sim) rand() uint32 {
	x := s.rng
	x ^= x << 13
	x ^= x >> 17
	x ^= x << 5
	s.rng = x
	return x
}

func (s *Sim) rf() float32           { return float32(s.rand()>>8) / (1 << 24) }
func (s *Sim) ri(n int) int          { return int(s.rand() % uint32(n)) }
func (s *Sim) chance(p float32) bool { return s.rf() < p }
func (s *Sim) sign() int {
	if s.rand()&1 == 0 {
		return -1
	}
	return 1
}

// Seed sets the random seed (for reproducible tests).
func (s *Sim) Seed(v uint32) {
	if v == 0 {
		v = 1
	}
	s.rng = v
}

// At returns the cell at (x,y). The pointer is valid until the next Step.
func (s *Sim) At(x, y int) *Cell { return &s.Cells[y*W+x] }

// put places a fresh cell of material m at index i.
func (s *Sim) put(i int, m Mat) {
	c := &s.Cells[i]
	t := mats[m].SpawnT
	if t == DefaultAmbient {
		t = s.Ambient // ordinary materials arrive at the current air temperature
	}
	*c = Cell{M: m, T: t, Shade: int8(s.ri(25) - 12), Flags: s.clock}
	switch m {
	case Fire:
		c.Life = int16(15 + s.ri(25))
	case Smoke:
		c.Life = int16(250 + s.ri(350))
	}
}

// become turns cell i into m, keeping its temperature.
func (s *Sim) become(i int, m Mat) {
	c := &s.Cells[i]
	t := c.T
	s.put(i, m)
	c.T = t
}

func (s *Sim) swap(i, j int) {
	c := s.Cells
	c[i], c[j] = c[j], c[i]
	c[i].Flags = c[i].Flags&^fClock | s.clock
	c[j].Flags = c[j].Flags&^fClock | s.clock
}

// Step advances the simulation one tick.
func (s *Sim) Step() {
	s.Frame++
	s.clock ^= fClock
	s.climateStep()
	s.Wind.step(s)
	for y := H - 1; y >= 0; y-- {
		ltr := s.rand()&1 == 0
		for k := 0; k < W; k++ {
			x := k
			if !ltr {
				x = W - 1 - k
			}
			s.update(x, y)
		}
	}
	s.heat()
	s.radiate()
	s.power()
}

func (s *Sim) update(x, y int) {
	i := y*W + x
	c := &s.Cells[i]
	if c.Flags&fClock == s.clock {
		return
	}
	c.Flags = c.Flags&^fClock | s.clock
	if c.M == Empty {
		if c.T > s.Ambient+8 || c.T < s.Ambient-8 {
			s.moveGas(x, y, i)
		}
		return
	}
	info := &mats[c.M]
	if c.M == Lava && c.Flags&fFixed != 0 {
		c.T = max(c.T, 1250) // a level's volcanic vent never cools
	}

	if s.phase(x, y, i) {
		return
	}
	if info.Flammable && c.Flags&fBurning == 0 && c.T >= info.AutoT {
		s.ignite(x, y, i)
		if c.M == Fire || c.M == Empty {
			return
		}
	}
	if c.Flags&fBurning != 0 && s.burn(x, y, i) {
		return
	}
	if s.react(x, y, i) {
		return
	}
	switch info.State {
	case SSolid:
		if info.Device != DevNone {
			s.device(x, y, i)
		} else if c.M == Roof {
			c.T += (s.Ambient - 25 - c.T) * 0.2 // the upper sky is cold
		} else if c.M == Trunk || c.M == Leaf {
			s.growTree(x, y, i)
		}
	case SPowder:
		if c.Flags&fFixed == 0 { // level layout (a dirt hill, a lava vent) stays put
			s.movePowder(x, y, i)
		}
	case SLiquid:
		if c.Flags&fFixed == 0 {
			s.moveLiquid(x, y, i)
		}
	case SGas:
		if c.M == CO2 && s.chance(co2Sink) {
			s.become(i, Empty) // soaked up by soil and water
			return
		}
		if c.M == Smoke {
			c.Life--
			if c.Life <= 0 {
				// soot settles back out of the air as ash
				if s.chance(0.25) {
					s.become(i, Ash)
				} else {
					s.become(i, Empty)
				}
				return
			}
		}
		s.moveGas(x, y, i)
	case SFire:
		s.updateFire(x, y, i)
	}
}

// emptyNear returns an empty neighbour, preferring upward ones, or -1.
func (s *Sim) emptyNear(x, y int) int {
	start := s.ri(8)
	// upward first
	for k := 0; k < 3; k++ {
		d := (start + k) % 3
		nx, ny := x+dx8[d], y+dy8[d]
		if in(nx, ny) && s.Cells[ny*W+nx].M == Empty {
			return ny*W + nx
		}
	}
	for k := 0; k < 8; k++ {
		d := (start + k) % 8
		nx, ny := x+dx8[d], y+dy8[d]
		if in(nx, ny) && s.Cells[ny*W+nx].M == Empty {
			return ny*W + nx
		}
	}
	return -1
}

func (s *Sim) spawnNear(x, y int, m Mat, t float32) {
	if j := s.emptyNear(x, y); j >= 0 {
		s.put(j, m)
		if t > s.Cells[j].T || m == Steam || m == Nitrogen || m == CO2 {
			s.Cells[j].T = t
		}
	}
}

// phase handles melting, freezing, boiling and condensing.
func (s *Sim) phase(x, y, i int) bool {
	c := &s.Cells[i]
	info := &mats[c.M]
	if info.High != Empty && c.T > info.HighT && s.chance(info.PhaseRate) {
		old := c.M
		switch old {
		case SaltWater:
			// water boils off, salt crystallises out
			s.become(i, Salt)
			s.spawnNear(x, y, Steam, 110)
			return true
		case Mud:
			s.become(i, Dirt)
			s.spawnNear(x, y, Steam, 110)
			return true
		}
		s.become(i, info.High)
		switch info.High {
		case Steam:
			c.T = float32(math.Max(float64(c.T), 110))
		case Water:
			c.T = 0.5 // latent heat of melting
		case Nitrogen:
			c.T = -190
			// liquid nitrogen expands ~700x when it boils
			s.spawnNear(x, y, Nitrogen, -190)
		case CO2:
			c.T = -75
		case FuelVapor:
			c.T = float32(math.Max(float64(c.T), 100))
		}
		return true
	}
	if info.Low != Empty && c.T < info.LowT && s.chance(info.PhaseRate) {
		s.become(i, info.Low)
		if info.Low == Ice {
			c.T = float32(math.Min(float64(c.T), -0.5))
		}
		return true
	}
	return false
}

// react handles chemistry with one random neighbour.
func (s *Sim) react(x, y, i int) bool {
	c := &s.Cells[i]
	k := s.ri(8)
	nx, ny := x+dx8[k], y+dy8[k]
	if !in(nx, ny) {
		return false
	}
	j := ny*W + nx
	n := &s.Cells[j]
	switch c.M {
	case Water:
		switch n.M {
		case Salt:
			if s.chance(0.05) {
				s.become(i, SaltWater)
				s.become(j, Empty)
				return true
			}
		case Dirt:
			if s.chance(0.004) {
				s.become(j, Mud)
				s.become(i, Empty)
				return true
			}
		case Metal:
			if canRust(n) && s.chance(0.0004) {
				s.become(j, Rust)
			}
		}
	case SaltWater:
		switch n.M {
		case Metal:
			if canRust(n) && s.chance(0.002) {
				s.become(j, Rust)
			}
		case Ice, Snow:
			if s.chance(0.02) {
				s.become(j, Water)
			}
		}
	case Salt:
		if (n.M == Ice || n.M == Snow) && s.chance(0.03) {
			s.become(j, SaltWater)
			s.become(i, Empty)
			return true
		}
	case Acid:
		if n.M == Empty || n.M == Acid || n.M == Fire || mats[n.M].AcidProof {
			return false
		}
		if mats[n.M].State == SGas || !s.chance(0.04) {
			return false
		}
		prod := Empty
		switch n.M {
		case Metal, MoltenMetal, Rust, Aluminum, MoltenAluminum, Tin, Lead:
			prod = Hydrogen // Fe + 2H+ -> Fe2+ + H2
		case Stone, Gravel, Salt:
			if s.chance(0.5) {
				prod = CO2
			}
		case Water, SaltWater:
			if !s.chance(0.05) {
				return false
			}
			prod = Water
		}
		s.become(j, prod)
		s.Cells[j].T += 20
		c.T += 15
		if s.chance(0.3) {
			s.become(i, Empty)
			return true
		}
	case Seed:
		if s.seedSprout(x, y, i) {
			return true
		}
	case FallenLeaf:
		if n.M == Dirt && s.chance(0.0003) {
			s.become(i, Dirt) // rots into soil
			return true
		}
	case Plant:
		switch n.M {
		case Water:
			if s.chance(0.03) {
				if g := s.emptyNear(x, y); g >= 0 && g < i+W {
					s.put(g, Plant)
					s.become(j, Empty)
				}
			}
		case CO2:
			if s.chance(0.01) {
				s.become(j, Oxygen)
			}
		case Chlorine:
			if s.chance(0.05) {
				s.become(i, Ash)
				return true
			}
		}
	case Chlorine:
		if n.M == Water && s.chance(0.002) {
			s.become(j, Acid)
			s.become(i, Empty)
			return true
		}
	case Gasoline, Ethanol:
		if n.M == Empty && ny < y && s.chance(mats[c.M].EvapRate) {
			s.become(i, FuelVapor)
			return true
		}
	}
	return false
}

// ---------- combustion ----------

func (s *Sim) ignite(x, y, i int) {
	c := &s.Cells[i]
	info := &mats[c.M]
	if !info.Flammable || c.Flags&fBurning != 0 {
		return
	}
	if info.State == SGas {
		// gas burns all at once: a flash of flame that expands the air
		prod := info.GasProducts[s.ri(2)]
		t := float32(math.Max(float64(c.T), float64(info.BurnT)))
		s.become(i, Fire)
		c.Aux, c.T, c.Life = prod, t, int16(5+s.ri(8))
		s.Wind.Push(x, y, 1.2)
		return
	}
	c.Flags |= fBurning
	c.Life = info.BurnLife + int16(s.ri(int(info.BurnLife)/4+1))
}

func (s *Sim) trySpread(x, y int, rate float32) {
	if !in(x, y) {
		return
	}
	i := y*W + x
	c := &s.Cells[i]
	info := &mats[c.M]
	if info.Flammable && c.Flags&fBurning == 0 && c.T >= info.FlashT && s.chance(info.Spread*rate) {
		s.ignite(x, y, i)
	}
}

// burn advances a burning cell; returns true if the cell is gone.
func (s *Sim) burn(x, y, i int) bool {
	c := &s.Cells[i]
	info := &mats[c.M]
	air, quench := false, false
	o2 := -1
	for k := 0; k < 8; k++ {
		nx, ny := x+dx8[k], y+dy8[k]
		if !in(nx, ny) {
			continue
		}
		j := ny*W + nx
		switch s.Cells[j].M {
		case Empty, Fire, Ash: // ash is porous: embers keep breathing under it
			air = true
		case Oxygen:
			air = true
			o2 = j
		case Water, SaltWater, Snow, CO2:
			quench = true
		}
	}
	if info.Explosive > 0 {
		air, quench = true, false // explosives carry their own oxidiser
	}
	if (quench && s.chance(0.15)) || (!air && s.chance(0.03)) {
		c.Flags &^= fBurning
		return false
	}
	if !air {
		return false // smouldering: no oxygen, no progress
	}
	burnT := info.BurnT
	speed := int16(1)
	if o2 >= 0 {
		// pure oxygen: hotter, faster
		burnT *= 1.4
		speed = 3
		if s.chance(0.15) {
			s.become(o2, CO2)
		}
	}
	if c.T < burnT {
		c.T += (burnT - c.T) * 0.3
	}
	if s.chance(info.FlameRate) {
		if j := s.emptyNear(x, y); j >= 0 {
			s.put(j, Fire)
			s.Cells[j].T = c.T
			s.Cells[j].Aux = info.FlameAux
			s.Cells[j].Life = int16(8 + s.ri(16))
		}
	}
	if s.chance(info.SmokeRate) {
		s.spawnNear(x, y, Smoke, c.T*0.4)
	}
	// each burnt cell of fuel releases a few cells of CO2 over its burn time
	if info.Carbon > 0 && s.chance(info.Carbon/float32(max(info.BurnLife, 1))) {
		s.spawnNear(x, y, CO2, c.T*0.5) // C + O2 -> CO2
	}
	k := s.ri(8)
	s.trySpread(x+dx8[k], y+dy8[k], 1)

	c.Life -= speed
	if c.Life > 0 {
		return false
	}
	if info.Explosive > 0 {
		s.Explode(x, y, info.Explosive)
		return true
	}
	t := c.T
	prod := Empty
	if info.BurnInto != Empty && s.chance(info.BurnChance) {
		prod = info.BurnInto
	}
	s.become(i, prod)
	s.Cells[i].T = t
	return true
}

func (s *Sim) updateFire(x, y, i int) {
	c := &s.Cells[i]
	c.Life--
	if c.Life <= 0 {
		prod := c.Aux
		if prod == Empty && s.chance(0.04) {
			prod = Smoke
		}
		t := c.T * 0.6
		s.become(i, prod)
		s.Cells[i].T = t
		return
	}
	if c.T < 800 {
		c.T = 800
	}
	for k := 0; k < 2; k++ {
		d := s.ri(8)
		nx, ny := x+dx8[d], y+dy8[d]
		if !in(nx, ny) {
			continue
		}
		switch s.Cells[ny*W+nx].M {
		case Water, SaltWater, Snow, Ice:
			s.Cells[ny*W+nx].T += 30
			s.become(i, Steam)
			return
		case CO2:
			if s.chance(0.3) {
				s.become(i, Empty)
				return
			}
		}
		s.trySpread(nx, ny, 1)
	}
	s.moveGas(x, y, i)
}

// Explode blasts everything within radius power.
func (s *Sim) Explode(cx, cy int, power float32) {
	rad := power * Res
	r := int(rad + 1)
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			x, y := cx+dx, cy+dy
			if !in(x, y) {
				continue
			}
			d := float32(math.Sqrt(float64(dx*dx + dy*dy)))
			if d > rad {
				continue
			}
			f := 1 - d/rad
			i := y*W + x
			c := &s.Cells[i]
			if c.M == Wall {
				continue
			}
			info := &mats[c.M]
			// a blast heats things toward its fireball temperature; it doesn't
			// stack, so a chain of charges can't cook the world to 10,000 °C
			if hot := s.Ambient + 1400*f; c.T < hot {
				c.T = hot
			}
			if info.Explosive > 0 && c.Flags&fBurning == 0 && !(dx == 0 && dy == 0) {
				// chain reaction, slightly delayed so it ripples outward
				c.Flags |= fBurning
				c.Life = int16(1 + s.ri(3))
				continue
			}
			switch info.State {
			case SSolid:
				if s.rf() < f*1.6-info.Hardness {
					if info.Debris != Empty {
						s.become(i, info.Debris)
					} else if f > 0.5 {
						s.become(i, Fire)
					} else {
						s.become(i, Empty)
					}
				}
			case SEmpty:
				if f > 0.25 && s.chance(0.8) {
					s.put(i, Fire)
					c.T = 1200 + 800*f
					c.Life = int16(6 + s.ri(20))
				}
			}
			if dx == 0 && dy == 0 {
				s.put(i, Fire)
				c.T = 2500
				continue
			}
			st := mats[c.M].State
			if st == SPowder || st == SLiquid {
				k := power * Res * f * 1.2 / d
				c.VX = clamp(c.VX+float32(dx)*k, -12*Res, 12*Res)
				c.VY = clamp(c.VY+float32(dy)*k-power*Res*f*0.3, -12*Res, 12*Res)
			}
			if mats[c.M].Flammable && c.Flags&fBurning == 0 && s.chance(f) {
				s.ignite(x, y, i)
			}
		}
	}
	s.Wind.Blast(cx, cy, power)
}

func clamp(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ---------- movement ----------

// canDisplace reports whether a falling/flowing particle of material src may
// move into cell j going in vertical direction dy.
func (s *Sim) canDisplace(src Mat, j, dy int) bool {
	d := s.Cells[j].M
	if d == Empty {
		return true
	}
	di := &mats[d]
	si := &mats[src]
	switch di.State {
	case SGas, SFire:
		return true
	case SLiquid:
		if dy > 0 && si.Density > di.Density && d != src {
			if si.State == SPowder {
				return s.chance(0.35)
			}
			return s.chance(0.6 * (1 - di.Viscosity*0.5))
		}
	case SPowder:
		// a liquid pushes through a much lighter powder, which floats up
		if si.State == SLiquid && dy >= 0 && di.Density*2 < si.Density && s.Cells[j].Flags&fFixed == 0 {
			return s.chance(0.3)
		}
	}
	return false
}

// travel moves cell i along its velocity. It returns the new index and
// whether it moved at all.
func (s *Sim) travel(x, y, i int) (int, bool) {
	c := &s.Cells[i]
	vx, vy := c.VX, c.VY
	ax, ay := abs32(vx), abs32(vy)
	m := ax
	if ay > m {
		m = ay
	}
	if m < 0.01 {
		return i, false
	}
	n := int(m + s.rf())
	if n < 1 {
		n = 1
	}
	sx, sy := vx/m, vy/m
	fx, fy := float32(x)+0.5, float32(y)+0.5
	cx, cy, ci := x, y, i
	moved := false
	for k := 0; k < n; k++ {
		fx += sx
		fy += sy
		nx, ny := int(math.Floor(float64(fx))), int(math.Floor(float64(fy)))
		if nx == cx && ny == cy {
			continue
		}
		if !in(nx, ny) || !s.canDisplace(s.Cells[ci].M, ny*W+nx, ny-cy) {
			cc := &s.Cells[ci]
			impact := abs32(cc.VX) + abs32(cc.VY)
			if cc.M == Nitro && impact > 4.5 {
				s.Explode(cx, cy, mats[Nitro].Explosive)
				return ci, true
			}
			if nx != cx {
				cc.VX *= -0.3
			}
			if ny != cy {
				cc.VY = 0
				cc.VX *= 0.7
			}
			break
		}
		ni := ny*W + nx
		s.swap(ci, ni)
		cx, cy, ci = nx, ny, ni
		moved = true
	}
	return ci, moved
}

// co2Sink is the chance per frame that a CO2 cell is absorbed by the
// ground and water: on its own, extra CO2 halves in about seven minutes.
const co2Sink = 1.0 / (60 * 60 * 10)

// canRust reports whether iron is bare enough to rust: level layout
// and the campaign's bought kits are protected, so boilers don't rot away.
func canRust(c *Cell) bool { return c.Flags&(fFixed|fBuilt) == 0 }

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func (s *Sim) applyForces(x, y int, c *Cell, info *MatInfo) {
	c.VY += gravity
	if c.VY > maxFall {
		c.VY = maxFall
	}
	c.VX *= 0.92
	if info.WindDrag > 0 {
		wu, wv := s.Wind.At(x, y)
		c.VX += (wu - c.VX) * info.WindDrag
		c.VY += wv * info.WindDrag
	}
}

func (s *Sim) movePowder(x, y, i int) {
	c := &s.Cells[i]
	m := c.M
	info := &mats[m]
	s.applyForces(x, y, c, info)
	ci, moved := s.travel(x, y, i)
	if moved || s.Cells[ci].M != m {
		return
	}
	// resting: slide down slopes
	if info.Slide < 1 && !s.chance(info.Slide) {
		return
	}
	if y+1 >= H {
		return
	}
	dir := s.sign()
	for k := 0; k < 2; k++ {
		nx := x + dir
		if nx >= 0 && nx < W && s.canDisplace(m, (y+1)*W+nx, 1) {
			s.swap(i, (y+1)*W+nx)
			return
		}
		dir = -dir
	}
}

func (s *Sim) moveLiquid(x, y, i int) {
	c := &s.Cells[i]
	m := c.M
	info := &mats[m]
	s.applyForces(x, y, c, info)
	ci, moved := s.travel(x, y, i)
	if moved || s.Cells[ci].M != m {
		return
	}
	// thermal convection: warmer liquid rises through cooler liquid of the same kind
	if y > 0 {
		up := &s.Cells[i-W]
		if up.M == m && c.T > up.T+3 && s.chance(0.5) {
			s.swap(i, i-W)
			return
		}
	}
	if y+1 < H {
		dir := s.sign()
		for k := 0; k < 2; k++ {
			nx := x + dir
			if nx >= 0 && nx < W && s.canDisplace(m, (y+1)*W+nx, 1) {
				s.swap(i, (y+1)*W+nx)
				return
			}
			dir = -dir
		}
	}
	if info.Viscosity > 0 && s.chance(info.Viscosity) {
		return
	}
	// flow sideways
	dir := s.sign()
	if c.VX > 0.3 {
		dir = 1
	} else if c.VX < -0.3 {
		dir = -1
	}
	for k := 0; k < 2; k++ {
		best := -1
		for d := 1; d <= info.Dispersion*3/2; d++ {
			nx := x + dir*d
			if nx < 0 || nx >= W {
				break
			}
			j := y*W + nx
			st := mats[s.Cells[j].M].State
			if st != SEmpty && st != SGas {
				break
			}
			best = j
			if y+1 < H && s.Cells[j+W].M == Empty {
				break // fall off the edge
			}
		}
		if best >= 0 {
			s.swap(i, best)
			s.Cells[best].VX = float32(dir) * 0.5
			return
		}
		dir = -dir
	}
}

func gasDensity(c *Cell) float32 {
	t := c.T
	if t < -250 {
		t = -250
	}
	return mats[c.M].Density * 293 / (t + 273)
}

// moveGas moves gases, fire and hot/cold air by buoyancy, wind and diffusion.
func (s *Sim) moveGas(x, y, i int) {
	c := &s.Cells[i]
	dens := gasDensity(c)
	const air = 1.2
	buoy := (air - dens) / (air + dens)
	wu, wv := s.Wind.At(x, y)
	jit := float32(0.9 * Res)
	if c.M == Empty {
		jit = 0.6 * Res
	}
	vx := wu + (s.rf()*2-1)*jit
	vy := -buoy*1.4*Res + wv + (s.rf()*2-1)*jit
	dx := stochRound(vx, s.rf())
	dy := stochRound(vy, s.rf())
	if dx == 0 && dy == 0 {
		return
	}
	n := iabs(dx)
	if iabs(dy) > n {
		n = iabs(dy)
	}
	if n > 4 {
		n = 4
	}
	fx, fy := float32(x)+0.5, float32(y)+0.5
	sx, sy := float32(dx)/float32(n), float32(dy)/float32(n)
	cx, cy, ci := x, y, i
	for k := 0; k < n; k++ {
		fx += sx
		fy += sy
		nx, ny := int(math.Floor(float64(fx))), int(math.Floor(float64(fy)))
		if nx == cx && ny == cy {
			continue
		}
		if !in(nx, ny) {
			if ny < 0 && s.OpenTop && s.Cells[ci].M != Empty && s.chance(0.5) {
				s.put(ci, Empty) // escapes into the sky
			}
			return
		}
		ni := ny*W + nx
		if !s.gasCanEnter(dens, s.Cells[ci].M, ni, ny-cy) {
			return
		}
		s.swap(ci, ni)
		cx, cy, ci = nx, ny, ni
	}
}

func (s *Sim) gasCanEnter(srcDens float32, src Mat, j, dy int) bool {
	d := &s.Cells[j]
	switch mats[d.M].State {
	case SEmpty:
		return true
	case SGas, SFire:
		if d.M == src {
			return false
		}
		dd := gasDensity(d)
		if dy < 0 {
			return srcDens < dd || s.chance(0.08)
		}
		if dy > 0 {
			return srcDens > dd || s.chance(0.08)
		}
		return s.chance(0.3)
	}
	return false
}

func stochRound(v, r float32) int {
	f := float32(math.Floor(float64(v)))
	if r < v-f {
		return int(f) + 1
	}
	return int(f)
}

func iabs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

var (
	condK   [NumMats][NumMats]float32 // pairwise conductance (harmonic mean)
	invCap  [NumMats]float32
	coolOf  [NumMats]float32
	tableOK bool
)

func buildHeatTables() {
	for a := range mats {
		invCap[a] = 1 / mats[a].HeatCap
		coolOf[a] = mats[a].CoolRate
		for b := range mats {
			condK[a][b] = harm(mats[a].Conduct, mats[b].Conduct)
		}
	}
	tableOK = true
}

// heat conducts heat between neighbouring cells and relaxes air/gases toward
// the ambient temperature.
func (s *Sim) heat() {
	if !tableOK {
		buildHeatTables()
	}
	// Rows are split into bands; even bands run in parallel, then odd ones.
	// A band only touches its own rows and the first row of the next band,
	// so bands running at the same time never share a cell.
	//
	// Heat spreads one cell per pass; at the finer resolution the same
	// distance on screen is Res times more cells, so several passes run each
	// frame to keep heating times the same (diffusion time grows as Res^2).
	// Rows of calm air at the outdoor temperature have nothing to conduct,
	// so they are skipped (most of the sky, most of the time).
	if len(s.rowBusy) != H {
		s.rowBusy = make([]bool, H)
	}
	amb := s.Ambient
	parallelRows(H, func(y0, y1 int) {
		for y := y0; y < y1; y++ {
			busy := false
			for _, c := range s.Cells[y*W : (y+1)*W] {
				if c.M != Empty || c.T > amb+0.5 || c.T < amb-0.5 {
					busy = true
					break
				}
			}
			s.rowBusy[y] = busy
		}
	})
	nb := workers * 2
	band := (H + nb - 1) / nb
	for pass := 0; pass < heatPasses; pass++ {
		for phase := 0; phase < 2; phase++ {
			var wg sync.WaitGroup
			for k := phase; k*band < H; k += 2 {
				wg.Add(1)
				go func(y0, y1 int) {
					defer wg.Done()
					s.heatRows(y0, y1, pass == 0)
				}(k*band, min((k+1)*band, H))
			}
			wg.Wait()
		}
	}
}

const heatPasses = 2 // ~Res^2

func (s *Sim) heatRows(y0, y1 int, relax bool) {
	cells := s.Cells
	amb := s.Ambient
	for y := y0; y < y1; y++ {
		row := y * W
		last := y+1 >= H
		if !s.rowBusy[y] && (last || !s.rowBusy[y+1]) {
			continue
		}
		for x := 0; x < W; x++ {
			i := row + x
			a := &cells[i]
			ka := &condK[a.M]
			ca := invCap[a.M]
			if x+1 < W {
				b := &cells[i+1]
				if d := a.T - b.T; d != 0 {
					f := d * ka[b.M]
					a.T -= f * ca
					b.T += f * invCap[b.M]
				}
			}
			if !last {
				b := &cells[i+W]
				if d := a.T - b.T; d != 0 {
					f := d * ka[b.M]
					a.T -= f * ca
					b.T += f * invCap[b.M]
				}
			}
			if cr := coolOf[a.M]; cr > 0 && relax {
				a.T += (amb - a.T) * cr
			}
			if a.T < -273 {
				a.T = -273
			} else if a.T > maxT {
				a.T = maxT
			}
		}
	}
}

// maxT is the hottest anything gets (the Heat tool's limit too).
const maxT = 6000

func harm(a, b float32) float32 {
	if a <= 0 || b <= 0 {
		return 0
	}
	return 2 * a * b / (a + b)
}
