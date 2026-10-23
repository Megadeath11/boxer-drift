package sim

import "math"

// Seasons.
const (
	Spring = iota
	Summer
	Autumn
	Winter
)

// SeasonNames are indexed by season.
var SeasonNames = [4]string{"Spring", "Summer", "Autumn", "Winter"}

// Climate drives the seasons, the weather and the greenhouse effect.
type Climate struct {
	Enabled    bool    // seasons cycle; otherwise it is always DefaultAmbient
	Locked     bool    // stay in the current season
	Weather    bool    // rain in spring/autumn, snow in winter
	YearFrames int     // frames per full year
	Phase      float64 // 0..1 through the year; 0 = start of spring

	// Pollution (updated every frame)
	CO2Cells, SmokeCells int
	Leaves               int
	PPM                  float32 // CO2 concentration shown to the player
	Warming              float32 // extra °C from the greenhouse effect
}

const (
	basePPM = 280 // pre-industrial CO2
	// ppm added per CO2 cell. One "cell" of pure CO2 in the open air is a lot.
	ppmPerCell = 0.5
	// PPMUnhealthy is where trees start to suffer.
	PPMUnhealthy = 1200
)

// Season returns the current season.
func (c *Climate) Season() int { return int(c.Phase*4) % 4 }

// SeasonProgress returns how far through the current season we are (0..1).
func (c *Climate) SeasonProgress() float64 { return c.Phase*4 - math.Floor(c.Phase*4) }

// SetSeason jumps to the start of a season.
func (c *Climate) SetSeason(k int) { c.Phase = float64(k%4) / 4 }

// growth is how fast plants grow and photosynthesise this season (0..1).
func (s *Sim) growth() float32 {
	if !s.Climate.Enabled {
		return 1
	}
	switch s.Climate.Season() {
	case Spring:
		return 1
	case Summer:
		return 1
	case Autumn:
		return 0.35
	}
	return 0
}

// climateStep advances the year, sets the outdoor temperature and makes
// weather.
func (s *Sim) climateStep() {
	c := &s.Climate
	c.PPM = basePPM + float32(c.CO2Cells)*ppmPerCell + float32(c.SmokeCells)*0.2
	// CO2 traps heat: about +1 °C per 150 ppm above pre-industrial, capped
	c.Warming = clamp((c.PPM-basePPM)/150, 0, 12)

	if !c.Enabled {
		s.Ambient = DefaultAmbient + c.Warming
		return
	}
	if !c.Locked && c.YearFrames > 0 {
		c.Phase += 1 / float64(c.YearFrames)
		if c.Phase >= 1 {
			c.Phase -= 1
		}
	}
	// coldest in mid-winter, warmest in mid-summer
	t := 10 + 16*math.Sin(2*math.Pi*(c.Phase-0.125))
	s.Ambient = float32(t) + c.Warming

	if !c.Weather {
		return
	}
	// precipitation falls from just under the top of the world
	top := 0
	for top < H-1 && s.Cells[top*W+W/2].M == Roof {
		top++
	}
	switch c.Season() {
	case Spring, Autumn:
		if s.chance(0.5) {
			s.drop(top, Water)
		}
	case Winter:
		if s.chance(0.6) {
			s.drop(top, Snow)
		}
	}
}

func (s *Sim) drop(y int, m Mat) {
	x := s.ri(W)
	i := y*W + x
	if s.Cells[i].M == Empty {
		s.put(i, m)
	}
}

// LeafUptake is CO2 cells absorbed per leaf cell per frame in summer.
const LeafUptake = 0.0008

// breathe lets the forest draw CO2 out of the (well-mixed) atmosphere:
// leaves anywhere take up CO2 from anywhere, faster in the growing season.
func (s *Sim) breathe(leaves int) {
	s.breath += float32(leaves) * LeafUptake * s.growth()
	for s.breath >= 1 && len(s.co2Sample) > 0 {
		s.breath--
		k := s.ri(len(s.co2Sample))
		i := s.co2Sample[k]
		s.co2Sample[k] = s.co2Sample[len(s.co2Sample)-1]
		s.co2Sample = s.co2Sample[:len(s.co2Sample)-1]
		if s.Cells[i].M == CO2 {
			s.become(int(i), Oxygen)
		}
	}
	if len(s.co2Sample) == 0 {
		s.breath = 0
	}
}

// ---------- trees ----------

// seedSprout lets a resting seed on soil start a tree.
func (s *Sim) seedSprout(x, y, i int) bool {
	if y+1 >= H || s.Ambient < 3 || !s.chance(0.01*s.growth()) {
		return false
	}
	switch s.Cells[i+W].M {
	case Dirt, Mud, FallenLeaf:
	default:
		return false
	}
	s.become(i, Trunk)
	c := &s.Cells[i]
	c.Life = int16(float32(16+s.ri(22)) * Res) // how much taller it will grow
	return true
}

// growTree runs a living trunk or leaf cell.
//
// A trunk cell with Life > 0 is a growing tip; Life == -1 marks the crown,
// which (re)grows a ball of leaves around it each spring and summer.
const (
	growMaxT = 50  // °C: trees don't push new growth into air hotter than this
	scorchT  = 100 // °C: a trunk this hot dies
)

func (s *Sim) growTree(x, y, i int) {
	c := &s.Cells[i]
	g := s.growth()
	if c.M == Leaf {
		s.leaf(x, y, i, g)
		return
	}
	if c.T > scorchT {
		// scorched: the tree dies and becomes dead wood, which burns once
		// and is gone, instead of growing fresh leaves into the heat forever
		s.become(i, Wood)
		return
	}
	switch {
	case c.Life > 0:
		if !s.chance(0.1 * g) {
			return
		}
		nx := x
		if s.chance(0.12) {
			nx += s.sign() // a little wiggle
		}
		ny := y - 1
		if !in(nx, ny) {
			c.Life = -1
			return
		}
		j := ny*W + nx
		st := mats[s.Cells[j].M].State
		if st != SEmpty && st != SGas && s.Cells[j].M != Leaf || s.Cells[j].T > growMaxT {
			c.Life = -1 // blocked: this is as tall as it gets
			return
		}
		life := c.Life - 1
		s.put(j, Trunk)
		s.Cells[j].Life = life
		if life == 0 {
			s.Cells[j].Life = -1
		}
		c.Life = 0
		// thicken the lower trunk
		if life > 12 && x+1 < W && s.Cells[i+1].M == Empty {
			s.put(i+1, Trunk)
		}
		// now and then a short branch sprouts sideways near the top
		if life > 2 && life < 14 && s.chance(0.18) {
			d := s.sign()
			for b := 1; b <= 2+s.ri(4); b++ {
				bx, by := x+d*b, y-b/2
				if !in(bx, by) || s.Cells[by*W+bx].M != Empty || s.Cells[by*W+bx].T > growMaxT {
					break
				}
				s.put(by*W+bx, Trunk)
			}
		}
	case c.Life == -1:
		// the crown fills a rounded canopy with leaves
		r := float32(8 * Res)
		for k := 0; k < 4; k++ {
			if !s.chance(0.5 * g) {
				continue
			}
			dx := (s.rf()*2 - 1) * r
			dy := (s.rf()*2-1)*r*0.75 - r*0.15
			if dx*dx+dy*dy*1.6 > r*r*(0.75+0.25*s.rf()) {
				continue
			}
			lx, ly := x+int(dx), y+int(dy)
			if in(lx, ly) && s.Cells[ly*W+lx].M == Empty && s.Cells[ly*W+lx].T <= growMaxT {
				s.put(ly*W+lx, Leaf)
			}
		}
	}
}

// leaf photosynthesises, filters smoke, and changes with the seasons.
func (s *Sim) leaf(x, y, i int, g float32) {
	cl := &s.Climate
	// autumn and winter: leaves drop; heavy pollution kills them too
	drop := float32(0)
	if cl.Enabled {
		switch cl.Season() {
		case Autumn:
			drop = 0.0004 + 0.003*float32(cl.SeasonProgress())
		case Winter:
			drop = 0.02
		}
	}
	if cl.PPM > PPMUnhealthy {
		drop += (cl.PPM - PPMUnhealthy) * 0.000002
	}
	if drop > 0 && s.chance(drop) {
		s.become(i, FallenLeaf)
		return
	}
	k := s.ri(8)
	nx, ny := x+dx8[k], y+dy8[k]
	if !in(nx, ny) {
		return
	}
	j := ny*W + nx
	switch s.Cells[j].M {
	case CO2: // 6CO2 + 6H2O -> C6H12O6 + 6O2
		if s.chance(0.25 * g) {
			s.become(j, Oxygen)
		}
	case Smoke:
		if s.chance(0.05) {
			s.become(j, Empty)
		}
	case Chlorine:
		if s.chance(0.05) {
			s.become(i, FallenLeaf)
		}
	}
}

// AddRoof puts a roof of the given thickness across the top of the world,
// or removes it (thickness 0). Gases then gather under it instead of
// escaping, and steam condenses on its cold underside.
func (s *Sim) AddRoof(thick int) {
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			i := y*W + x
			if y < thick {
				s.put(i, Roof)
				s.Cells[i].Flags |= fFixed
			} else if s.Cells[i].M == Roof {
				s.Cells[i] = Cell{T: s.Ambient, Flags: s.clock}
			}
		}
	}
	s.OpenTop = thick == 0
}
