package sim

import "math"

const (
	batteryCap       = 3000 // stored power units (kept in Cell.Life)
	batteryCharge    = 2    // max charge per frame
	batteryDischarge = 6    // max discharge per frame
)

// device runs a power device cell each frame: generators compute their
// output (kept in VX, smoothed), consumers react to being powered.
func (s *Sim) device(x, y, i int) {
	c := &s.Cells[i]
	info := &mats[c.M]
	switch info.Device {
	case DevTurbine:
		out := float32(0)
		for k := 0; k < 8; k++ {
			nx, ny := x+dx8[k], y+dy8[k]
			if !in(nx, ny) {
				continue
			}
			j := ny*W + nx
			if s.Cells[j].M == Steam {
				// steam pushing through the blades gives up its energy and
				// condenses back into water
				out += 1.5
				if s.chance(0.06) {
					s.become(j, Water)
				}
			}
		}
		wu, wv := s.Wind.At(x, y)
		// wind turbines face the wind: they catch horizontal wind above a
		// cut-in speed, with output rising steeply with speed. Rising hot
		// air (a vertical draught) doesn't drive them.
		_ = wv
		if v := abs32(wu) - 0.4; v > 0 {
			out += 1.5 * v * v
		}
		c.VX = c.VX*0.85 + out*0.15
		c.VY += c.VX * 0.08 // blade angle, for drawing
	case DevThermoGen:
		hi, lo := float32(-1e9), float32(1e9)
		for k := 1; k < 7; k++ { // 4-neighbours: indices 1,3,4,6
			if k == 2 || k == 5 {
				continue
			}
			nx, ny := x+dx8[k], y+dy8[k]
			if !in(nx, ny) {
				continue
			}
			t := s.Cells[ny*W+nx].T
			hi = max(hi, t)
			lo = min(lo, t)
		}
		out := float32(0)
		if hi > lo+30 {
			out = (hi - lo - 30) * 0.004 // Seebeck effect
		}
		c.VX = c.VX*0.85 + out*0.15
	case DevHeater:
		if c.Flags&fPowered != 0 && c.T < 1100 {
			c.T += 8
		}
	case DevLamp:
		if c.Flags&fPowered != 0 && c.T < 70 {
			c.T += 0.5
		}
	}
}

// PowerStats summarises the electric grid for the UI.
type PowerStats struct {
	Generated, Used   float32
	Beacons, BeaconOn int
	Stored            float32
}

// power finds every connected circuit of conductive cells and shares the
// generated power among its consumers, using batteries as a buffer.
func (s *Sim) power() {
	n := len(s.Cells)
	if len(s.visit) != n {
		s.visit = make([]uint32, n)
	}
	s.visitMark++
	if s.visitMark == 0 {
		for i := range s.visit {
			s.visit[i] = 0
		}
		s.visitMark = 1
	}
	st := PowerStats{}
	s.beaconAt = s.beaconAt[:0]
	co2, smoke, leaves := 0, 0, 0
	s.co2Sample = s.co2Sample[:0]
	for i := range s.Cells {
		c := &s.Cells[i]
		c.Flags &^= fPowered
		switch c.M {
		case Beacon:
			s.beaconAt = append(s.beaconAt, int32(i))
		case CO2:
			co2++
			// reservoir-sample some CO2 cells for the forest to breathe in
			if len(s.co2Sample) < cap(s.co2Sample) {
				s.co2Sample = append(s.co2Sample, int32(i))
			} else if k := s.ri(co2); k < len(s.co2Sample) {
				s.co2Sample[k] = int32(i)
			}
		case Smoke:
			smoke++
		case Leaf:
			leaves++
		}
	}
	s.Climate.CO2Cells, s.Climate.SmokeCells, s.Climate.Leaves = co2, smoke, leaves
	s.breathe(leaves)
	if len(s.beaconAt) > 0 {
		st.Beacons = s.labelBeacons()
	}
	s.blobMark++
	for seed := range s.Cells {
		if s.visit[seed] == s.visitMark || !mats[s.Cells[seed].M].Conductive {
			continue
		}
		// flood-fill one circuit (4-connected)
		s.members = s.members[:0]
		s.members = append(s.members, int32(seed))
		s.visit[seed] = s.visitMark
		for k := 0; k < len(s.members); k++ {
			i := int(s.members[k])
			x, y := i%W, i/W
			for d := 0; d < 4; d++ {
				nx, ny := x, y
				switch d {
				case 0:
					nx--
				case 1:
					nx++
				case 2:
					ny--
				case 3:
					ny++
				}
				if !in(nx, ny) {
					continue
				}
				j := ny*W + nx
				if s.visit[j] != s.visitMark && mats[s.Cells[j].M].Conductive {
					s.visit[j] = s.visitMark
					s.members = append(s.members, int32(j))
				}
			}
		}
		s.resolveCircuit(&st)
	}
	s.Stats = st
}

// labelBeacons gives each connected beacon its own id (in s.blob) and
// returns how many beacons there are. A beacon is one object however many
// pixels it covers. Only beacon cells' entries in s.blob are meaningful.
func (s *Sim) labelBeacons() int {
	if len(s.blob) != len(s.Cells) {
		s.blob = make([]int32, len(s.Cells))
	}
	for _, i := range s.beaconAt {
		s.blob[i] = -1
	}
	n := 0
	var stack []int32
	for _, seed := range s.beaconAt {
		if s.blob[seed] >= 0 {
			continue
		}
		s.blob[seed] = int32(n)
		stack = append(stack[:0], seed)
		for len(stack) > 0 {
			i := int(stack[len(stack)-1])
			stack = stack[:len(stack)-1]
			x, y := i%W, i/W
			for k := 0; k < 8; k++ {
				nx, ny := x+dx8[k], y+dy8[k]
				if !in(nx, ny) {
					continue
				}
				j := ny*W + nx
				if mats[s.Cells[j].M].Device == DevBeacon && s.blob[j] < 0 {
					s.blob[j] = int32(n)
					stack = append(stack, int32(j))
				}
			}
		}
		n++
	}
	if len(s.blobDone) < n {
		s.blobDone = make([]uint32, n)
		s.blobOn = make([]bool, n)
	}
	return n
}

// demandOf is a cell's power demand; a beacon's whole demand sits on the
// first of its cells met in this circuit.
func (s *Sim) demandOf(m int32) float32 {
	c := &s.Cells[m]
	info := &mats[c.M]
	if info.Device != DevBeacon {
		return info.Demand
	}
	b := s.blob[m]
	if s.blobDone[b] == s.blobMark {
		return 0
	}
	s.blobDone[b] = s.blobMark
	return info.Demand
}

func (s *Sim) resolveCircuit(st *PowerStats) {
	gen, demand := float32(0), float32(0)
	stored := float32(0)
	s.blobMark++
	for _, m := range s.members {
		c := &s.Cells[m]
		switch mats[c.M].Device {
		case DevTurbine, DevThermoGen:
			gen += c.VX
		case DevBattery:
			stored += float32(c.Life)
		}
		demand += s.demandOf(m)
	}
	// batteries cover a shortfall
	avail := gen
	if gen < demand {
		need := demand - gen
		// draw evenly from every charged battery, so a bank keeps its full
		// output until it is nearly flat (a few passes mop up cells that run dry)
		for pass := 0; pass < 3 && need > 0.001; pass++ {
			n := 0
			for _, m := range s.members {
				c := &s.Cells[m]
				if mats[c.M].Device == DevBattery && c.Life > 0 {
					n++
				}
			}
			if n == 0 {
				break
			}
			share := need / float32(n)
			for _, m := range s.members {
				c := &s.Cells[m]
				if mats[c.M].Device != DevBattery || c.Life <= 0 || need <= 0 {
					continue
				}
				take := min(float32(c.Life), batteryDischarge, share, need)
				c.Life -= s.roundRandom(take)
				need -= take
				avail += take
			}
		}
	}
	// run consumers in order until the power runs out
	used := float32(0)
	s.blobMark++
	for _, m := range s.members {
		c := &s.Cells[m]
		isBeacon := mats[c.M].Device == DevBeacon
		if isBeacon && s.blobDone[s.blob[m]] == s.blobMark {
			if s.blobOn[s.blob[m]] {
				c.Flags |= fPowered
			}
			continue
		}
		d := s.demandOf(m)
		if d <= 0 {
			continue
		}
		on := avail-used >= d-0.001
		if on {
			used += d
			c.Flags |= fPowered
		}
		if isBeacon {
			s.blobOn[s.blob[m]] = on
			if on {
				st.BeaconOn++
			}
		}
	}
	// spare power charges batteries
	spare := gen - used
	for _, m := range s.members {
		if spare <= 0 {
			break
		}
		c := &s.Cells[m]
		if mats[c.M].Device != DevBattery || c.Life >= batteryCap {
			continue
		}
		add := min(spare, batteryCharge, float32(batteryCap-c.Life))
		c.Life += s.roundRandom(add)
		spare -= add
	}
	if used > 0 {
		// current flows: wires show it, and resistive ones warm up
		for _, m := range s.members {
			c := &s.Cells[m]
			c.Flags |= fPowered
			if r := mats[c.M].Resist; r > 0 {
				c.T += used * r * 0.003
			}
		}
	}
	st.Generated += gen
	st.Used += used
	st.Stored += stored
}

// roundRandom rounds v up or down at random in proportion to its fraction,
// so small charges and drains add up correctly on average.
func (s *Sim) roundRandom(v float32) int16 {
	f := float32(math.Floor(float64(v)))
	if s.rf() < v-f {
		f++
	}
	return int16(f)
}
