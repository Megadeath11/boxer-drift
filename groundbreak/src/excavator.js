// 20 t class hydraulic crawler excavator.
// Kinematics: swing -> boom (5.0 m chord) -> stick (2.9 m) -> bucket (1.45 m tip radius)
// Digging: the bucket shell is rasterised into the soil grid each hydraulic step.
// Cut volume costs energy (soil specific cutting energy); joint motion is scaled
// back to what the hydraulic power / force limits allow, and bedrock blocks it.
import * as THREE from 'three';
import { Machine } from './machine.js';
import { Contact, resolveContact } from './physics.js';
import { EXCAVATOR as S, SOIL, SITE } from './config.js';
import { clamp, smoothstep, DEG } from './util.js';
import {
  materials, rbox, box, cyl, profile, plan, plate, HydCylinder, Strut, marker, TrackBelt,
  sprocket, idler, shadowAll,
} from './parts.js';
import { makeTextTexture } from './textures.js';

const LB = S.boomChord, LS = S.stickLength;
const FOOT = new THREE.Vector3(-0.12, 0.85, 1.0); // boom foot pin in upper frame
const UPPER_Y = 0.05;                              // upper frame bottom in body frame
const SHELL = [[1.45, 0.0], [1.38, 0.22], [1.24, 0.45], [1.02, 0.66], [0.74, 0.82], [0.42, 0.88],
  [0.14, 0.80], [-0.04, 0.58], [-0.08, 0.32], [0.0, 0.10]];
const L1 = [LS - 0.45, 0.14]; // rocker pin on stick
const B2 = [-0.12, 0.40];     // coupler pin on bucket (u,w)
const ROCKER = 0.52, COUPLER = 0.56;

const LIM = {
  a1: [-55 * DEG, 62 * DEG],
  r2: [-160 * DEG, -28 * DEG],
  r3: [-155 * DEG, 32 * DEG],
};
const RATE = { swing: S.swingSpeed * 2 * Math.PI / 60, a1: 0.62, r2: 0.78, r3: 1.15 };
const ACC = { swing: 0.9, a1: 2.2, r2: 2.6, r3: 3.5 };

function shellSamples() {
  const pts = [];
  for (let i = 0; i < SHELL.length - 1; i++) {
    const [u0, w0] = SHELL[i], [u1, w1] = SHELL[i + 1];
    const L = Math.hypot(u1 - u0, w1 - w0);
    const n = Math.max(1, Math.ceil(L / 0.08));
    for (let k = 0; k < n; k++) pts.push([u0 + (u1 - u0) * k / n, w0 + (w1 - w0) * k / n]);
  }
  pts.push(SHELL[SHELL.length - 1]);
  // interior fill points so that the whole bucket volume rasterises on the 12.5 cm grid
  const inside = (u, w) => {
    let c = false;
    for (let i = 0, j = SHELL.length - 1; i < SHELL.length; j = i++) {
      const [ui, wi] = SHELL[i], [uj, wj] = SHELL[j];
      if ((wi > w) !== (wj > w) && u < ((uj - ui) * (w - wi)) / (wj - wi) + ui) c = !c;
    }
    return c;
  };
  for (let u = 0.0; u <= 1.45; u += 0.11) for (let w = 0.02; w <= 0.9; w += 0.11) if (inside(u, w)) pts.push([u, w]);
  const out = [];
  for (let x = -0.62; x <= 0.621; x += 0.1) for (const [u, w] of pts) out.push([x, u, w]);
  return out;
}
const SAMPLES = shellSamples();

export class Excavator extends Machine {
  constructor(scene, terrain, particles) {
    super(scene, terrain, particles, S.operatingMass, new THREE.Vector3(62000, 72000, 30000));
    this.spec = S;
    this.kind = 'excavator';
    this.trackPrint = { hw: 0.15, type: 3 };
    this.label = S.name;
    this.idleRpm = 950; this.maxRpm = 1800;
    this.radius = 11;
    this.swing = 0; this.a1 = 28 * DEG; this.r2 = -115 * DEG; this.r3 = -105 * DEG;
    this.w = { swing: 0, a1: 0, r2: 0, r3: 0 };
    this.load = 0; this.loadType = 0;
    this.pressure = 0; this.stall = 0;
    this.reaction = new THREE.Vector3();
    this.reactionPoint = new THREE.Vector3();
    this._hydAcc = 0;
    this.highSpeed = false;
    this.throttle = 0.9;
    this._mUpper = new THREE.Matrix4();
    this.tracksOffset = { L: 0, R: 0 };
    this.build();
    this.setupPhysics();
  }

  get capacity() { return S.bucketCapacity; }

  onReset() {
    for (const k of ['swing', 'a1', 'r2', 'r3']) if (!Number.isFinite(this[k])) this[k] = { swing: 0, a1: 0.5, r2: -2.0, r3: -1.8 }[k];
    for (const k in this.w) this.w[k] = 0;
    this.reaction.set(0, 0, 0);
  }

  // ---------------------------------------------------------------- model
  build() {
    const m = materials();
    const R = this.root;
    // -------- undercarriage (ground-referenced coordinates) --------
    const U = new THREE.Group(); U.position.y = -1.0; R.add(U); this.under = U;
    const gauge = S.trackGauge / 2;
    const circles = [{ z: -1.825, y: 0.45, r: 0.40 }, { z: 1.825, y: 0.45, r: 0.40 }];
    this.belts = [];
    for (const s of [1, -1]) {
      const x = s * gauge;
      const fr = new THREE.Mesh(profile([[-1.55, 0.26], [1.5, 0.26], [1.62, 0.5], [1.45, 0.66], [-1.45, 0.66], [-1.62, 0.5]], 0.42, 0.03), m.paint);
      fr.position.x = x; U.add(fr);
      // track guard plates
      box(0.03, 0.18, 2.8, m.darkSteel, x + s * 0.2, 0.25, 0, U);
      const spr = sprocket(0.37, 0.24, 21, m.darkSteel); spr.position.set(x, 0.45, -1.825); U.add(spr);
      cyl(0.28, 0.32, m.paint, 'x', x - s * 0.3, 0.45, -1.825, U, 20);
      const idl = idler(0.36, 0.26, m.darkSteel); idl.position.set(x, 0.45, 1.825); U.add(idl);
      box(0.3, 0.2, 0.6, m.paint, x, 0.5, 1.25, U); // recoil housing
      for (let i = 0; i < 7; i++) {
        const r = idler(0.105, 0.3, m.darkSteel); r.position.set(x, 0.165, -1.32 + i * 0.44); U.add(r);
      }
      for (const z of [-0.6, 0.6]) { const r = idler(0.085, 0.2, m.darkSteel); r.position.set(x, 0.765, z); U.add(r); }
      const belt = new TrackBelt(U, circles, { x, shoeWidth: S.shoeWidth, pitch: 0.19 });
      this.belts.push(belt);
    }
    // car body + X frame
    rbox(1.5, 0.45, 1.6, 0.06, m.paint, 0, 0.8, 0, U);
    for (const s of [1, -1]) for (const z of [-0.55, 0.55]) {
      const leg = rbox(0.9, 0.34, 0.5, 0.05, m.paint, s * 0.7, 0.62, z, U);
      leg.rotation.y = s * z > 0 ? 0.35 : -0.35;
    }
    cyl(0.8, 0.16, m.darkSteel, 'y', 0, 1.0, 0, U, 40);
    cyl(0.68, 0.05, m.black, 'y', 0, 1.09, 0, U, 40);

    // -------- upper structure --------
    const Up = new THREE.Group(); Up.position.y = UPPER_Y; R.add(Up); this.upper = Up;
    rbox(2.6, 0.26, 3.4, 0.05, m.darkSteel, 0, 0.13, -0.5, Up);
    // counterweight: rear arc at tail swing radius
    const cw = [];
    cw.push([1.47, -1.72]);
    for (let i = 0; i <= 16; i++) {
      const x = 1.47 - (2.94 * i) / 16;
      cw.push([x, -Math.sqrt(S.tailSwingRadius ** 2 - x * x)]);
    }
    cw.push([-1.47, -1.72]);
    const cwM = new THREE.Mesh(plan(cw, 1.12, 0.06), m.paint);
    cwM.position.y = 0.02; Up.add(cwM);
    // rear reflector strips + lettering on counterweight
    for (const s of [1, -1]) box(0.28, 0.08, 0.02, m.redLens, s * 0.85, 0.75, -2.72 + Math.abs(s * 0.85) * 0.14, Up);
    const label = new THREE.Mesh(new THREE.PlaneGeometry(1.1, 0.26), new THREE.MeshStandardMaterial({ map: makeTextTexture('EX-20', { color: '#151515' }), transparent: true, roughness: 0.5 }));
    label.position.set(0, 0.45, -2.84); label.rotation.y = Math.PI; Up.add(label);
    // engine compartment & hood
    rbox(2.92, 1.2, 1.32, 0.08, m.paint, 0, 0.85, -1.15, Up);
    for (let i = 0; i < 7; i++) box(0.02, 0.5, 0.9, m.black, 1.47, 0.85, -1.15 + 0, Up).position.z = -1.5 + i * 0.12;
    for (let i = 0; i < 6; i++) box(1.4, 0.015, 0.05, m.black, -0.5, 1.455, -1.55 + i * 0.13, Up);
    // right-front tank / toolbox
    rbox(1.18, 1.08, 1.55, 0.07, m.paint, -0.86, 0.79, 0.2, Up);
    box(0.5, 0.06, 0.4, m.darkSteel, -0.8, 1.36, -0.1, Up); // filler cap pad
    cyl(0.08, 0.05, m.black, 'y', -0.7, 1.38, 0.5, Up);
    // walkway / anti-skid
    box(1.2, 0.02, 1.5, m.darkSteel, -0.86, 1.34, 0.2, Up);
    // exhaust and air pre-cleaner
    cyl(0.065, 0.55, m.black, 'y', -0.55, 1.7, -1.05, Up);
    this.exhaust = marker(Up, -0.55, 2.0, -1.05);
    cyl(0.12, 0.28, m.black, 'y', 0.6, 1.58, -1.4, Up, 16);
    cyl(0.14, 0.06, m.black, 'y', 0.6, 1.75, -1.4, Up, 16);
    // handrails
    const rail = (x0, z0, x1, z1, y) => {
      const a = new THREE.Vector3(x0, y, z0), b = new THREE.Vector3(x1, y, z1);
      const g = new THREE.CylinderGeometry(0.018, 0.018, a.distanceTo(b), 8);
      const mm = new THREE.Mesh(g, m.paint);
      mm.position.copy(a).add(b).multiplyScalar(0.5);
      mm.quaternion.setFromUnitVectors(new THREE.Vector3(0, 1, 0), b.clone().sub(a).normalize());
      Up.add(mm);
    };
    rail(-1.42, -0.5, -1.42, 0.9, 2.0); rail(-1.42, -0.5, -1.42, -0.5, 1.33);
    rail(-1.42, 0.9, -1.42, 0.9, 1.33);
    for (const z of [-0.5, 0.2, 0.9]) rail(-1.42, z, -1.42, z, 2.0);
    for (const z of [-0.5, 0.2, 0.9]) { const g = new THREE.CylinderGeometry(0.018, 0.018, 0.67, 8); const mm = new THREE.Mesh(g, m.paint); mm.position.set(-1.42, 1.66, z); Up.add(mm); }
    // mirror
    cyl(0.012, 0.5, m.black, 'y', -1.42, 2.2, 0.9, Up, 6);
    box(0.02, 0.28, 0.2, m.chrome, -1.42, 2.45, 0.9, Up).rotation.y = 0.6;
    // boom foot bracket
    const br = new THREE.Mesh(profile([[0.4, 0.2], [1.62, 0.2], [1.6, 0.5], [1.2, 1.05], [0.6, 1.05], [0.4, 0.7]], 0.82, 0.03), m.paint);
    br.position.x = FOOT.x; Up.add(br);
    cyl(0.1, 0.95, m.steel, 'x', FOOT.x, FOOT.y, FOOT.z, Up);

    // -------- cab (left front) --------
    const C = new THREE.Group(); C.position.set(0.86, 0.3, 0.27); Up.add(C); this.cab = C;
    rbox(1.0, 0.22, 1.66, 0.04, m.paint, 0, 0.11, 0, C);
    const post = (x, z, h = 1.38) => box(0.06, h, 0.06, m.paint, x, 0.22 + h / 2, z, C);
    post(0.47, 0.8); post(-0.47, 0.8); post(0.47, -0.8); post(-0.47, -0.8); post(0.47, 0.05);
    rbox(1.04, 0.1, 1.72, 0.04, m.paint, 0, 1.62, 0, C);
    box(0.9, 0.04, 1.5, m.black, 0, 1.69, -0.05, C); // roof guard
    for (let i = 0; i < 4; i++) box(0.04, 0.04, 1.5, m.darkSteel, -0.33 + i * 0.22, 1.72, -0.05, C);
    // glass panels
    const glass = (w, h, x, y, z, ry) => { const g = new THREE.Mesh(new THREE.PlaneGeometry(w, h), m.glass); g.position.set(x, y, z); g.rotation.y = ry; g.renderOrder = 2; C.add(g); return g; };
    const fg = glass(0.9, 1.32, 0, 0.92, 0.82, 0); fg.rotation.x = -0.05;
    glass(1.56, 1.3, 0.48, 0.92, 0, Math.PI / 2);
    glass(1.56, 1.3, -0.48, 0.92, 0, -Math.PI / 2);
    box(0.94, 0.6, 0.04, m.paint, 0, 0.5, -0.81, C); glass(0.9, 0.7, 0, 1.25, -0.81, Math.PI);
    // wiper
    box(0.02, 0.7, 0.02, m.black, 0.2, 1.0, 0.84, C).rotation.z = 0.4;
    // interior: seat, consoles, joysticks, monitor
    rbox(0.5, 0.12, 0.5, 0.04, m.seat, 0, 0.5, -0.05, C);
    rbox(0.5, 0.65, 0.12, 0.05, m.seat, 0, 0.85, -0.32, C).rotation.x = -0.12;
    for (const s of [1, -1]) {
      rbox(0.16, 0.25, 0.55, 0.03, m.interior, s * 0.35, 0.52, 0.05, C);
      cyl(0.016, 0.16, m.black, 'y', s * 0.35, 0.72, 0.2, C, 8);
      const knob = new THREE.Mesh(new THREE.SphereGeometry(0.035, 10, 8), m.black); knob.position.set(s * 0.35, 0.8, 0.2); C.add(knob);
    }
    box(0.04, 0.5, 0.04, m.black, 0.1, 0.45, 0.6, C).rotation.x = 0.4; box(0.04, 0.5, 0.04, m.black, -0.1, 0.45, 0.6, C).rotation.x = 0.4; // travel levers
    const mon = box(0.22, 0.15, 0.03, new THREE.MeshStandardMaterial({ color: 0x0c2030, emissive: 0x15506a, emissiveIntensity: 0.6 }), 0.38, 1.1, 0.72, C);
    mon.rotation.y = -0.4;
    // roof lights + beacon
    for (const s of [1, -1]) { box(0.14, 0.09, 0.06, m.black, s * 0.3, 1.63, 0.86, C); box(0.11, 0.07, 0.01, m.lightLens, s * 0.3, 1.63, 0.895, C); }
    this.beacon = cyl(0.07, 0.12, m.beacon, 'y', -0.35, 1.76, -0.6, C, 12);
    this.cabCam = marker(C, 0, 1.12, 0.05);
    this.lookAhead = marker(C, 0, 0.5, 6);

    // -------- boom / stick / bucket --------
    const boom = new THREE.Group(); boom.position.copy(FOOT); Up.add(boom); this.boom = boom;
    const bp = [[-0.30, -0.18], [0.4, -0.32], [2.6, 0.42], [4.6, -0.12], [5.18, -0.2], [5.28, 0.0], [5.18, 0.2],
      [4.6, 0.26], [2.55, 1.18], [0.5, 0.44], [-0.3, 0.3], [-0.42, 0.06]];
    const boomM = new THREE.Mesh(profile(bp, 0.6, 0.035), m.paint); boom.add(boomM);
    box(0.03, 0.3, 3.4, m.darkSteel, 0.31, 0.45, 2.3, boom).rotation.x = 0.0; // weld seam / reinforcement
    cyl(0.12, 0.7, m.darkSteel, 'x', 0, 0, 0, boom); cyl(0.11, 0.8, m.steel, 'x', 0, 0, LB, boom);
    // hydraulic lines along boom
    for (const s of [0.12, -0.12]) {
      const hp = new THREE.CatmullRomCurve3([new THREE.Vector3(s, 0.45, 0.3), new THREE.Vector3(s, 1.27, 2.55), new THREE.Vector3(s, 0.36, 4.6), new THREE.Vector3(s, 0.4, 5.2)]);
      boom.add(new THREE.Mesh(new THREE.TubeGeometry(hp, 40, 0.025, 6), m.hose));
    }
    // boom work light
    box(0.16, 0.12, 0.08, m.black, 0.3, 1.0, 1.9, boom); box(0.13, 0.09, 0.01, m.lightLens, 0.3, 1.0, 1.945, boom);
    const boomLug = marker(boom, 0, 0.15, 1.9);
    const stickBase = marker(boom, 0, 0.98, 1.6);
    box(0.36, 0.25, 0.4, m.paint, 0, 0.95, 1.6, boom);
    box(0.44, 0.2, 0.35, m.paint, 0, 0.1, 1.9, boom);

    const stick = new THREE.Group(); stick.position.set(0, 0, LB); boom.add(stick); this.stick = stick;
    const sp = [[-0.95, 0.18], [-0.85, 0.44], [-0.3, 0.46], [0.4, 0.37], [2.75, 0.17], [3.02, 0.05], [3.0, -0.12],
      [2.7, -0.2], [0.3, -0.30], [-0.5, -0.22], [-0.95, -0.02]];
    stick.add(new THREE.Mesh(profile(sp, 0.46, 0.03), m.paint));
    cyl(0.1, 0.62, m.steel, 'x', 0, 0, LS, stick);
    cyl(0.08, 0.55, m.steel, 'x', L1[1] * 0 + 0, L1[1], L1[0], stick);
    const stickTail = marker(stick, 0, 0.32, -0.85);
    const bucketBase = marker(stick, 0, 0.5, 0.0);
    box(0.3, 0.2, 0.35, m.paint, 0, 0.48, 0.05, stick);
    this.L1m = marker(stick, 0, L1[1], L1[0]);
    this.Jm = marker(stick, 0, 0.4, LS - 0.3);

    const bucket = new THREE.Group(); bucket.position.set(0, 0, LS); stick.add(bucket); this.bucket = bucket;
    this.buildBucket(bucket);
    this.B2m = marker(bucket, 0, B2[1], B2[0]);

    // cylinders
    this.cyls = [];
    for (const s of [0.42, -0.42]) {
      const a = marker(Up, FOOT.x + s, 0.18, 1.72);
      const b = marker(boom, s, 0.15, 1.9);
      this.cyls.push(new HydCylinder(Up, a, b, { rBarrel: 0.095, rRod: 0.055, barrel: 1.05 }));
    }
    void boomLug;
    this.cyls.push(new HydCylinder(boom, stickBase, stickTail, { rBarrel: 0.1, rRod: 0.06, barrel: 2.25 }));
    this.cyls.push(new HydCylinder(stick, bucketBase, this.Jm, { rBarrel: 0.085, rRod: 0.05, barrel: 1.45 }));
    // bucket linkage plates
    this.links = [new Strut(stick, this.L1m, this.Jm, 0.42, 0.08, m.paint), new Strut(stick, this.Jm, this.B2m, 0.44, 0.1, m.darkSteel)];

    shadowAll(this.root);
    for (const b of this.belts) { b.shoes.castShadow = true; }
    // collision volumes
    this.addBox(U, 0, 0.45, 0, gauge + 0.3, 0.45, 2.2);
    this.addBox(Up, 0, 0.75, -1.0, 1.47, 0.72, 1.4);
    this.addBox(C, 0, 0.9, 0, 0.5, 0.8, 0.83);
    this.addBox(boom, 0, 0.35, 2.6, 0.3, 0.3, 2.6);
    this.addBox(stick, 0, 0.05, 1.3, 0.24, 0.25, 1.5);
    this.bucketBox = this.addBox(bucket, 0, 0.45, 0.72, 0.65, 0.45, 0.72);
  }

  buildBucket(B) {
    const m = materials();
    const W = S.bucketWidth;
    // shell plate
    B.add(new THREE.Mesh(plate(SHELL, 0.035, W), m.paint));
    // wear strips on the back
    for (const x of [-0.4, 0, 0.4]) {
      const ws = new THREE.Mesh(plate(SHELL.slice(1, 7).map(([u, w]) => [u, w + 0.025]), 0.02, 0.08), m.wornSteel);
      ws.position.x = x; B.add(ws);
    }
    // side plates
    const sideShape = SHELL.concat([[0.7, -0.02]]);
    for (const s of [1, -1]) {
      const sp = new THREE.Mesh(profile(sideShape, 0.03, 0.006), m.paint); sp.position.x = s * (W / 2 + 0.01); B.add(sp);
      // side cutter
      const sc = new THREE.Mesh(profile([[1.0, -0.02], [1.42, -0.02], [1.3, 0.25], [1.0, 0.2]], 0.035, 0.005), m.wornSteel);
      sc.position.x = s * (W / 2 + 0.03); B.add(sc);
    }
    // ears / lugs
    for (const s of [0.24, -0.24]) {
      const e = new THREE.Mesh(profile([[-0.22, -0.02], [0.25, -0.02], [0.12, 0.5], [-0.12, 0.55], [-0.26, 0.3]], 0.05, 0.01), m.paint);
      e.position.x = s; B.add(e);
    }
    cyl(0.08, 0.62, m.steel, 'x', 0, 0, 0, B);
    cyl(0.065, 0.6, m.steel, 'x', 0, B2[1], B2[0], B);
    // top rear plate (bucket "wrapper")
    box(W, 0.035, 0.32, m.paint, 0, 0.06, 0.15, B);
    // cutting edge + teeth
    box(W, 0.035, 0.16, m.wornSteel, 0, 0.012, 1.38, B).rotation.x = -0.12;
    const toothG = new THREE.ConeGeometry(0.055, 0.26, 4); toothG.rotateX(Math.PI / 2); toothG.rotateZ(Math.PI / 4); toothG.scale(1, 0.55, 1);
    for (let i = 0; i < 5; i++) {
      const x = -0.52 + i * 0.26;
      const adapter = box(0.09, 0.07, 0.16, m.darkSteel, x, 0.02, 1.42, B);
      adapter.rotation.x = -0.12;
      const t = new THREE.Mesh(toothG, m.wornSteel);
      t.position.set(x, 0.0, 1.6); t.rotation.x = -0.12; B.add(t);
    }
    // contents
    const inner = SHELL.map(([u, w]) => [u - 0.01, w - 0.03]).concat([[0.05, 0.05]]);
    const cg = profile(inner.map(([u, w]) => [u, w - 0.88]), W - 0.08, 0.02, 4);
    this.contentMat = new THREE.MeshStandardMaterial({ color: 0x6a4a2c, roughness: 1 });
    const cm = new THREE.Mesh(cg, this.contentMat);
    this.content = new THREE.Group(); this.content.position.y = 0.88; this.content.add(cm); B.add(this.content);
    const heap = new THREE.Mesh(new THREE.SphereGeometry(1, 16, 10, 0, Math.PI * 2, 0, Math.PI / 2), this.contentMat);
    heap.rotation.x = Math.PI; // dome pointing to -w (out of the mouth)
    this.heap = heap; heap.position.set(0, 0.02, 0.72); B.add(heap);
    this.content.visible = false; heap.visible = false;
  }

  setupPhysics() {
    const gauge = S.trackGauge / 2;
    for (const s of [1, -1]) for (const dx of [-0.2, 0.2]) {
      for (let i = 0; i < 9; i++) {
        this.addContact(s * gauge + dx, -1.0, -1.825 + i * (3.65 / 8), { k: 2.0e5, c: 1.3e4, area: 0.12 });
      }
      for (const z of [-1, 1]) {
        this.addContact(s * gauge + dx, -0.78, z * 2.12, { k: 2e5, c: 1.3e4, area: 0.1 });
        this.addContact(s * gauge + dx, -0.45, z * 2.22, { k: 2e5, c: 1.3e4, area: 0.1 });
      }
    }
    // bucket shell contacts: the bucket is a solid body pressing on the soil
    this.bucketContacts = [];
    for (const [u, w] of [[1.45, 0.0], [1.02, 0.66], [0.42, 0.88], [-0.04, 0.58]]) for (const x of [-0.55, 0, 0.55]) {
      const c = new Contact(new THREE.Vector3(), { k: 3.5e5, c: 2.5e4, area: 0.15 });
      c.uw = [u, w]; c.lat = x;
      this.bucketContacts.push(c);
    }
    const self = this;
    const v = new THREE.Vector3();
    const arm = (s, t) => v.set(FOOT.x, FOOT.y + t, FOOT.z + s);
    this.masses = [
      { m: 8200, pos: () => v.set(0, -0.55, 0).clone() },
      { m: 10500, pos: () => self.upperToBody(new THREE.Vector3(0, 0.75, -0.85)) },
      { m: 1900, pos: () => { const c = Math.cos(self.a1), sn = Math.sin(self.a1); return self.upperToBody(arm(2.5 * c - 0.4 * sn, 2.5 * sn + 0.4 * c).clone()); } },
      { m: 1000, pos: () => { const p = self.armPoints(); return self.upperToBody(arm((p.boomTip[0] + p.pin[0]) / 2, (p.boomTip[1] + p.pin[1]) / 2).clone()); } },
      { m: () => 900 + self.load * SOIL.densLoose[Math.round(self.loadType)], pos: () => { const p = self.armPoints(); const s = p.pin[0] + p.u[0] * 0.7 + p.w[0] * 0.45, t = p.pin[1] + p.u[1] * 0.7 + p.w[1] * 0.45; return self.upperToBody(arm(s, t).clone()); } },
    ];
  }

  upperToBody(v) {
    const c = Math.cos(this.swing), s = Math.sin(this.swing);
    const x = v.x * c + v.z * s, z = -v.x * s + v.z * c;
    return v.set(x, v.y + UPPER_Y, z);
  }

  /** Arm geometry in the arm plane (s forward, t up) relative to the boom foot. */
  armPoints(a1 = this.a1, r2 = this.r2, r3 = this.r3) {
    const tip = [LB * Math.cos(a1), LB * Math.sin(a1)];
    const a2 = a1 + r2;
    const pin = [tip[0] + LS * Math.cos(a2), tip[1] + LS * Math.sin(a2)];
    const a3 = a2 + r3;
    const u = [Math.cos(a3), Math.sin(a3)], w = [-Math.sin(a3), Math.cos(a3)];
    return { boomTip: tip, pin, u, w, a2, a3 };
  }

  // world matrix of the upper structure for a given swing
  upperMatrix(swing, out) {
    const q = new THREE.Quaternion().setFromAxisAngle(new THREE.Vector3(0, 1, 0), swing);
    const t = new THREE.Matrix4().makeRotationFromQuaternion(q).setPosition(0, UPPER_Y, 0);
    const b = new THREE.Matrix4().compose(this.body.pos, this.body.quat, new THREE.Vector3(1, 1, 1));
    return out.multiplyMatrices(b, t);
  }

  /** World positions of rasterised bucket samples for a pose. */
  bucketWorld(pose, out) {
    const M = this.upperMatrix(pose.swing, this._mUpper);
    const ap = this.armPoints(pose.a1, pose.r2, pose.r3);
    const e = M.elements;
    let n = 0;
    for (const [x, u, w] of SAMPLES) {
      const s = ap.pin[0] + ap.u[0] * u + ap.w[0] * w;
      const t = ap.pin[1] + ap.u[1] * u + ap.w[1] * w;
      const lx = FOOT.x + x, ly = FOOT.y + t, lz = FOOT.z + s;
      out[n++] = e[0] * lx + e[4] * ly + e[8] * lz + e[12];
      out[n++] = e[1] * lx + e[5] * ly + e[9] * lz + e[13];
      out[n++] = e[2] * lx + e[6] * ly + e[10] * lz + e[14];
    }
    // tip + mouth normal for convenience
    const tipS = ap.pin[0] + ap.u[0] * 1.45, tipT = ap.pin[1] + ap.u[1] * 1.45;
    const lip = new THREE.Vector3(FOOT.x, FOOT.y + tipT, FOOT.z + tipS).applyMatrix4(M);
    const back = new THREE.Vector3(FOOT.x, FOOT.y + ap.pin[1] + ap.w[1] * 0.1, FOOT.z + ap.pin[0] + ap.w[0] * 0.1).applyMatrix4(M);
    const mouth = new THREE.Vector3(0, -ap.w[1], -ap.w[0]).applyMatrix4(new THREE.Matrix4().extractRotation(M)).normalize();
    const along = new THREE.Vector3(0, ap.u[1], ap.u[0]).applyMatrix4(new THREE.Matrix4().extractRotation(M)).normalize();
    return { tip: lip, back, mouth, along };
  }

  /** Build per-node envelope (min/max y) touched by the bucket. */
  envelope(pose) {
    const T = this.terrain;
    if (!this._wbuf) {
      this._wbuf = new Float32Array(SAMPLES.length * 3); this._env = new Map();
      this._eLo = new Float32Array(T.W * T.W); this._eHi = new Float32Array(T.W * T.W);
      this._eStamp = new Uint32Array(T.W * T.W); this._eList = new Int32Array(SAMPLES.length); this._stamp = 0;
    }
    const info = this.bucketWorld(pose, this._wbuf);
    const b = this._wbuf, lo = this._eLo, hi = this._eHi, st = this._eStamp, list = this._eList;
    const stamp = ++this._stamp;
    let n = 0;
    for (let i = 0; i < SAMPLES.length; i++) {
      const x = b[i * 3], y = b[i * 3 + 1], z = b[i * 3 + 2];
      const k = T.nearestIdx(x, z);
      if (k < 0) continue;
      if (st[k] !== stamp) { st[k] = stamp; lo[k] = y; hi[k] = y; list[n++] = k; }
      else { if (y < lo[k]) lo[k] = y; if (y > hi[k]) hi[k] = y; }
    }
    // Map-compatible view used by cost() / cutting: [k, [lo, hi]]
    const env = this._env; env.clear();
    for (let i = 0; i < n; i++) { const k = list[i]; env.set(k, [lo[k], hi[k]]); }
    info.env = env;
    return info;
  }

  /** Energy (J) to move bucket into `pose`, or -1 if bedrock blocks it. */
  cost(pose) {
    const info = this.envelope(pose);
    const T = this.terrain;
    let E = 0, n = 0;
    for (const [k, [lo, hi]] of info.env) {
      const H = T.bank[k] + T.loose[k];
      const cut = Math.min(H - lo, hi - lo + 0.25);
      if (cut <= 0.002) continue;
      n++;
      const c = T.cutCost(k, cut);
      if (c < 0) {
        // allow if not deeper into rock than the current pose already is
        const curPen = this._rockPen?.get(k) ?? 0;
        if (T.rock[k] - lo > curPen + 0.01) return { E: -1, info };
        continue;
      }
      E += c;
    }
    // a full bucket can't take more: soil packs ahead of it and resistance climbs steeply
    const full = this.load / (this.capacity * 1.08);
    if (full > 0.9) E *= 1 + (full - 0.9) * 40;
    // shoving soil with the back/side of the bucket is far harder than scooping it
    if (this._scoopHint === false) E *= 3;
    return { E, info, n };
  }

  // ---------------------------------------------------------------- simulation
  control(inp, dt) {
    this.inp = inp;
    if (inp.toggleSpeed) this.highSpeed = !this.highSpeed;
  }

  step(dt) {
    const inp = this.inp || {};
    this._hydAcc += dt;
    while (this._hydAcc >= 1 / 60) { this._hydAcc -= 1 / 60; this.hydStep(1 / 60, inp); }
    this.applyGravity();
    // digging reaction on the machine (soil pushes back on the bucket)
    if (this.reaction.lengthSq() > 1) this.body.addForceAt(this.reaction, this.reactionPoint);
    const T = this.terrain;
    for (const c of this.bucketContacts) {
      const wp = this.body.toWorld(c.local, c.world);
      const pen = T.heightAt(wp.x, wp.z) - wp.y;
      if (pen <= 0 || pen > 0.3) { c.zl = c.zt = 0; c.Fn = 0; continue; }
      resolveContact(this.body, c, T, dt, { belt: 0, muLong: 0.55, muLat: 0.55 });
    }
    const vmax = (this.highSpeed ? S.travelSpeedHi : S.travelSpeedLo) / 3.6;
    const tl = clamp(inp.trackL || 0, -1, 1), tr = clamp(inp.trackR || 0, -1, 1);
    this._beltL = (this._beltL || 0) + (tl * vmax - (this._beltL || 0)) * Math.min(1, dt * 3);
    this._beltR = (this._beltR || 0) + (tr * vmax - (this._beltR || 0)) * Math.min(1, dt * 3);
    this.trackStep(dt, this._beltL, this._beltR, S.enginePower * 0.8, 0.9);
    this.body.integrate(dt);
  }

  hydStep(dt, inp) {
    const T = this.terrain;
    // joystick -> commanded joint velocities with flow sharing between functions
    const ax = (v) => (Number.isFinite(v) ? clamp(v, -1, 1) : 0); // ignore bad/NaN input
    const cmd = {
      swing: ax(inp.swing) * RATE.swing,
      a1: ax(inp.boom) * RATE.a1,
      r2: ax(inp.stick) * RATE.r2,
      r3: ax(inp.bucket) * RATE.r3,
    };
    const rpmF = clamp((this.rpm - 700) / (this.maxRpm - 700), 0.2, 1);
    let flow = 0;
    for (const k in cmd) { cmd[k] *= rpmF; flow += Math.abs(cmd[k]) / RATE[k]; }
    const travel = Math.abs(inp.trackL || 0) + Math.abs(inp.trackR || 0);
    flow += travel * 0.5;
    const share = flow > 1.7 ? 1.7 / flow : 1;
    for (const k in cmd) cmd[k] *= share;
    // swing acceleration reduces with load at reach (swing torque limit)
    const loadF = 1 / (1 + this.load * 0.25);
    for (const k of ['swing', 'a1', 'r2', 'r3']) {
      const acc = ACC[k] * (k === 'swing' ? loadF : 1);
      const d = cmd[k] - this.w[k];
      const braking = cmd[k] === 0 || Math.sign(cmd[k]) !== Math.sign(this.w[k]);
      const mx = acc * dt * (braking && this.w[k] !== 0 ? (k === 'swing' ? 2.4 : 1.8) : 1);
      this.w[k] += clamp(d, -mx, mx);
    }
    // cushion near cylinder end-stops
    for (const k of ['a1', 'r2', 'r3']) {
      const [lo, hi] = LIM[k];
      const v = this[k];
      if (this.w[k] > 0) this.w[k] = Math.min(this.w[k], (hi - v) * 6);
      if (this.w[k] < 0) this.w[k] = Math.max(this.w[k], (lo - v) * 6);
    }
    const p0 = { swing: this.swing, a1: this.a1, r2: this.r2, r3: this.r3 };
    const p1 = {
      swing: p0.swing + this.w.swing * dt,
      a1: clamp(p0.a1 + this.w.a1 * dt, ...LIM.a1),
      r2: clamp(p0.r2 + this.w.r2 * dt, ...LIM.r2),
      r3: clamp(p0.r3 + this.w.r3 * dt, ...LIM.r3),
    };
    const lerpPose = (t) => ({ swing: p0.swing + (p1.swing - p0.swing) * t, a1: p0.a1 + (p1.a1 - p0.a1) * t, r2: p0.r2 + (p1.r2 - p0.r2) * t, r3: p0.r3 + (p1.r3 - p0.r3) * t });
    const before = this.bucketWorld(p0, this._tmpBuf || (this._tmpBuf = new Float32Array(SAMPLES.length * 3)));
    const Pmax = S.hydraulicPower * dt * (0.4 + 0.6 * rpmF);
    const Fmax = 165e3;
    // overlap that already exists at the current pose (caused by the machine body
    // moving, not the hydraulics) is resisted by the bucket contacts, not charged here
    {
      const ti = this.bucketWorld(p1, this._tmpBuf).tip.sub(before.tip);
      const L = ti.length();
      this._scoopHint = L < 1e-4 || this.w.r3 < -0.08 || ti.dot(before.mouth) / L + 0.6 * ti.dot(before.along) / L > 0.05;
    }
    const r0 = this.cost(p0);
    const E0 = Math.max(0, r0.E);
    const ok = (res, pose) => {
      if (res.E < 0) return false;
      const ti = this.bucketWorld(pose, this._tmpBuf);
      const dist = ti.tip.distanceTo(before.tip) + 0.002;
      return res.E - E0 <= Math.min(Pmax, Fmax * dist * 1.2);
    };
    let lam = 1;
    let res = this.cost(p1);
    if (!ok(res, p1)) {
      let lo = 0, hi = 1;
      for (let it = 0; it < 5; it++) {
        const mid = (lo + hi) / 2;
        const r = this.cost(lerpPose(mid));
        if (ok(r, lerpPose(mid))) lo = mid; else hi = mid;
      }
      lam = lo;
      res = this.cost(lerpPose(lam));
      if (res.E < 0) { lam = 0; res = this.cost(p0); }
      // the joints that were pushing are stalled: bleed their velocity (relief valve)
      for (const k of ['swing', 'a1', 'r2', 'r3']) this.w[k] *= Math.max(lam, 0.35);
    }
    const pose = lerpPose(lam);
    const after = res.info;
    // apply the cut
    const moveDir = after.tip.clone().sub(before.tip);
    const moveLen = moveDir.length();
    // stall reaction comes from the boom/stick/bucket cylinders; a blocked swing only
    // pushes with swing-motor torque (~10 kN at the bucket), so leave swing out of it
    const intended = this.bucketWorld({ ...p1, swing: p0.swing }, this._tmpBuf).tip.sub(before.tip);
    if (moveLen > 1e-5) moveDir.divideScalar(moveLen);
    const curling = this.w.r3 < -0.08;
    const scoop = moveDir.dot(after.mouth) + 0.6 * moveDir.dot(after.along) > 0.05 || curling;
    let cutVol = 0, energy = 0, touching = 0;
    this._rockPen = this._rockPen || new Map();
    this._rockPen.clear();
    const center = after.tip.clone().add(after.back).multiplyScalar(0.5);
    // a bucket that isn't moving doesn't cut: soil that slumps against it just rests there
    const still = moveLen < 0.003;
    for (const [k, [lo, hi]] of after.env) {
      const H = T.bank[k] + T.loose[k];
      const pen = T.rock[k] - lo;
      if (pen > 0) this._rockPen.set(k, pen);
      let cut = Math.min(H - lo, hi - lo + 0.25);
      if (cut <= 0.002) continue;
      touching++;
      if (still) continue;
      const c = T.cutCost(k, cut);
      if (c > 0) energy += c;
      const vol = T.take(k, cut);
      // the teeth leave grooves in the bank along the direction of the cut
      if (scoop && T.loose[k] < 0.01) {
        // grooves follow the bucket's travel; smoothed so one pass leaves straight parallel marks
        const hx = Math.abs(moveDir.x) + Math.abs(moveDir.z) > 0.2 ? moveDir : after.along;
        const sd = this._cutDir || (this._cutDir = new THREE.Vector2(hx.x, hx.z));
        if (sd.x * hx.x + sd.y * hx.z < 0) sd.multiplyScalar(-1);
        sd.lerp(new THREE.Vector2(hx.x, hx.z), 0.08);
        T.mark(k, sd.x, sd.y, 1, 1);
      }
      const type = T.lastType;
      if (vol <= 0) continue;
      cutVol += vol;
      this.st = this.st || { cut: 0, into: 0, pushed: 0, steps: 0 }; this.st.cut += vol;
      const room = this.capacity * 1.08 - this.load;
      if (scoop && room > 0) {
        const into = Math.min(room, vol);
        this.loadType = (this.loadType * this.load + type * into) / (this.load + into);
        this.load += into;
        this.st.into += into;
        const rest = vol - into;
        if (rest > 0) this.pushAside(k, rest, type, moveDir, center);
      } else {
        this.pushAside(k, vol, type, moveDir, center);
      }
    }
    // the bucket lip smooths what it passes over: shave nodes left proud of their
    // neighbours (sampling stair-steps), so cut faces come out as clean planes
    if (!still && touching) {
      const W = T.W;
      for (const [k, [lo]] of after.env) {
        const H = T.bank[k] + T.loose[k];
        if (H > lo + 0.3) continue; // not under the bucket
        const avg = (T.bank[k - 1] + T.loose[k - 1] + T.bank[k + 1] + T.loose[k + 1] + T.bank[k - W] + T.loose[k - W] + T.bank[k + W] + T.loose[k + W]) * 0.25;
        const ex = H - avg - 0.008;
        if (ex > 0.002 && T.bank[k] - ex > T.rock[k]) {
          const v = T.take(k, ex * 0.6);
          if (v > 0) {
            const room = this.capacity * 1.08 - this.load;
            if (scoop && room > 0) { const into = Math.min(room, v); this.loadType = (this.loadType * this.load + T.lastType * into) / (this.load + into); this.load += into; if (v > into) this.pushAside(k, v - into, T.lastType, moveDir, center); }
            else this.pushAside(k, v, T.lastType, moveDir, center);
          }
        }
      }
    }
    this.swing = pose.swing; this.a1 = pose.a1; this.r2 = pose.r2; this.r3 = pose.r3;
    const ap = this.armPoints();
    for (const c of this.bucketContacts) {
      const [u, w] = c.uw;
      const sx = ap.pin[0] + ap.u[0] * u + ap.w[0] * w, ty = ap.pin[1] + ap.u[1] * u + ap.w[1] * w;
      this.upperToBody(c.local.set(FOOT.x + c.lat, FOOT.y + ty, FOOT.z + sx));
    }
    // reaction force & gauges
    const iLen = intended.length();
    const Fr = Fmax * 0.6;
    let F = moveLen > 1e-4 ? clamp((energy - E0) / Math.max(moveLen, 0.004), 0, Fr) : 0;
    if (lam < 0.999 && iLen > 1e-4) F = Math.max(F, Fr * (1 - lam));
    const target = new THREE.Vector3();
    if (F > 0 && iLen > 1e-5) target.copy(intended).normalize().multiplyScalar(-F);
    this.reaction.lerp(target, 0.3);
    this.reactionPoint.copy(after.tip);
    const p = lam < 0.999 ? 1 : clamp(F / Fmax, 0, 1);
    this.pressure += (p - this.pressure) * 0.25;
    this.stall = lam < 0.999 && iLen > 1e-4 ? 1 : this.stall * 0.9;
    this.touching = touching;
    this.digRate = cutVol / dt;
    // spill when bucket out of the ground
    if (touching === 0 && this.load > 0) {
      const my = after.mouth.y;
      const cap = this.capacity * smoothstep(-0.45, 0.75, my) * 1.05;
      if (this.load > cap) {
        const ex = this.load - cap;
        const out = Math.min(this.load, ex * Math.min(1, dt * 5) + 0.15 * dt);
        this.load -= out;
        const lip = after.tip.y < after.back.y ? after.tip : after.back;
        const pv = new THREE.Vector3();
        this.body.pointVel(lip, pv);
        const swingV = new THREE.Vector3();
        const om = new THREE.Vector3(0, this.w.swing, 0).applyQuaternion(this.body.quat);
        swingV.crossVectors(om, lip.clone().sub(this.body.pos));
        pv.add(swingV);
        const P = lip.clone().addScaledVector(after.mouth, 0.15);
        this.particles.emit(P.x, P.y, P.z, pv.x + after.mouth.x * 0.4, pv.y, pv.z + after.mouth.z * 0.4, out, this.loadType, 0.6);
        this.pouring = out / dt;
      } else this.pouring = 0;
    } else this.pouring = 0;
    if (this.load < 1e-4) this.load = 0;
    // engine demand
    const hydLoad = clamp(flow * 0.35 + this.pressure * 0.6 + travel * 0.2, 0, 1.2);
    this.throttle = inp.throttle ?? this.throttle;
    this.updateEngine(dt, hydLoad);
    this.hydLoad = hydLoad;
  }

  pushAside(k, vol, type, dir, center) {
    const T = this.terrain;
    const j = (k / T.W) | 0, i = k - j * T.W;
    const x = T.nodeX(i), z = T.nodeX(j);
    let dx = x - center.x, dz = z - center.z;
    const L = Math.hypot(dx, dz) || 1;
    dx /= L; dz /= L;
    const hx = dir.x, hz = dir.z;
    const hl = Math.hypot(hx, hz);
    // shoved soil ends up clear of the bucket, mostly ahead of its motion
    let ox = dx * 0.9, oz = dz * 0.9;
    if (hl > 0.2) { ox = dx * 0.3 + (hx / hl) * 1.1; oz = dz * 0.3 + (hz / hl) * 1.1; }
    T.deposit(x + ox, z + oz, vol, type);
  }

  // ---------------------------------------------------------------- visuals
  updateVisuals(dt) {
    this.syncRoot();
    this.upper.rotation.y = this.swing;
    this.boom.rotation.x = -this.a1;
    this.stick.rotation.x = -this.r2;
    this.bucket.rotation.x = -this.r3;
    // four-bar bucket linkage
    const c = Math.cos(this.r3), s = Math.sin(this.r3);
    const bx = LS + B2[0] * c - B2[1] * s, by = B2[0] * s + B2[1] * c;
    let dx = bx - L1[0], dy = by - L1[1];
    let d = Math.hypot(dx, dy);
    d = clamp(d, Math.abs(ROCKER - COUPLER) + 1e-3, ROCKER + COUPLER - 1e-3);
    const ux = dx / Math.hypot(dx, dy), uy = dy / Math.hypot(dx, dy);
    const a = (ROCKER * ROCKER - COUPLER * COUPLER + d * d) / (2 * d);
    const h = Math.sqrt(Math.max(0, ROCKER * ROCKER - a * a));
    const jx = L1[0] + a * ux - h * uy, jy = L1[1] + a * uy + h * ux;
    this.Jm.position.set(0, jy, jx);
    this.root.updateMatrixWorld();
    for (const cy of this.cyls) cy.update();
    for (const l of this.links) l.update();
    // tracks
    const r = 0.40;
    void r;
    this.belts[0].update(-(this.beltActualL || 0) * dt);
    this.belts[1].update(-(this.beltActualR || 0) * dt);
    // bucket contents
    const f = this.load / this.capacity;
    this.content.visible = f > 0.01;
    this.content.scale.y = clamp(f, 0.02, 1);
    this.heap.visible = f > 1.0;
    const hf = clamp((f - 1) / 0.12, 0, 1);
    this.heap.scale.set(0.6, 0.05 + 0.3 * hf, 0.65);
    const col = [0x4f3a26, 0x7c5230, 0x77695a, 0x606060][Math.round(this.loadType)];
    this.contentMat.color.setHex(col);
    this.beacon.material.emissiveIntensity = 0.5 + 0.8 * Math.max(0, Math.sin(performance.now() * 0.012));
    this.emitExhaust(dt);
  }

  /** Bucket tip world position (for grade control HUD). */
  tipWorld() {
    const info = this.bucketWorld(this, this._tmpBuf || (this._tmpBuf = new Float32Array(SAMPLES.length * 3)));
    return info.tip;
  }

  hud() {
    const tip = this.tipWorld();
    const p = SITE.pit;
    const inPit = tip.x > p.x0 - 1 && tip.x < p.x1 + 1 && tip.z > p.z0 - 1 && tip.z < p.z1 + 1;
    const dens = SOIL.densLoose[Math.round(this.loadType)];
    return {
      title: S.name,
      sub: S.className,
      rows: [
        ['Engine', `${Math.round(this.rpm)} rpm`],
        ['Hydraulic pressure', `${Math.round(60 + this.pressure * 290)} bar`],
        ['Bucket load', `${this.load.toFixed(2)} m³ · ${(this.load * dens / 1000).toFixed(2)} t`],
        ['Bucket material', this.load > 0.02 ? SOIL.names[Math.round(this.loadType)] : '—'],
        ['Swing', `${((this.swing / DEG) % 360).toFixed(0)}°`],
        ['Travel', `${this.highSpeed ? 'Hi' : 'Lo'} · ${Math.abs(this.speedKmh()).toFixed(1)} km/h`],
        ['Tip elevation', `${tip.y.toFixed(2)} m`],
      ],
      gauge: this.pressure,
      stall: this.stall > 0.5,
      grade: inPit ? tip.y - this.gradeTarget : null,
    };
  }
}
