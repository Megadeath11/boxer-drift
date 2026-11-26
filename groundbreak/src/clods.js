// 3D soil clods and pebbles scattered on loose spoil (and on gravelly cut faces)
// near the camera. Positions are derived deterministically from a world-space
// hash, so clods stay put as you move, and they are re-seated whenever the soil
// under them changes.
import * as THREE from 'three';
import { SOIL } from './config.js';

const MAX = 12000;
const SPACING = 0.22;   // candidate grid (m)
const RADIUS = 26;      // around the focus point (m)

function hash(i, j, s) {
  let h = Math.imul(i, 73856093) ^ Math.imul(j, 19349663) ^ Math.imul(s, 83492791);
  h = Math.imul(h ^ (h >>> 13), 1274126177);
  return ((h ^ (h >>> 16)) >>> 0) / 4294967296;
}

const COLS = [
  [0.13, 0.085, 0.05], // topsoil
  [0.42, 0.24, 0.11],  // clay
  [0.36, 0.32, 0.26],  // till
  [0.33, 0.32, 0.31],  // rock
];

export class Clods {
  constructor(scene, terrain) {
    this.T = terrain;
    // lumpy, slightly flattened rock shape
    const g = new THREE.IcosahedronGeometry(1, 1);
    const p = g.attributes.position;
    for (let i = 0; i < p.count; i++) {
      const x = p.getX(i), y = p.getY(i), z = p.getZ(i);
      const k = 0.78 + 0.32 * Math.abs(Math.sin(x * 5.1 + y * 3.7 + z * 4.3));
      p.setXYZ(i, x * k, y * k * 0.62, z * k);
    }
    g.computeVertexNormals();
    this.mesh = new THREE.InstancedMesh(g, new THREE.MeshStandardMaterial({ roughness: 0.95, metalness: 0, color: 0xffffff }), MAX);
    this.mesh.instanceColor = new THREE.InstancedBufferAttribute(new Float32Array(MAX * 3), 3);
    this.mesh.castShadow = true;
    this.mesh.receiveShadow = true;
    this.mesh.frustumCulled = false;
    this.mesh.count = 0;
    scene.add(this.mesh);
    this._m = new THREE.Matrix4(); this._q = new THREE.Quaternion(); this._s = new THREE.Vector3();
    this._p = new THREE.Vector3(); this._e = new THREE.Euler(); this._c = new THREE.Color();
    this._lastV = -1; this._lastC = new THREE.Vector3(1e9, 0, 0); this._t = 0;
  }

  update(dt, focus) {
    this._t -= dt;
    const T = this.T;
    const moved = this._lastC.distanceToSquared(focus) > 2.5;
    if (!moved && (T.version === this._lastV || this._t > 0)) return;
    this._t = 0.35;
    this._lastV = T.version; this._lastC.copy(focus);
    let n = 0;
    const i0 = Math.floor((focus.x - RADIUS) / SPACING), i1 = Math.ceil((focus.x + RADIUS) / SPACING);
    const j0 = Math.floor((focus.z - RADIUS) / SPACING), j1 = Math.ceil((focus.z + RADIUS) / SPACING);
    const R2 = RADIUS * RADIUS;
    for (let j = j0; j <= j1 && n < MAX; j++) {
      for (let i = i0; i <= i1 && n < MAX; i++) {
        const h0 = hash(i, j, 1);
        const x = (i + hash(i, j, 2)) * SPACING, z = (j + hash(i, j, 3)) * SPACING;
        const dx = x - focus.x, dz = z - focus.z;
        if (dx * dx + dz * dz > R2) continue;
        const k = T.nearestIdx(x, z);
        if (k < 0) continue;
        const loose = T.loose[k];
        let type, density, sizeMul;
        if (loose > 0.025) {
          type = T.ltype[k];
          density = Math.min(1, loose * 5) * (0.45 + 0.35 * Math.min(1, type));
          sizeMul = 0.6 + 0.5 * Math.min(type, 2) / 2;
          // compacted / tracked spoil has fewer standing clods
          if (T.mtype[k] === 3 || T.mtype[k] === 4) density *= 0.25;
        } else {
          const bt = T.bankType(k, T.bank[k]);
          if (bt < SOIL.GRAVEL || T.orig[k] - T.bank[k] < 0.05) continue;
          type = bt; density = 0.35; sizeMul = 0.45;
        }
        if (h0 > density) continue;
        // falloff in count with distance keeps far areas cheap
        const dist = Math.sqrt(dx * dx + dz * dz);
        if (dist > 12 && hash(i, j, 9) > 12 / dist) continue;
        const r = (0.025 + 0.09 * Math.pow(hash(i, j, 4), 2.2)) * sizeMul * (dist > 12 ? 1.3 : 1);
        const y = T.heightAt(x, z) - r * 0.25;
        this._e.set(hash(i, j, 5) * 6.28, hash(i, j, 6) * 6.28, hash(i, j, 7) * 0.6);
        this._q.setFromEuler(this._e);
        this._s.set(r * (0.8 + 0.5 * hash(i, j, 8)), r, r * (0.8 + 0.5 * hash(i, j, 10)));
        this._m.compose(this._p.set(x, y, z), this._q, this._s);
        this.mesh.setMatrixAt(n, this._m);
        const a = Math.floor(Math.min(type, 3)), b = Math.min(3, a + 1), f = Math.min(type, 3) - a;
        const v = 0.7 + 0.55 * hash(i, j, 11);
        this._c.setRGB((COLS[a][0] * (1 - f) + COLS[b][0] * f) * v, (COLS[a][1] * (1 - f) + COLS[b][1] * f) * v, (COLS[a][2] * (1 - f) + COLS[b][2] * f) * v);
        this.mesh.setColorAt(n, this._c);
        n++;
      }
    }
    this.mesh.count = n;
    this.mesh.instanceMatrix.needsUpdate = true;
    this.mesh.instanceColor.needsUpdate = true;
  }
}
