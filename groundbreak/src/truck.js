// 6x4 tandem-axle vocational dump truck with 16 yd³ body.
// Ray-cast suspension per wheel station, engine torque curve, 6-speed automatic
// with torque converter, air brakes, and a granular load simulated on a grid
// inside the body that shifts on slopes and slides out the tailgate when tipped.
import * as THREE from 'three';
import { Machine } from './machine.js';
import { TRUCK as S, SOIL, SITE } from './config.js';
import { clamp, DEG } from './util.js';
import { materials, rbox, box, cyl, profile, HydCylinder, marker, shadowAll } from './parts.js';
import { resolveContact, Contact } from './physics.js';
import { makeTextTexture } from './textures.js';

const R = S.tyreRadius;
const RIDE = 1.1;               // body origin height above ground at rest
const MOUNT_Y = -RIDE + R + 0.3; // suspension mount (local y)
const REST = 0.3 + 0.08;         // rest length incl. static sag
const HINGE = new THREE.Vector3(0, 0.1, -3.8);
const BED_NX = 10, BED_NZ = 20, BED_CS = 0.245;
const BED_W = S.bodyWidth, BED_L = S.bodyLength, BED_SIDE = S.bodySide;
const FLOOR_Y = 0.12; // body floor above hinge

function torqueCurve(rpm) {
  // N*m: 450 hp / 1650 lb-ft class heavy diesel
  if (rpm < 600) return 900;
  if (rpm < 1200) return 1300 + (S.peakTorque - 1300) * (rpm - 600) / 600;
  if (rpm < 1600) return S.peakTorque;
  if (rpm < 2100) return S.peakTorque - (S.peakTorque - 1530) * (rpm - 1600) / 500;
  return Math.max(0, 1530 - (rpm - 2100) * 8);
}

export class Truck extends Machine {
  constructor(scene, terrain, particles) {
    super(scene, terrain, particles, S.emptyMass, new THREE.Vector3(80000, 85000, 14000));
    this.spec = S;
    this.kind = 'truck';
    this.label = S.name;
    this.idleRpm = 650; this.maxRpm = 2100;
    this.radius = 7;
    this.gear = 1; this.reverse = false; this.shiftT = 0;
    this.steer = 0; this.tip = 0; // body angle (rad)
    this.parkBrake = true;
    this.bed = new Float32Array(BED_NX * BED_NZ);
    this.bedType = 1;
    this.dumpedInPad = 0;
    this._bedAcc = 0;
    this.tailgate = 0;
    this.build();
    this.setupPhysics();
  }

  build() {
    const m = materials();
    const G = new THREE.Group(); G.position.y = -RIDE; this.root.add(G); this.G = G; // ground coords
    const fy = 1.02; // frame rail top
    // frame rails + crossmembers
    for (const s of [1, -1]) box(0.09, 0.28, 8.4, m.black, s * 0.44, fy - 0.14, 0.25, G);
    for (let z = -3.8; z < 4.3; z += 0.9) box(0.8, 0.12, 0.08, m.black, 0, fy - 0.2, z, G);
    // front bumper, tow hooks
    rbox(2.45, 0.32, 0.25, 0.05, m.chrome, 0, 0.72, 4.62, G);
    for (const s of [1, -1]) box(0.08, 0.12, 0.12, m.paint, s * 0.6, 0.6, 4.78, G);
    // hood
    G.add(new THREE.Mesh(profile([[2.95, 1.15], [4.45, 1.2], [4.56, 1.55], [4.46, 1.95], [3.1, 2.15], [2.95, 2.15]], 1.25, 0.08), m.truckPaint));
    // grille
    rbox(1.12, 0.78, 0.06, 0.03, m.chrome, 0, 1.55, 4.53, G);
    for (let i = 0; i < 8; i++) box(1.0, 0.03, 0.04, m.black, 0, 1.24 + i * 0.09, 4.57, G);
    // fenders over steer axle
    for (const s of [1, -1]) {
      const f = new THREE.Mesh(profile([[2.25, 1.05], [2.5, 1.5], [3.5, 1.55], [3.85, 1.1], [3.65, 1.05], [3.4, 1.38], [2.6, 1.36], [2.45, 1.05]], 0.5, 0.03), m.truckPaint);
      f.position.x = s * 0.98; G.add(f);
      box(0.3, 0.18, 0.1, m.black, s * 0.95, 1.4, 3.82, G);
      box(0.25, 0.14, 0.02, m.lightLens, s * 0.95, 1.4, 3.88, G);
      box(0.12, 0.08, 0.02, m.amberLens, s * 1.12, 1.48, 3.75, G);
    }
    // cab
    const C = new THREE.Group(); G.add(C); this.cab = C;
    C.add(new THREE.Mesh(profile([[1.35, 1.1], [2.95, 1.1], [2.95, 2.2], [2.75, 3.15], [1.4, 3.2], [1.35, 3.0]], 2.3, 0.08), m.truckPaint));
    const glass = (w, h, x, y, z, ry, rx = 0) => { const g = new THREE.Mesh(new THREE.PlaneGeometry(w, h), m.glass); g.position.set(x, y, z); g.rotation.set(rx, ry, 0, 'YXZ'); g.renderOrder = 2; C.add(g); return g; };
    glass(2.05, 0.92, 0, 2.66, 2.875, 0, -0.208);
    for (const s of [1, -1]) {
      glass(0.95, 0.75, s * 1.155, 2.62, 2.25, s * Math.PI / 2);
      box(0.04, 0.05, 0.95, m.chrome, s * 1.17, 1.85, 2.0, C);
      cyl(0.015, 0.6, m.chrome, 'x', s * 1.45, 2.4, 2.9, C, 6);
      rbox(0.08, 0.4, 0.22, 0.02, m.black, s * 1.75, 2.4, 2.9, C);
      box(0.35, 0.04, 0.55, m.wornSteel, s * 1.08, 0.62, 2.25, C);
      cyl(0.32, 1.1, m.chrome, 'z', s * 0.82, 0.72, 1.45, C, 24);
    }
    // vertical exhaust stack on the right side of the cab
    cyl(0.065, 1.95, m.chrome, 'y', -1.22, 2.4, 1.55, C, 14);
    box(0.14, 0.9, 0.14, m.black, -1.22, 1.4, 1.55, C);
    this.exhaust = marker(C, -1.22, 3.4, 1.55);
    rbox(1.5, 0.1, 0.25, 0.04, m.black, 0, 3.24, 2.55, C);
    for (let i = -2; i <= 2; i++) box(0.08, 0.05, 0.05, m.amberLens, i * 0.3, 3.24, 2.72, C);
    // interior
    rbox(0.5, 0.15, 0.5, 0.04, m.seat, 0.55, 1.55, 1.95, C);
    rbox(0.5, 0.7, 0.12, 0.05, m.seat, 0.55, 1.95, 1.68, C);
    const wheel = new THREE.Mesh(new THREE.TorusGeometry(0.22, 0.025, 8, 24), m.black);
    wheel.position.set(0.55, 2.05, 2.5); wheel.rotation.x = -1.0; C.add(wheel); this.steerWheel = wheel;
    box(2.0, 0.35, 0.3, m.interior, 0, 1.9, 2.75, C);
    this.cabCam = marker(C, 0.55, 2.55, 2.0);
    const dec = new THREE.Mesh(new THREE.PlaneGeometry(0.9, 0.22), new THREE.MeshStandardMaterial({ map: makeTextTexture('DT-16', { color: '#f2f2f2' }), transparent: true }));
    for (const s of [1, -1]) { const d = dec.clone(); d.position.set(s * 1.205, 1.55, 2.2); d.rotation.y = s * Math.PI / 2; C.add(d); }
    // rear lights & mud flaps
    for (const s of [1, -1]) {
      box(0.25, 0.12, 0.05, m.redLens, s * 0.95, 0.85, -3.92, G);
      const flap = box(0.6, 0.55, 0.02, m.rubber, s * 0.95, 0.5, -3.75, G);
      void flap;
    }
    // ---- dump body ----
    const BD = new THREE.Group(); BD.position.copy(HINGE).add(new THREE.Vector3(0, RIDE, 0)); G.add(BD); this.bodyGroup = BD;
    const t = 0.012;
    const fl = FLOOR_Y;
    box(BED_W + 0.04, 0.06, BED_L, m.bodyPaint, 0, fl - 0.03, BED_L / 2, BD);
    for (const s of [1, -1]) {
      box(0.05, BED_SIDE, BED_L, m.bodyPaint, s * (BED_W / 2 + 0.025), fl + BED_SIDE / 2, BED_L / 2, BD);
      box(0.12, 0.12, BED_L + 0.1, m.bodyPaint, s * (BED_W / 2 + 0.06), fl + BED_SIDE - 0.02, BED_L / 2, BD); // top rail
      for (let i = 0; i < 6; i++) box(0.06, BED_SIDE - 0.1, 0.1, m.bodyPaint, s * (BED_W / 2 + 0.07), fl + BED_SIDE / 2, 0.4 + i * 0.85, BD); // side ribs
      box(0.12, 0.2, 0.25, m.redLens, s * (BED_W / 2 + 0.05), fl + 0.15, 0.05, BD).scale.set(0.5, 0.5, 0.3);
    }
    // long sills under floor
    for (const s of [1, -1]) box(0.1, 0.22, BED_L, m.black, s * 0.44, fl - 0.17, BED_L / 2, BD);
    // front wall + cab shield
    box(BED_W + 0.1, BED_SIDE + 0.35, 0.05, m.bodyPaint, 0, fl + (BED_SIDE + 0.35) / 2, BED_L, BD);
    const shield = box(BED_W + 0.1, 0.05, 0.24, m.bodyPaint, 0, fl + BED_SIDE + 0.35, BED_L + 0.1, BD);
    shield.rotation.x = -0.1;
    for (let i = 0; i < 4; i++) box(0.06, BED_SIDE + 0.3, 0.08, m.bodyPaint, -0.9 + i * 0.6, fl + BED_SIDE / 2, BED_L + 0.05, BD);
    // tailgate (top-hinged)
    const TG = new THREE.Group(); TG.position.set(0, fl + BED_SIDE + 0.02, -0.04); BD.add(TG); this.tailGroup = TG;
    box(BED_W + 0.04, BED_SIDE + 0.02, 0.06, m.bodyPaint, 0, -(BED_SIDE + 0.02) / 2, 0, TG);
    for (let i = 0; i < 4; i++) box(BED_W, 0.06, 0.08, m.bodyPaint, 0, -0.15 - i * 0.3, -0.05, TG);
    const stripe = new THREE.Mesh(new THREE.PlaneGeometry(BED_W, 0.18), m.stripes); stripe.position.set(0, -BED_SIDE + 0.12, -0.081); stripe.rotation.y = Math.PI; TG.add(stripe);
    cyl(0.04, BED_W + 0.3, m.steel, 'x', 0, 0, 0, TG);
    // hinge pivots
    cyl(0.06, 1.0, m.steel, 'x', 0, 0, 0, BD);
    // ---- load mesh ----
    const lg = new THREE.PlaneGeometry(BED_W - 0.02, BED_L - 0.02, BED_NX - 1, BED_NZ - 1);
    lg.rotateX(-Math.PI / 2);
    lg.translate(0, fl, BED_L / 2);
    this.loadGeo = lg;
    this.loadMat = new THREE.MeshStandardMaterial({ color: 0x7c5230, roughness: 1, map: this.detailTex || null });
    this.loadMesh = new THREE.Mesh(lg, this.loadMat);
    this.loadMesh.castShadow = true; this.loadMesh.receiveShadow = true;
    BD.add(this.loadMesh);
    // hoist: front-mount telescopic cylinder
    const hoistA = marker(G, 0, 0.95, 1.22);
    const hoistB = marker(BD, 0, 0.55, BED_L + 0.12);
    this.hoist = new HydCylinder(G, hoistA, hoistB, { rBarrel: 0.13, rRod: 0.09, barrel: 0.6, mat: m.black, rodMat: m.chrome });
    // ---- wheels ----
    this.wheelDefs = [];
    const addWheel = (x, z, steer, driven, dual) => {
      const w = new THREE.Group();
      G.add(w);
      const spin = new THREE.Group(); w.add(spin);
      const mk = (ox) => {
        const tyre = new THREE.Mesh(tyreGeo(), m.tyre); tyre.position.x = ox; spin.add(tyre);
        const rim = new THREE.Mesh(rimGeo(), m.wornSteel); rim.position.x = ox; spin.add(rim);
      };
      if (dual) { mk(0.17); mk(-0.17); } else mk(0);
      const hub = cyl(0.12, dual ? 0.66 : 0.38, m.darkSteel, 'x', 0, 0, 0, spin, 12);
      void hub;
      this.wheelDefs.push({ x, z, steer, driven, dual, group: w, spin, comp: 0, compV: 0, rot: 0, contact: new Contact(new THREE.Vector3(x, 0, z), { k: 1, c: 1, area: dual ? 0.12 : 0.06 }) });
    };
    for (const s of [1, -1]) addWheel(s * S.trackFront / 2, 3.0, true, false, false);
    for (const z of [-1.715, -3.085]) for (const s of [1, -1]) addWheel(s * S.trackRear / 2, z, false, true, true);
    // axles
    box(1.9, 0.14, 0.14, m.black, 0, R, 3.0, G);
    for (const z of [-1.715, -3.085]) { box(1.6, 0.2, 0.2, m.black, 0, R, z, G); cyl(0.28, 0.4, m.black, 'x', 0, R, z, G); }
    for (const s of [1, -1]) box(0.1, 0.12, 1.9, m.black, s * 0.45, R + 0.2, -2.4, G); // walking beams
    shadowAll(this.root);
    // collision
    this.addBox(G, 0, 2.15, 2.15, 1.2, 1.05, 0.82);
    this.addBox(G, 0, 1.5, 3.85, 1.15, 0.55, 0.85);
    this.addBox(G, 0, 0.75, -0.9, 1.25, 0.3, 3.0);
    this.bedBox = this.addBox(BD, 0, fl + BED_SIDE / 2, BED_L / 2, BED_W / 2 + 0.1, BED_SIDE / 2 + 0.05, BED_L / 2);
  }

  setupPhysics() {
    const self = this;
    this.masses = [
      { m: S.emptyMass - 3600, pos: () => new THREE.Vector3(0, 0.0, 1.2) },
      { m: 3600, pos: () => self.bedLocalToBody(new THREE.Vector3(0, 0.5, BED_L / 2)) },
      { m: () => self.loadMass(), pos: () => self.loadCOM() },
    ];
  }

  bedLocalToBody(v) {
    // rotate about hinge by tip angle (front up)
    const c = Math.cos(this.tip), s = Math.sin(this.tip);
    const y = v.y * c + v.z * s, z = -v.y * s + v.z * c;
    return v.set(v.x + HINGE.x, y + HINGE.y, z + HINGE.z);
  }

  loadVolume() { let s = 0; for (let i = 0; i < this.bed.length; i++) s += this.bed[i]; return s * BED_CS * BED_CS; }
  loadMass() { return this.loadVolume() * SOIL.densLoose[Math.round(this.bedType)]; }
  loadCOM() {
    let sx = 0, sz = 0, sy = 0, sm = 0;
    for (let j = 0; j < BED_NZ; j++) for (let i = 0; i < BED_NX; i++) {
      const h = this.bed[j * BED_NX + i];
      if (h <= 0) continue;
      sx += h * this.cellX(i); sz += h * this.cellZ(j); sy += h * (FLOOR_Y + h / 2); sm += h;
    }
    if (sm <= 0) return this.bedLocalToBody(new THREE.Vector3(0, 0.5, BED_L / 2));
    return this.bedLocalToBody(new THREE.Vector3(sx / sm, sy / sm, sz / sm));
  }
  cellX(i) { return -BED_W / 2 + (i + 0.5) * (BED_W / BED_NX); }
  cellZ(j) { return (j + 0.5) * (BED_L / BED_NZ); }

  /** Particle receiver: catch falling soil into the body. */
  tryCatch(x, y, z, vol, type) {
    if (Math.abs(x - this.body.pos.x) > 6 || Math.abs(z - this.body.pos.z) > 6) return false;
    if (!this._bedInv) return false;
    const p = new THREE.Vector3(x, y, z).applyMatrix4(this._bedInv);
    if (p.x < -BED_W / 2 || p.x > BED_W / 2 || p.z < 0 || p.z > BED_L) {
      // landing on the side rails / cab shield -> deflects; let it fall
      return false;
    }
    const i = clamp(Math.floor((p.x + BED_W / 2) / (BED_W / BED_NX)), 0, BED_NX - 1);
    const j = clamp(Math.floor(p.z / (BED_L / BED_NZ)), 0, BED_NZ - 1);
    const k = j * BED_NX + i;
    if (p.y > FLOOR_Y + this.bed[k] + 0.05) return false;
    if (p.y < FLOOR_Y - 0.5) return false;
    const cellA = (BED_W / BED_NX) * (BED_L / BED_NZ);
    const total = this.loadVolume();
    this.bedType = (this.bedType * total + type * vol) / (total + vol);
    this.bed[k] += vol / cellA;
    if (Math.random() < 0.15) this.particles.dust.puff(x, y + 0.2, z, 0.8, type);
    return true;
  }

  control(inp) { this.inp = inp; if (inp.park) this.parkBrake = !this.parkBrake; }

  step(dt) {
    const inp = this.inp || {};
    this._bedAcc += dt;
    while (this._bedAcc >= 1 / 60) { this._bedAcc -= 1 / 60; this.bedStep(1 / 60, inp); }
    this.applyGravity();
    const fwd = new THREE.Vector3(0, 0, 1).applyQuaternion(this.body.quat);
    const up = new THREE.Vector3(0, 1, 0).applyQuaternion(this.body.quat);
    const v = this.body.vel.dot(fwd);
    this.vlong = v;
    // driver: W accelerate, S brake (then reverse when stopped)
    let thr = clamp(inp.throttle || 0, 0, 1), brk = clamp(inp.brake || 0, 0, 1);
    if (this.reverse) { const t = thr; thr = brk; brk = t; }
    if (Math.abs(v) < 0.4 && (inp.brake || 0) > 0.5 && !this.reverse && (inp.throttle || 0) < 0.1) { this._revT = (this._revT || 0) + dt; if (this._revT > 0.5) { this.reverse = true; this._revT = 0; } }
    else if (Math.abs(v) < 0.4 && (inp.throttle || 0) > 0.5 && this.reverse) { this._revT = (this._revT || 0) + dt; if (this._revT > 0.5) { this.reverse = false; this._revT = 0; } }
    else this._revT = 0;
    if (thr > 0.05 || brk > 0.05) this.parkBrake = false;
    // steering
    const steerTarget = clamp(inp.steer || 0, -1, 1) * 38 * DEG * (1 / (1 + Math.abs(v) * 0.04));
    this.steer += clamp(steerTarget - this.steer, -0.9 * dt, 0.9 * dt);
    // drivetrain
    const ratio = this.reverse ? S.reverse : S.gears[this.gear - 1];
    const wheelRpm = Math.abs(v) / R * 60 / (2 * Math.PI);
    const shaftRpm = wheelRpm * ratio * S.finalDrive;
    const stall = 1250;
    const tcRpm = this.idleRpm + (stall - this.idleRpm) * thr;
    const locked = shaftRpm > 1100 && this.gear >= 2;
    const rpm = locked ? Math.max(shaftRpm, this.idleRpm) : Math.max(shaftRpm, tcRpm);
    this.rpm += (Math.min(rpm, 2150) - this.rpm) * Math.min(1, dt * 8);
    // torque converter multiplication
    const sr = clamp(shaftRpm / Math.max(this.rpm, 1), 0, 1);
    const tcMul = locked ? 1 : 1 + 0.7 * (1 - sr);
    const gov = this.rpm > 2100 ? 0 : 1;
    const Teng = torqueCurve(this.rpm) * thr * gov;
    let Fdrive = Teng * tcMul * ratio * S.finalDrive * 0.84 / R * (this.reverse ? -1 : 1);
    // flywheel power limit (450 hp minus accessories / driveline losses)
    const Pmax = S.enginePower * 0.62 * clamp(this.rpm / 1500, 0.3, 1);
    if (Math.abs(Fdrive * v) > Pmax) Fdrive = Math.sign(Fdrive) * Pmax / Math.abs(v);
    if (this.shiftT > 0) { this.shiftT -= dt; Fdrive *= 0.25; }
    // automatic shifting
    if (!this.reverse && this.shiftT <= 0) {
      if (shaftRpm > 1950 && this.gear < 6) { this.gear++; this.shiftT = 0.45; }
      else if (shaftRpm < 1000 && this.gear > 1) { this.gear--; this.shiftT = 0.35; }
    }
    if (this.reverse) this.gear = 1;
    this.engineLoad += (thr * (0.4 + 0.6 * clamp(1 - Math.abs(v) / 15, 0, 1)) - this.engineLoad) * Math.min(1, dt * 4);
    const holding = this.parkBrake || (thr < 0.05 && Math.abs(v) < 0.25 && brk < 0.05);
    // suspension + tyres
    let driven = 0;
    for (const w of this.wheelDefs) if (w.driven && w.ground) driven++;
    const P = new THREE.Vector3(), Wp = new THREE.Vector3(), N = new THREE.Vector3(), V = new THREE.Vector3();
    let nGround = 0, slip = 0;
    for (const w of this.wheelDefs) {
      // mount point in world
      P.set(w.x, MOUNT_Y, w.z);
      this.body.toWorld(P, Wp);
      const gy = this.terrain.heightAt(Wp.x, Wp.z);
      const dist = (Wp.y - gy) / Math.max(up.y, 0.3); // along body down
      const comp = REST + R - dist;
      w.ground = comp > 0;
      const prev = w.comp;
      w.comp = clamp(comp, -0.2, 0.35);
      w.compV = (w.comp - prev) / dt;
      if (!w.ground) { w.contact.ground = false; w.contact.Fn = 0; continue; }
      nGround++;
      const k = w.driven ? 4.6e5 : 3.4e5, c = w.driven ? 3.2e4 : 2.4e4;
      let Fs = k * Math.min(comp, 0.33) + c * w.compV;
      if (comp > 0.33) Fs += (comp - 0.33) * 4e6; // bump stop
      Fs = Math.max(0, Fs);
      // tyre patch
      const cp = Wp.clone().addScaledVector(up, -(dist));
      cp.y = gy;
      this.terrain.normalAt(cp.x, cp.z, N);
      // suspension force along body up
      this.body.addForceAt(up.clone().multiplyScalar(Fs), cp);
      // friction: emulate normal force via contact solver with forced Fn
      const ct = w.contact;
      ct.local.set(w.x, -RIDE, w.z);
      const dir = new THREE.Vector3(Math.sin(w.steer ? this.steer : 0), 0, Math.cos(w.steer ? this.steer : 0));
      ct.dir.copy(dir);
      const loose = this.terrain.looseAt(cp.x, cp.z);
      const mu = 0.75 - Math.min(0.25, loose * 0.8);
      const rr = 0.03 + Math.min(0.18, loose * 0.6);
      const wheelFn = Fs;
      const brakeCap = (holding ? 1 : brk) * 0.85 * wheelFn;
      const mode = brakeCap > 1
        ? { belt: 0, muLong: mu, muLat: mu * 0.95, longCap: brakeCap, rollRes: rr }
        : { free: true, drive: w.driven && driven ? Fdrive / driven : 0, muLong: mu, muLat: mu * 0.95, rollRes: rr };
      mode.print = { hw: w.dual ? 0.3 : 0.15, type: 4 };
      tyreFriction(this.body, ct, cp, N, wheelFn, dt, mode, this.terrain);
      slip += ct.slip;
      w.rotV = this.body.pointVel(cp, V).dot(new THREE.Vector3().copy(dir).applyQuaternion(this.body.quat)) / R;
      if (w.driven && Math.abs(Fdrive / Math.max(driven, 1)) > mu * wheelFn * 1.05) { w.rotV += Math.sign(Fdrive) * 4; w.spinning = true; } else w.spinning = false;
    }
    this.grounded = nGround > 0;
    this.trackSlip = nGround ? slip / nGround : 0;
    this.body.integrate(dt);
  }

  bedStep(dt, inp) {
    // hoist
    const want = clamp(inp.bed || 0, -1, 1);
    const rate = want > 0 ? (S.maxTip * DEG) / 13 : (S.maxTip * DEG) / 9;
    this.tip = clamp(this.tip + want * rate * dt, 0, S.maxTip * DEG);
    this.hoistActive = Math.abs(want) > 0.05 && this.tip > 0 && this.tip < S.maxTip * DEG;
    // tailgate swings open by gravity once the body is raised
    const tgTarget = this.tip > 6 * DEG ? Math.min(this.tip * 1.1, 1.1) : 0;
    this.tailgate += (tgTarget - this.tailgate) * Math.min(1, dt * 3);
    // granular flow inside the body (effective height along world vertical)
    const bedM = this._bedM;
    if (!bedM) return;
    const e = bedM.elements;
    const ux = e[1], uy = e[5], uz = e[9]; // world Y component of bed local axes (x,y,z)
    const tanR = Math.tan(SOIL.REPOSE_LOOSE_DEG * DEG);
    const b = this.bed, NX = BED_NX, NZ = BED_NZ;
    const dx = BED_W / NX, dz = BED_L / NZ;
    const eff = (i, j) => this.cellX(i) * ux + (FLOOR_Y + b[j * NX + i]) * uy + this.cellZ(j) * uz;
    const flow = (ka, kb, Ya, Yb, d) => {
      let dh = Ya - Yb, src = ka, dst = kb;
      if (dh < 0) { dh = -dh; src = kb; dst = ka; }
      const lim = tanR * d;
      if (dh > lim && b[src] > 0) {
        const m = Math.min(b[src], (dh - lim) * 0.25 / Math.max(uy, 0.3));
        b[src] -= m; b[dst] += m;
      }
    };
    for (let it = 0; it < 2; it++) {
      for (let j = 0; j < NZ; j++) for (let i = 0; i < NX; i++) {
        const k = j * NX + i;
        if (i < NX - 1) flow(k, k + 1, eff(i, j), eff(i + 1, j), dx);
        if (j < NZ - 1) flow(k, k + NX, eff(i, j), eff(i, j + 1), dz);
      }
    }
    // the whole load slides on the steel floor once the body is steeper than the
    // soil-on-steel friction angle (~28 deg), even below the soil's repose angle
    const floorTan = uz / Math.max(uy, 0.05);
    const slideRate = clamp((floorTan - Math.tan(28 * DEG)) * 2.5, 0, 3) * dt;
    if (slideRate > 0) {
      for (let j = 0; j < NZ; j++) for (let i = 0; i < NX; i++) {
        const k = j * NX + i;
        if (b[k] <= 0) continue;
        const m = b[k] * Math.min(0.5, slideRate * (BED_L / NZ) * 4);
        if (j > 0) { b[k] -= m; b[k - NX] += m; }
      }
    }
    // outflow over the tail (tailgate open) and over the side rails
    const cellA = dx * dz;
    const out = (k, vol, lx, ly, lz, vx) => {
      if (vol <= 0) return;
      b[k] -= vol / cellA;
      const wp = new THREE.Vector3(lx, ly, lz).applyMatrix4(bedM);
      const vel = new THREE.Vector3();
      this.body.pointVel(wp, vel);
      this.particles.emit(wp.x, wp.y, wp.z, vel.x + vx.x, vel.y + vx.y, vel.z + vx.z, vol, this.bedType, 0.35);
      if (this.inPad(wp.x, wp.z)) this.dumpedInPad += vol;
      this.dumpedTotal = (this.dumpedTotal || 0) + vol;
    };
    const back = new THREE.Vector3(-e[8], -e[9], -e[10]).multiplyScalar(1.2);
    if (this.tailgate > 0.15) {
      for (let i = 0; i < NX; i++) {
        const k = i; // j = 0 row at tail
        const Y0 = eff(i, 0);
        const Ylip = this.cellX(i) * ux + FLOOR_Y * uy + (-dz * 0.5) * uz;
        const ex = Y0 - Ylip - tanR * dz * 0.5;
        if ((ex > 0 || slideRate > 0) && b[k] > 0) {
          const m = Math.min(b[k], Math.max(0, ex) * 0.5 / Math.max(uy, 0.3) + 0.002 + b[k] * Math.min(0.5, slideRate * 4)) * cellA;
          out(k, m, this.cellX(i), FLOOR_Y + b[k] * 0.5, -0.1, back);
        }
      }
    }
    for (let j = 0; j < NZ; j++) for (const i of [0, NX - 1]) {
      const k = j * NX + i;
      if (b[k] > BED_SIDE + 0.05) {
        const m = (b[k] - BED_SIDE) * 0.3 * cellA;
        const side = i === 0 ? -1 : 1;
        out(k, m, side * (BED_W / 2 + 0.15), FLOOR_Y + BED_SIDE + 0.05, this.cellZ(j), new THREE.Vector3(e[0] * side, 0, e[2] * side));
      }
    }
    // front over cab shield
    for (let i = 0; i < NX; i++) {
      const k = (NZ - 1) * NX + i;
      if (b[k] > BED_SIDE + 0.4) out(k, (b[k] - BED_SIDE - 0.4) * 0.3 * cellA, this.cellX(i), FLOOR_Y + BED_SIDE + 0.45, BED_L + 0.1, new THREE.Vector3());
    }
    for (let k = 0; k < b.length; k++) if (b[k] < 1e-5) b[k] = 0;
  }

  inPad(x, z) { const p = SITE.pad; return x > p.x0 && x < p.x1 && z > p.z0 && z < p.z1; }

  updateVisuals(dt) {
    this.syncRoot();
    this.bodyGroup.rotation.x = -this.tip;
    this.tailGroup.rotation.x = -this.tailgate * 0.9;
    // wheels
    for (const w of this.wheelDefs) {
      const sag = clamp(w.comp, -0.2, 0.33);
      w.group.position.y = MOUNT_Y + RIDE - (REST - sag);
      w.group.rotation.y = w.steer ? this.steer : 0;
      w.rot += (w.rotV || 0) * dt;
      w.spin.rotation.x = w.rot;
    }
    this.steerWheel.rotation.z = -this.steer * 12;
    this.root.updateMatrixWorld();
    this.hoist.update();
    // body matrix (for particles + bed flow)
    this._bedM = this.bodyGroup.matrixWorld.clone();
    this._bedInv = this._bedM.clone().invert();
    // load mesh
    const pos = this.loadGeo.attributes.position;
    const b = this.bed;
    let any = false;
    for (let j = 0; j < BED_NZ; j++) for (let i = 0; i < BED_NX; i++) {
      // PlaneGeometry rotated: rows go along -z after rotation; map index carefully
      const vi = j * BED_NX + i;
      const h = b[j * BED_NX + i];
      if (h > 0.005) any = true;
      pos.setY(vi, FLOOR_Y + (h > 0.005 ? h : -0.03));
    }
    pos.needsUpdate = true;
    this.loadGeo.computeVertexNormals();
    this.loadMesh.visible = any;
    const col = [0x4f3a26, 0x7c5230, 0x77695a, 0x606060][Math.round(clamp(this.bedType, 0, 3))];
    this.loadMat.color.setHex(col);
    this.emitExhaust(dt);
    for (const w of this.wheelDefs) if (w.spinning && Math.random() < 0.3) {
      const p = new THREE.Vector3(w.x, -RIDE + 0.1, w.z - 0.4).applyQuaternion(this.body.quat).add(this.body.pos);
      this.particles.dust.puff(p.x, p.y, p.z, 0.9, 0);
    }
  }

  hud() {
    const vol = this.loadVolume();
    const mass = this.loadMass();
    const gvw = S.emptyMass + mass;
    return {
      title: S.name,
      sub: S.className,
      rows: [
        ['Engine', `${Math.round(this.rpm)} rpm`],
        ['Gear', this.reverse ? 'R' : `${this.gear}${this.shiftT > 0 ? ' (shifting)' : ''}`],
        ['Speed', `${Math.abs(this.vlong * 3.6 || 0).toFixed(1)} km/h`],
        ['Payload', `${vol.toFixed(2)} m³ · ${(mass / 1000).toFixed(1)} t`],
        ['Gross weight', `${(gvw / 1000).toFixed(1)} t / ${(S.gvwr / 1000).toFixed(1)} t${gvw > S.gvwr ? '  OVERLOAD' : ''}`],
        ['Body', this.tip > 0.01 ? `raised ${(this.tip / DEG).toFixed(0)}°` : 'down'],
        ['Park brake', this.parkBrake ? 'ON' : 'off'],
      ],
      gauge: clamp(mass / (S.gvwr - S.emptyMass), 0, 1.2),
      stall: gvw > S.gvwr,
    };
  }
}

// --- tyre friction: uses the contact bristle model with an externally given normal force
const _f = new THREE.Vector3(), _l = new THREE.Vector3(), _v = new THREE.Vector3(), _F = new THREE.Vector3();
function tyreFriction(body, c, p, n, Fn, dt, mode, terrain) {
  body.pointVel(p, _v);
  body.dirToWorld(c.dir, _f);
  _f.addScaledVector(n, -_f.dot(n)).normalize();
  _l.crossVectors(n, _f);
  const vl = _v.dot(_f), vt = _v.dot(_l);
  const B = 0.02;
  // lateral: slip-angle-like at speed, bristle at low speed
  const muT = mode.muLat * Fn;
  c.zt += vt * dt;
  let Ft = -(muT / B * c.zt + Fn * 0.25 * vt);
  let slip = 0;
  if (Math.abs(Ft) > muT) { Ft = Math.sign(Ft) * muT; c.zt = Math.sign(c.zt) * B; slip = Math.abs(vt); }
  let Fl;
  if (mode.free) {
    Fl = mode.drive - mode.rollRes * Fn * Math.tanh(vl / 0.3);
    c.zl = 0;
    const cap = mode.muLong * Fn;
    if (Math.abs(Fl) > cap) { Fl = Math.sign(Fl) * cap; slip = Math.max(slip, 1); }
  } else {
    const cap = Math.min(mode.longCap, mode.muLong * Fn);
    c.zl += vl * dt;
    Fl = -(cap / B * c.zl + Fn * 0.25 * vl);
    if (Math.abs(Fl) > cap) { Fl = Math.sign(Fl) * cap; c.zl = Math.sign(c.zl) * B; slip = Math.max(slip, Math.abs(vl) * 0.3); }
    Fl -= mode.rollRes * Fn * Math.tanh(vl / 0.3);
  }
  const mag = Math.hypot(Fl / mode.muLong, Ft / mode.muLat);
  if (mag > Fn) { const s = Fn / mag; Fl *= s; Ft *= s; }
  c.slip = slip;
  _F.copy(_f).multiplyScalar(Fl).addScaledVector(_l, Ft);
  body.addForceAt(_F, p);
  terrain.compact(p.x, p.z, Fn / c.area * 0.6, dt, slip, mode.print ? { dx: _f.x, dz: _f.z, hw: mode.print.hw, type: mode.print.type } : undefined);
  c.world.copy(p); c.Fn = Fn; c.ground = true;
}
void resolveContact;

let _tyre = null, _rim = null;
function tyreGeo() {
  if (_tyre) return _tyre;
  const pts = [];
  const r = R, w = 0.3, ri = 0.29;
  pts.push(new THREE.Vector2(ri, -w / 2 + 0.02));
  pts.push(new THREE.Vector2(r - 0.06, -w / 2));
  pts.push(new THREE.Vector2(r - 0.01, -w / 2 + 0.04));
  pts.push(new THREE.Vector2(r, -w / 2 + 0.08));
  pts.push(new THREE.Vector2(r, w / 2 - 0.08));
  pts.push(new THREE.Vector2(r - 0.01, w / 2 - 0.04));
  pts.push(new THREE.Vector2(r - 0.06, w / 2));
  pts.push(new THREE.Vector2(ri, w / 2 - 0.02));
  const g = new THREE.LatheGeometry(pts, 40);
  g.rotateZ(Math.PI / 2);
  // uv for tread bump (u around)
  _tyre = g;
  return g;
}
function rimGeo() {
  if (_rim) return _rim;
  const pts = [new THREE.Vector2(0.05, -0.12), new THREE.Vector2(0.2, -0.13), new THREE.Vector2(0.29, -0.14), new THREE.Vector2(0.29, 0.14), new THREE.Vector2(0.2, 0.06), new THREE.Vector2(0.05, 0.07)];
  const g = new THREE.LatheGeometry(pts, 24);
  g.rotateZ(Math.PI / 2);
  _rim = g;
  return g;
}
