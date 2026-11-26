// Minimal rigid-body dynamics with penalty ground contacts and a bristle
// (elasto-plastic) friction model: gives true static friction so machines park
// on slopes without creeping, while still sliding when the friction limit is hit.
import * as THREE from 'three';

const _r = new THREE.Vector3();
const _q = new THREE.Quaternion();
const _wl = new THREE.Vector3();
const _tl = new THREE.Vector3();
const _iw = new THREE.Vector3();

export class RigidBody {
  constructor(mass, inertia) {
    this.mass = mass;
    this.I = inertia.clone();
    this.pos = new THREE.Vector3();
    this.quat = new THREE.Quaternion();
    this.vel = new THREE.Vector3();
    this.angVel = new THREE.Vector3();
    this.force = new THREE.Vector3();
    this.torque = new THREE.Vector3();
  }
  setMass(m, inertia) { this.mass = m; if (inertia) this.I.copy(inertia); }
  toWorld(v, out) { return out.copy(v).applyQuaternion(this.quat).add(this.pos); }
  dirToWorld(v, out) { return out.copy(v).applyQuaternion(this.quat); }
  toLocal(v, out) { _q.copy(this.quat).invert(); return out.copy(v).sub(this.pos).applyQuaternion(_q); }
  pointVel(p, out) {
    _r.subVectors(p, this.pos);
    return out.crossVectors(this.angVel, _r).add(this.vel);
  }
  addForce(f) { this.force.add(f); }
  addForceAt(f, p) {
    this.force.add(f);
    _r.subVectors(p, this.pos);
    this.torque.x += _r.y * f.z - _r.z * f.y;
    this.torque.y += _r.z * f.x - _r.x * f.z;
    this.torque.z += _r.x * f.y - _r.y * f.x;
  }
  integrate(dt) {
    this.vel.addScaledVector(this.force, dt / this.mass);
    _q.copy(this.quat).invert();
    _wl.copy(this.angVel).applyQuaternion(_q);
    _tl.copy(this.torque).applyQuaternion(_q);
    _iw.set(this.I.x * _wl.x, this.I.y * _wl.y, this.I.z * _wl.z);
    // gyroscopic term
    _tl.x -= _wl.y * _iw.z - _wl.z * _iw.y;
    _tl.y -= _wl.z * _iw.x - _wl.x * _iw.z;
    _tl.z -= _wl.x * _iw.y - _wl.y * _iw.x;
    _wl.x += (_tl.x / this.I.x) * dt;
    _wl.y += (_tl.y / this.I.y) * dt;
    _wl.z += (_tl.z / this.I.z) * dt;
    this.angVel.copy(_wl).applyQuaternion(this.quat);
    // light aerodynamic / internal damping
    this.angVel.multiplyScalar(1 - 0.4 * dt);
    this.vel.multiplyScalar(1 - 0.01 * dt);
    this.pos.addScaledVector(this.vel, dt);
    const w = this.angVel, q = this.quat, h = 0.5 * dt;
    const qx = q.x, qy = q.y, qz = q.z, qw = q.w;
    q.x += h * (w.x * qw + w.y * qz - w.z * qy);
    q.y += h * (w.y * qw + w.z * qx - w.x * qz);
    q.z += h * (w.z * qw + w.x * qy - w.y * qx);
    q.w += h * (-w.x * qx - w.y * qy - w.z * qz);
    q.normalize();
    this.force.set(0, 0, 0);
    this.torque.set(0, 0, 0);
  }
}

const _p = new THREE.Vector3(), _n = new THREE.Vector3(), _v = new THREE.Vector3();
const _f = new THREE.Vector3(), _l = new THREE.Vector3(), _F = new THREE.Vector3();
const BRISTLE = 0.012; // m of elastic tangential deflection before sliding

/**
 * A ground contact point (track roller / tyre patch).
 * opts: k (N/m), c (Ns/m), area (m^2), dir (local forward Vector3)
 */
export class Contact {
  constructor(local, opts = {}) {
    this.local = local.clone();
    this.k = opts.k ?? 2e5;
    this.c = opts.c ?? 1.5e4;
    this.area = opts.area ?? 0.1;
    this.dir = opts.dir ?? new THREE.Vector3(0, 0, 1);
    this.zl = 0; this.zt = 0; // bristle deflection
    this.Fn = 0; this.slip = 0;
    this.world = new THREE.Vector3();
    this.normal = new THREE.Vector3(0, 1, 0);
    this.Flong = 0; this.Flat = 0;
    this.ground = false;
  }
}

/**
 * Resolve one contact against the terrain and apply forces to `body`.
 * mode.belt:      target ground-relative longitudinal speed (tracks / braked wheel = 0)
 * mode.free:      true => longitudinal is free-rolling with explicit force `drive`
 * mode.muLong / muLat: friction coefficients
 * mode.longCap:   optional cap on |Flong| (brake torque / engine)
 */
export function resolveContact(body, c, terrain, dt, mode, worldPointOverride) {
  const p = worldPointOverride || body.toWorld(c.local, c.world);
  if (worldPointOverride) c.world.copy(p);
  const h = terrain.heightAt(p.x, p.z);
  const pen = h - p.y;
  if (pen <= 0) { c.Fn = 0; c.zl = 0; c.zt = 0; c.ground = false; c.slip = 0; return 0; }
  c.ground = true;
  terrain.normalAt(p.x, p.z, _n);
  c.normal.copy(_n);
  const depth = pen * _n.y;
  body.pointVel(p, _v);
  const vn = _v.dot(_n);
  let Fn = c.k * depth - c.c * vn;
  if (Fn < 0) Fn = 0;
  const maxFn = c.k * 0.25;
  if (Fn > maxFn) Fn = maxFn;
  c.Fn = Fn;
  body.dirToWorld(c.dir, _f);
  _f.addScaledVector(_n, -_f.dot(_n)).normalize();
  _l.crossVectors(_n, _f);
  const vl = _v.dot(_f), vt = _v.dot(_l);
  // lateral bristle
  const muT = mode.muLat * Fn;
  const Kt = muT / BRISTLE, Ct = Fn * 2.5 / 9.81 * 9;
  c.zt += vt * dt;
  let Ft = -(Kt * c.zt + Ct * vt);
  let slipT = 0;
  if (Math.abs(Ft) > muT) { Ft = Math.sign(Ft) * muT; c.zt = Math.sign(c.zt) * BRISTLE; slipT = Math.abs(vt); }
  // longitudinal
  let Fl, slipL = 0;
  if (mode.free) {
    Fl = (mode.drive || 0) - (mode.rollRes || 0.03) * Fn * Math.tanh(vl / 0.15);
    c.zl = 0;
  } else {
    const sv = vl - (mode.belt || 0);
    let muL = mode.muLong * Fn;
    if (mode.longCap !== undefined && mode.longCap < muL) muL = mode.longCap;
    const Kl = Math.max(muL, 1) / BRISTLE, Cl = Ct;
    c.zl += sv * dt;
    Fl = -(Kl * c.zl + Cl * sv);
    if (Math.abs(Fl) > muL) { Fl = Math.sign(Fl) * muL; c.zl = Math.sign(c.zl) * BRISTLE; slipL = Math.abs(sv); }
    if (mode.rollRes) Fl -= mode.rollRes * Fn * Math.tanh(vl / 0.15);
  }
  // friction ellipse
  const lim = mode.muLong * Fn;
  const mag = Math.hypot(Fl / Math.max(mode.muLong, 0.01), Ft / Math.max(mode.muLat, 0.01));
  if (mag > Fn && mag > 0) { const s = Fn / mag; Fl *= s; Ft *= s; slipL = Math.max(slipL, Math.abs(vl - (mode.belt || 0)) * 0.5); }
  c.Flong = Fl; c.Flat = Ft;
  c.slip = Math.max(slipL, slipT);
  _F.copy(_n).multiplyScalar(Fn).addScaledVector(_f, Fl).addScaledVector(_l, Ft);
  body.addForceAt(_F, p);
  // soil response: compaction, ruts, grass damage
  terrain.compact(p.x, p.z, Fn / c.area, dt, c.slip,
    mode.print ? { dx: _f.x, dz: _f.z, hw: mode.print.hw, type: mode.print.type } : undefined);
  void lim;
  return Fn;
}

// ---------- machine-to-machine collisions (OBB vs sample points) ----------
const _a = new THREE.Vector3(), _b = new THREE.Vector3(), _m4 = new THREE.Matrix4(), _inv = new THREE.Matrix4();
const _vp = new THREE.Vector3(), _vq = new THREE.Vector3(), _dv = new THREE.Vector3(), _nW = new THREE.Vector3();

/**
 * boxes: [{c: Vector3 centre(local), h: Vector3 half extents, obj?: Object3D (local frame override)}]
 * points: array of local points for each machine (box corners)
 */
export function collideMachines(A, B) {
  // points of A inside boxes of B and vice versa
  pointsVsBoxes(A, B);
  pointsVsBoxes(B, A);
}

function pointsVsBoxes(P, Q) {
  if (P.body.pos.distanceTo(Q.body.pos) > P.radius + Q.radius) return;
  const pts = P.getCollisionPoints();
  for (const box of Q.getCollisionBoxes()) {
    _inv.copy(box.matrixWorld).invert();
    for (const wp of pts) {
      _a.copy(wp).applyMatrix4(_inv);
      const dx = box.h.x - Math.abs(_a.x), dy = box.h.y - Math.abs(_a.y), dz = box.h.z - Math.abs(_a.z);
      if (dx <= 0 || dy <= 0 || dz <= 0) continue;
      // push out along axis of least penetration (in box frame)
      let pen, nl;
      if (dx < dy && dx < dz) { pen = dx; nl = _b.set(Math.sign(_a.x), 0, 0); }
      else if (dy < dz) { pen = dy; nl = _b.set(0, Math.sign(_a.y), 0); }
      else { pen = dz; nl = _b.set(0, 0, Math.sign(_a.z)); }
      _nW.copy(nl).transformDirection(box.matrixWorld);
      P.body.pointVel(wp, _vp);
      Q.body.pointVel(wp, _vq);
      _dv.subVectors(_vp, _vq);
      const vn = _dv.dot(_nW);
      // stiff but bounded: steel-on-steel contact that can shove a machine without launching it
      const k = 1.2e6, c = 4e4;
      let F = k * Math.min(pen, 0.15) - c * vn;
      if (F < 0) F = 0;
      if (F > 1.2e5) F = 1.2e5;
      // tangential damping (friction-ish)
      _dv.addScaledVector(_nW, -vn);
      const ft = Math.min(F * 0.5, _dv.length() * 4e4);
      const tdir = _dv.lengthSq() > 1e-8 ? _dv.normalize() : _dv.set(0, 0, 0);
      _vp.copy(_nW).multiplyScalar(F).addScaledVector(tdir, -ft);
      P.body.addForceAt(_vp, wp);
      _vp.multiplyScalar(-1);
      Q.body.addForceAt(_vp, wp);
    }
  }
}
void _m4;
