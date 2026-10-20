package sim

import "testing"

func TestSnapshotRestoresTheWorld(t *testing.T) {
	s := New()
	s.Fill(100, 100, 120, 110, Water, false)
	for k := 0; k < 30; k++ {
		s.Step()
	}
	sn := s.Snapshot()
	before := append([]Cell(nil), s.Cells...)
	frame := s.Frame

	s.Paint(200, 50, 8, Sand, false, -1)
	s.Explode(110, 100, 6)
	for k := 0; k < 60; k++ {
		s.Step()
	}
	s.Restore(sn)
	if s.Frame != frame {
		t.Fatalf("frame %d, want %d", s.Frame, frame)
	}
	for i := range before {
		if s.Cells[i] != before[i] {
			t.Fatalf("cell %d differs after restore: %+v, want %+v", i, s.Cells[i], before[i])
		}
	}
	s.Step() // and the world carries on from there
}
