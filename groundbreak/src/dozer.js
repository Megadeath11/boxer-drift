// Medium track-type tractor with semi-U blade and multi-shank ripper.
// Blade: cuts soil above the cutting edge and carries it as a growing pile ahead
// of the moldboard (which spills round the ends and over the top). Resistance =
// soil-on-soil friction of the carried pile + specific cutting resistance; when it
// exceeds track traction the tracks slip, like a real dozer.
import * as THREE from 'three';
import { Machine } from './machine.js';
import { DOZER as S, SOIL } from './config.js';
import { clamp, DEG } from './util.js';
import { materials, rbox, box, cyl, profile, plate, HydCylinder, Strut, marker, TrackBelt, sprocket, idler, shadowAll } from './parts.js';
import { makeTextTexture } from './textures.js';

const TRUN = { y: 0.55, z: -0.1 }; // push-arm trunnion (ground coords)
const EDGE0 = { y: 0.0, z: 2.95 };
const ARM_L = Math.hypot(EDGE0.z - TRUN.z, EDGE0.y - TRUN.y);
const TH0 = Math.atan2(EDGE0.y - TRUN.y, EDGE0.z - TRUN.z);
const THMIN = Math.asin((-S.bladeDig - TRUN.y) / ARM_L), THMAX = Math.asin((S.bladeLift - TRUN.y) / ARM_L);
const MOLD = [[0.0, 0.0], [-0.08, 0.25], [-0.16, 0.55], [-0.2, 0.85], [-0.17, 1.1], [-0.08, 1.3], [0.06, 1.42]];
const WING = 25 * DEG;

export class Dozer extends Machine {
  constructor(scene, terrain, particles) {
    super(scene, terrain, particles, S.operatingMass, new THREE.Vector3(56000, 56000, 23000));
    this.spec = S;
    this.kind = 'dozer';
    this.label = S.name;
    this.idleRpm = 800; this.maxRpm = 1900;
    this.radius = 6;
    this.theta = THMIN + 0.15; this.tilt = 0; this.rip = 0;
    this.bladeForce = 0; this.pile = 0;
    this._hydAcc = 0;
    this.drag = new THREE.Vector3(); this.dragPt = new THREE.Vector3();
    this.lift = new THREE.Vector3();
    this.throttle = 1;
    this.build();
    this.setupPhysics();
  }

  build() {
    const m = materials();
    const B = new THREE.Group(); B.position.y = -1.0; this.root.add(B); this.B = B;
    const gauge = S.trackGauge / 2;
    const circles = [{ z: -1.335, y: 0.33, r: 0.33 }, { z: 1.335, y: 0.33, r: 0.33 }, { z: -0.95, y: 1.22, r: 0.40 }];
    this.belts = [];
    for (const s of [1, -1]) {
      const x = s * gauge;
      this.belts.push(new TrackBelt(B, circles, { x, shoeWidth: S.shoeWidth, pitch: 0.2, grouser: 0.06 }));
      const spr = sprocket(0.37, 0.22, 23, m.darkSteel); spr.position.set(x, 1.22, -0.95); B.add(spr);
      for (const z of [-1.335, 1.335]) { const id = idler(0.3, 0.24, m.darkSteel); id.position.set(x, 0.33, z); B.add(id); }
      for (let i = 0; i < 6; i++) { const r = idler(0.1, 0.26, m.darkSteel); r.position.set(x, 0.16, -0.95 + i * 0.38); B.add(r); }
      const fr = new THREE.Mesh(profile([[-1.2, 0.22], [1.2, 0.22], [1.35, 0.42], [1.1, 0.58], [-0.6, 0.62], [-1.1, 0.55], [-1.35, 0.42]], 0.36, 0.03), m.paint);
      fr.position.x = x; B.add(fr);
      // equaliser / pivot shaft housing
      cyl(0.14, 0.5, m.darkSteel, 'x', s * (gauge - 0.45), 0.55, -0.1, B);
      // final drive housing (elevated sprocket)
      cyl(0.32, 0.3, m.paint, 'x', s * (gauge - 0.3), 1.22, -0.95, B);
      // fender & steps
      const fp = new THREE.Mesh(profile([[-1.7, 1.72], [-0.1, 1.72], [0.6, 1.25], [0.6, 1.2], [-0.1, 1.66], [-1.7, 1.66]], 0.66, 0.01), m.paint);
      fp.position.x = s * (gauge + 0.02); B.add(fp);
      box(0.6, 0.02, 0.3, m.darkSteel, s * gauge, 1.73, -1.2, B);
    }
    // main case + engine compartment
    rbox(1.12, 0.7, 3.2, 0.06, m.paint, 0, 0.98, -0.1, B);
    rbox(1.22, 0.98, 1.95, 0.1, m.paint, 0, 1.76, 0.98, B);
    for (let i = 0; i < 8; i++) for (const s of [1, -1]) box(0.015, 0.45, 0.05, m.black, s * 0.615, 1.8, 0.35 + i * 0.14, B);
    // radiator guard + grille
    rbox(1.36, 1.3, 0.26, 0.06, m.paint, 0, 1.55, 2.02, B);
    for (let i = 0; i < 9; i++) box(1.12, 0.035, 0.05, m.black, 0, 1.06 + i * 0.12, 2.16, B);
    box(1.2, 0.3, 0.5, m.darkSteel, 0, 0.72, 2.0, B); // crankcase guard
    // lift cylinder brackets on radiator guard
    for (const s of [1, -1]) box(0.14, 0.26, 0.3, m.paint, s * 0.6, 2.0, 1.95, B);
    // exhaust & precleaner
    cyl(0.07, 0.75, m.black, 'y', 0.32, 2.6, 1.45, B);
    this.exhaust = marker(B, 0.32, 3.0, 1.45);
    cyl(0.13, 0.32, m.black, 'y', -0.32, 2.4, 1.6, B, 16);
    // cab
    const C = new THREE.Group(); C.position.set(0, 1.78, -0.68); B.add(C); this.cab = C;
    rbox(1.72, 0.12, 1.55, 0.04, m.paint, 0, 0.06, 0, C);
    for (const [x, z] of [[0.83, 0.74], [-0.83, 0.74], [0.83, -0.74], [-0.83, -0.74]]) box(0.08, 1.28, 0.08, m.paint, x, 0.7, z, C);
    rbox(1.82, 0.12, 1.66, 0.05, m.paint, 0, 1.36, 0, C);
    box(1.6, 0.06, 1.4, m.black, 0, 1.44, 0, C);
    const glass = (w, h, x, y, z, ry) => { const g = new THREE.Mesh(new THREE.PlaneGeometry(w, h), m.glass); g.position.set(x, y, z); g.rotation.y = ry; g.renderOrder = 2; C.add(g); };
    glass(1.6, 1.2, 0, 0.72, 0.75, 0); glass(1.6, 1.2, 0, 0.72, -0.75, Math.PI);
    glass(1.42, 1.2, 0.84, 0.72, 0, Math.PI / 2); glass(1.42, 1.2, -0.84, 0.72, 0, -Math.PI / 2);
    rbox(0.52, 0.12, 0.5, 0.04, m.seat, 0, 0.42, -0.2, C);
    rbox(0.52, 0.62, 0.12, 0.05, m.seat, 0, 0.76, -0.46, C).rotation.x = -0.12;
    for (const s of [1, -1]) { rbox(0.16, 0.3, 0.5, 0.03, m.interior, s * 0.38, 0.35, -0.1, C); cyl(0.016, 0.16, m.black, 'y', s * 0.38, 0.56, 0.05, C, 8); }
    for (const s of [1, -1]) { box(0.14, 0.09, 0.06, m.black, s * 0.6, 1.37, 0.85, C); box(0.11, 0.07, 0.01, m.lightLens, s * 0.6, 1.37, 0.885, C); }
    this.beacon = cyl(0.07, 0.12, m.beacon, 'y', 0.6, 1.5, -0.5, C, 12);
    this.cabCam = marker(C, 0, 1.0, -0.15);
    // rear: fuel tank, hitch, ripper frame
    rbox(1.55, 0.65, 0.5, 0.06, m.paint, 0, 1.45, -1.78, B);
    for (const s of [1, -1]) box(0.14, 0.12, 0.02, m.redLens, s * 0.6, 1.6, -2.04, B);
    box(0.9, 0.5, 0.4, m.paint, 0, 0.85, -1.85, B);
    for (const s of [1, -1]) {
      const rf = new THREE.Mesh(profile([[-1.85, 0.6], [-2.35, 0.7], [-2.4, 1.15], [-1.85, 1.2]], 0.18, 0.02), m.paint);
      rf.position.x = s * 0.55; B.add(rf);
    }
    const RP = new THREE.Group(); B.add(RP); this.ripper = RP;
    rbox(2.2, 0.28, 0.32, 0.04, m.paint, 0, 0.95, -2.7, RP);
    this.shanks = [];
    for (const x of [-0.85, 0, 0.85]) {
      const sh = new THREE.Mesh(profile([[-2.62, 1.0], [-2.78, 1.0], [-2.85, 0.2], [-2.75, -0.15], [-2.62, -0.05], [-2.7, 0.25]], 0.09, 0.01), m.paint);
      sh.position.x = x; RP.add(sh);
      box(0.08, 0.1, 0.18, m.wornSteel, x, -0.12, -2.68, RP).rotation.x = 0.6;
      this.shanks.push(x);
    }
    this.ripperTop = marker(B, 0, 1.25, -2.2);
    this.ripperArm = marker(RP, 0, 1.05, -2.65);
    // ---- blade ----
    const BL = new THREE.Group(); B.add(BL); this.blade = BL;
    const center = new THREE.Mesh(plate(MOLD, 0.04, 2.4), m.paint); BL.add(center);
    for (const s of [1, -1]) {
      const wg = new THREE.Group();
      wg.position.set(s * 1.2, 0, 0); wg.rotation.y = s * -WING; BL.add(wg);
      const w = new THREE.Mesh(plate(MOLD, 0.04, 0.48), m.paint); w.position.x = s * 0.24; wg.add(w);
      const eb = box(0.48, 0.22, 0.05, m.wornSteel, s * 0.24, 0.1, 0.0, wg); eb.rotation.x = 0.25;
      const ep = new THREE.Mesh(profile(MOLD.concat([[-0.55, 1.3], [-0.55, 0.2]]), 0.03, 0.005), m.paint); ep.position.x = s * 0.48; wg.add(ep);
    }
    box(2.42, 0.22, 0.05, m.wornSteel, 0, 0.1, 0.0, BL).rotation.x = 0.25;
    // back bracing
    rbox(2.3, 0.9, 0.22, 0.04, m.paint, 0, 0.72, -0.36, BL);
    for (const x of [-0.9, -0.3, 0.3, 0.9]) box(0.06, 1.0, 0.26, m.paint, x, 0.72, -0.28, BL);
    box(2.6, 0.08, 0.32, m.paint, 0, 1.38, -0.12, BL);
    const decal = new THREE.Mesh(new THREE.PlaneGeometry(0.9, 0.22), new THREE.MeshStandardMaterial({ map: makeTextTexture('DZ-6', { color: '#151515' }), transparent: true }));
    decal.position.set(0, 1.0, -0.476); decal.rotation.y = Math.PI; BL.add(decal);
    this.bladeArmPt = [];
    this.cyls = []; this.struts = [];
    for (const s of [1, -1]) {
      const tr = marker(B, s * 1.32, TRUN.y, TRUN.z);
      const ba = marker(BL, s * 1.32, 0.25, -0.38);
      this.struts.push(new Strut(B, tr, ba, 0.16, 0.26, m.paint));
      const cyA = marker(B, s * 0.6, 2.05, 1.92), cyB = marker(BL, s * 0.6, 1.1, -0.45);
      this.cyls.push(new HydCylinder(B, cyA, cyB, { rBarrel: 0.08, rRod: 0.05, barrel: 1.0 }));
    }
    // tilt cylinder
    this.cyls.push(new HydCylinder(B, marker(B, 1.32, 0.7, 0.8), marker(BL, 0.9, 1.25, -0.45), { rBarrel: 0.07, rRod: 0.04, barrel: 1.0 }));
    this.cyls.push(new HydCylinder(B, this.ripperTop, this.ripperArm, { rBarrel: 0.08, rRod: 0.05, barrel: 0.35 }));
    shadowAll(this.root);
    // collision
    this.addBox(B, 0, 1.2, 0, gauge + 0.3, 0.75, 1.75);
    this.addBox(C, 0, 0.7, 0, 0.9, 0.7, 0.8);
    this.bladeBox = this.addBox(BL, 0, 0.7, -0.15, 1.63, 0.7, 0.25);
    this.addBox(RP, 0, 0.55, -2.7, 1.1, 0.5, 0.25);
  }

  setupPhysics() {
    const gauge = S.trackGauge / 2;
    for (const s of [1, -1]) for (const dx of [-0.15, 0.15]) {
      for (let i = 0; i < 7; i++) this.addContact(s * gauge + dx, -1.0, -1.335 + i * (2.67 / 6), { k: 2.4e5, c: 1.4e4, area: 0.107 });
      for (const z of [-1, 1]) this.addContact(s * gauge + dx, -0.72, z * 1.6, { k: 2.4e5, c: 1.4e4, area: 0.08 });
    }
    const self = this;
    this.masses = [
      { m: 17600, pos: () => new THREE.Vector3(0, 0.0, 0.05) },
      { m: 2300, pos: () => { const e = self.edgeLocal(); return new THREE.Vector3(0, e.y - 1.0 + 0.6, e.z - 0.3); } },
      { m: 600, pos: () => new THREE.Vector3(0, -0.3, -2.7) },
    ];
  }

  edgeLocal(theta = this.theta) {
    return { y: TRUN.y + ARM_L * Math.sin(theta), z: TRUN.z + ARM_L * Math.cos(theta) };
  }

  control(inp) { this.inp = inp; }

  step(dt) {
    const inp = this.inp || {};
    this._hydAcc += dt;
    while (this._hydAcc >= 1 / 60) { this._hydAcc -= 1 / 60; this.hydStep(1 / 60, inp); }
    this.applyGravity();
    if (this.drag.lengthSq() > 1) this.body.addForceAt(this.drag, this.dragPt);
    if (this.lift.lengthSq() > 1) this.body.addForceAt(this.lift, this.dragPt);
    if (this.ripDrag) this.body.addForceAt(this.ripDrag, this.ripPt);
    const drive = clamp(inp.drive || 0, -1, 1), steer = clamp(inp.steer || 0, -1, 1);
    const vmax = drive >= 0 ? S.speedFwd / 3.6 : S.speedRev / 3.6;
    const v = drive * vmax * (inp.slow ? 0.35 : 1);
    const k = Math.max(Math.abs(v) * 0.55, 0.65);
    const tl = v - steer * k, tr = v + steer * k;
    this._bl = (this._bl || 0) + (tl - (this._bl || 0)) * Math.min(1, dt * 2.5);
    this._br = (this._br || 0) + (tr - (this._br || 0)) * Math.min(1, dt * 2.5);
    this.trackStep(dt, this._bl, this._br, S.enginePower * 0.85, 0.92);
    this.body.integrate(dt);
  }

  bladeMatrix(out) {
    const e = this.edgeLocal();
    const q = new THREE.Quaternion().setFromEuler(new THREE.Euler(-(this.theta - TH0), 0, this.tilt, 'XYZ'));
    const local = new THREE.Matrix4().compose(new THREE.Vector3(0, e.y - 1.0, e.z), q, new THREE.Vector3(1, 1, 1));
    const bm = new THREE.Matrix4().compose(this.body.pos, this.body.quat, new THREE.Vector3(1, 1, 1));
    return out.multiplyMatrices(bm, local);
  }

  hydStep(dt, inp) {
    const T = this.terrain;
    // blade lift / tilt and ripper hydraulics
    const lr = 0.24, tr = 0.35;
    this.theta = clamp(this.theta + clamp(inp.blade || 0, -1, 1) * lr * dt, THMIN, THMAX);
    const maxTilt = Math.atan(S.maxTilt / (S.bladeWidth / 2));
    this.tilt = clamp(this.tilt + clamp(inp.tilt || 0, -1, 1) * tr * dt, -maxTilt, maxTilt);
    this.rip = clamp(this.rip + clamp(inp.ripper || 0, -1, 1) * 0.5 * dt, 0, 1);

    const M = this.bladeMatrix(this._bm || (this._bm = new THREE.Matrix4()));
    const P = new THREE.Vector3(), Q = new THREE.Vector3();
    const fwd = new THREE.Vector3(0, 0, 1).applyQuaternion(this.body.quat);
    const fh = new THREE.Vector3(fwd.x, 0, fwd.z).normalize();
    const vf = this.body.vel.dot(fh);
    const seen = this._seen || (this._seen = new Set()); seen.clear();
    let Fcut = 0, Frock = 0, pile = 0, nCut = 0, cutDepthSum = 0;
    const pileSeen = this._pileSeen || (this._pileSeen = new Set()); pileSeen.clear();
    const local = (x, y, z) => {
      // wings swept forward
      const ax = Math.abs(x);
      let zz = z;
      if (ax > 1.2) zz += (ax - 1.2) * Math.tan(WING);
      return P.set(x, y, zz).applyMatrix4(M);
    };
    for (let x = -1.6; x <= 1.601; x += 0.2) {
      // cutting strip just behind the edge
      for (const d of [-0.12, 0.0]) {
        local(x, 0, d);
        const k = T.nearestIdx(P.x, P.z);
        if (k < 0 || seen.has(k)) continue;
        seen.add(k);
        const H = T.bank[k] + T.loose[k];
        if (H > P.y + 0.003) {
          const ex = H - P.y;
          const rockTop = T.rock[k];
          if (rockTop > P.y) { Frock += (rockTop - P.y) * 4e6 * 0.2; }
          const vol = T.take(k, ex);
          if (vol > 0) {
            const tp = T.lastType;
            local(x, 0, 0.45 + Math.random() * 0.2);
            T.deposit(P.x, P.z, vol, tp);
            if (Math.random() < 0.06 * Math.min(1, vol * 50)) this.particles.dust.puff(P.x, P.y + 0.6, P.z, 1.2, tp);
          }
        }
      }
      // resistance from undisturbed bank ahead of the edge
      local(x, 0, 0.3);
      const ka = T.nearestIdx(P.x, P.z);
      if (ka >= 0) {
        const edgeY = P.y;
        const dep = T.bank[ka] - edgeY;
        if (dep > 0) {
          const t = T.bankType(ka, T.bank[ka] - dep * 0.5);
          Fcut += dep * 0.2 * SOIL.energyBank[t] * 0.5;
          cutDepthSum += dep; nCut++;
        }
        if (T.rock[ka] > edgeY) Frock += (T.rock[ka] - edgeY) * 4e6 * 0.2;
      }
      // the carried pile (loose soil in front of the moldboard)
      for (let d = 0.25; d <= 1.8; d += 0.25) {
        local(x, 0, d);
        const k = T.nearestIdx(P.x, P.z);
        if (k < 0 || pileSeen.has(k)) continue;
        pileSeen.add(k);
        const ex = T.bank[k] + T.loose[k] - P.y;
        if (ex > 0) pile += Math.min(T.loose[k], ex) * T.cellA;
        // spill over the top of the blade
        if (d <= 0.5) {
          Q.set(x, 1.45, 0).applyMatrix4(M);
          const over = T.bank[k] + T.loose[k] - Q.y;
          if (over > 0.05) {
            const vol = T.take(k, Math.min(over, T.loose[k]));
            local(x, 0, -0.9);
            if (vol > 0) this.particles.emit(Q.x, Q.y + 0.1, Q.z, -fh.x * 0.8, 0.5, -fh.z * 0.8, vol, T.lastType, 0.3);
          }
        }
      }
    }
    this.pile = pile;
    const Fpush = pile * 1550 * 9.81 * 0.6;
    const F = Fpush + Fcut + Math.min(Frock, 1.5e6);
    this.bladeForce = F;
    local(0, 0.3, 0);
    this.dragPt.copy(P);
    const dir = -Math.tanh(vf / 0.06);
    this.drag.copy(fh).multiplyScalar(F * dir);
    // pressing the edge into hard ground lifts the front of the tractor
    const avgCut = nCut ? cutDepthSum / nCut : 0;
    const up = Math.min(Fcut * 0.35 + Math.min(Frock, 4e5) * 0.5, 1.2e5) * (avgCut > 0.02 || Frock > 0 ? 1 : 0);
    this.lift.set(0, up, 0);

    // ripper shanks
    this.ripDrag = null;
    const ripDepth = 0.45 - this.rip * (0.45 + S.rippeDepth);
    let Frip = 0;
    for (const x of this.shanks) {
      const tip = new THREE.Vector3(x, ripDepth - 1.0, -2.72).applyQuaternion(this.body.quat).add(this.body.pos);
      for (const ox of [-0.25, 0, 0.25]) {
        const p = new THREE.Vector3(x + ox, ripDepth - 1.0 + Math.abs(ox) * 0.6, -2.72).applyQuaternion(this.body.quat).add(this.body.pos);
        const k = T.nearestIdx(p.x, p.z);
        if (k < 0) continue;
        if (T.bank[k] > p.y + 0.02) {
          const dep = T.bank[k] - p.y;
          const rockDep = Math.max(0, T.rock[k] - p.y);
          if (ox === 0) Frip += (dep - rockDep) * 0.3 * SOIL.energyBank[SOIL.CLAY] * 0.8 + rockDep * 0.3 * 2.4e6;
          if (Math.abs(vf) > 0.05) T.loosenTo(k, p.y);
        }
      }
      void tip;
    }
    if (Frip > 0) {
      this.ripDrag = fh.clone().multiplyScalar(-Frip * Math.tanh(vf / 0.06));
      this.ripPt = new THREE.Vector3(0, -0.8, -2.7).applyQuaternion(this.body.quat).add(this.body.pos);
    }
    this.ripForce = Frip;
    const load = clamp((this.tractive || 0) / (S.operatingMass * 9.81 * 0.75), 0, 1.2);
    this.throttle = 1;
    this.updateEngine(dt, Math.max(load, Math.abs(inp.drive || 0) * 0.3));
  }

  updateVisuals(dt) {
    this.syncRoot();
    const e = this.edgeLocal();
    this.blade.position.set(0, e.y, e.z);
    this.blade.rotation.set(-(this.theta - TH0), 0, this.tilt, 'XYZ');
    const ripDepth = 0.45 - this.rip * (0.45 + S.rippeDepth);
    this.ripper.position.y = ripDepth - 0.45 + 0.45; // shank tips at ripDepth relative to design
    this.ripper.position.y = ripDepth + 0.15;
    this.root.updateMatrixWorld();
    for (const s of this.struts) s.update();
    for (const c of this.cyls) c.update();
    this.belts[0].update(-(this.beltActualL || 0) * dt);
    this.belts[1].update(-(this.beltActualR || 0) * dt);
    this.beacon.material.emissiveIntensity = 0.5 + 0.8 * Math.max(0, Math.sin(performance.now() * 0.012 + 1));
    this.emitExhaust(dt);
    // track slip kicks dust
    if (this.trackSlip > 0.25 && Math.random() < 0.5) {
      const p = new THREE.Vector3((Math.random() > 0.5 ? 1 : -1) * 0.99, -0.9, -1.4).applyQuaternion(this.body.quat).add(this.body.pos);
      this.particles.dust.puff(p.x, p.y, p.z, 1.0, 0);
    }
  }

  hud() {
    const e = this.edgeLocal();
    const pileT = this.pile * 1.55;
    return {
      title: S.name,
      sub: S.className,
      rows: [
        ['Engine', `${Math.round(this.rpm)} rpm`],
        ['Ground speed', `${this.speedKmh().toFixed(1)} km/h`],
        ['Blade edge', `${(e.y).toFixed(2)} m (${e.y < 0 ? 'cutting' : 'raised'})`],
        ['Blade tilt', `${(Math.sin(this.tilt) * S.bladeWidth / 2).toFixed(2)} m`],
        ['Blade load', `${this.pile.toFixed(2)} m³ · ${pileT.toFixed(1)} t`],
        ['Drawbar / resistance', `${(this.bladeForce / 1000).toFixed(0)} kN`],
        ['Ripper', this.rip > 0.4 ? `in ground (${(this.ripForce / 1000).toFixed(0)} kN)` : 'raised'],
        ['Track slip', `${Math.round(clamp(this.trackSlip * 100, 0, 100))}%`],
      ],
      gauge: clamp(this.bladeForce / (S.operatingMass * 9.81 * 0.9), 0, 1),
      stall: this.trackSlip > 0.4,
    };
  }
}
