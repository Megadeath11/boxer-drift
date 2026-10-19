package sim

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func count(s *Sim, m Mat) int {
	n := 0
	for i := range s.Cells {
		if s.Cells[i].M == m {
			n++
		}
	}
	return n
}

func meanY(s *Sim, m Mat) float64 {
	sum, n := 0.0, 0
	for i := range s.Cells {
		if s.Cells[i].M == m {
			sum += float64(i / W)
			n++
		}
	}
	if n == 0 {
		return -1
	}
	return sum / float64(n)
}

func fill(s *Sim, x0, y0, x1, y1 int, m Mat) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			s.put(y*W+x, m)
		}
	}
}

func run(s *Sim, n int) {
	for k := 0; k < n; k++ {
		s.Step()
	}
}

func TestSandFallsAndPiles(t *testing.T) {
	s := New()
	fill(s, 170, 10, 190, 30, Sand)
	n := count(s, Sand)
	run(s, 300)
	if got := count(s, Sand); got != n {
		t.Fatalf("sand not conserved: %d -> %d", n, got)
	}
	if my := meanY(s, Sand); my < H-25 {
		t.Fatalf("sand did not fall: mean y %.1f", my)
	}
}

func TestOilFloatsOnWater(t *testing.T) {
	s := New()
	fill(s, 100, 100, 260, 120, Oil)
	fill(s, 100, 60, 260, 80, Water)
	run(s, 900)
	if mo, mw := meanY(s, Oil), meanY(s, Water); mo >= mw {
		t.Fatalf("oil (y=%.1f) should be above water (y=%.1f)", mo, mw)
	}
}

func TestMercurySinks(t *testing.T) {
	s := New()
	fill(s, 150, 150, 210, H, Water)
	fill(s, 170, 120, 190, 130, Mercury)
	run(s, 600)
	if mm, mw := meanY(s, Mercury), meanY(s, Water); mm <= mw {
		t.Fatalf("mercury (y=%.1f) should sink below water (y=%.1f)", mm, mw)
	}
}

func TestWaterBoilsIntoSteamThatRises(t *testing.T) {
	s := New()
	s.OpenTop = false
	fill(s, 150, H-10, 210, H, Water)
	for k := 0; k < 400; k++ {
		s.AddHeat(180, H-5, 30, 40)
		s.Step()
	}
	if count(s, Steam) == 0 {
		t.Fatal("no steam produced")
	}
	if my := meanY(s, Steam); my > H-20 {
		t.Fatalf("steam did not rise: mean y %.1f", my)
	}
}

func TestWaterFreezes(t *testing.T) {
	s := New()
	fill(s, 150, H-10, 210, H, Water)
	for k := 0; k < 300; k++ {
		s.AddHeat(180, H-5, 30, -30)
		s.Step()
	}
	if count(s, Ice) == 0 {
		t.Fatal("water did not freeze")
	}
}

func TestSaltDissolves(t *testing.T) {
	s := New()
	fill(s, 150, H-20, 210, H, Water)
	fill(s, 170, H-30, 190, H-25, Salt)
	run(s, 600)
	if count(s, SaltWater) == 0 {
		t.Fatal("salt did not dissolve into salt water")
	}
}

func TestGasolineBurns(t *testing.T) {
	s := New()
	fill(s, 150, H-8, 210, H, Gasoline)
	run(s, 30)
	before := count(s, Gasoline)
	s.Ignite(180, H-10, 4)
	run(s, 600)
	if after := count(s, Gasoline); after > before/4 {
		t.Fatalf("gasoline barely burned: %d -> %d", before, after)
	}
}

func TestDieselHarderToLightThanGasoline(t *testing.T) {
	burned := func(m Mat) int {
		s := New()
		s.Seed(7)
		fill(s, 150, H-8, 210, H, m)
		run(s, 20)
		before := count(s, m)
		s.Ignite(152, H-10, 2)
		run(s, 120)
		return before - count(s, m)
	}
	g, d := burned(Gasoline), burned(Diesel)
	if d >= g {
		t.Fatalf("diesel burned as fast as gasoline: gasoline %d, diesel %d", g, d)
	}
}

func TestHydrogenRisesCO2Sinks(t *testing.T) {
	s := New()
	s.OpenTop = false
	fill(s, 160, 90, 200, 110, Hydrogen)
	fill(s, 100, 90, 140, 110, CO2)
	run(s, 400)
	if my := meanY(s, Hydrogen); my > 60 {
		t.Fatalf("hydrogen did not rise: %.1f", my)
	}
	if my := meanY(s, CO2); my < 120 {
		t.Fatalf("co2 did not sink: %.1f", my)
	}
}

func TestTNTExplodes(t *testing.T) {
	s := New()
	fill(s, 100, H-40, 260, H, Stone)
	fill(s, 178, H-44, 182, H-40, TNT)
	stone := count(s, Stone)
	s.Ignite(180, H-46, 3)
	run(s, 60)
	if count(s, TNT) != 0 {
		t.Fatal("tnt did not detonate")
	}
	if count(s, Stone) >= stone {
		t.Fatal("explosion did not break any stone")
	}
}

func TestLavaMeetsWaterMakesStone(t *testing.T) {
	s := New()
	fill(s, 150, H-15, 210, H, Water)
	fill(s, 165, H-35, 195, H-25, Lava)
	run(s, 900)
	if count(s, Stone) == 0 {
		t.Fatal("lava never cooled into stone")
	}
	if count(s, Steam)+count(s, Water) == 0 {
		t.Fatal("water vanished")
	}
}

func TestWindCarriesSmoke(t *testing.T) {
	s := New()
	s.Wind.Global = 2.5
	fill(s, 60, 100, 80, 120, Smoke)
	for i := range s.Cells {
		if s.Cells[i].M == Smoke {
			s.Cells[i].Life = 3000
		}
	}
	run(s, 120)
	sum, n := 0, 0
	for i := range s.Cells {
		if s.Cells[i].M == Smoke {
			sum += i % W
			n++
		}
	}
	if n == 0 || float64(sum)/float64(n) < 120 {
		t.Fatalf("wind did not carry smoke downwind (n=%d)", n)
	}
}

func TestFireMakesUpdraft(t *testing.T) {
	s := New()
	fill(s, 160, H-6, 200, H, Coal)
	s.Ignite(180, H-8, 20)
	for i := range s.Cells {
		if s.Cells[i].M == Coal {
			s.ignite(i%W, i/W, i)
		}
	}
	run(s, 200)
	_, v := s.Wind.At(180, H-30)
	if v > -0.05 {
		t.Fatalf("no updraft above fire: v=%.3f", v)
	}
}

func BenchmarkStep(b *testing.B) {
	s := New()
	fill(s, 0, H-60, W, H, Water)
	fill(s, 50, 20, 300, 60, Sand)
	fill(s, 100, 80, 200, 100, Smoke)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Step()
	}
}

// TestSnapshot renders a demo scene to PNG when SNAPSHOT_DIR is set.
func TestSnapshot(t *testing.T) {
	dir := os.Getenv("SNAPSHOT_DIR")
	if dir == "" {
		t.Skip("set SNAPSHOT_DIR to write snapshots")
	}
	s := New()
	fill(s, 0, H-12, W, H, Stone)
	fill(s, 20, H-40, 22, H-12, Glass)
	fill(s, 80, H-40, 82, H-12, Glass)
	fill(s, 22, H-30, 80, H-12, Water)
	fill(s, 22, H-36, 80, H-30, Oil)
	fill(s, 110, H-30, 140, H-12, Wood)
	fill(s, 150, H-20, 200, H-12, Gasoline)
	fill(s, 230, H-25, 260, H-12, Sand)
	fill(s, 280, 40, 300, 60, Hydrogen)
	fill(s, 300, H-30, 340, H-12, Lava)
	fill(s, 300, 60, 340, 80, Snow)
	run(s, 60)
	s.Ignite(125, H-32, 3)
	s.Ignite(175, H-22, 3)
	for k := 0; k < 4; k++ {
		run(s, 60)
		pix := make([]byte, W*H*4)
		for v, name := range []string{"normal", "heat"} {
			s.Render(pix, v)
			img := &image.RGBA{Pix: pix, Stride: W * 4, Rect: image.Rect(0, 0, W, H)}
			f, err := os.Create(filepath.Join(dir, name+"_"+string(rune('0'+k))+".png"))
			if err != nil {
				t.Fatal(err)
			}
			png.Encode(f, img)
			f.Close()
		}
	}
}
