// Physical soil clods.
//
// When the bucket's steel cuts the ground, the cut soil becomes real clods at the
// place it was cut. Whether a clod ends up in the bucket is purely geometric: it
// stays there only if the bucket shell is around it. Clods are simulated with
// position-based dynamics (Macklin et al., "Unified particle physics"): gravity,
// clod-clod contact with static/kinetic friction, and contact with the terrain,
// the bucket (shell + side plates + lip/teeth), the truck body (floor, sides,
// front, swinging tailgate) and machine collision boxes. Every contact pushes back
// on the machine involved, so a loaded bucket or truck body really is heavy.
//
// Clods resting on open ground for a long while (or crushed under tracks/tyres)
// merge back into the terrain's loose soil layer, keeping the simulation bounded.
import * as THREE from 'three';
import { SOIL, SITE } from './config.js';

const MAX = 6000;
const PACK = 0.6;          // clods fill ~60 % of a loose heap's volume
const MU_S = 0.75, MU_K = 0.55; // clod friction (clay on clay / steel)
const CELL = 0.3;
const MERGE_AFTER = 45;    // s asleep on open ground before becoming terrain
const COLS = [[0.13, 0.085, 0.05], [0.42, 0.22, 0.10], [0.36, 0.32, 0.26], [0.33, 0.32, 0.31]];

export class SoilChunks {
  constructor(scene, terrain) {
    this.T = terrain;
    this.n = 0;
    this.px = new Float32Array(MAX); this.py = new Float32Array(MAX); this.pz = new Float32Array(MAX);
    this.qx = new Float32Array(MAX); this.qy = new Float32Array(MAX); this.qz = new Float32Array(MAX); // predicted
    this.vx = new Float32Array(MAX); this.vy = new Float32Array(MAX); this.vz = new Float32Array(MAX);
    this.r = new Float32Array(MAX); this.vol = new Float32Array(MAX); this.type = new Float32Array(MAX);
    this.sleep = new Float32Array(MAX); this.asleep = new Uint8Array(MAX);
    this.rot = new Float32Array(MAX * 4);
    this.where = new Uint8Array(MAX); // 0 ground/air, 1 bucket, 2 truck body
    this.sx = new Float32Array(MAX); this.sy = new Float32Array(MAX); this.sz = new Float32Array(MAX); this.sn = new Float32Array(MAX);
    this.colliders = [];
    this.padDumped = 0;
    this.spawnCarry = 0;
    // spatial hash
    this.hashSize = 16384;
    this.cellStart = new Int32Array(this.hashSize + 1);
    this.cellIdx = new Int32Array(MAX);
    this.cellOf = new Int32Array(MAX);
    // rendering
    const g = new THREE.IcosahedronGeometry(1, 1);
    const p = g.attributes.position;
    for (let i = 0; i < p.count; i++) {
      const x = p.getX(i), y = p.getY(i), z = p.getZ(i);
      const k = 0.8 + 0.28 * Math.abs(Math.sin(x * 4.1 + y * 6.3 + z * 3.7)) - 0.08 * Math.abs(Math.cos(x * 9 - z * 7));
      p.setXYZ(i, x * k, y * k * 0.85, z * k);
    }
    g.computeVertexNormals();
    this.mesh = new THREE.InstancedMesh(g, new THREE.MeshStandardMaterial({ roughness: 0.96, metalness: 0 }), MAX);
    this.mesh.instanceColor = new THREE.InstancedBufferAttribute(new Float32Array(MAX * 3), 3);
    this.mesh.castShadow = true; this.mesh.receiveShadow = true;
    this.mesh.frustumCulled = false;
    this.mesh.count = 0;
    scene.add(this.mesh);
    this._m = new THREE.Matrix4(); this._q = new THREE.Quaternion(); this._v = new THREE.Vector3(); this._s = new THREE.Vector3(); this._c = new THREE.Color();
    this._axis = new THREE.Vector3(); this._dq = new THREE.Quaternion();
  }

  // ---------------------------------------------------------------- spawning
  /** Loose volume represented by one clod of radius r. */
  static volFor(r) { return (4 / 3) * Math.PI * r * r * r / PACK; }

  /** Is there room for a clod of radius r at (x,y,z)? */
  free(x, y, z, r) {
    const T = this;
    const ci = Math.floor(x / CELL), cj = Math.floor(y / CELL), ck = Math.floor(z / CELL);
    for (let a = -1; a <= 1; a++) for (let b = -1; b <= 1; b++) for (let c = -1; c <= 1; c++) {
      const h = this._hash(ci + a, cj + b, ck + c);
      for (let s = this.cellStart[h]; s < this.cellStart[h + 1]; s++) {
        const j = this.cellIdx[s];
        if (j >= this.n) continue;
        const dx = T.px[j] - x, dy = T.py[j] - y, dz = T.pz[j] - z, rr = (T.r[j] + r) * 0.75;
        if (dx * dx + dy * dy + dz * dz < rr * rr) return false;
      }
    }
    // also against clods spawned this step (not yet hashed)
    for (let j = this._hashedN; j < this.n; j++) {
      const dx = T.px[j] - x, dy = T.py[j] - y, dz = T.pz[j] - z, rr = (T.r[j] + r) * 0.75;
      if (dx * dx + dy * dy + dz * dz < rr * rr) return false;
    }
    return true;
  }

  add(x, y, z, vx, vy, vz, r, type) {
    if (this.n >= MAX) this._makeRoom();
    if (this.n >= MAX) return -1;
    const i = this.n++;
    this.px[i] = this.qx[i] = x; this.py[i] = this.qy[i] = y; this.pz[i] = this.qz[i] = z;
    this.vx[i] = vx; this.vy[i] = vy; this.vz[i] = vz;
    this.r[i] = r; this.vol[i] = SoilChunks.volFor(r); this.type[i] = type;
    this.sleep[i] = 0; this.asleep[i] = 0; this.where[i] = 0;
    const a = Math.random() * 6.28, b = Math.random() * 6.28;
    this._q.setFromEuler(new THREE.Euler(a, b, a * 0.5));
    this.rot[i * 4] = this._q.x; this.rot[i * 4 + 1] = this._q.y; this.rot[i * 4 + 2] = this._q.z; this.rot[i * 4 + 3] = this._q.w;
    return i;
  }

  /**
   * Soil just cut from a terrain column (x,z between heights lo..top) turns into clods.
   * accept(x,y,z) says whether the spot is inside the bucket cavity. Returns the loose
   * volume that could not become a clod (no room / outside the cavity) so the caller
   * can shove it aside as terrain.
   */
  fromCut(x, z, lo, top, vol, type, accept, vel) {
    let left = vol + this.spawnCarry;
    this.spawnCarry = 0;
    let tries = 0;
    let rejected = 0;
    const cs = this.T.res;
    while (left > 0 && tries < 28) {
      tries++;
      const r = 0.06 + 0.07 * Math.pow(Math.random(), 1.6);
      const v = SoilChunks.volFor(r);
      if (v > left) { if (left < v * 0.5 || tries > 20) { this.spawnCarry = left; left = 0; break; } }
      const sx = x + (Math.random() - 0.5) * cs, sz = z + (Math.random() - 0.5) * cs;
      const sy = lo + r + 0.02 + Math.random() * Math.max(0, top - lo - r);
      const at = accept(sx, sy, sz, r);
      const st = this.cutStats || (this.cutStats = { ok: 0, outside: 0, full: 0 });
      if (!at) { st.outside += Math.min(v, left); rejected += Math.min(v, left); left -= v; continue; }
      let spot = null;
      for (const c of at) if (this.free(c[0], c[1], c[2], r)) { spot = c; break; }
      if (!spot) { st.full += Math.min(v, left); rejected += Math.min(v, left); left -= v; continue; }
      st.ok += v;
      this.add(spot[0], spot[1], spot[2], vel.x, vel.y, vel.z, r, type);
      left -= v;
    }
    return rejected + Math.max(0, left);
  }

  /** Clods resting on open ground merge back into terrain; oldest first. */
  _makeRoom() {
    let best = -1, bt = -1;
    for (let i = 0; i < this.n; i++) if (this.where[i] === 0 && this.asleep[i] && this.sleep[i] > bt) { bt = this.sleep[i]; best = i; }
    if (best < 0) for (let i = 0; i < this.n; i++) if (this.where[i] === 0 && this.sleep[i] > bt) { bt = this.sleep[i]; best = i; }
    if (best >= 0) this._merge(best);
  }

  _merge(i) {
    const x = this.px[i], z = this.pz[i];
    if (this.T.deposit(x, z, this.vol[i], this.type[i])) {
      const p = SITE.pad;
      if (x > p.x0 && x < p.x1 && z > p.z0 && z < p.z1) this.padDumped += this.vol[i];
    }
    this._remove(i);
  }

  _remove(i) {
    const l = --this.n;
    if (i === l) return;
    for (const a of [this.px, this.py, this.pz, this.qx, this.qy, this.qz, this.vx, this.vy, this.vz, this.r, this.vol, this.type, this.sleep, this.asleep, this.where]) a[i] = a[l];
    for (let c = 0; c < 4; c++) this.rot[i * 4 + c] = this.rot[l * 4 + c];
  }

  // ---------------------------------------------------------------- hash
  _hash(i, j, k) { return ((Math.imul(i, 92837111) ^ Math.imul(j, 689287499) ^ Math.imul(k, 283923481)) >>> 0) % this.hashSize; }
  _build(usePred) {
    const X = usePred ? this.qx : this.px, Y = usePred ? this.qy : this.py, Z = usePred ? this.qz : this.pz;
    const cs = this.cellStart; cs.fill(0);
    for (let i = 0; i < this.n; i++) {
      const h = this._hash(Math.floor(X[i] / CELL), Math.floor(Y[i] / CELL), Math.floor(Z[i] / CELL));
      this.cellOf[i] = h; cs[h + 1]++;
    }
    for (let h = 0; h < this.hashSize; h++) cs[h + 1] += cs[h];
    const fill = this._fill || (this._fill = new Int32Array(this.hashSize));
    fill.set(cs.subarray(0, this.hashSize));
    for (let i = 0; i < this.n; i++) this.cellIdx[fill[this.cellOf[i]]++] = i;
    this._hashedN = this.n;
  }

  // ---------------------------------------------------------------- simulation
  /**
   * colliders: [{ kind:'bucket'|'bed'|'box', M, Mprev, inv, machine, crush, ... }]
   * Returns nothing; accumulates reaction forces into collider.machine._chunkF/_chunkT.
   */
  step(dt, colliders) {
    this._dt = dt;
    const n = this.n;
    if (!n) { this._build(false); return; }
    const T = this.T;
    const { px, py, pz, qx, qy, qz, vx, vy, vz, r } = this;
    // wake clods around moving colliders
    for (const c of colliders) {
      if (!c.moving) continue;
      const b = c.aabb;
      for (let i = 0; i < n; i++) {
        if (!this.asleep[i]) continue;
        if (px[i] > b[0] && px[i] < b[3] && py[i] > b[1] && py[i] < b[4] && pz[i] > b[2] && pz[i] < b[5]) { this.asleep[i] = 0; this.sleep[i] = 0; }
      }
    }
    // predict
    for (let i = 0; i < n; i++) {
      if (this.asleep[i]) { qx[i] = px[i]; qy[i] = py[i]; qz[i] = pz[i]; continue; }
      vy[i] -= 9.81 * dt;
      // light air drag
      vx[i] *= 0.999; vz[i] *= 0.999;
      qx[i] = px[i] + vx[i] * dt; qy[i] = py[i] + vy[i] * dt; qz[i] = pz[i] + vz[i] * dt;
    }
    this._build(true);
    for (const c of colliders) { c._F = c._F || new THREE.Vector3(); c._F.set(0, 0, 0); c._Tq = c._Tq || new THREE.Vector3(); c._Tq.set(0, 0, 0); }
    const ITER = 3;
    this.where.fill(0, 0, n);
    this.sx.fill(0, 0, n); this.sy.fill(0, 0, n); this.sz.fill(0, 0, n); this.sn.fill(0, 0, n);
    for (let it = 0; it < ITER; it++) {
      const last = it === ITER - 1;
      // clod-clod
      for (let i = 0; i < n; i++) {
        if (this.asleep[i]) continue; // sleeping clods are handled from their awake neighbours
        const ci = Math.floor(qx[i] / CELL), cj = Math.floor(qy[i] / CELL), ck = Math.floor(qz[i] / CELL);
        const wi = 1 / r[i] ** 3;
        for (let a = -1; a <= 1; a++) for (let b = -1; b <= 1; b++) for (let cc = -1; cc <= 1; cc++) {
          const h = this._hash(ci + a, cj + b, ck + cc);
          for (let s = this.cellStart[h]; s < this.cellStart[h + 1]; s++) {
            const j = this.cellIdx[s];
            if (j === i || (j < i && !this.asleep[j])) continue;
            const dx = qx[i] - qx[j], dy = qy[i] - qy[j], dz = qz[i] - qz[j];
            const rr = r[i] + r[j];
            const d2 = dx * dx + dy * dy + dz * dz;
            if (d2 >= rr * rr || d2 < 1e-12) continue;
            const wj = this.asleep[j] ? 0 : 1 / r[j] ** 3;
            const W = wi + wj;
            if (W === 0) continue;
            const d = Math.sqrt(d2), pen = rr - d;
            const nx = dx / d, ny = dy / d, nz = dz / d;
            // only a clod that is actually moving wakes a sleeping one (settling jitter doesn't)
            if (pen > 0.004 && this.asleep[j] && wi && vx[i] * vx[i] + vy[i] * vy[i] + vz[i] * vz[i] > 0.09) { this.asleep[j] = 0; this.sleep[j] = 0; }
            if (pen > 0.004 && this.asleep[i] && wj && vx[j] * vx[j] + vy[j] * vy[j] + vz[j] * vz[j] > 0.09) { this.asleep[i] = 0; this.sleep[i] = 0; }
            const ai = wi / W, aj = wj / W;
            qx[i] += nx * pen * ai; qy[i] += ny * pen * ai; qz[i] += nz * pen * ai;
            qx[j] -= nx * pen * aj; qy[j] -= ny * pen * aj; qz[j] -= nz * pen * aj;
            if (last) {
              this.sx[i] += vx[j]; this.sy[i] += vy[j]; this.sz[i] += vz[j]; this.sn[i]++;
              this.sx[j] += vx[i]; this.sy[j] += vy[i]; this.sz[j] += vz[i]; this.sn[j]++;
              // position-level Coulomb friction
              let rx = (qx[i] - px[i]) - (qx[j] - px[j]), ry = (qy[i] - py[i]) - (qy[j] - py[j]), rz = (qz[i] - pz[i]) - (qz[j] - pz[j]);
              const rn = rx * nx + ry * ny + rz * nz;
              rx -= rn * nx; ry -= rn * ny; rz -= rn * nz;
              const rt = Math.sqrt(rx * rx + ry * ry + rz * rz);
              if (rt > 1e-9) {
                const k = rt < MU_S * pen ? 1 : Math.min(1, MU_K * pen / rt);
                qx[i] -= rx * k * ai; qy[i] -= ry * k * ai; qz[i] -= rz * k * ai;
                qx[j] += rx * k * aj; qy[j] += ry * k * aj; qz[j] += rz * k * aj;
              }
            }
          }
        }
      }
      // colliders
      for (let i = 0; i < n; i++) {
        if (this.asleep[i] && it > 0) continue;
        // terrain
        const h = T.heightAt(qx[i], qz[i]);
        if (qy[i] - r[i] < h) {
          const pen = h - (qy[i] - r[i]);
          qy[i] += pen;
          if (last) { this._fric(i, pen, 0, 1, 0, 0, 0, 0, 0.9); this.sn[i] += 2; }
        }
        const X = qx[i], Y = qy[i], Z = qz[i], R = r[i];
        for (const c of colliders) {
          const bb = c.aabb;
          if (X < bb[0] - R || X > bb[3] + R || Y < bb[1] - R || Y > bb[4] + R || Z < bb[2] - R || Z > bb[5] + R) continue;
          const res = c.kind === 'bucket' ? this._bucket(i, c, last) : c.kind === 'bed' ? this._bed(i, c, last) : this._box(i, c, last);
          if (res === 2 && c.crush) { this._crushList = this._crushList || []; this._crushList.push(i); }
        }
      }
    }
    // velocities, sleeping, bookkeeping
    const idt = 1 / dt;
    for (let i = 0; i < n; i++) {
      if (this.asleep[i]) { this.sleep[i] += dt; continue; }
      let nvx = (qx[i] - px[i]) * idt, nvy = (qy[i] - py[i]) * idt, nvz = (qz[i] - pz[i]) * idt;
      // angular clods don't roll like marbles: motion relative to whatever they rest
      // on (other clods, steel, ground) is strongly resisted
      if (this.sn[i] > 0) {
        const k = 1 / this.sn[i], sxv = this.sx[i] * k, syv = this.sy[i] * k, szv = this.sz[i] * k;
        const keep = 0.72;
        nvx = sxv + (nvx - sxv) * keep; nvz = szv + (nvz - szv) * keep;
        if (nvy > syv) nvy = syv + (nvy - syv) * keep;
      }
      // keep the solver from injecting energy on deep overlaps
      const sp = Math.sqrt(nvx * nvx + nvy * nvy + nvz * nvz);
      if (sp > 7) { const s = 7 / sp; nvx *= s; nvy *= s; nvz *= s; }
      // rolling tumble for rendering
      const hv = Math.hypot(nvx, nvz);
      if (hv > 0.02) {
        this._axis.set(nvz / hv, 0, -nvx / hv);
        this._dq.setFromAxisAngle(this._axis, hv * dt / r[i]);
        this._q.set(this.rot[i * 4], this.rot[i * 4 + 1], this.rot[i * 4 + 2], this.rot[i * 4 + 3]).premultiply(this._dq);
        this.rot[i * 4] = this._q.x; this.rot[i * 4 + 1] = this._q.y; this.rot[i * 4 + 2] = this._q.z; this.rot[i * 4 + 3] = this._q.w;
      }
      vx[i] = nvx; vy[i] = nvy; vz[i] = nvz;
      px[i] = qx[i]; py[i] = qy[i]; pz[i] = qz[i];
      // relative-to-support stillness (clods riding a moving body stay awake)
      if (sp < 0.12) { this.sleep[i] += dt; if (this.sleep[i] > 0.6) { this.asleep[i] = 1; vx[i] = vy[i] = vz[i] = 0; } }
      else this.sleep[i] = 0;
      if (py[i] < -50) { this._remove(i); i--; }
    }
    // crushed under tracks / tyres, or long-resting on open ground -> terrain
    if (this._crushList && this._crushList.length) {
      const set = [...new Set(this._crushList)].sort((a, b) => b - a);
      for (const i of set) if (i < this.n) this._merge(i);
      this._crushList.length = 0;
    }
    for (let i = this.n - 1; i >= 0; i--) if (this.asleep[i] && this.where[i] === 0 && this.sleep[i] > MERGE_AFTER) this._merge(i);
    this._build(false);
    // resting (asleep) clods inside the bucket / truck body still weigh on it
    {
      const bucket = colliders.find((c) => c.kind === 'bucket'), bed = colliders.find((c) => c.kind === 'bed');
      const g = 9.81 * dt * dt;
      for (let i = 0; i < this.n; i++) {
        if (!this.asleep[i] || !this.where[i]) continue;
        const c = this.where[i] === 1 ? bucket : bed;
        if (!c || !c.machine) continue;
        const m = this.vol[i] * PACK * SOIL.densBank[Math.round(this.type[i])];
        const Fy = -m * g;
        c._F.y += Fy;
        const rx = this.px[i] - c.machine.body.pos.x, rz = this.pz[i] - c.machine.body.pos.z;
        c._Tq.x += -rz * Fy; c._Tq.z += rx * Fy;
      }
    }
    // reaction forces -> machines
    const k = 1 / (dt * dt);
    for (const c of colliders) {
      if (!c.machine) continue;
      c.machine._chunkF.addScaledVector(c._F, k);
      c.machine._chunkT.addScaledVector(c._Tq, k);
    }
  }

  /** Coulomb friction against a surface whose own motion this step is (sx,sy,sz). */
  _fric(i, pen, nx, ny, nz, sx, sy, sz, mu) {
    let rx = (this.qx[i] - this.px[i]) - sx, ry = (this.qy[i] - this.py[i]) - sy, rz = (this.qz[i] - this.pz[i]) - sz;
    const rn = rx * nx + ry * ny + rz * nz;
    rx -= rn * nx; ry -= rn * ny; rz -= rn * nz;
    const rt = Math.sqrt(rx * rx + ry * ry + rz * rz);
    if (rt < 1e-9) return [0, 0, 0];
    const k = rt < mu * pen * 1.4 ? 1 : Math.min(1, mu * pen / rt);
    this.qx[i] -= rx * k; this.qy[i] -= ry * k; this.qz[i] -= rz * k;
    return [rx * k, ry * k, rz * k];
  }

  /** Apply a collider correction to clod i and the equal-and-opposite push to the machine. */
  _push(i, c, dx, dy, dz, last, nx, ny, nz, pen) {
    this.qx[i] += dx; this.qy[i] += dy; this.qz[i] += dz;
    let fx = dx, fy = dy, fz = dz;
    if (last) {
      // surface motion at the contact point
      const mv = c.motionAt(this.qx[i], this.qy[i], this.qz[i]);
      const f = this._fric(i, pen, nx, ny, nz, mv.x, mv.y, mv.z, c.mu ?? MU_K);
      const idt = 1 / this._dt;
      this.sx[i] += mv.x * idt * 2; this.sy[i] += mv.y * idt * 2; this.sz[i] += mv.z * idt * 2; this.sn[i] += 2;
      fx += f[0]; fy += f[1]; fz += f[2];
    }
    if (c.machine && !this.asleep[i]) {
      const m = this.vol[i] * PACK * SOIL.densBank[Math.round(this.type[i])];
      // reaction on the machine: F = -m * dx / dt^2 (scaled in step())
      const Fx = -fx * m, Fy = -fy * m, Fz = -fz * m;
      c._F.x += Fx; c._F.y += Fy; c._F.z += Fz;
      const rx = this.qx[i] - c.machine.body.pos.x, ry = this.qy[i] - c.machine.body.pos.y, rz = this.qz[i] - c.machine.body.pos.z;
      c._Tq.x += ry * Fz - rz * Fy; c._Tq.y += rz * Fx - rx * Fz; c._Tq.z += rx * Fy - ry * Fx;
    }
  }

  // --- bucket: shell polyline (u,w) extruded across x, side plates, open mouth
  _bucket(i, c, last) {
    const r = this.r[i];
    const e = c.inv.elements;
    const X = this.qx[i], Y = this.qy[i], Z = this.qz[i];
    // local: x lateral, y = w (toward back), z = u (pin -> tip)
    let lx = e[0] * X + e[4] * Y + e[8] * Z + e[12];
    let ly = e[1] * X + e[5] * Y + e[9] * Z + e[13];
    let lz = e[2] * X + e[6] * Y + e[10] * Z + e[14];
    if (lz < -0.4 - r || lz > 1.9 + r || ly < -0.6 - r || ly > 1.1 + r || Math.abs(lx) > c.halfW + 0.05 + r) return 0;
    const P = this.px[i], Q = this.py[i], R = this.pz[i];
    const ox = e[0] * P + e[4] * Q + e[8] * R + e[12], oy = e[1] * P + e[5] * Q + e[9] * R + e[13], oz = e[2] * P + e[6] * Q + e[10] * R + e[14];
    const M = c.M.elements;
    const toWorld = (dx, dy, dz) => [M[0] * dx + M[4] * dy + M[8] * dz, M[1] * dx + M[5] * dy + M[9] * dz, M[2] * dx + M[6] * dy + M[10] * dz];
    const tw = 0.03;
    let hit = 0;
    if (Math.abs(lx) < c.halfW) {
      const S = c.shell;
      for (let s = 0; s < S.length - 1; s++) {
        const au = S[s][0], aw = S[s][1], bu = S[s + 1][0], bw = S[s + 1][1];
        const eu = bu - au, ew = bw - aw;
        const L2 = eu * eu + ew * ew;
        let t = ((lz - au) * eu + (ly - aw) * ew) / L2;
        t = t < 0 ? 0 : t > 1 ? 1 : t;
        const cu = au + eu * t, cw = aw + ew * t;
        let du = lz - cu, dw = ly - cw;
        let d = Math.sqrt(du * du + dw * dw);
        const lim = r + tw;
        if (d >= lim) continue;
        // which side was the clod on before? (thin wall: never let it tunnel)
        const sPrev = eu * (oy - aw) - ew * (oz - au);
        const sCur = eu * (ly - aw) - ew * (lz - au);
        let nu, nw;
        if (d > 1e-6) { nu = du / d; nw = dw / d; } else { const L = Math.sqrt(L2); nu = ew / L; nw = -eu / L; }
        if (t > 0 && t < 1 && sPrev * sCur < 0) { nu = -nu; nw = -nw; d = -d; }
        const pen = lim - d;
        ly += nw * pen; lz += nu * pen;
        const [wx, wy, wz] = toWorld(0, nw * pen, nu * pen);
        const [nx, ny, nz] = toWorld(0, nw, nu);
        this._push(i, c, wx, wy, wz, last, nx, ny, nz, pen);
        hit = 1;
      }
    }
    // side plates (only within the bucket profile)
    if (c.inside(lz, ly)) {
      const ax = Math.abs(lx), wall = c.halfW + 0.02;
      if (Math.abs(ax - wall) < r + 0.02) {
        const prevIn = Math.abs(ox) < wall;
        const sgn = Math.sign(lx) || 1;
        const target = prevIn ? (wall - r - 0.02) : (wall + r + 0.02);
        const dx = sgn * target - lx;
        if ((prevIn && ax > target) || (!prevIn && ax < target)) {
          const [wx, wy, wz] = toWorld(dx, 0, 0);
          const [nx, ny, nz] = toWorld(Math.sign(dx), 0, 0);
          this._push(i, c, wx, wy, wz, last, nx, ny, nz, Math.abs(dx));
          lx += dx; hit = 1;
        }
      }
      if (Math.abs(lx) < wall) this.where[i] = 1;
    } else if (ly < 0.02 && ly > -0.7 && lz > -0.1 && lz < 1.5 && Math.abs(lx) < c.halfW) {
      this.where[i] = 1; // heaped above the mouth
    }
    return hit;
  }

  // --- truck body: floor, two sides, front wall, top-hinged tailgate
  _bed(i, c, last) {
    const r = this.r[i];
    const e = c.inv.elements;
    const X = this.qx[i], Y = this.qy[i], Z = this.qz[i];
    let lx = e[0] * X + e[4] * Y + e[8] * Z + e[12];
    let ly = e[1] * X + e[5] * Y + e[9] * Z + e[13];
    let lz = e[2] * X + e[6] * Y + e[10] * Z + e[14];
    const { hw, len, floor, side, front } = c;
    if (Math.abs(lx) > hw + 0.3 || lz < -1.5 || lz > len + 0.4 || ly < floor - 0.6 || ly > floor + side + 1.2) return 0;
    const P = this.px[i], Q = this.py[i], R = this.pz[i];
    const ox = e[0] * P + e[4] * Q + e[8] * R + e[12], oy = e[1] * P + e[5] * Q + e[9] * R + e[13], oz = e[2] * P + e[6] * Q + e[10] * R + e[14];
    const M = c.M.elements;
    const toWorld = (dx, dy, dz) => [M[0] * dx + M[4] * dy + M[8] * dz, M[1] * dx + M[5] * dy + M[9] * dz, M[2] * dx + M[6] * dy + M[10] * dz];
    const push = (dx, dy, dz, nx, ny, nz) => {
      const pen = Math.sqrt(dx * dx + dy * dy + dz * dz);
      const [wx, wy, wz] = toWorld(dx, dy, dz);
      const [nwx, nwy, nwz] = toWorld(nx, ny, nz);
      this._push(i, c, wx, wy, wz, last, nwx, nwy, nwz, pen);
      lx += dx; ly += dy; lz += dz;
    };
    let hit = 0;
    const inPlan = Math.abs(lx) < hw && lz > 0 && lz < len;
    // floor (plate 6 cm thick)
    if (inPlan && ly - r < floor && oy > floor - 0.08) { push(0, floor - (ly - r), 0, 0, 1, 0); hit = 1; }
    // sides
    const top = floor + side;
    if (lz > -0.05 && lz < len + 0.05 && ly < top + r * 0.5 && ly > floor - 0.1) {
      for (const sg of [1, -1]) {
        const wx = sg * hw;
        const prevIn = sg * ox < hw;
        const d = sg * (lx - wx);
        if (prevIn && d > -r) { push(-sg * (d + r), 0, 0, -sg, 0, 0); hit = 1; }
        else if (!prevIn && d < r && d > -0.1) { push(sg * (r - d), 0, 0, sg, 0, 0); hit = 1; }
      }
    }
    // front wall
    if (Math.abs(lx) < hw + 0.05 && ly < front && ly > floor - 0.1) {
      const prevIn = oz < len;
      const d = lz - len;
      if (prevIn && d > -r) { push(0, 0, -(d + r), 0, 0, -1); hit = 1; }
      else if (!prevIn && d < r && d > -0.1) { push(0, 0, r - d, 0, 0, 1); hit = 1; }
    }
    // tailgate: plate from the top hinge, swung back by c.gate radians
    {
      const hy = top + 0.02, hz = -0.04, L = side + 0.02;
      const dy = -Math.cos(c.gate), dz = -Math.sin(c.gate);
      if (Math.abs(lx) < hw) {
        const ry = ly - hy, rz = lz - hz;
        let t = ry * dy + rz * dz; t = t < 0 ? 0 : t > L ? L : t;
        const cy = hy + dy * t, cz = hz + dz * t;
        let ey = ly - cy, ez = lz - cz;
        let d = Math.sqrt(ey * ey + ez * ez);
        const lim = r + 0.03;
        if (d < lim) {
          // gate normal pointing into the bed
          const ny0 = -dz, nz0 = dy; // rotate (dy,dz) by +90 -> (−dz, dy)
          const sPrev = (oy - hy) * ny0 + (oz - hz) * nz0, sCur = ey * ny0 + ez * nz0;
          let ny, nz;
          if (d > 1e-6) { ny = ey / d; nz = ez / d; } else { ny = ny0; nz = nz0; }
          if (t > 0 && t < L && sPrev * sCur < 0) { ny = -ny; nz = -nz; d = -d; }
          const pen = lim - d;
          push(0, ny * pen, nz * pen, 0, ny, nz);
          hit = 1;
        }
      }
    }
    if (Math.abs(lx) < hw + 0.05 && lz > -0.05 && lz < len + 0.05 && ly > floor - 0.1 && ly < top + 1.0) this.where[i] = 2;
    return hit;
  }

  // --- machine collision box (kinematic). Returns 2 when the clod is under a crushing part.
  _box(i, c, last) {
    const r = this.r[i];
    const e = c.inv.elements;
    const X = this.qx[i], Y = this.qy[i], Z = this.qz[i];
    const lx = e[0] * X + e[4] * Y + e[8] * Z + e[12];
    const ly = e[1] * X + e[5] * Y + e[9] * Z + e[13];
    const lz = e[2] * X + e[6] * Y + e[10] * Z + e[14];
    const h = c.h;
    const dx = h.x + r - Math.abs(lx), dy = h.y + r - Math.abs(ly), dz = h.z + r - Math.abs(lz);
    if (dx <= 0 || dy <= 0 || dz <= 0) return 0;
    if (c.crush && ly < -h.y + r * 1.2) return 2; // squeezed under a track / tyre
    let ax = 0, ay = 0, az = 0, pen;
    if (dx < dy && dx < dz) { ax = Math.sign(lx) || 1; pen = dx; }
    else if (dy < dz) { ay = Math.sign(ly) || 1; pen = dy; }
    else { az = Math.sign(lz) || 1; pen = dz; }
    const M = c.M.elements;
    const nx = M[0] * ax + M[4] * ay + M[8] * az, ny = M[1] * ax + M[5] * ay + M[9] * az, nz = M[2] * ax + M[6] * ay + M[10] * az;
    this._push(i, c, nx * pen, ny * pen, nz * pen, last, nx, ny, nz, pen);
    return 1;
  }

  // ---------------------------------------------------------------- queries / render
  volumeWhere(code) {
    let s = 0, t = 0;
    for (let i = 0; i < this.n; i++) if (this.where[i] === code) { s += this.vol[i]; t += this.type[i] * this.vol[i]; }
    return { vol: s, type: s > 0 ? t / s : 0 };
  }

  render() {
    const n = this.n;
    for (let i = 0; i < n; i++) {
      const r = this.r[i];
      this._q.set(this.rot[i * 4], this.rot[i * 4 + 1], this.rot[i * 4 + 2], this.rot[i * 4 + 3]);
      this._m.compose(this._v.set(this.px[i], this.py[i], this.pz[i]), this._q, this._s.set(r * 1.08, r * 1.08, r * 1.08));
      this.mesh.setMatrixAt(i, this._m);
      const t = Math.min(3, this.type[i]), a = Math.floor(t), b = Math.min(3, a + 1), f = t - a;
      const v = 0.55 + 0.4 * ((Math.imul(i + 7, 2654435761) >>> 24) / 255);
      this._c.setRGB((COLS[a][0] * (1 - f) + COLS[b][0] * f) * v, (COLS[a][1] * (1 - f) + COLS[b][1] * f) * v, (COLS[a][2] * (1 - f) + COLS[b][2] * f) * v);
      this.mesh.setColorAt(i, this._c);
    }
    this.mesh.count = n;
    this.mesh.instanceMatrix.needsUpdate = true;
    this.mesh.instanceColor.needsUpdate = true;
  }
}
