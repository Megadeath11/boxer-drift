package sim

import "testing"

func TestRenderAndHeatConcurrencySafe(t *testing.T) {
	s := New()
	fill(s, 0, H-60, W, H, Water)
	fill(s, 100, 80, 200, 100, Fire)
	fill(s, 300, H-90, 340, H-60, Lava)
	pix := make([]byte, W*H*4)
	glow := make([]byte, (W/3)*(H/3)*4)
	for k := 0; k < 5; k++ {
		s.Step()
		s.Render(pix, ViewNormal)
		s.Glow(glow, 3)
	}
}
