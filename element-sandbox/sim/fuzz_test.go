package sim

import (
	"math"
	"math/rand"
	"testing"
)

// TestChaosKeepsWorldValid throws random materials and tools at the world
// and checks that nothing ends up in an impossible state.
func TestChaosKeepsWorldValid(t *testing.T) {
	frames := 2000
	if testing.Short() {
		frames = 600
	}
	for _, seed := range []int64{1, 2} {
		r := rand.New(rand.NewSource(seed))
		s := New()
		s.Climate = Climate{Enabled: true, Weather: true, YearFrames: 1200}
		s.AddRoof(4)
		for f := 0; f < frames; f++ {
			if f%4 == 0 {
				x, y, rad := r.Intn(W), r.Intn(H), r.Intn(8)
				switch k := r.Intn(20); {
				case k < 13:
					s.Paint(x, y, rad, Mat(1+r.Intn(int(NumMats)-1)), r.Intn(2) == 0, -1)
				case k == 13:
					s.Erase(x, y, rad)
				case k == 14:
					s.Ignite(x, y, rad)
				case k == 15:
					s.AddHeat(x, y, rad, 400)
				case k == 16:
					s.AddHeat(x, y, rad, -400)
				case k == 17:
					s.Explode(x, y, float32(2+r.Intn(10)))
				case k == 18:
					s.Wind.AddForce(x, y, 8, r.Float32()*6-3, r.Float32()*6-3)
				default:
					s.Wind.Global = r.Float32()*8 - 4
				}
			}
			s.Step()
			if f%50 == 0 || f == frames-1 {
				checkWorld(t, s, seed, f)
			}
		}
	}
}

func checkWorld(t *testing.T, s *Sim, seed int64, f int) {
	t.Helper()
	bad := func(v float32) bool { return math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) }
	for i := range s.Cells {
		c := &s.Cells[i]
		switch {
		case c.M >= NumMats:
			t.Fatalf("seed %d frame %d: cell %d has invalid material %d", seed, f, i, c.M)
		case bad(c.T) || bad(c.VX) || bad(c.VY):
			t.Fatalf("seed %d frame %d: cell %d (%s) has T=%v V=(%v,%v)", seed, f, i, mats[c.M].Name, c.T, c.VX, c.VY)
		case c.T < -274 || c.T > maxT+500:
			t.Fatalf("seed %d frame %d: cell %d (%s) at impossible %v °C", seed, f, i, mats[c.M].Name, c.T)
		case c.M == Battery && (c.Life < 0 || c.Life > batteryCap):
			t.Fatalf("seed %d frame %d: battery holds %d", seed, f, c.Life)
		}
	}
	for _, v := range []float32{s.Ambient, s.Climate.PPM, s.Stats.Generated, s.Stats.Used} {
		if bad(v) {
			t.Fatalf("seed %d frame %d: global value is %v", seed, f, v)
		}
	}
	for i := range s.Wind.U {
		if bad(s.Wind.U[i]) || bad(s.Wind.V[i]) {
			t.Fatalf("seed %d frame %d: wind field went bad", seed, f)
		}
	}
}
