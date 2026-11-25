// Deformable layered soil terrain.
//
// Each grid node stores a column:  rock (undiggable bedrock top)
//                                  bank (undisturbed / compacted soil top, >= rock)
//                                  loose (thickness of loose, excavated material on top)
// Loose material flows whenever the local slope exceeds its angle of repose; bank
// material can stand much steeper but fails (collapses into loose) past its limit.
// All transfers conserve volume, with bank->loose swell and loose->bank compaction.

import * as THREE from 'three';
import { fbm, clamp, smoothstep } from './util.js';
import { SOIL, SITE } from './config.js';

const TILE = 16;

export function baseHeight(x, z) {
  const r = Math.hypot(x, z);
  let h = 1.3 * fbm(x * 0.011 + 3.1, z * 0.011 - 1.7, 4, 11) + 0.22 * fbm(x * 0.05, z * 0.05, 3, 5)
        + 0.03 * fbm(x * 0.5, z * 0.5, 2, 9);
  const f = smoothstep(28, 70, r);
  h = h * (0.35 + 0.65 * f) + 0.01 * x;
  // distant hills
  const hill = smoothstep(90, 260, r);
  h += hill * (10 * (fbm(x * 0.004 + 9, z * 0.004 - 4, 4, 41) * 0.5 + 0.5) + 4);
  return h;
}

export class Terrain {
  constructor() {
    this.size = SITE.size;
    this.res = SITE.res;
    this.N = Math.round(this.size / this.res);
    this.W = this.N + 1;
    this.half = this.size / 2;
    this.cellA = this.res * this.res;
    const n = this.W * this.W;
    this.bank = new Float32Array(n);
    this.loose = new Float32Array(n);
    this.rock = new Float32Array(n);
    this.orig = new Float32Array(n);
    this.ltype = new Float32Array(n);
    this.dist = new Float32Array(n);
    this.TW = Math.ceil(this.W / TILE);
    this.tileActive = new Uint8Array(this.TW * this.TW);
    this.dirtyMin = this.W; this.dirtyMax = -1;
    this.tanL = Math.tan(SOIL.REPOSE_LOOSE_DEG * Math.PI / 180);
    this.tanB = Math.tan(SOIL.REPOSE_BANK_DEG * Math.PI / 180);
    this.stats = { excavatedBank: 0 };
    this._init();
  }

  _init() {
    const { W, res, half } = this;
    const p = SITE.pit;
    for (let j = 0; j < W; j++) for (let i = 0; i < W; i++) {
      const x = i * res - half, z = j * res - half;
      const k = j * W + i;
      const h = baseHeight(x, z);
      this.bank[k] = h; this.orig[k] = h;
      let rd = 3.7 + 2.0 * fbm(x * 0.03 + 7, z * 0.03, 3, 21);
      // shallow bedrock knoll in the north-east
      const dx = x - 24, dz = z + 22;
      rd -= 2.6 * Math.exp(-(dx * dx + dz * dz) / (2 * 9 * 9));
      // keep rock just below the design grade in the pit so the floor can be reached
      const inPit = x > p.x0 - 3 && x < p.x1 + 3 && z > p.z0 - 3 && z < p.z1 + 3;
      if (inPit) rd = Math.max(rd, p.depth + 0.55 + 0.4 * (fbm(x * 0.2, z * 0.2, 2, 3) * 0.5 + 0.5));
      this.rock[k] = h - Math.max(0.9, rd);
    }
  }

  // ---------- indexing ----------
  idx(i, j) { return j * this.W + i; }
  nodeX(i) { return i * this.res - this.half; }
  toI(x) { return (x + this.half) / this.res; }
  inside(x, z, m = 0) {
    return x > -this.half + m && x < this.half - m && z > -this.half + m && z < this.half - m;
  }
  nearestIdx(x, z) {
    const i = Math.round(this.toI(x)), j = Math.round(this.toI(z));
    if (i < 1 || j < 1 || i > this.N - 1 || j > this.N - 1) return -1;
    return j * this.W + i;
  }
  H(k) { return this.bank[k] + this.loose[k]; }

  heightAt(x, z) {
    let fi = this.toI(x), fj = this.toI(z);
    if (fi < 0 || fj < 0 || fi > this.N || fj > this.N) return baseHeight(x, z);
    let i = Math.floor(fi), j = Math.floor(fj);
    if (i >= this.N) i = this.N - 1;
    if (j >= this.N) j = this.N - 1;
    const u = fi - i, v = fj - j;
    const W = this.W, k = j * W + i;
    const b = this.bank, l = this.loose;
    const h00 = b[k] + l[k], h10 = b[k + 1] + l[k + 1];
    const h01 = b[k + W] + l[k + W], h11 = b[k + W + 1] + l[k + W + 1];
    return (h00 * (1 - u) + h10 * u) * (1 - v) + (h01 * (1 - u) + h11 * u) * v;
  }
  bankAt(x, z) { const k = this.nearestIdx(x, z); return k < 0 ? baseHeight(x, z) : this.bank[k]; }
  rockAt(x, z) { const k = this.nearestIdx(x, z); return k < 0 ? -99 : this.rock[k]; }
  looseAt(x, z) { const k = this.nearestIdx(x, z); return k < 0 ? 0 : this.loose[k]; }

  normalAt(x, z, out) {
    const e = this.res;
    const hl = this.heightAt(x - e, z), hr = this.heightAt(x + e, z);
    const hd = this.heightAt(x, z - e), hu = this.heightAt(x, z + e);
    return out.set(hl - hr, 2 * e, hd - hu).normalize();
  }

  /** Material type of the bank surface at node k when cut down to height y. */
  bankType(k, y) {
    const d = this.orig[k] - y;
    if (y - this.rock[k] < 0.02) return SOIL.ROCK;
    if (d < SOIL.topsoilDepth) return SOIL.TOPSOIL;
    if (y - this.rock[k] < SOIL.gravelBand) return SOIL.GRAVEL;
    return SOIL.CLAY;
  }
  /** Material at the very surface (for UI). */
  surfaceType(k) {
    if (this.loose[k] > 0.02) return Math.round(this.ltype[k]);
    return this.bankType(k, this.bank[k]);
  }

  // ---------- modification ----------
  touch(k) {
    const j = (k / this.W) | 0, i = k - j * this.W;
    if (j < this.dirtyMin) this.dirtyMin = j;
    if (j > this.dirtyMax) this.dirtyMax = j;
    const ti = (i / TILE) | 0, tj = (j / TILE) | 0;
    this.tileActive[tj * this.TW + ti] = 90;
  }

  /** Estimate cutting `depth` metres off node k: returns energy (J) or -1 if blocked by rock. */
  cutCost(k, depth) {
    const lc = Math.min(depth, this.loose[k]);
    const rem = depth - lc;
    if (rem <= 0) return lc * SOIL.energyLoose * this.cellA;
    const avail = this.bank[k] - this.rock[k];
    if (rem > avail + 0.01) return -1;
    const t = this.bankType(k, this.bank[k] - rem * 0.5);
    return (lc * SOIL.energyLoose + rem * SOIL.energyBank[t]) * this.cellA;
  }

  /**
   * Remove `depth` metres from the top of column k. Returns loose volume (m^3);
   * this.lastType holds its material type.
   */
  take(k, depth) {
    if (depth <= 0) return 0;
    const lc = Math.min(depth, this.loose[k]);
    let bc = Math.min(depth - lc, this.bank[k] - this.rock[k]);
    if (bc < 0) bc = 0;
    let type = this.ltype[k];
    if (bc > 0) {
      const bt = this.bankType(k, this.bank[k] - bc * 0.5);
      type = (type * lc + bt * bc * SOIL.SWELL) / (lc + bc * SOIL.SWELL);
      this.bank[k] -= bc;
      this.stats.excavatedBank += bc * this.cellA;
    }
    this.loose[k] -= lc;
    this.dist[k] = 1;
    this.lastType = type;
    this.touch(k);
    return (lc + bc * SOIL.SWELL) * this.cellA;
  }

  addLoose(k, thick, type) {
    if (thick <= 0) return;
    const l = this.loose[k];
    const nl = l + thick;
    this.ltype[k] = (this.ltype[k] * l + type * thick) / nl;
    this.loose[k] = nl;
    this.dist[k] = 1;
    this.touch(k);
  }

  /** Deposit loose volume at a world position, bilinearly spread over 4 nodes. */
  deposit(x, z, vol, type) {
    const fi = this.toI(x), fj = this.toI(z);
    let i = Math.floor(fi), j = Math.floor(fj);
    if (i < 1 || j < 1 || i >= this.N - 1 || j >= this.N - 1) return false;
    const u = fi - i, v = fj - j;
    const t = vol / this.cellA;
    const W = this.W, k = j * W + i;
    this.addLoose(k, t * (1 - u) * (1 - v), type);
    this.addLoose(k + 1, t * u * (1 - v), type);
    this.addLoose(k + W, t * (1 - u) * v, type);
    this.addLoose(k + W + 1, t * u * v, type);
    return true;
  }

  /** Machine traffic: compress loose soil, flatten grass. pressure in Pa. */
  compact(x, z, pressure, dt, slip) {
    const k = this.nearestIdx(x, z);
    if (k < 0) return;
    const l = this.loose[k];
    if (l > 0.002) {
      const rate = clamp(pressure / 9e4, 0, 3) * dt * 1.6;
      const dl = Math.min(l, l * rate + 0.0005 * rate);
      this.loose[k] = l - dl;
      this.bank[k] += dl * SOIL.COMPACT;
      this.touch(k);
    }
    const d = this.dist[k];
    if (d < 1) {
      const nd = Math.min(1, d + dt * (0.35 + slip * 3) * clamp(pressure / 5e4, 0, 2));
      if (nd - d > 0.004 || nd === 1) { this.dist[k] = nd; this.touch(k); } else this.dist[k] = nd;
    }
  }

  /** Ripper / tyre spin: loosen bank down to y (bank -> loose with swell). */
  loosenTo(k, y) {
    if (k < 0) return 0;
    const b = this.bank[k];
    if (y >= b) return 0;
    let cut = b - y;
    let rockCut = 0;
    if (y < this.rock[k]) { rockCut = this.rock[k] - y; this.rock[k] = y; }
    const t = this.bankType(k, b - cut * 0.5);
    const tt = rockCut > 0 ? SOIL.ROCK : t;
    this.bank[k] = y;
    this.addLoose(k, cut * SOIL.SWELL, tt);
    this.stats.excavatedBank += cut * this.cellA;
    return cut;
  }

  // ---------- granular relaxation ----------
  _pair(a, b, d) {
    const bank = this.bank, loose = this.loose;
    const Ha = bank[a] + loose[a], Hb = bank[b] + loose[b];
    let dh = Ha - Hb, src = a, dst = b;
    if (dh < 0) { dh = -dh; src = b; dst = a; }
    let moved = 0;
    const lim = this.tanL * d;
    if (dh > lim && loose[src] > 1e-5) {
      let m = (dh - lim) * 0.22;
      if (m > loose[src]) m = loose[src];
      const lt = this.ltype[src];
      loose[src] -= m;
      const l2 = loose[dst], nl = l2 + m;
      this.ltype[dst] = (this.ltype[dst] * l2 + lt * m) / nl;
      loose[dst] = nl;
      this.dist[dst] = 1;
      moved += m;
      dh -= 2 * m;
    }
    // bank failure (cut face collapse)
    const bd = bank[src] - (bank[dst] + loose[dst]);
    const limB = this.tanB * d;
    if (bd > limB) {
      let m = (bd - limB) * 0.06 * (0.6 + Math.random() * 0.8);
      const av = bank[src] - this.rock[src];
      if (m > av) m = av;
      if (m > 1e-5) {
        const t = this.bankType(src, bank[src] - m * 0.5);
        bank[src] -= m;
        const add = m * SOIL.SWELL;
        const l2 = loose[dst], nl = l2 + add;
        this.ltype[dst] = (this.ltype[dst] * l2 + t * add) / nl;
        loose[dst] = nl;
        this.dist[src] = 1; this.dist[dst] = 1;
        moved += m;
      }
    }
    return moved;
  }

  relax(iters = 2) {
    const { W, TW } = this;
    const r = this.res, rd = r * Math.SQRT2;
    for (let it = 0; it < iters; it++) {
      for (let tj = 0; tj < TW; tj++) for (let ti = 0; ti < TW; ti++) {
        const t = tj * TW + ti;
        if (!this.tileActive[t]) continue;
        let moved = 0;
        const i0 = Math.max(1, ti * TILE), i1 = Math.min(W - 2, ti * TILE + TILE - 1);
        const j0 = Math.max(1, tj * TILE), j1 = Math.min(W - 2, tj * TILE + TILE - 1);
        for (let j = j0; j <= j1; j++) {
          for (let i = i0; i <= i1; i++) {
            const k = j * W + i;
            moved += this._pair(k, k + 1, r);
            moved += this._pair(k, k + W, r);
            moved += this._pair(k, k + W + 1, rd);
            moved += this._pair(k, k + W - 1, rd);
          }
        }
        if (moved > 1e-4) {
          if (j0 < this.dirtyMin) this.dirtyMin = j0 - 1;
          if (j1 > this.dirtyMax) this.dirtyMax = j1 + 1;
          this.tileActive[t] = 90;
          // wake neighbours (flow crosses tile borders)
          for (let dj = -1; dj <= 1; dj++) for (let di = -1; di <= 1; di++) {
            const ni = ti + di, nj = tj + dj;
            if (ni < 0 || nj < 0 || ni >= TW || nj >= TW) continue;
            const q = nj * TW + ni;
            if (this.tileActive[q] < 20) this.tileActive[q] = 20;
          }
        } else if (it === iters - 1) {
          this.tileActive[t]--;
        }
      }
    }
  }

  // ---------- rendering ----------
  buildMesh(detailTex) {
    const { W, res, half } = this;
    const n = W * W;
    const pos = new Float32Array(n * 3);
    const nrm = new Float32Array(n * 3);
    const soil = new Float32Array(n * 4);
    for (let j = 0; j < W; j++) for (let i = 0; i < W; i++) {
      const k = j * W + i;
      pos[k * 3] = i * res - half;
      pos[k * 3 + 2] = j * res - half;
    }
    const idx = new Uint32Array(this.N * this.N * 6);
    let o = 0;
    for (let j = 0; j < this.N; j++) for (let i = 0; i < this.N; i++) {
      const a = j * W + i, b = a + 1, c = a + W, d = c + 1;
      idx[o++] = a; idx[o++] = c; idx[o++] = b;
      idx[o++] = b; idx[o++] = c; idx[o++] = d;
    }
    const g = new THREE.BufferGeometry();
    g.setAttribute('position', new THREE.BufferAttribute(pos, 3).setUsage(THREE.DynamicDrawUsage));
    g.setAttribute('normal', new THREE.BufferAttribute(nrm, 3).setUsage(THREE.DynamicDrawUsage));
    g.setAttribute('soil', new THREE.BufferAttribute(soil, 4).setUsage(THREE.DynamicDrawUsage));
    g.setIndex(new THREE.BufferAttribute(idx, 1));
    this.geo = g;
    this.material = makeSoilMaterial(detailTex);
    this.mesh = new THREE.Mesh(g, this.material);
    this.mesh.receiveShadow = true;
    this.mesh.castShadow = true;
    this.mesh.frustumCulled = false;
    this.dirtyMin = 0; this.dirtyMax = W - 1;
    this.updateMesh();
    g.computeBoundingSphere();
    return this.mesh;
  }

  updateMesh() {
    if (this.dirtyMax < this.dirtyMin) return;
    const { W, res } = this;
    const j0 = Math.max(0, this.dirtyMin - 1), j1 = Math.min(W - 1, this.dirtyMax + 1);
    const pos = this.geo.attributes.position.array;
    const nrm = this.geo.attributes.normal.array;
    const soil = this.geo.attributes.soil.array;
    const b = this.bank, l = this.loose;
    for (let j = j0; j <= j1; j++) {
      for (let i = 0; i < W; i++) {
        const k = j * W + i;
        const h = b[k] + l[k];
        pos[k * 3 + 1] = h;
        const kl = i > 0 ? k - 1 : k, kr = i < W - 1 ? k + 1 : k;
        const kd = j > 0 ? k - W : k, ku = j < W - 1 ? k + W : k;
        let nx = (b[kl] + l[kl]) - (b[kr] + l[kr]);
        let nz = (b[kd] + l[kd]) - (b[ku] + l[ku]);
        let ny = 2 * res;
        const inv = 1 / Math.sqrt(nx * nx + ny * ny + nz * nz);
        nrm[k * 3] = nx * inv; nrm[k * 3 + 1] = ny * inv; nrm[k * 3 + 2] = nz * inv;
        // soil attribute: x looseness, y surface type, z disturbance, w cut depth
        const lf = clamp(l[k] / 0.12, 0, 1);
        const bt = this.bankType(k, b[k]);
        let btc = bt;
        const d = this.orig[k] - b[k];
        if (bt === SOIL.CLAY || bt === SOIL.TOPSOIL) btc = smoothstep(SOIL.topsoilDepth - 0.12, SOIL.topsoilDepth + 0.1, d);
        else if (bt === SOIL.GRAVEL) btc = 1 + smoothstep(SOIL.gravelBand, SOIL.gravelBand * 0.5, b[k] - this.rock[k]);
        soil[k * 4] = lf;
        soil[k * 4 + 1] = lf > 0.5 ? this.ltype[k] : btc * (1 - lf) + this.ltype[k] * lf;
        soil[k * 4 + 2] = this.dist[k];
        soil[k * 4 + 3] = d;
      }
    }
    const start = j0 * W, count = (j1 - j0 + 1) * W;
    for (const [name, size] of [['position', 3], ['normal', 3], ['soil', 4]]) {
      const a = this.geo.attributes[name];
      a.clearUpdateRanges();
      a.addUpdateRange(start * size, count * size);
      a.needsUpdate = true;
    }
    this.dirtyMin = W; this.dirtyMax = -1;
  }
}

// ------------------------------------------------------------------------------------
// Soil shader: physically-based standard material with procedural, triplanar soil
// layers (topsoil / clay / gravelly till / rock), grass on undisturbed ground,
// wet-dark cut faces, survey paint lines and an optional cut/fill grade overlay.
export function makeSoilMaterial(detailTex) {
  const mat = new THREE.MeshStandardMaterial({ color: 0xffffff, roughness: 0.93, metalness: 0 });
  const p = SITE.pit, d = SITE.pad;
  mat.userData.uniforms = {
    uDetail: { value: detailTex },
    uPit: { value: new THREE.Vector4(p.x0, p.z0, p.x1, p.z1) },
    uPad: { value: new THREE.Vector4(d.x0, d.z0, d.x1, d.z1) },
    uPitElev: { value: 0 },
    uPadElev: { value: 0 },
    uGrade: { value: 0 },
  };
  mat.onBeforeCompile = (sh) => {
    Object.assign(sh.uniforms, mat.userData.uniforms);
    sh.vertexShader = sh.vertexShader
      .replace('#include <common>', `#include <common>
        attribute vec4 soil; varying vec4 vSoil; varying vec3 vWPos; varying vec3 vWNrm;`)
      .replace('#include <worldpos_vertex>', `#include <worldpos_vertex>
        vSoil = soil; vWPos = (modelMatrix * vec4(transformed,1.0)).xyz; vWNrm = normalize(mat3(modelMatrix) * objectNormal);`);
    sh.fragmentShader = sh.fragmentShader
      .replace('#include <common>', `#include <common>
        uniform sampler2D uDetail; uniform vec4 uPit; uniform vec4 uPad;
        uniform float uPitElev; uniform float uPadElev; uniform float uGrade;
        varying vec4 vSoil; varying vec3 vWPos; varying vec3 vWNrm;
        float gH; float gRough;
        vec4 tri(vec3 p, vec3 bw, float s) {
          return texture2D(uDetail, p.zy * s) * bw.x + texture2D(uDetail, p.xz * s) * bw.y + texture2D(uDetail, p.xy * s) * bw.z;
        }
        float rectEdge(vec2 p, vec4 r) {
          vec2 c = (r.xy + r.zw) * 0.5; vec2 hs = (r.zw - r.xy) * 0.5;
          vec2 q = abs(p - c) - hs; return abs(max(q.x, q.y));
        }
        bool inRect(vec2 p, vec4 r) { return p.x > r.x && p.x < r.z && p.y > r.y && p.y < r.w; }
        vec3 gbPerturb(vec3 surf_pos, vec3 surf_norm, vec2 dHdxy, float fd) {
          vec3 vSigmaX = normalize(dFdx(surf_pos)); vec3 vSigmaY = normalize(dFdy(surf_pos));
          vec3 R1 = cross(vSigmaY, surf_norm); vec3 R2 = cross(surf_norm, vSigmaX);
          float fDet = dot(vSigmaX, R1) * fd;
          vec3 vGrad = sign(fDet) * (dHdxy.x * R1 + dHdxy.y * R2);
          return normalize(abs(fDet) * surf_norm - vGrad);
        }`)
      .replace('#include <map_fragment>', `
        vec3 wn = normalize(vWNrm);
        vec3 bw = pow(abs(wn), vec3(4.0)); bw /= (bw.x + bw.y + bw.z);
        vec4 d1 = tri(vWPos, bw, 0.55);
        vec4 d2 = tri(vWPos, bw, 0.11);
        vec4 d3 = tri(vWPos, bw, 1.9);
        float loose = vSoil.x; float st = vSoil.y; float dist = vSoil.z; float cutd = vSoil.w;
        vec3 cTop = vec3(0.20, 0.135, 0.085);
        vec3 cClay = vec3(0.50, 0.32, 0.17);
        vec3 cGrav = vec3(0.47, 0.42, 0.35);
        vec3 cRock = vec3(0.36, 0.35, 0.34);
        vec3 soilc = mix(cTop, cClay, smoothstep(0.0, 1.0, st));
        soilc = mix(soilc, cGrav, smoothstep(1.0, 2.0, st));
        soilc = mix(soilc, cRock, smoothstep(2.0, 3.0, st));
        // banding in cut faces (strata), clods and pebbles
        float strata = sin(vWPos.y * 9.0 + d2.b * 6.0) * 0.5 + 0.5;
        float face = 1.0 - smoothstep(0.55, 0.85, wn.y);
        soilc *= 1.0 + face * (strata - 0.5) * 0.18 * (1.0 - loose);
        float pebble = smoothstep(0.55, 0.9, d1.g);
        vec3 pebc = mix(vec3(0.42,0.40,0.37), vec3(0.62,0.58,0.52), d3.a);
        float pebAmt = pebble * (0.15 + 0.6 * smoothstep(1.0, 2.2, st)) * (0.5 + 0.5*loose);
        soilc = mix(soilc, pebc, pebAmt);
        soilc *= 0.78 + 0.42 * d1.r * (0.7 + 0.3 * d2.b);
        soilc *= mix(1.0, 0.82 + 0.4 * d3.r, loose);
        // freshly cut bank is darker / moister; compacted tracks smoother
        soilc *= mix(1.0, 0.82, smoothstep(0.2, 1.5, cutd) * (1.0 - loose));
        // grass on undisturbed original ground: two rotated low-frequency samples break up tiling
        vec2 rp = mat2(0.8, -0.6, 0.6, 0.8) * vWPos.xz;
        vec4 m1 = texture2D(uDetail, rp * 0.013);
        vec4 m2 = texture2D(uDetail, vWPos.xz * 0.0041 + 0.37);
        float macro = m1.b * 0.6 + m2.b * 0.7 - 0.15;
        vec3 grassA = vec3(0.115, 0.165, 0.055); vec3 grassB = vec3(0.24, 0.235, 0.10); vec3 dry = vec3(0.36, 0.31, 0.17);
        vec3 grass = mix(grassA, grassB, smoothstep(0.2, 0.8, macro + (d2.b - 0.5) * 0.4));
        grass = mix(grass, dry, smoothstep(0.62, 0.9, m2.r * 0.6 + d2.r * 0.5) * 0.55);
        grass *= 0.6 + 0.55 * d3.r + 0.25 * (d1.g - 0.3);
        float grassAmt = (1.0 - smoothstep(0.15, 0.85, dist)) * smoothstep(0.55, 0.8, wn.y);
        grassAmt *= smoothstep(0.1, 0.35, d2.r + d1.b * 0.4 + 0.2 + m1.r * 0.2);
        vec3 col = mix(soilc, grass, grassAmt);
        // trampled grass -> muddy brown
        col = mix(col, cTop * (0.8 + 0.3 * d1.r), smoothstep(0.2, 0.6, dist) * (1.0 - smoothstep(0.85,1.0,dist)) * 0.6);
        // survey spray paint (orange: excavation limit, blue: fill pad)
        float ePit = rectEdge(vWPos.xz, uPit); float ePad = rectEdge(vWPos.xz, uPad);
        float paintNoise = d3.a * 0.5 + 0.5;
        float pitLine = (1.0 - smoothstep(0.05, 0.09, ePit)) * step(cutd, 0.08) * paintNoise;
        float padLine = (1.0 - smoothstep(0.05, 0.09, ePad)) * (1.0 - smoothstep(0.1, 0.3, loose)) * paintNoise;
        col = mix(col, vec3(0.95, 0.32, 0.02), pitLine);
        col = mix(col, vec3(0.05, 0.25, 0.85), padLine);
        // grade-control overlay
        if (uGrade > 0.5) {
          float tgt = -1000.0;
          if (inRect(vWPos.xz, uPit)) tgt = uPitElev;
          if (inRect(vWPos.xz, uPad)) tgt = uPadElev;
          if (tgt > -999.0) {
            float dd = vWPos.y - tgt;
            vec3 gc = dd > 0.08 ? mix(vec3(0.9,0.5,0.1), vec3(0.9,0.05,0.05), clamp(dd/1.5,0.0,1.0))
                    : (dd < -0.08 ? mix(vec3(0.1,0.5,0.95), vec3(0.15,0.1,0.8), clamp(-dd/1.0,0.0,1.0)) : vec3(0.1,0.85,0.2));
            float iso = smoothstep(0.03, 0.0, abs(fract(dd * 4.0) - 0.5) - 0.47);
            col = mix(col, gc, 0.55) * (1.0 - iso * 0.35);
          }
        }
        diffuseColor.rgb *= col;
        gH = d1.r * 0.55 + d1.g * 0.6 * (0.4 + loose) + d3.r * 0.25;
        gRough = 0.86 + 0.1 * d1.r - 0.1 * (1.0 - loose) * face;
      `)
      .replace('#include <roughnessmap_fragment>', `#include <roughnessmap_fragment>
        roughnessFactor = gRough;`)
      .replace('#include <normal_fragment_maps>', `#include <normal_fragment_maps>
        normal = gbPerturb(-vViewPosition, normal, vec2(dFdx(gH), dFdy(gH)) * 0.9, faceDirection);`);
  };
  return mat;
}
