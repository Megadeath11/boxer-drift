package campaign

import (
	"bufio"
	"encoding/gob"
	"io"
	"math"

	"elementsandbox/sim"
)

// State is a campaign in progress. The world itself lives in the sim.
type State struct {
	Stage int
	Money float64
	Built map[string]int // kits bought, by name (for the tutorial)
	Step  int            // tutorial steps completed in this stage
	Frame int            // frames since the stage began
	Hold  int            // frames the goal has been met in a row
	Won   bool           // the current stage is complete
	Paid  float64        // paid for buildings cleared when this stage began
}

// Start builds the valley and begins the first stage.
func Start(s *sim.Sim) *State {
	s.Clear()
	s.Wind.Global = 0
	World(s)
	c := &State{Built: map[string]int{}}
	c.Begin(s, 0)
	return c
}

// Cur is the current stage.
func (c *State) Cur() *Stage { return &Stages[c.Stage] }

// Begin starts stage k: its setup runs on the world as it stands, and its
// money is handed over.
func (c *State) Begin(s *sim.Sim, k int) {
	c.Stage, c.Step, c.Frame, c.Hold, c.Won, c.Paid = k, 0, 0, 0, false, 0
	st := c.Cur()
	c.Money += st.Grant
	if q := st.Clear; q != nil {
		x0, y0, x1, y1 := sc(q[0]), sc(q[1]), sc(q[2]), sc(q[3])
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				cell := s.At(x, y)
				if cell.IsFixed() {
					continue
				}
				if cell.IsBuilt() {
					c.Paid += cellPrice[cell.M]
				}
				s.Fill(x, y, x+1, y+1, sim.Empty, false)
			}
		}
		if st.Cover > 0 {
			c.Paid *= st.Cover
		}
		c.Paid = math.Round(c.Paid)
		c.Money += c.Paid
	}
	if st.Setup != nil {
		st.Setup(s)
	}
}

// Next moves on to the stage after a won one. It reports false after the last.
func (c *State) Next(s *sim.Sim) bool {
	if c.Stage+1 >= len(Stages) {
		return false
	}
	c.Begin(s, c.Stage+1)
	return true
}

// Final reports whether the last stage has been won.
func (c *State) Final() bool { return c.Won && c.Stage == len(Stages)-1 }

// Update runs one frame: weather, the world, the checklist and the goal.
func (c *State) Update(s *sim.Sim, selected string) {
	st := c.Cur()
	if st.Tick != nil {
		st.Tick(s, c.Frame)
	}
	s.Step()
	c.Frame++
	ctx := &Ctx{S: s, Selected: selected, Built: c.Built}
	for c.Step < len(st.Steps) && st.Steps[c.Step].Done(ctx) {
		c.Step++
	}
	if c.Won {
		return
	}
	if c.Step >= len(st.Steps) && st.Won(s) {
		c.Hold++
	} else {
		c.Hold = 0
	}
	if c.Hold >= st.Hold {
		c.Won = true
	}
}

// Affordable is how many cells of m the money buys.
func (c *State) Affordable(m sim.Mat) int {
	p := cellPrice[m]
	if p <= 0 {
		return 0
	}
	return int(math.Floor(c.Money/p + 1e-9))
}

// Paint buys and places material m in a circle; it returns cells placed.
func (c *State) Paint(s *sim.Sim, x, y, r int, m sim.Mat) int {
	n := s.Build(x, y, r, m, c.Affordable(m))
	c.Money -= float64(n) * cellPrice[m]
	return n
}

// BuyKit drops kit k at (x, y) if it fits and the money covers it.
func (c *State) BuyKit(s *sim.Sim, k *Kit, x, y int) bool {
	cost := k.Cost()
	if c.Money < cost {
		return false
	}
	y, ok := k.Drop(s, x, y)
	if !ok {
		return false
	}
	k.Place(s, x, y)
	c.Money -= cost
	c.Built[k.Name]++
	return true
}

// Demolish pulls down the player's buildings in a circle, refunding part
// of their value. It returns the refund.
func (c *State) Demolish(s *sim.Sim, x, y, r int) float64 {
	removed := s.Demolish(x, y, r)
	got := 0.0
	for m, n := range removed {
		got += float64(n) * cellPrice[sim.Mat(m)] * Refund
	}
	c.Money += got
	return got
}

// Save writes the campaign and its world.
func (c *State) Save(w io.Writer, s *sim.Sim) error {
	if err := gob.NewEncoder(w).Encode(c); err != nil {
		return err
	}
	return s.Save(w)
}

// Load reads a campaign written by Save into s.
func Load(r io.Reader, s *sim.Sim) (*State, error) {
	br := bufio.NewReader(r) // one buffered reader, so gob can't read ahead into the world
	c := &State{}
	if err := gob.NewDecoder(br).Decode(c); err != nil {
		return nil, err
	}
	if c.Stage < 0 || c.Stage >= len(Stages) {
		return nil, io.ErrUnexpectedEOF
	}
	if c.Built == nil {
		c.Built = map[string]int{}
	}
	if err := s.Load(br); err != nil {
		return nil, err
	}
	tinRoofs(s)
	return c, nil
}

// tinRoofs updates saves from before house roofs conducted: the only fixed
// wall above the valley floor is a house roof, and it becomes metal.
func tinRoofs(s *sim.Sim) {
	for y := sc(150); y < sc(ground); y++ {
		for x := 0; x < sim.W; x++ {
			if c := s.At(x, y); c.M == sim.Wall && c.IsFixed() {
				s.Fill(x, y, x+1, y+1, sim.Metal, true)
			}
		}
	}
}
