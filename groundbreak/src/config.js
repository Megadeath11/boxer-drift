// Real-world reference data used by the simulation.
// Machines are generic models built to match published spec-sheet figures of
// common machines in each class (sources noted per field).

export const SOIL = {
  // Bank -> loose volume swell factor (typical common earth / loam: 1.20-1.30)
  SWELL: 1.25,
  // Fraction of loose thickness remaining after compaction by machine traffic
  COMPACT: 0.84,
  // Angle of repose of loose, dry-ish excavated soil: 34-38 degrees
  REPOSE_LOOSE_DEG: 36,
  // Undisturbed cohesive ground can stand steep (short-term) cut faces
  REPOSE_BANK_DEG: 74,
  // Material indices
  TOPSOIL: 0, CLAY: 1, GRAVEL: 2, ROCK: 3,
  names: ['Topsoil', 'Clay', 'Gravelly till', 'Ripped rock'],
  // Specific cutting energy (J/m^3 of bank material). Tuned so that a 20 t
  // excavator fills a 1.2 m^3 bucket in ~3-6 s in clay, like real cycle times.
  energyBank: [120e3, 260e3, 430e3, 900e3],
  energyLoose: 30e3,
  // Densities (kg/m^3). Bank: in-situ; loose: after excavation
  densBank: [1600, 1900, 2050, 2600],
  densLoose: [1300, 1520, 1750, 1750],
  topsoilDepth: 0.35,
  gravelBand: 0.7, // weathered/gravelly layer just above bedrock
};

// Job site layout (metres, world X/Z). The playable deformable grid is 96 m square.
export const SITE = {
  size: 96,
  res: 0.25,
  // Foundation excavation for a small building: 12 m x 9 m x 2.5 m deep
  pit: { x0: 2, z0: 2, x1: 14, z1: 11, depth: 2.5 },
  // Fill pad to be built from the spoil: 14 m x 12 m, +0.5 m above existing grade
  pad: { x0: -32, z0: -26, x1: -18, z1: -14, lift: 0.5 },
  tolerance: 0.08,
};

// 20-tonne class hydraulic excavator (reference: CAT 320 / Komatsu PC210 spec sheets)
export const EXCAVATOR = {
  name: 'EX-20 Hydraulic Excavator',
  className: '20 t class crawler excavator',
  operatingMass: 22500,        // kg  (CAT 320: 22,500 kg)
  enginePower: 129e3,          // W   (CAT C4.4: 129 kW net)
  hydraulicPower: 85e3,        // W available to the work tools
  boomLength: 5.7,             // m (reach boom). Pin-to-pin chord used: 5.0 m
  boomChord: 5.0,
  stickLength: 2.9,            // m
  bucketCapacity: 1.19,        // m^3 heaped (SAE)
  bucketWidth: 1.3,            // m
  bucketTipRadius: 1.45,       // m pin to tooth tip
  trackGauge: 2.38,            // m
  shoeWidth: 0.6,              // m
  trackLength: 4.45,           // m overall
  tumblerCenters: 3.65,        // m
  upperWidth: 2.95,            // m
  tailSwingRadius: 2.83,       // m
  cabHeight: 2.95,             // m
  counterweightClearance: 1.05,// m
  swingSpeed: 11.25,           // rpm
  travelSpeedHi: 5.7,          // km/h
  travelSpeedLo: 3.4,          // km/h
  drawbarPull: 205e3,          // N
  bucketDiggingForce: 140e3,   // N (ISO)
  maxDigDepth: 6.72,           // m
  maxReachGround: 9.86,        // m
};

// Medium track-type tractor (reference: CAT D6 / Komatsu D61 spec sheets)
export const DOZER = {
  name: 'DZ-6 Track-Type Tractor',
  className: 'Medium dozer, SU blade + multi-shank ripper',
  operatingMass: 20500,        // kg
  enginePower: 160e3,          // W (D6: 160 kW net)
  trackGauge: 1.98,            // m
  shoeWidth: 0.56,             // m
  trackOnGround: 2.67,         // m
  bladeWidth: 3.26,            // m (SU)
  bladeHeight: 1.40,           // m
  bladeCapacity: 5.6,          // m^3 (SAE J1265)
  bladeLift: 1.10,             // m max lift above ground
  bladeDig: 0.50,              // m max dig depth
  maxTilt: 0.40,               // m
  speedFwd: 10.6,              // km/h
  speedRev: 13.0,              // km/h
  rippeDepth: 0.55,            // m max penetration
};

// Tandem-axle vocational dump truck (reference: Kenworth T880 / Mack Granite + 16 yd body)
export const TRUCK = {
  name: 'DT-16 Tandem Dump Truck',
  className: '6x4 vocational dump truck, 16 yd³ body',
  emptyMass: 13600,            // kg
  gvwr: 33100,                 // kg
  enginePower: 336e3,          // W (450 hp)
  peakTorque: 2237,            // N*m @ 1200 rpm (1650 lb-ft)
  wheelbase: 5.3,              // m front axle to tandem centre
  tandemSpread: 1.37,          // m
  trackFront: 2.05,            // m
  trackRear: 1.85,             // m
  overallWidth: 2.59,          // m
  tyreRadius: 0.54,            // m (315/80R22.5)
  bodyLength: 4.9,             // m inside
  bodyWidth: 2.3,              // m inside
  bodySide: 1.15,              // m
  bodyCapacity: 12.2,          // m^3 struck (16 yd^3)
  maxTip: 50,                  // deg
  gears: [4.70, 2.21, 1.53, 1.00, 0.76, 0.67], // Allison 4500 RDS
  reverse: 5.55,
  finalDrive: 4.88,
};
