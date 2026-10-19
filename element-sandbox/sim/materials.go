package sim

import "image/color"

// Mat identifies a material type stored in a cell.
type Mat uint8

// State is the physical state of a material and decides how it moves.
type State uint8

const (
	SEmpty State = iota
	SSolid
	SPowder
	SLiquid
	SGas
	SFire
)

const (
	Empty Mat = iota

	// Solids
	Wall
	Stone
	Wood
	Metal
	Glass
	Ice
	Plant
	TNT
	DryIce

	// Powders
	Sand
	Dirt
	Gravel
	Salt
	Snow
	Gunpowder
	Coal
	Sawdust
	Thermite
	Ash
	Rust

	// Liquids
	Water
	SaltWater
	Oil
	Gasoline
	Diesel
	Ethanol
	Lava
	Acid
	Mercury
	LiquidN2
	MoltenGlass
	MoltenMetal
	Mud
	Nitro

	// Gases
	Steam
	Smoke
	Hydrogen
	Oxygen
	Methane
	Propane
	CO2
	Nitrogen
	Chlorine
	FuelVapor

	// Energy
	Fire

	// Metals (Iron is Metal above)
	Copper
	Aluminum
	Lead
	Tin
	Titanium
	Silver
	Gold
	Platinum

	// Molten metals (Molten Iron is MoltenMetal above)
	MoltenCopper
	MoltenAluminum
	MoltenLead
	MoltenTin
	MoltenTitanium
	MoltenSilver
	MoltenGold
	MoltenPlatinum

	// Power
	Turbine
	ThermoGen
	Battery
	Lamp
	Heater
	Beacon
	House

	// Nature and climate
	Roof
	Seed
	Trunk
	Leaf
	FallenLeaf

	NumMats
)

// Device is the electrical role of a material.
type Device uint8

const (
	DevNone Device = iota
	DevTurbine
	DevThermoGen
	DevBattery
	DevLamp
	DevHeater
	DevBeacon
)

// Palette groups.
const (
	GroupSolids  = "SOLIDS"
	GroupMetals  = "METALS"
	GroupPower   = "POWER"
	GroupPowders = "POWDERS"
	GroupLiquids = "LIQUIDS"
	GroupMolten  = "MOLTEN METALS"
	GroupGases   = "GASES"
	GroupNature  = "NATURE"
)

// MatInfo holds the physical and chemical properties of a material.
type MatInfo struct {
	Name  string
	Desc  string
	State State
	Color color.RGBA
	Alpha float32 // gas opacity when rendered

	Density float32 // kg/m^3 (gases at 20°C); decides what floats/sinks
	Conduct float32 // thermal conductivity, 0..0.2 (simulation units)
	HeatCap float32 // >= 1; higher = slower to heat up/cool down
	SpawnT  float32 // temperature when placed, °C

	// Phase changes: becomes High above HighT, Low below LowT.
	High      Mat
	HighT     float32
	Low       Mat
	LowT      float32
	PhaseRate float32

	// Combustion
	Flammable   bool
	FlashT      float32 // min temperature to catch from an adjacent flame
	AutoT       float32 // auto-ignition temperature
	BurnT       float32 // flame temperature while burning
	Spread      float32 // chance per contact to catch from a flame
	BurnLife    int16
	FlameRate   float32
	SmokeRate   float32
	BurnInto    Mat
	BurnChance  float32
	FlameAux    Mat    // product left by flames this burns with (colours the flame)
	GasProducts [2]Mat // flammable gases turn into these after burning
	Explosive   float32
	EvapRate    float32 // slow evaporation into FuelVapor at room temperature

	// Motion
	Dispersion int     // liquids: how far they flow sideways per frame
	Viscosity  float32 // liquids: chance to skip flowing
	Slide      float32 // powders: chance to slide down slopes
	WindDrag   float32 // how strongly wind pushes it
	Hardness   float32 // resistance to explosions
	Debris     Mat     // what it shatters into
	CoolRate   float32 // relaxation toward ambient (air and gases)

	// Electricity
	Conductive bool    // carries electric power
	Resist     float32 // electrical resistance: heats up when carrying power
	Device     Device
	Demand     float32 // power a consumer needs to run
	AcidProof  bool
	Carbon     bool // burning it releases CO2

	Group string // palette group ("" = hidden)
}

var mats [NumMats]MatInfo

// Info returns the properties of material m.
func Info(m Mat) *MatInfo { return &mats[m] }

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{r, g, b, 255} }

func def(m Mat, name, desc string, st State, col color.RGBA, dens float32) *MatInfo {
	mi := &mats[m]
	*mi = MatInfo{
		Name: name, Desc: desc, State: st, Color: col, Density: dens,
		Conduct: 0.04, HeatCap: 1, SpawnT: DefaultAmbient, PhaseRate: 0.05,
		Hardness: 0.5, Alpha: 1,
	}
	switch st {
	case SSolid:
		mi.Group = GroupSolids
	case SPowder:
		mi.Group = GroupPowders
	case SGas:
		mi.Group = GroupGases
		mi.Conduct = 0.03
		mi.CoolRate = 0.002
		mi.Alpha = 0.4
		mi.WindDrag = 1
	case SLiquid:
		mi.Group = GroupLiquids
		mi.Dispersion = 5
	case SEmpty:
		mi.Conduct = 0.02
		mi.CoolRate = 0.004
	}
	return mi
}

func flammable(mi *MatInfo, flash, auto, burnT, spread float32, life int16, flame, smoke float32) {
	mi.Flammable = true
	mi.FlashT, mi.AutoT, mi.BurnT, mi.Spread = flash, auto, burnT, spread
	mi.BurnLife, mi.FlameRate, mi.SmokeRate = life, flame, smoke
}

func init() {
	m := def(Empty, "Air", "Air. Carries heat, hot air rises.", SEmpty, rgb(14, 16, 22), 1.2)
	m.HeatCap = 1

	// ---- Solids ----
	m = def(Wall, "Wall", "Indestructible, heat-insulating wall.", SSolid, rgb(85, 88, 98), 9999)
	m.Conduct, m.Hardness, m.AcidProof = 0.001, 99, true

	m = def(Stone, "Stone", "Rock. Melts into lava around 1200°C.", SSolid, rgb(118, 116, 112), 2600)
	m.Conduct, m.Hardness, m.Debris = 0.06, 0.6, Gravel
	m.High, m.HighT, m.PhaseRate = Lava, 1200, 0.01

	m = def(Wood, "Wood", "Ignites ~300°C, burns slowly to ash and smoke.", SSolid, rgb(118, 78, 40), 600)
	m.Conduct, m.Hardness, m.Debris = 0.01, 0.3, Sawdust
	flammable(m, 150, 300, 650, 0.03, 500, 0.25, 0.02)
	m.BurnInto, m.BurnChance = Ash, 0.35

	m = def(Metal, "Iron", "Conducts heat fast. Melts at 1538°C. Rusts in water.", SSolid, rgb(148, 153, 163), 7800)
	m.Conduct, m.HeatCap, m.Hardness = 0.2, 1.2, 0.9
	m.High, m.HighT, m.PhaseRate = MoltenMetal, 1538, 0.02
	m.Group, m.Conductive, m.Resist = GroupMetals, true, 0.6

	m = def(Glass, "Glass", "Acid-proof. Softens and melts ~1400°C.", SSolid, rgb(160, 205, 215), 2500)
	m.Hardness, m.Debris, m.AcidProof = 0.2, Sand, true
	m.High, m.HighT, m.PhaseRate = MoltenGlass, 1400, 0.02

	m = def(Ice, "Ice", "Frozen water. Melts above 0°C. Salt melts it.", SSolid, rgb(170, 220, 250), 917)
	m.Conduct, m.HeatCap, m.SpawnT, m.Hardness, m.Debris = 0.1, 2, -10, 0.3, Snow
	m.High, m.HighT, m.PhaseRate = Water, 0, 0.03

	m = def(Plant, "Plant", "Grows by drinking water. Turns CO2 into oxygen.", SSolid, rgb(40, 160, 55), 700)
	m.Conduct, m.Hardness = 0.02, 0.1
	flammable(m, 100, 250, 500, 0.05, 60, 0.2, 0.03)
	m.BurnInto, m.BurnChance = Ash, 0.2

	m = def(TNT, "TNT", "High explosive. Detonates from fire or ~240°C.", SSolid, rgb(200, 40, 40), 1650)
	m.Conduct, m.Hardness = 0.02, 0.5
	flammable(m, 0, 240, 2500, 0.15, 2, 0, 0)
	m.Explosive = 10

	m = def(DryIce, "Dry Ice", "Solid CO2 at -78°C. Sublimates into heavy cold fog.", SSolid, rgb(222, 230, 240), 1560)
	m.Conduct, m.SpawnT, m.Hardness = 0.05, -80, 0.3
	m.High, m.HighT, m.PhaseRate = CO2, -78.5, 0.02

	// ---- Powders ----
	m = def(Sand, "Sand", "Silica. Melts into glass at ~1700°C.", SPowder, rgb(220, 195, 120), 1600)
	m.Slide, m.WindDrag, m.Hardness = 1, 0.03, 0
	m.High, m.HighT, m.PhaseRate = MoltenGlass, 1700, 0.02

	m = def(Dirt, "Dirt", "Soil. Turns to mud when soaked with water.", SPowder, rgb(110, 75, 45), 1300)
	m.Conduct, m.Slide, m.WindDrag = 0.03, 0.4, 0.02

	m = def(Gravel, "Gravel", "Crushed rock. Melts into lava ~1200°C.", SPowder, rgb(128, 124, 120), 1800)
	m.Conduct, m.Slide = 0.05, 0.6
	m.High, m.HighT, m.PhaseRate = Lava, 1200, 0.01

	m = def(Salt, "Salt", "NaCl. Dissolves in water, melts ice.", SPowder, rgb(240, 240, 240), 2160)
	m.Conduct, m.Slide, m.WindDrag = 0.05, 0.8, 0.03

	m = def(Snow, "Snow", "Light frozen flakes. Blows in the wind, melts at 0°C.", SPowder, rgb(250, 250, 255), 300)
	m.Conduct, m.HeatCap, m.SpawnT, m.Slide, m.WindDrag = 0.02, 1.5, -5, 0.3, 0.3
	m.High, m.HighT, m.PhaseRate = Water, 0, 0.02

	m = def(Gunpowder, "Gunpowder", "Black powder. Any flame sets it off.", SPowder, rgb(58, 58, 62), 1000)
	m.Conduct, m.Slide, m.WindDrag = 0.03, 0.8, 0.05
	flammable(m, -100, 300, 2000, 0.9, 3, 0.8, 0.3)
	m.Explosive = 2.5

	m = def(Coal, "Coal", "Hard to light, burns very long and hot.", SPowder, rgb(32, 32, 34), 1100)
	m.Conduct, m.Slide = 0.03, 0.4
	flammable(m, 300, 450, 1100, 0.02, 1800, 0.06, 0.01)
	m.BurnInto, m.BurnChance = Ash, 0.12

	m = def(Sawdust, "Sawdust", "Fine wood. Floats, catches fire easily.", SPowder, rgb(210, 180, 130), 250)
	m.Conduct, m.Slide, m.WindDrag = 0.01, 0.4, 0.35
	flammable(m, 100, 250, 600, 0.2, 50, 0.4, 0.05)
	m.BurnInto, m.BurnChance = Ash, 0.3

	m = def(Thermite, "Thermite", "Needs ~900°C to light. Burns at 2700°C into molten iron.", SPowder, rgb(150, 92, 80), 2500)
	m.Conduct, m.Slide = 0.05, 0.7
	flammable(m, 900, 900, 2700, 0.5, 120, 0.5, 0.02)
	m.BurnInto, m.BurnChance = MoltenMetal, 0.6

	m = def(Ash, "Ash", "Burnt remains. Very light, blows away.", SPowder, rgb(160, 160, 160), 600)
	m.Conduct, m.Slide, m.WindDrag = 0.01, 0.5, 0.4

	m = def(Rust, "Rust", "Iron oxide, formed when iron meets water.", SPowder, rgb(150, 70, 30), 2000)
	m.Slide, m.WindDrag = 0.6, 0.03

	// ---- Liquids ----
	m = def(Water, "Water", "Boils at 100°C, freezes at 0°C. Puts out fires.", SLiquid, rgb(40, 100, 220), 1000)
	m.Conduct, m.HeatCap, m.Dispersion = 0.1, 3, 6
	m.High, m.HighT, m.PhaseRate = Steam, 100, 0.04
	m.Low, m.LowT = Ice, 0

	m = def(SaltWater, "Salt Water", "Brine. Freezes at -21°C. Boils off leaving salt.", SLiquid, rgb(60, 118, 200), 1025)
	m.Conduct, m.HeatCap, m.Dispersion = 0.1, 3, 6
	m.High, m.HighT, m.PhaseRate = Steam, 101, 0.04
	m.Low, m.LowT = Ice, -21
	m.Conductive, m.Resist = true, 20 // dissolved ions carry current, badly

	m = def(Oil, "Oil", "Thick crude oil. Floats on water, burns smoky.", SLiquid, rgb(66, 48, 20), 900)
	m.Conduct, m.HeatCap, m.Dispersion, m.Viscosity = 0.03, 1.5, 3, 0.3
	flammable(m, 200, 300, 900, 0.1, 300, 0.3, 0.12)

	m = def(Gasoline, "Gasoline", "Flash point -43°C: lights instantly. Gives off heavy vapor.", SLiquid, rgb(230, 210, 100), 740)
	m.Conduct, m.HeatCap, m.Dispersion = 0.03, 1.5, 7
	flammable(m, -43, 250, 1000, 0.6, 70, 0.5, 0.03)
	m.High, m.HighT, m.PhaseRate, m.EvapRate = FuelVapor, 95, 0.03, 0.0006

	m = def(Diesel, "Diesel", "Flash point 52°C: hard to light cold, burns long and sooty.", SLiquid, rgb(170, 140, 40), 840)
	m.Conduct, m.HeatCap, m.Dispersion, m.Viscosity = 0.03, 1.5, 5, 0.1
	flammable(m, 52, 210, 950, 0.12, 220, 0.35, 0.1)
	m.High, m.HighT, m.PhaseRate = FuelVapor, 280, 0.03

	m = def(Ethanol, "Ethanol", "Alcohol. Burns with a clean blue flame. Boils 78°C.", SLiquid, rgb(200, 222, 232), 789)
	m.Conduct, m.HeatCap, m.Dispersion = 0.04, 2, 7
	flammable(m, 13, 365, 900, 0.4, 90, 0.45, 0)
	m.FlameAux = CO2
	m.High, m.HighT, m.PhaseRate, m.EvapRate = FuelVapor, 78, 0.03, 0.0004

	m = def(Lava, "Lava", "Molten rock at 1200°C. Cools into stone.", SLiquid, rgb(255, 90, 0), 2600)
	m.Conduct, m.HeatCap, m.SpawnT, m.Dispersion, m.Viscosity = 0.06, 1.5, 1200, 2, 0.75
	m.Low, m.LowT, m.PhaseRate = Stone, 700, 0.01

	m = def(Acid, "Acid", "Dissolves almost anything except glass. Metal fizzes hydrogen.", SLiquid, rgb(120, 255, 60), 1500)
	m.Conduct, m.HeatCap = 0.08, 2

	m = def(Mercury, "Mercury", "Liquid metal, 13.5x denser than water. Conducts electricity.", SLiquid, rgb(190, 195, 205), 13500)
	m.Conduct, m.Conductive, m.Resist = 0.15, true, 3

	m = def(LiquidN2, "Liquid N2", "Liquid nitrogen at -196°C. Flash-freezes, boils violently.", SLiquid, rgb(200, 232, 255), 808)
	m.Conduct, m.HeatCap, m.SpawnT, m.Dispersion = 0.05, 1.5, -200, 7
	m.High, m.HighT, m.PhaseRate = Nitrogen, -196, 0.03

	m = def(MoltenGlass, "Molten Glass", "Liquid glass. Sets into glass below 1000°C.", SLiquid, rgb(255, 170, 60), 2400)
	m.Conduct, m.SpawnT, m.Dispersion, m.Viscosity = 0.05, 1600, 1, 0.9
	m.Low, m.LowT, m.PhaseRate = Glass, 1000, 0.02

	m = def(MoltenMetal, "Molten Iron", "Liquid iron at 1600°C+. Solidifies below 1500°C.", SLiquid, rgb(255, 140, 40), 7000)
	m.Conduct, m.SpawnT, m.Dispersion, m.Viscosity = 0.2, 1700, 3, 0.3
	m.Low, m.LowT, m.PhaseRate = Metal, 1500, 0.03
	m.Group, m.Conductive, m.Resist = GroupMolten, true, 1.2

	m = def(Mud, "Mud", "Wet dirt. Dries back to dirt when heated.", SLiquid, rgb(88, 60, 35), 1600)
	m.Conduct, m.HeatCap, m.Dispersion, m.Viscosity = 0.06, 2, 2, 0.8
	m.High, m.HighT, m.PhaseRate = Dirt, 100, 0.02

	m = def(Nitro, "Nitro", "Nitroglycerin. Explodes on impact or when heated. Don't drop it.", SLiquid, rgb(230, 230, 180), 1600)
	m.Conduct, m.Dispersion = 0.03, 4
	flammable(m, 0, 200, 2000, 0.5, 1, 0, 0)
	m.Explosive = 7

	// ---- Gases ----
	m = def(Steam, "Steam", "Water vapor. Rises, condenses into rain below 100°C.", SGas, rgb(205, 205, 215), 0.6)
	m.SpawnT, m.CoolRate, m.Alpha = 110, 0.0012, 0.55
	m.Low, m.LowT, m.PhaseRate = Water, 98, 0.08

	m = def(Smoke, "Smoke", "Soot from fires. Drifts with the wind and fades.", SGas, rgb(62, 62, 66), 1.1)
	m.Alpha = 0.8

	m = def(Hydrogen, "Hydrogen", "Lightest gas. Explosive in air, burns into steam.", SGas, rgb(200, 220, 255), 0.09)
	flammable(m, -300, 500, 2100, 0.95, 0, 0, 0)
	m.GasProducts, m.Alpha = [2]Mat{Steam, Steam}, 0.22

	m = def(Oxygen, "Oxygen", "Makes fires burn faster and much hotter.", SGas, rgb(150, 200, 255), 1.43)
	m.Alpha = 0.25

	m = def(Methane, "Methane", "Natural gas. Lighter than air, burns blue.", SGas, rgb(170, 230, 170), 0.66)
	flammable(m, -300, 580, 1950, 0.8, 0, 0, 0)
	m.GasProducts, m.Alpha = [2]Mat{Steam, CO2}, 0.3

	m = def(Propane, "Propane", "Heavier than air: pools on the floor. Very flammable.", SGas, rgb(230, 200, 160), 1.88)
	flammable(m, -300, 470, 1980, 0.8, 0, 0, 0)
	m.GasProducts, m.Alpha = [2]Mat{CO2, Steam}, 0.3

	m = def(CO2, "CO2", "Heavy, sinks. Smothers fires. Freezes at -78°C.", SGas, rgb(120, 120, 135), 1.98)
	m.Low, m.LowT, m.PhaseRate, m.Alpha = DryIce, -78.5, 0.005, 0.3

	m = def(Nitrogen, "Nitrogen", "Inert. 78% of the air. Liquefies at -196°C.", SGas, rgb(170, 170, 215), 1.25)
	m.Low, m.LowT, m.PhaseRate, m.Alpha = LiquidN2, -196, 0.01, 0.2

	m = def(Chlorine, "Chlorine", "Toxic heavy gas. Kills plants, forms acid in water.", SGas, rgb(190, 230, 80), 3.2)
	m.Alpha = 0.5

	m = def(FuelVapor, "Fuel Vapor", "Evaporated fuel. Heavy and explosive.", SGas, rgb(200, 180, 110), 3.0)
	flammable(m, -300, 280, 1500, 0.9, 0, 0, 0)
	m.GasProducts, m.Alpha = [2]Mat{Smoke, CO2}, 0.3

	// ---- Energy ----
	m = def(Fire, "Fire", "Flame. Ignites anything flammable it touches.", SFire, rgb(255, 160, 40), 0.3)
	m.Conduct, m.SpawnT, m.WindDrag = 0.08, 1000, 1

	// ---- Metals ----
	// name, molten form, colour, density, thermal conductivity, melting point,
	// electrical resistance (relative), hardness, acid-proof, acid gives H2
	metal := func(m, molten Mat, name, desc string, col color.RGBA, dens, cond, melt, resist, hard float32, acidProof bool) {
		mi := def(m, name, desc, SSolid, col, dens)
		mi.Group, mi.Conductive, mi.Resist = GroupMetals, true, resist
		mi.Conduct, mi.HeatCap, mi.Hardness, mi.AcidProof = cond, 1.1, hard, acidProof
		mi.High, mi.HighT, mi.PhaseRate = molten, melt, 0.02
		hot := color.RGBA{uint8(min(255, int(col.R)/2+140)), uint8(min(255, int(col.G)/2+70)), uint8(int(col.B) / 3), 255}
		ml := def(molten, "Molten "+name, "Liquid "+name+". Solidifies below its melting point.", SLiquid, hot, dens*0.93)
		ml.Group, ml.Conductive, ml.Resist = GroupMolten, true, resist*2
		ml.Conduct, ml.HeatCap, ml.Dispersion, ml.Viscosity = cond, 1.1, 3, 0.25
		ml.SpawnT, ml.AcidProof = melt+150, acidProof
		ml.Low, ml.LowT, ml.PhaseRate = m, melt-10, 0.03
	}
	metal(Copper, MoltenCopper, "Copper", "Best value wire: conducts heat and electricity very well. Melts 1085°C.",
		rgb(190, 110, 55), 8960, 0.2, 1085, 0.1, 0.8, false)
	metal(Aluminum, MoltenAluminum, "Aluminum", "Light metal, good conductor. Melts at only 660°C. Acid fizzes hydrogen.",
		rgb(196, 202, 210), 2700, 0.2, 660, 0.16, 0.6, false)
	metal(Lead, MoltenLead, "Lead", "Very dense, soft, poor conductor. Melts at 327°C.",
		rgb(88, 94, 110), 11340, 0.04, 327, 1.3, 0.4, false)
	metal(Tin, MoltenTin, "Tin", "Soft metal used in solder. Melts at just 232°C.",
		rgb(200, 200, 188), 7300, 0.07, 232, 0.65, 0.4, false)
	metal(Titanium, MoltenTitanium, "Titanium", "Strong and acid-proof. Poor conductor. Melts 1668°C.",
		rgb(160, 166, 178), 4500, 0.03, 1668, 2.5, 0.97, true)
	metal(Silver, MoltenSilver, "Silver", "The best electrical conductor of all metals. Melts 962°C.",
		rgb(222, 225, 232), 10490, 0.2, 962, 0.08, 0.6, false)
	metal(Gold, MoltenGold, "Gold", "Precious. Never corrodes, acid can't touch it. Great conductor. Melts 1064°C.",
		rgb(255, 196, 40), 19300, 0.2, 1064, 0.12, 0.5, true)
	metal(Platinum, MoltenPlatinum, "Platinum", "Rarest precious metal, densest here. Acid-proof. Melts 1768°C.",
		rgb(206, 214, 222), 21450, 0.1, 1768, 0.55, 0.7, true)

	// ---- Power ----
	dev := func(m Mat, name, desc string, col color.RGBA, d Device, demand float32) *MatInfo {
		mi := def(m, name, desc, SSolid, col, 3000)
		mi.Group, mi.Conductive, mi.Device, mi.Demand = GroupPower, true, d, demand
		mi.Hardness, mi.Conduct = 0.7, 0.05
		return mi
	}
	dev(Turbine, "Turbine", "Spins when steam or wind flows past it. Cools the steam back to water.",
		rgb(112, 122, 134), DevTurbine, 0)
	dev(ThermoGen, "Thermo Gen", "Thermoelectric generator: power from a hot side and a cold side.",
		rgb(70, 66, 92), DevThermoGen, 0).Conduct = 0.06
	dev(Battery, "Battery", "Stores spare power and releases it when generators fall short.",
		rgb(40, 70, 45), DevBattery, 0)
	dev(Lamp, "Lamp", "Lights up when powered. Uses 0.2 power per pixel.",
		rgb(95, 90, 55), DevLamp, 0.2)
	m = dev(Heater, "Heater", "Electric heating coil. Uses 0.5 power per pixel, gets red hot.",
		rgb(110, 45, 35), DevHeater, 0.5)
	m.Conduct = 0.12
	m = dev(Beacon, "Beacon", "Level goal. Each beacon needs 12 power. Power them all to win.",
		rgb(40, 75, 55), DevBeacon, 12)
	m.Hardness = 99
	// a house is a beacon in disguise: one connected house needs 12 power
	m = dev(House, "House", "A home in the town. Each house needs 12 power to light up.",
		rgb(150, 122, 92), DevBeacon, 12)
	m.Hardness, m.Conduct = 99, 0.02

	// ---- Nature ----
	m = def(Roof, "Roof", "The sky's ceiling. Cold: steam condenses on it and rains back down.", SSolid, rgb(46, 52, 70), 9999)
	m.Group, m.Conduct, m.Hardness, m.AcidProof = "", 0.05, 99, true

	m = def(Seed, "Tree Seed", "Plant on dirt or mud. Grows into a tree in warm seasons.", SPowder, rgb(140, 100, 50), 700)
	m.Group, m.Slide, m.WindDrag = GroupNature, 0.3, 0.05
	flammable(m, 150, 250, 500, 0.1, 20, 0.2, 0.02)

	m = def(Trunk, "Tree Trunk", "Living wood. Grows a crown of leaves every spring.", SSolid, rgb(96, 64, 36), 700)
	m.Group, m.Conduct, m.Hardness, m.Debris = GroupNature, 0.01, 0.35, Sawdust
	flammable(m, 150, 300, 650, 0.03, 500, 0.25, 0.02)
	m.BurnInto, m.BurnChance = Ash, 0.35

	m = def(Leaf, "Leaves", "Absorb CO2 and give off oxygen, fastest in summer. Fall in autumn.", SSolid, rgb(60, 150, 50), 400)
	m.Group, m.Conduct, m.Hardness = GroupNature, 0.02, 0.05
	flammable(m, 120, 230, 500, 0.08, 25, 0.3, 0.05)
	m.BurnInto, m.BurnChance = Ash, 0.1

	m = def(FallenLeaf, "Fallen Leaves", "Dry leaves. Blow around, burn easily, rot into soil.", SPowder, rgb(170, 100, 40), 150)
	m.Group, m.Conduct, m.Slide, m.WindDrag = GroupNature, 0.01, 0.3, 0.45
	flammable(m, 80, 220, 550, 0.4, 25, 0.4, 0.06)
	m.BurnInto, m.BurnChance = Ash, 0.3

	// carbon-based fuels give off CO2 when they burn
	for _, c := range []Mat{Wood, Plant, Coal, Sawdust, Oil, Gasoline, Diesel, Ethanol, Seed, Trunk, Leaf, FallenLeaf} {
		mats[c].Carbon = true
	}
}
