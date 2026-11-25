// Procedurally generated textures (no external image assets needed).
import * as THREE from 'three';
import { fbm, vnoise, rng, clamp } from './util.js';

function worleyP(u, v, cells, seed) {
  // periodic worley distance (F1) in cell units
  const x = u * cells, y = v * cells;
  const xi = Math.floor(x), yi = Math.floor(y);
  let best = 9;
  for (let j = -1; j <= 1; j++) for (let i = -1; i <= 1; i++) {
    const cx = xi + i, cy = yi + j;
    const wx = ((cx % cells) + cells) % cells, wy = ((cy % cells) + cells) % cells;
    const px = cx + (vnoise(wx * 7.13, wy * 3.71, seed) * 0.5 + 0.5);
    const py = cy + (vnoise(wx * 5.37, wy * 9.11, seed + 3) * 0.5 + 0.5);
    const d = Math.hypot(px - x, py - y);
    if (d < best) best = d;
  }
  return best;
}

/**
 * Soil detail texture. Channels:
 *  R: fine grain fbm, G: pebbles/clods (worley), B: medium fbm, A: very fine speckle
 */
export function makeDetailTexture(size = 512) {
  const data = new Uint8Array(size * size * 4);
  for (let y = 0; y < size; y++) {
    for (let x = 0; x < size; x++) {
      const u = x / size, v = y / size;
      const r = fbm(u * 16, v * 16, 5, 1, 16) * 0.5 + 0.5;
      const w = worleyP(u, v, 22, 7);
      const w2 = worleyP(u, v, 57, 13);
      const g = clamp(1 - w * 1.6, 0, 1) * 0.65 + clamp(1 - w2 * 1.8, 0, 1) * 0.35;
      const b = fbm(u * 4, v * 4, 4, 5, 4) * 0.5 + 0.5;
      const a = vnoise(u * 180, v * 180, 9, 180) * 0.5 + 0.5;
      const o = (y * size + x) * 4;
      data[o] = r * 255; data[o + 1] = g * 255; data[o + 2] = b * 255; data[o + 3] = a * 255;
    }
  }
  const tex = new THREE.DataTexture(data, size, size, THREE.RGBAFormat);
  tex.wrapS = tex.wrapT = THREE.RepeatWrapping;
  tex.magFilter = THREE.LinearFilter;
  tex.minFilter = THREE.LinearMipmapLinearFilter;
  tex.generateMipmaps = true;
  tex.anisotropy = 8;
  tex.needsUpdate = true;
  return tex;
}

/** Subtle paint wear / grime colour map for machine bodies (multiplies base colour). */
export function makeGrimeTexture(size = 256) {
  const c = document.createElement('canvas');
  c.width = c.height = size;
  const g = c.getContext('2d');
  const img = g.createImageData(size, size);
  for (let y = 0; y < size; y++) for (let x = 0; x < size; x++) {
    const u = x / size, v = y / size;
    const n = fbm(u * 6, v * 6, 5, 31, 6) * 0.5 + 0.5;
    const s = vnoise(u * 64, v * 64, 4, 64) * 0.5 + 0.5;
    const streak = fbm(u * 24, v * 2, 3, 77, 0) * 0.5 + 0.5;
    let k = 0.86 + 0.14 * n - 0.06 * Math.max(0, s - 0.8) * 5 - 0.07 * Math.max(0, streak - 0.62) * 4;
    k = clamp(k, 0.55, 1);
    const o = (y * size + x) * 4;
    img.data[o] = k * 255; img.data[o + 1] = k * 250; img.data[o + 2] = k * 240; img.data[o + 3] = 255;
  }
  g.putImageData(img, 0, 0);
  const t = new THREE.CanvasTexture(c);
  t.wrapS = t.wrapT = THREE.RepeatWrapping;
  t.colorSpace = THREE.SRGBColorSpace;
  t.anisotropy = 4;
  return t;
}

/** Diagonal hazard stripes (yellow/black). */
export function makeStripeTexture() {
  const c = document.createElement('canvas');
  c.width = 128; c.height = 32;
  const g = c.getContext('2d');
  g.fillStyle = '#f2c200'; g.fillRect(0, 0, 128, 32);
  g.fillStyle = '#151515';
  for (let i = -2; i < 10; i++) {
    g.beginPath();
    g.moveTo(i * 24, 32); g.lineTo(i * 24 + 12, 32); g.lineTo(i * 24 + 44, 0); g.lineTo(i * 24 + 32, 0); g.closePath(); g.fill();
  }
  const t = new THREE.CanvasTexture(c);
  t.wrapS = t.wrapT = THREE.RepeatWrapping;
  t.colorSpace = THREE.SRGBColorSpace;
  return t;
}

/** Chain-link fence alpha pattern. */
export function makeChainLinkTexture() {
  const c = document.createElement('canvas');
  c.width = c.height = 64;
  const g = c.getContext('2d');
  g.clearRect(0, 0, 64, 64);
  g.strokeStyle = 'rgba(200,205,210,1)';
  g.lineWidth = 2.2;
  g.beginPath();
  g.moveTo(0, 32); g.lineTo(32, 0); g.lineTo(64, 32); g.lineTo(32, 64); g.closePath();
  g.moveTo(-32, 32); g.lineTo(0, 0); g.moveTo(64, 0); g.lineTo(96, 32);
  g.stroke();
  const t = new THREE.CanvasTexture(c);
  t.wrapS = t.wrapT = THREE.RepeatWrapping;
  t.anisotropy = 4;
  return t;
}

/** Soft round sprite for dust. */
export function makeDustSprite() {
  const c = document.createElement('canvas');
  c.width = c.height = 64;
  const g = c.getContext('2d');
  const grd = g.createRadialGradient(32, 32, 0, 32, 32, 32);
  grd.addColorStop(0, 'rgba(255,255,255,0.9)');
  grd.addColorStop(0.4, 'rgba(255,255,255,0.45)');
  grd.addColorStop(1, 'rgba(255,255,255,0)');
  g.fillStyle = grd; g.fillRect(0, 0, 64, 64);
  const t = new THREE.CanvasTexture(c);
  return t;
}

/** Simple text decal, e.g. model numbers on machines. */
export function makeTextTexture(text, { w = 256, h = 64, color = '#111', bg = null, font = 'bold 44px Arial' } = {}) {
  const c = document.createElement('canvas');
  c.width = w; c.height = h;
  const g = c.getContext('2d');
  if (bg) { g.fillStyle = bg; g.fillRect(0, 0, w, h); }
  g.fillStyle = color; g.font = font; g.textAlign = 'center'; g.textBaseline = 'middle';
  g.fillText(text, w / 2, h / 2 + 2);
  const t = new THREE.CanvasTexture(c);
  t.colorSpace = THREE.SRGBColorSpace;
  t.anisotropy = 4;
  return t;
}

/** Tread pattern bump for tyres (grey-scale). */
export function makeTreadTexture() {
  const c = document.createElement('canvas');
  c.width = 64; c.height = 256;
  const g = c.getContext('2d');
  g.fillStyle = '#000'; g.fillRect(0, 0, 64, 256);
  g.fillStyle = '#fff';
  for (let i = 0; i < 16; i++) {
    const y = i * 16;
    g.save(); g.translate(16, y + 8); g.rotate(0.5); g.fillRect(-14, -4, 28, 8); g.restore();
    g.save(); g.translate(48, y + 16); g.rotate(-0.5); g.fillRect(-14, -4, 28, 8); g.restore();
  }
  const t = new THREE.CanvasTexture(c);
  t.wrapS = t.wrapT = THREE.RepeatWrapping;
  return t;
}

export function makeRandom(seed) { return rng(seed); }
