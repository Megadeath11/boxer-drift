package sim

// Snapshot is an in-memory copy of the world, for undo. Unlike Save it is
// uncompressed and keeps the air flow, so restoring one is instant and seamless.
type Snapshot struct {
	cells   []Cell
	frame   int
	climate Climate
	ambient float32
	openTop bool
	global  float32
	u, v    []float32
}

// Snapshot copies the world.
func (s *Sim) Snapshot() *Snapshot {
	return &Snapshot{
		cells:   append([]Cell(nil), s.Cells...),
		frame:   s.Frame,
		climate: s.Climate,
		ambient: s.Ambient,
		openTop: s.OpenTop,
		global:  s.Wind.Global,
		u:       append([]float32(nil), s.Wind.U...),
		v:       append([]float32(nil), s.Wind.V...),
	}
}

// Restore puts the world back the way it was when the snapshot was taken.
func (s *Sim) Restore(sn *Snapshot) {
	copy(s.Cells, sn.cells)
	s.Frame, s.Climate, s.Ambient, s.OpenTop = sn.frame, sn.climate, sn.ambient, sn.openTop
	s.Wind.Global = sn.global
	copy(s.Wind.U, sn.u)
	copy(s.Wind.V, sn.v)
}
