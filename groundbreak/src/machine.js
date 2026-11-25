// Base class shared by all machines: rigid body, contacts, collision volumes,
// diesel engine state and camera anchors.
import * as THREE from 'three';
import { RigidBody, Contact, resolveContact } from './physics.js';
import { clamp } from './util.js';

const _g = new THREE.Vector3();
const _p = new THREE.Vector3();
const _corner = new THREE.Vector3();

export class Machine {
  constructor(scene, terrain, particles, mass, inertia) {
    this.scene = scene;
    this.terrain = terrain;
    this.particles = particles;
    this.body = new RigidBody(mass, inertia);
    this.root = new THREE.Group();
    scene.add(this.root);
    this.contacts = [];
    this.boxes = [];
    this.masses = []; // {m, obj?, local: Vector3, getter?}
    this.radius = 6;
    this.rpm = 800;
    this.idleRpm = 800; this.maxRpm = 2000;
    this.engineLoad = 0;
    this.engineOn = true;
    this.throttle = 0;
    this._bodyM = new THREE.Matrix4();
    this._rootInv = new THREE.Matrix4();
    this.cabCam = new THREE.Object3D();
    this.lookTarget = new THREE.Vector3();
    this.exhaust = new THREE.Object3D();
    this._smokeT = 0;
  }

  place(x, z, yaw, lift = 1.2) {
    this.body.pos.set(x, this.terrain.heightAt(x, z) + lift, z);
    this.body.quat.setFromAxisAngle(new THREE.Vector3(0, 1, 0), yaw);
    this.body.vel.set(0, 0, 0); this.body.angVel.set(0, 0, 0);
    this.syncRoot();
  }

  addBox(parent, cx, cy, cz, hx, hy, hz) {
    const o = new THREE.Object3D();
    o.position.set(cx, cy, cz);
    o.userData.h = new THREE.Vector3(hx, hy, hz);
    parent.add(o);
    const rec = { obj: o, h: o.userData.h, lm: new THREE.Matrix4(), matrixWorld: new THREE.Matrix4() };
    this.boxes.push(rec);
    return rec;
  }

  addContact(x, y, z, opts) {
    const c = new Contact(new THREE.Vector3(x, y, z), opts);
    this.contacts.push(c);
    return c;
  }

  /** Safety net: if the physics state ever becomes non-finite, put the machine back on its feet. */
  guard() {
    const b = this.body;
    const ok = [b.pos.x, b.pos.y, b.pos.z, b.quat.x, b.quat.w, b.vel.x, b.vel.y, b.angVel.x, b.angVel.y].every(Number.isFinite);
    if (ok) { this._lastGood = this._lastGood || { p: new THREE.Vector3(), q: new THREE.Quaternion() }; this._lastGood.p.copy(b.pos); this._lastGood.q.copy(b.quat); return; }
    const g = this._lastGood;
    b.pos.copy(g ? g.p : new THREE.Vector3()); b.quat.copy(g ? g.q : new THREE.Quaternion());
    b.vel.set(0, 0, 0); b.angVel.set(0, 0, 0); b.force.set(0, 0, 0); b.torque.set(0, 0, 0);
    for (const c of this.contacts) { c.zl = c.zt = 0; }
    if (this.onReset) this.onReset();
    console.warn(this.label + ': physics state reset');
  }

  syncRoot() {
    this.root.position.copy(this.body.pos);
    this.root.quaternion.copy(this.body.quat);
  }

  /** After the scene graph is updated each frame, cache box transforms relative to the body. */
  refreshCollisionLocal() {
    this.root.updateMatrixWorld();
    this._rootInv.copy(this.root.matrixWorld).invert();
    for (const b of this.boxes) b.lm.multiplyMatrices(this._rootInv, b.obj.matrixWorld);
  }

  _updateBodyMatrix() {
    this._bodyM.compose(this.body.pos, this.body.quat, _corner.set(1, 1, 1));
  }
  getCollisionBoxes() {
    this._updateBodyMatrix();
    for (const b of this.boxes) b.matrixWorld.multiplyMatrices(this._bodyM, b.lm);
    return this.boxes;
  }
  getCollisionPoints() {
    this._updateBodyMatrix();
    if (!this._pts) this._pts = [];
    let n = 0;
    for (const b of this.boxes) {
      for (let sx = -1; sx <= 1; sx++) for (let sy = -1; sy <= 1; sy += 2) for (let sz = -1; sz <= 1; sz++) {
        if (!this._pts[n]) this._pts[n] = new THREE.Vector3();
        _corner.set(sx * b.h.x, sy * b.h.y, sz * b.h.z).applyMatrix4(b.lm).applyMatrix4(this._bodyM);
        this._pts[n++].copy(_corner);
      }
    }
    this._pts.length = n;
    return this._pts;
  }

  /** Gravity applied per component so the centre of mass moves with the work tool / payload. */
  applyGravity() {
    let total = 0;
    for (const m of this.masses) {
      const mass = typeof m.m === 'function' ? m.m() : m.m;
      total += mass;
      const lp = m.pos();
      this.body.toWorld(lp, _p);
      _g.set(0, -9.81 * mass, 0);
      this.body.addForceAt(_g, _p);
    }
    this.body.mass = total;
    return total;
  }

  /** Tracked running gear: belts speeds for left (+x) and right (-x) sides. */
  trackStep(dt, beltL, beltR, power, mu = 0.85) {
    // engine power limit based on last step's tractive force
    const FL = Math.abs(this._FL || 0), FR = Math.abs(this._FR || 0);
    const demand = FL * Math.abs(beltL) + FR * Math.abs(beltR);
    const k = demand > power ? power / demand : 1;
    this._powerFactor = (this._powerFactor ?? 1) + (k - (this._powerFactor ?? 1)) * Math.min(1, dt * 20);
    const bl = beltL * this._powerFactor, br = beltR * this._powerFactor;
    let sumL = 0, sumR = 0, slip = 0, nGround = 0;
    for (const c of this.contacts) {
      const left = c.local.x > 0;
      const belt = left ? bl : br;
      resolveContact(this.body, c, this.terrain, dt, { belt, muLong: mu, muLat: mu * 0.8, rollRes: 0.015 });
      if (left) sumL += c.Flong; else sumR += c.Flong;
      if (c.ground) { nGround++; slip += c.slip; }
    }
    this._FL = sumL; this._FR = sumR;
    this.trackSlip = nGround ? slip / nGround : 0;
    this.grounded = nGround > 0;
    this.beltActualL = bl; this.beltActualR = br;
    this.tractive = Math.abs(sumL) + Math.abs(sumR);
    return this._powerFactor;
  }

  /** Simple diesel model shared by machines. */
  updateEngine(dt, demandLoad, governed = true) {
    const target = governed ? this.idleRpm + (this.maxRpm - this.idleRpm) * clamp(this.throttle, 0, 1) : this.rpm;
    const droop = 160 * clamp(demandLoad, 0, 1.3);
    const tgt = target - droop;
    this.rpm += (tgt - this.rpm) * Math.min(1, dt * 3);
    this.engineLoad += (clamp(demandLoad, 0, 1) - this.engineLoad) * Math.min(1, dt * 4);
  }

  emitExhaust(dt) {
    this._smokeT -= dt;
    const load = this.engineLoad;
    if (this._smokeT <= 0) {
      this._smokeT = 0.12 - load * 0.08;
      this.exhaust.getWorldPosition(_p);
      this.particles.dust.smoke(_p.x, _p.y, _p.z, load);
    }
  }

  speedKmh() {
    const f = new THREE.Vector3(0, 0, 1).applyQuaternion(this.body.quat);
    return this.body.vel.dot(f) * 3.6;
  }
}
