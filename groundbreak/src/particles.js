// Falling soil (each particle carries real volume which is deposited where it
// lands: terrain, truck body, ...) plus cosmetic dust.
import * as THREE from 'three';
import { makeDustSprite } from './textures.js';
import { SOIL } from './config.js';

const MAX = 2500;
const SOIL_COL = [new THREE.Color(0x4f3a26), new THREE.Color(0x7c5230), new THREE.Color(0x77695a), new THREE.Color(0x606060)];

export class SoilParticles {
  constructor(scene, terrain) {
    this.terrain = terrain;
    this.receivers = []; // objects with tryCatch(particle) -> bool
    this.p = new Float32Array(MAX * 3);
    this.v = new Float32Array(MAX * 3);
    this.vol = new Float32Array(MAX);
    this.type = new Float32Array(MAX);
    this.size = new Float32Array(MAX);
    this.rot = new Float32Array(MAX);
    this.n = 0;
    const g = new THREE.IcosahedronGeometry(1, 0);
    const pos = g.attributes.position;
    for (let i = 0; i < pos.count; i++) {
      const k = 0.75 + ((i * 7919) % 13) / 26;
      pos.setXYZ(i, pos.getX(i) * k, pos.getY(i) * k * 0.8, pos.getZ(i) * k);
    }
    g.computeVertexNormals();
    this.mesh = new THREE.InstancedMesh(g, new THREE.MeshStandardMaterial({ roughness: 1, color: 0xffffff }), MAX);
    this.mesh.instanceColor = new THREE.InstancedBufferAttribute(new Float32Array(MAX * 3), 3);
    this.mesh.castShadow = true;
    this.mesh.frustumCulled = false;
    this.mesh.count = 0;
    scene.add(this.mesh);
    this.dust = new Dust(scene);
    this._m = new THREE.Matrix4(); this._q = new THREE.Quaternion(); this._s = new THREE.Vector3(); this._e = new THREE.Euler();
    this._pp = new THREE.Vector3();
    this.flowRate = 0; // m^3/s landing, for audio
    this._landed = 0;
  }

  /** Emit `vol` m^3 of soil around point (x,y,z) with velocity v and spread. */
  emit(x, y, z, vx, vy, vz, vol, type, spread = 0.25) {
    if (vol <= 0) return;
    const per = 0.012;
    let count = Math.max(1, Math.round(vol / per));
    if (count > 40) count = 40;
    const each = vol / count;
    for (let c = 0; c < count; c++) {
      if (this.n >= MAX) { // out of particles: deposit immediately
        this.terrain.deposit(x, z, each, type);
        continue;
      }
      const i = this.n++;
      this.p[i * 3] = x + (Math.random() - 0.5) * spread;
      this.p[i * 3 + 1] = y + (Math.random() - 0.5) * spread * 0.5;
      this.p[i * 3 + 2] = z + (Math.random() - 0.5) * spread;
      this.v[i * 3] = vx + (Math.random() - 0.5) * 0.6;
      this.v[i * 3 + 1] = vy + (Math.random() - 0.5) * 0.4;
      this.v[i * 3 + 2] = vz + (Math.random() - 0.5) * 0.6;
      this.vol[i] = each;
      this.type[i] = type;
      // visual clod size: smaller than volume-equivalent (many small clumps)
      this.size[i] = Math.cbrt(each) * (0.35 + Math.random() * 0.4);
      this.rot[i] = Math.random() * 6.28;
    }
  }

  update(dt) {
    const T = this.terrain;
    let landed = 0;
    for (let i = 0; i < this.n; i++) {
      const o = i * 3;
      this.v[o + 1] -= 9.81 * dt;
      // air drag on clods
      this.v[o] *= 1 - 0.3 * dt; this.v[o + 2] *= 1 - 0.3 * dt;
      this.p[o] += this.v[o] * dt; this.p[o + 1] += this.v[o + 1] * dt; this.p[o + 2] += this.v[o + 2] * dt;
      this.rot[i] += dt * 4;
      let caught = false;
      if (this.v[o + 1] < 0) {
        for (const r of this.receivers) {
          if (r.tryCatch(this.p[o], this.p[o + 1], this.p[o + 2], this.vol[i], this.type[i])) { caught = true; break; }
        }
      }
      let dead = caught;
      if (!caught) {
        const h = T.heightAt(this.p[o], this.p[o + 2]);
        if (this.p[o + 1] <= h) {
          if (!T.deposit(this.p[o], this.p[o + 2], this.vol[i], this.type[i])) { /* off-grid: lost */ }
          dead = true;
          if (Math.random() < 0.25) this.dust.puff(this.p[o], h + 0.1, this.p[o + 2], 0.6, this.type[i]);
        }
      }
      if (dead) {
        landed += this.vol[i];
        const last = --this.n;
        if (i !== last) {
          this.p[o] = this.p[last * 3]; this.p[o + 1] = this.p[last * 3 + 1]; this.p[o + 2] = this.p[last * 3 + 2];
          this.v[o] = this.v[last * 3]; this.v[o + 1] = this.v[last * 3 + 1]; this.v[o + 2] = this.v[last * 3 + 2];
          this.vol[i] = this.vol[last]; this.type[i] = this.type[last]; this.size[i] = this.size[last]; this.rot[i] = this.rot[last];
        }
        i--;
      }
    }
    this._landed += landed;
    this.flowRate += ((landed / Math.max(dt, 1e-3)) - this.flowRate) * Math.min(1, dt * 4);
    // instance matrices
    const col = new THREE.Color();
    for (let i = 0; i < this.n; i++) {
      const s = this.size[i];
      this._e.set(this.rot[i], this.rot[i] * 0.7, 0);
      this._q.setFromEuler(this._e);
      this._s.set(s, s, s);
      this._pp.set(this.p[i * 3], this.p[i * 3 + 1], this.p[i * 3 + 2]);
      this._m.compose(this._pp, this._q, this._s);
      this.mesh.setMatrixAt(i, this._m);
      const t = this.type[i];
      const a = Math.floor(t), b = Math.min(3, a + 1);
      col.copy(SOIL_COL[a]).lerp(SOIL_COL[b], t - a).multiplyScalar(0.8 + ((i * 31) % 10) / 25);
      this.mesh.setColorAt(i, col);
    }
    this.mesh.count = this.n;
    this.mesh.instanceMatrix.needsUpdate = true;
    if (this.mesh.instanceColor) this.mesh.instanceColor.needsUpdate = true;
    this.dust.update(dt);
  }
  get inFlight() { let s = 0; for (let i = 0; i < this.n; i++) s += this.vol[i]; return s; }
}

const DMAX = 1200;
class Dust {
  constructor(scene) {
    this.p = new Float32Array(DMAX * 3);
    this.v = new Float32Array(DMAX * 3);
    this.life = new Float32Array(DMAX);
    this.max = new Float32Array(DMAX);
    this.sz = new Float32Array(DMAX);
    this.n = 0;
    const g = new THREE.BufferGeometry();
    this.aPos = new THREE.BufferAttribute(new Float32Array(DMAX * 3), 3);
    this.aSize = new THREE.BufferAttribute(new Float32Array(DMAX), 1);
    this.aAlpha = new THREE.BufferAttribute(new Float32Array(DMAX), 1);
    this.aCol = new THREE.BufferAttribute(new Float32Array(DMAX * 3), 3);
    this.col = new Float32Array(DMAX * 3);
    g.setAttribute('position', this.aPos);
    g.setAttribute('size', this.aSize);
    g.setAttribute('alpha', this.aAlpha);
    g.setAttribute('color', this.aCol);
    this.geo = g;
    const mat = new THREE.ShaderMaterial({
      uniforms: { map: { value: makeDustSprite() }, scale: { value: 600 } },
      vertexShader: `attribute float size; attribute float alpha; attribute vec3 color; varying float vA; varying vec3 vC;
        uniform float scale;
        void main(){ vA = alpha; vC = color; vec4 mv = modelViewMatrix * vec4(position,1.0);
          gl_PointSize = size * scale / -mv.z; gl_Position = projectionMatrix * mv; }`,
      fragmentShader: `uniform sampler2D map; varying float vA; varying vec3 vC;
        void main(){ vec4 t = texture2D(map, gl_PointCoord); gl_FragColor = vec4(vC, t.a * vA);
        #include <colorspace_fragment>
        }`,
      transparent: true, depthWrite: false,
    });
    this.points = new THREE.Points(g, mat);
    this.points.frustumCulled = false;
    this.points.renderOrder = 5;
    scene.add(this.points);
  }
  puff(x, y, z, size = 1, type = 0, vx = 0, vz = 0) {
    if (this.n >= DMAX) return;
    const i = this.n++;
    this.p[i * 3] = x; this.p[i * 3 + 1] = y; this.p[i * 3 + 2] = z;
    this.v[i * 3] = vx + (Math.random() - 0.5) * 0.8; this.v[i * 3 + 1] = 0.3 + Math.random() * 0.5; this.v[i * 3 + 2] = vz + (Math.random() - 0.5) * 0.8;
    this.max[i] = this.life[i] = 1.8 + Math.random() * 2.2;
    this.sz[i] = size * (0.6 + Math.random() * 0.8);
    const c = SOIL_COL[Math.min(3, Math.round(type))];
    const k = 1.9;
    this.col[i * 3] = Math.min(1, c.r * k + 0.25); this.col[i * 3 + 1] = Math.min(1, c.g * k + 0.22); this.col[i * 3 + 2] = Math.min(1, c.b * k + 0.2);
  }
  /** Exhaust smoke uses dust puffs too. */
  smoke(x, y, z, amount) {
    if (this.n >= DMAX) return;
    const i = this.n++;
    this.p[i * 3] = x; this.p[i * 3 + 1] = y; this.p[i * 3 + 2] = z;
    this.v[i * 3] = (Math.random() - 0.5) * 0.3 + 0.4; this.v[i * 3 + 1] = 1.5 + Math.random(); this.v[i * 3 + 2] = (Math.random() - 0.5) * 0.3;
    this.max[i] = this.life[i] = 1.2 + amount;
    this.sz[i] = 0.25 + amount * 0.5;
    const g = 0.12 + (1 - amount) * 0.35;
    this.col[i * 3] = g; this.col[i * 3 + 1] = g; this.col[i * 3 + 2] = g;
  }
  update(dt) {
    for (let i = 0; i < this.n; i++) {
      this.life[i] -= dt;
      if (this.life[i] <= 0) {
        const l = --this.n;
        if (i !== l) {
          for (let c = 0; c < 3; c++) { this.p[i * 3 + c] = this.p[l * 3 + c]; this.v[i * 3 + c] = this.v[l * 3 + c]; this.col[i * 3 + c] = this.col[l * 3 + c]; }
          this.life[i] = this.life[l]; this.max[i] = this.max[l]; this.sz[i] = this.sz[l];
        }
        i--; continue;
      }
      const o = i * 3;
      this.v[o] *= 1 - dt * 0.6; this.v[o + 1] *= 1 - dt * 0.9; this.v[o + 2] *= 1 - dt * 0.6;
      this.p[o] += (this.v[o] + 0.25) * dt; this.p[o + 1] += this.v[o + 1] * dt; this.p[o + 2] += (this.v[o + 2] + 0.1) * dt;
      this.sz[i] += dt * 0.9;
    }
    const pa = this.aPos.array, sa = this.aSize.array, aa = this.aAlpha.array, ca = this.aCol.array;
    for (let i = 0; i < this.n; i++) {
      pa[i * 3] = this.p[i * 3]; pa[i * 3 + 1] = this.p[i * 3 + 1]; pa[i * 3 + 2] = this.p[i * 3 + 2];
      ca[i * 3] = this.col[i * 3]; ca[i * 3 + 1] = this.col[i * 3 + 1]; ca[i * 3 + 2] = this.col[i * 3 + 2];
      sa[i] = this.sz[i];
      const t = this.life[i] / this.max[i];
      aa[i] = Math.min(1, (1 - t) * 6) * t * 0.35;
    }
    this.geo.setDrawRange(0, this.n);
    this.aPos.needsUpdate = this.aSize.needsUpdate = this.aAlpha.needsUpdate = this.aCol.needsUpdate = true;
  }
}
export { SOIL };
