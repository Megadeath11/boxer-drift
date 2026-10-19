package sim

import (
	"compress/gzip"
	"encoding/gob"
	"io"
)

// saved is what Save writes: the world and its weather, not the derived
// state (wind field, power stats) that rebuilds itself in a frame or two.
type saved struct {
	Cells   []Cell
	OpenTop bool
	Frame   int
	Climate Climate
	Ambient float32
	Global  float32
}

// Save writes the world, compressed.
func (s *Sim) Save(w io.Writer) error {
	zw := gzip.NewWriter(w)
	err := gob.NewEncoder(zw).Encode(saved{s.Cells, s.OpenTop, s.Frame, s.Climate, s.Ambient, s.Wind.Global})
	if cerr := zw.Close(); err == nil {
		err = cerr
	}
	return err
}

// Load replaces the world with one written by Save.
func (s *Sim) Load(r io.Reader) error {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	var v saved
	if err := gob.NewDecoder(zr).Decode(&v); err != nil {
		return err
	}
	if len(v.Cells) != len(s.Cells) {
		return io.ErrUnexpectedEOF
	}
	copy(s.Cells, v.Cells)
	s.OpenTop, s.Frame, s.Climate, s.Ambient = v.OpenTop, v.Frame, v.Climate, v.Ambient
	s.Wind.reset()
	s.Wind.Global = v.Global
	return nil
}
