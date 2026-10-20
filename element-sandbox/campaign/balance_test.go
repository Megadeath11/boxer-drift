package campaign

import (
	"bytes"
	"os"
	"testing"

	"elementsandbox/sim"
)

// The balance report plays each stage several ways, from the same starting
// point, and logs which ways win and what they leave in the bank. It's slow,
// so it only runs with BALANCE=1:
//
//	BALANCE=1 go test ./campaign/ -run Balance -v -timeout 60m

type world struct {
	c *State
	s *sim.Sim
}

func fork(t *testing.T, w world) world {
	t.Helper()
	var b bytes.Buffer
	if err := w.c.Save(&b, w.s); err != nil {
		t.Fatal(err)
	}
	s := sim.New()
	c, err := Load(&b, s)
	if err != nil {
		t.Fatal(err)
	}
	return world{c, s}
}

// attempt runs the stage for up to secs and reports whether it was won.
func attempt(w world, secs int) (bool, float64) {
	for f := 0; f < 60*secs; f++ {
		w.c.Update(w.s, "")
		if w.c.Won {
			return true, float64(f) / 60
		}
	}
	return false, float64(secs)
}

// strategy is one way to play a stage: it spends money and builds, and
// reports false if it couldn't afford or place what it wanted.
type strategy struct {
	name string
	do   func(w world) bool
}

func tryBuy(w world, k *Kit, x, y int) (int, bool) {
	y, ok := k.Drop(w.s, x, y)
	if !ok || !w.c.BuyKit(w.s, k, x, y) {
		return y, false
	}
	return y, true
}

func line(w world, x0, y0, x1, y1 int) {
	n := max(abs(x1-x0), abs(y1-y0))
	for k := 0; k <= n; k++ {
		w.c.Paint(w.s, x0+(x1-x0)*k/max(n, 1), y0+(y1-y0)*k/max(n, 1), 0, sim.Copper)
	}
}

func stoke(w world, x, y, rows int) {
	for yy := y - rows + 1; yy <= y; yy++ {
		for xx := x - 11; xx <= x+10; xx++ {
			w.c.Paint(w.s, xx, yy, 0, sim.Coal)
		}
	}
	for k := 0; k < 40; k++ {
		w.s.Ignite(x-11+k%22, y-rows-1, 3)
		w.c.Update(w.s, "")
	}
}

// boilerAt is where the intended play puts the stage-2 boiler.
var boilerX = sc(150)

const boilerY = 278 // where it lands on the valley floor

func report(t *testing.T, base world, secs int, strats []strategy) map[string]world {
	out := map[string]world{}
	t.Logf("== %s: starts with $%.0f", base.c.Cur().Name, base.c.Money)
	for _, st := range strats {
		w := fork(t, base)
		before := w.c.Money
		if !st.do(w) {
			t.Logf("   %-34s couldn't afford it ($%.0f)", st.name, w.c.Money)
			continue
		}
		spent := before - w.c.Money
		won, at := attempt(w, secs)
		verdict := "FAILS"
		if won {
			verdict = "wins "
		}
		t.Logf("   %-34s %s after %4.0fs  spent $%4.0f  left $%4.0f  (houses %d/%d, %.0f ppm)",
			st.name, verdict, at, spent, w.c.Money, w.s.Stats.BeaconOn, w.s.Stats.Beacons, w.s.Climate.PPM)
		out[st.name] = w
	}
	return out
}

func TestBalanceReport(t *testing.T) {
	if os.Getenv("BALANCE") == "" {
		t.Skip("set BALANCE=1 for the balance report")
	}
	s := sim.New()
	c := Start(s)
	w0 := world{c, s}

	// 1. First Light
	r1 := report(t, w0, 60, []strategy{
		{"hill turbine + thin wire", func(w world) bool {
			w.c.Update(w.s, WindTurbine.Name)
			if _, ok := tryBuy(w, WindTurbine, sc(62), sc(60)); !ok {
				return false
			}
			line(w, sc(62)+2, 150, 492, 150)
			line(w, 492, 150, 492, 277)
			line(w, 492, 277, 494, 277)
			return true
		}},
		{"valley turbine + thin wire", func(w world) bool {
			w.c.Update(w.s, WindTurbine.Name)
			if _, ok := tryBuy(w, WindTurbine, sc(120), 200); !ok {
				return false
			}
			line(w, sc(120)+2, 260, 492, 260)
			line(w, 492, 260, 492, 277)
			line(w, 492, 277, 494, 277)
			return true
		}},
	})
	w1 := r1["hill turbine + thin wire"]
	w1.c.Next(w1.s)
	line(w1, 380, 277, 494, 277) // wire the new houses together

	// 2. Growing Town
	steam := func(rows int) func(w world) bool {
		return func(w world) bool {
			y, ok := tryBuy(w, SteamBoiler, boilerX, 230)
			if !ok {
				return false
			}
			line(w, boilerX+15, y-42, boilerX+15, 151)
			stoke(w, boilerX, y, rows)
			return true
		}
	}
	r2 := report(t, w1, 60, []strategy{
		{"nothing", func(w world) bool { return true }},
		{"two more turbines", func(w world) bool {
			for _, x := range []int{215, 240} {
				if _, ok := tryBuy(w, WindTurbine, sc(x), 200); !ok {
					return false
				}
				line(w, sc(x)+2, 250, sc(x)+2, 151)
			}
			return true
		}},
		{"boiler + 3 rows of coal", steam(3)},
		{"boiler + 5 rows of coal", steam(5)},
	})
	w2 := r2["boiler + 5 rows of coal"]
	w2.c.Next(w2.s)
	line(w2, sc(212), 277, 494, 277) // wire in the two new houses

	// 3. Still Nights
	bank := func(x int) func(w world) bool {
		return func(w world) bool {
			y, ok := tryBuy(w, BatteryBank, sc(x), 230)
			if !ok {
				return false
			}
			line(w, sc(x)+9, y-6, sc(x)+9, 151)
			return true
		}
	}
	all := func(fs ...func(w world) bool) func(w world) bool {
		return func(w world) bool {
			for _, f := range fs {
				if !f(w) {
					return false
				}
			}
			return true
		}
	}
	more := func(w world) bool { stoke(w, boilerX, boilerY, 5); return true }
	turbine := func(x int) func(w world) bool {
		return func(w world) bool {
			y, ok := tryBuy(w, WindTurbine, sc(x), 100)
			if !ok {
				return false
			}
			line(w, sc(x)+2, y-20, sc(x)+2, 150)
			return true
		}
	}
	r3 := report(t, w2, 90, []strategy{
		{"nothing", func(w world) bool { return true }},
		{"more coal", more},
		{"battery bank", bank(130)},
		{"battery bank + more coal", all(bank(130), more)},
		{"2nd hill turbine + more coal", all(turbine(48), more)},
		{"bank + turbine + more coal", all(bank(130), turbine(48), more)},
	})
	w3 := r3["battery bank + more coal"]
	if w3.c == nil || !w3.c.Won {
		t.Log("intended stage 3 play didn't win; stopping")
		return
	}
	w3.c.Next(w3.s)

	// 4. Clean Air
	seeds := func(n int) func(w world) bool {
		return func(w world) bool {
			return plant(w.c, w.s, n) == n
		}
	}
	r4 := report(t, w3, 180, []strategy{
		{"nothing", func(w world) bool { return true }},
		{"5 seeds", seeds(5)},
		{"10 seeds", seeds(10)},
		{"10 seeds + more coal", all(seeds(10), more)},
		{"10 seeds + bank", all(seeds(10), bank(100))},
	})
	var w4 world
	for _, name := range []string{"10 seeds", "10 seeds + bank", "10 seeds + more coal"} {
		if w := r4[name]; w.c != nil && w.c.Won {
			w4 = w
			t.Logf("   (going on with %q)", name)
			break
		}
	}
	if w4.c == nil {
		t.Log("no stage 4 play won; stopping")
		return
	}
	if p := os.Getenv("BALANCE_SAVE"); p != "" { // a start point for stage 5 experiments
		if f, err := os.Create(p); err == nil {
			w4.c.Save(f, w4.s)
			f.Close()
		}
	}
	w4.c.Next(w4.s)

	// 5. Hot Springs
	t.Logf("   (paid $%.0f for buildings the quake destroyed)", w4.c.Paid)
	mend := func(w world) bool {
		line(w, sc(112), 150, sc(216), 150)
		line(w, sc(168), 277, sc(216), 277)
		return true
	}
	vent := func(x int) func(w world) bool {
		return func(w world) bool {
			y, ok := tryBuy(w, SteamBoiler, sc(x), 230)
			if !ok {
				return false
			}
			line(w, sc(x)+15, y-42, sc(x)+15, 151)
			return true
		}
	}
	report(t, w4, 120, []strategy{
		{"rewire only", mend},
		{"rewire + vent boiler", all(mend, vent(140))},
		{"rewire + vent boiler + bank", all(mend, vent(140), bank(100))},
		{"rewire + 2 vent boilers", all(mend, vent(127), vent(151))},
	})
}

// TestBalanceStage5 replays stage 5 from a snapshot saved by the report
// (BALANCE_SAVE=file), for quick experiments.
func TestBalanceStage5(t *testing.T) {
	p := os.Getenv("BALANCE_LOAD")
	if p == "" {
		t.Skip("set BALANCE_LOAD to a snapshot saved with BALANCE_SAVE")
	}
	f, err := os.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	s := sim.New()
	c, err := Load(f, s)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	base := world{c, s}
	base.c.Next(base.s)
	t.Logf("paid $%.0f; starts with $%.0f", base.c.Paid, base.c.Money)
	for _, name := range []string{"none", "1 vent", "2 vents", "1 vent + bank", "2 vents + bank"} {
		w := fork(t, base)
		line(w, sc(112), 150, sc(216), 150)
		line(w, sc(168), 277, sc(216), 277)
		ok := true
		buyB := func(x int) {
			y, got := tryBuy(w, SteamBoiler, sc(x), 230)
			ok = ok && got
			line(w, sc(x)+15, y-42, sc(x)+15, 151)
		}
		switch name {
		case "1 vent", "1 vent + bank":
			buyB(140)
		case "2 vents", "2 vents + bank":
			buyB(127)
			buyB(151)
		}
		if name == "1 vent + bank" || name == "2 vents + bank" {
			y, got := tryBuy(w, BatteryBank, sc(100), 230)
			ok = ok && got
			line(w, sc(100)+9, y-6, sc(100)+9, 151)
		}
		lo, hi := float32(1e9), float32(0)
		won, at := false, 0.0
		for f := 0; f < 60*90; f++ {
			w.c.Update(w.s, "")
			if f > 600 {
				lo, hi = min(lo, w.s.Stats.Generated), max(hi, w.s.Stats.Generated)
			}
			if w.c.Won && !won {
				won, at = true, float64(f)/60
			}
		}
		t.Logf("%-16s placed %v won %v at %.0fs, power %.0f..%.0f, left $%.0f", name, ok, won, at, lo, hi, w.c.Money)
	}
}
