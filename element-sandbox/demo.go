package main

import "elementsandbox/sim"

// The main menu runs a small live scene behind the buttons.

const demoLoop = 60 * 40 // rebuild the scene every 40 seconds

func newDemo() *sim.Sim {
	s := sim.New()
	buildDemo(s)
	return s
}

func buildDemo(s *sim.Sim) {
	const W, H = sim.W, sim.H
	s.Clear()
	s.OpenTop = true
	s.Wind.Global = 0.6
	s.Fill(0, H-12, W, H, sim.Wall, true)
	// volcanic vent with a waterfall pouring onto it
	s.Fill(20, H-60, 26, H-12, sim.Wall, true)
	s.Fill(150, H-60, 156, H-12, sim.Wall, true)
	s.Fill(26, H-45, 150, H-12, sim.Lava, true)
	s.Fill(40, 40, 110, 46, sim.Stone, true)
	// a coal fire under an iron plate holding a gold ingot
	s.Fill(200, H-40, 204, H-12, sim.Wall, true)
	s.Fill(316, H-40, 320, H-12, sim.Wall, true)
	s.Fill(204, H-30, 316, H-12, sim.Coal, false)
	s.Fill(200, H-44, 320, H-40, sim.Metal, true)
	s.Fill(240, H-56, 280, H-44, sim.Gold, false)
	// a glass tank of water with oil on top
	s.Fill(370, H-90, 374, H-12, sim.Glass, true)
	s.Fill(500, H-90, 504, H-12, sim.Glass, true)
	s.Fill(374, H-60, 500, H-12, sim.Water, false)
	s.Fill(374, H-75, 500, H-60, sim.Oil, false)
	for x := 206; x < 316; x += 4 {
		s.Ignite(x, H-31, 2)
	}
}

func stepDemo(s *sim.Sim) {
	if s.Frame%demoLoop == demoLoop-1 {
		buildDemo(s)
	}
	if s.Frame%demoLoop < 30 { // keep the coal lit at the start
		for x := 206; x < 316; x += 6 {
			s.Ignite(x, sim.H-31, 2)
		}
	}
	s.Paint(70, 34, 3, sim.Water, false, 6)
	if s.Frame%3 == 0 {
		s.Paint(440, 10, 2, sim.Sand, false, 2)
		s.Paint(150+(s.Frame*7)%250, 6, 2, sim.Snow, false, 1)
	}
	s.Step()
}
