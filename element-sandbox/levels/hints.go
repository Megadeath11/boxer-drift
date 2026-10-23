package levels

import "strings"

// Hints for the harder levels, by level number. Each starts as a nudge and
// gets more specific, so players take only as much help as they want.
var hints = map[string][]string{
	"8": {
		"Hilltops catch the most wind.",
		"A battery cell only charges a little each frame: a big bank charges much faster.",
		"About 20 battery cells, all wired into the same grid as the beacons, will bridge the calm.",
	},
	"9": {
		"Melting snow puts fires out: clear the firebox first.",
		"The pot's iron isn't wired to the turbine lid: the Thermo Gen needs its own wire to the line.",
		"Stand the Thermo Gen against the fire with the snowdrift on its other side, under the pot.",
	},
	"10": {
		"TNT goes off from a spark: you don't need a big bomb.",
		"Once the water is in, bridge the gap in the wire. No eraser, so draw carefully.",
	},
	"11": {
		"Only part of the wire ever touches the lava.",
		"Copper is fine overhead: use platinum just for the part that dips into the pit.",
	},
	"12": {
		"The boiler covers some beacons; wind covers the rest.",
		"Wherever the wire goes under the dipping wall, it sits in acid: make that part gold.",
		"Turbines on the hilltop, a battery bank for the calm, everything in one grid.",
	},
	"13": {
		"A boiler is an iron pot of water with Turbine blades for a lid.",
		"Leave an open firebox under the pot. Pile coal right up against the pot's bottom: a gap wastes heat.",
		"Light the coal with Fire, and wire from the turbine lid to the beacon.",
	},
	"14": {
		"Lava can boil water as well as any fire.",
		"Build the pot so its iron bottom sits directly on the lava.",
		"One wire can serve both beacons: run it high, and drop down to each.",
	},
	"15": {
		"Glass is acid-proof.",
		"A glass roof over the wire keeps the rain off, but acid pools on it and pours off the ends.",
		"Butt the glass against the shelter wall, and run it past the point where the wire turns down.",
	},
	"16": {
		"The hilltops get the most wind.",
		"Batteries charge slowly and the gusts are short: you need a big bank.",
		"You'll need nearly all of your turbine and battery budget.",
	},
	"17": {
		"There's no fuel: the hot ground is your steady power.",
		"Where the wire passes under the dipping wall it's in acid: use gold there.",
		"The boiler can't do it alone: high turbines and a battery bank cover the gusts and calms.",
		"Don't forget the beacon in the air: run a wire up to it.",
	},
}

func init() {
	for k := range All {
		num, _, _ := strings.Cut(All[k].Name, ".")
		if h, ok := hints[num]; ok {
			All[k].Hints = h
		}
	}
}
