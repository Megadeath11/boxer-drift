// Shared materials and geometry helpers for building the machines.
import * as THREE from 'three';
import { RoundedBoxGeometry } from '../vendor/RoundedBoxGeometry.js';
import { makeGrimeTexture, makeStripeTexture, makeTreadTexture } from './textures.js';

let M = null;
export function materials() {
  if (M) return M;
  const grime = makeGrimeTexture();
  grime.repeat.set(0.6, 0.6);
  const stripes = makeStripeTexture();
  const tread = makeTreadTexture();
  M = {
    paint: new THREE.MeshPhysicalMaterial({ color: 0xe8a50a, map: grime, roughness: 0.42, metalness: 0.05, clearcoat: 0.5, clearcoatRoughness: 0.35 }),
    paintDirty: new THREE.MeshPhysicalMaterial({ color: 0xb38a2e, map: grime, roughness: 0.75, metalness: 0.05 }),
    truckPaint: new THREE.MeshPhysicalMaterial({ color: 0xc41a14, map: grime, roughness: 0.3, metalness: 0.15, clearcoat: 0.9, clearcoatRoughness: 0.12 }),
    bodyPaint: new THREE.MeshPhysicalMaterial({ color: 0x8a8d90, map: grime, roughness: 0.55, metalness: 0.4, clearcoat: 0.2 }),
    black: new THREE.MeshStandardMaterial({ color: 0x1b1b1b, roughness: 0.6, metalness: 0.2 }),
    darkSteel: new THREE.MeshStandardMaterial({ color: 0x2b2a28, roughness: 0.55, metalness: 0.65, map: grime }),
    steel: new THREE.MeshStandardMaterial({ color: 0x75726d, roughness: 0.38, metalness: 0.85, map: grime }),
    wornSteel: new THREE.MeshStandardMaterial({ color: 0x8d8780, roughness: 0.32, metalness: 0.9 }),
    chrome: new THREE.MeshStandardMaterial({ color: 0xdedede, roughness: 0.08, metalness: 1.0 }),
    rubber: new THREE.MeshStandardMaterial({ color: 0x151515, roughness: 0.92, metalness: 0 }),
    tyre: new THREE.MeshStandardMaterial({ color: 0x161616, roughness: 0.9, metalness: 0, bumpMap: tread, bumpScale: 3 }),
    hose: new THREE.MeshStandardMaterial({ color: 0x0e0e0e, roughness: 0.7 }),
    glass: new THREE.MeshPhysicalMaterial({ color: 0x223038, roughness: 0.05, metalness: 0.1, transmission: 0, transparent: true, opacity: 0.38, side: THREE.DoubleSide, envMapIntensity: 1.6, depthWrite: false }),
    seat: new THREE.MeshStandardMaterial({ color: 0x202020, roughness: 0.85 }),
    interior: new THREE.MeshStandardMaterial({ color: 0x3a3a38, roughness: 0.8 }),
    stripes: new THREE.MeshStandardMaterial({ map: stripes, roughness: 0.5 }),
    lightLens: new THREE.MeshStandardMaterial({ color: 0xfff6d8, emissive: 0xfff2c0, emissiveIntensity: 0.4, roughness: 0.1 }),
    redLens: new THREE.MeshStandardMaterial({ color: 0x8a0000, emissive: 0xff1100, emissiveIntensity: 0.25, roughness: 0.2 }),
    amberLens: new THREE.MeshStandardMaterial({ color: 0xff9900, emissive: 0xff7700, emissiveIntensity: 0.5, roughness: 0.2 }),
    beacon: new THREE.MeshStandardMaterial({ color: 0xff8800, emissive: 0xff6600, emissiveIntensity: 1.0, roughness: 0.2, transparent: true, opacity: 0.85 }),
    white: new THREE.MeshStandardMaterial({ color: 0xe8e8e4, roughness: 0.5 }),
    rust: new THREE.MeshStandardMaterial({ color: 0x5a3a22, roughness: 0.9, metalness: 0.3 }),
    dirt: new THREE.MeshStandardMaterial({ color: 0x5a3e26, roughness: 1 }),
  };
  return M;
}

export function shadowAll(obj) {
  obj.traverse((o) => { if (o.isMesh) { o.castShadow = true; o.receiveShadow = true; } });
  return obj;
}

/** Rounded box, positioned by centre. */
export function rbox(w, h, l, r, mat, x = 0, y = 0, z = 0, parent) {
  const g = new RoundedBoxGeometry(w, h, l, 2, Math.min(r, w / 2 - 1e-3, h / 2 - 1e-3, l / 2 - 1e-3));
  const m = new THREE.Mesh(g, mat);
  m.position.set(x, y, z);
  if (parent) parent.add(m);
  return m;
}
export function box(w, h, l, mat, x = 0, y = 0, z = 0, parent) {
  const m = new THREE.Mesh(new THREE.BoxGeometry(w, h, l), mat);
  m.position.set(x, y, z);
  if (parent) parent.add(m);
  return m;
}
/** Cylinder whose axis lies along `axis` ('x' | 'y' | 'z'). */
export function cyl(r, len, mat, axis = 'x', x = 0, y = 0, z = 0, parent, seg = 20, r2) {
  const g = new THREE.CylinderGeometry(r, r2 ?? r, len, seg);
  if (axis === 'x') g.rotateZ(Math.PI / 2);
  if (axis === 'z') g.rotateX(Math.PI / 2);
  const m = new THREE.Mesh(g, mat);
  m.position.set(x, y, z);
  if (parent) parent.add(m);
  return m;
}

/**
 * Side-profile extrusion: points are [forward, up] pairs in the machine's side
 * plane. Returned geometry has forward = +z, up = +y and is centred on x.
 */
export function profile(points, width, bevel = 0.02, curveSeg = 8) {
  const s = new THREE.Shape(points.map(([a, b]) => new THREE.Vector2(a, b)));
  const depth = Math.max(0.001, width - 2 * bevel);
  const g = new THREE.ExtrudeGeometry(s, { depth, bevelEnabled: bevel > 0, bevelThickness: bevel, bevelSize: bevel, bevelSegments: 2, curveSegments: curveSeg });
  g.translate(0, 0, -depth / 2);
  g.rotateY(-Math.PI / 2);
  g.computeVertexNormals();
  return g;
}

/** Plan-view extrusion: points are [x, z]; extruded upward by height from y=0. */
export function plan(points, height, bevel = 0.02) {
  const s = new THREE.Shape(points.map(([a, b]) => new THREE.Vector2(a, b)));
  const depth = Math.max(0.001, height - 2 * bevel);
  const g = new THREE.ExtrudeGeometry(s, { depth, bevelEnabled: bevel > 0, bevelThickness: bevel, bevelSize: bevel, bevelSegments: 2, curveSegments: 12 });
  g.rotateX(Math.PI / 2); // shape y -> z, extrusion -> -y
  g.translate(0, depth + bevel, 0);
  g.computeVertexNormals();
  return g;
}

/** Thin curved plate from a polyline (side view), extruded across width. */
export function plate(points, thickness, width) {
  // offset polyline to build closed shape
  const out = [], inn = [];
  for (let i = 0; i < points.length; i++) {
    const p = points[i];
    const a = points[Math.max(0, i - 1)], b = points[Math.min(points.length - 1, i + 1)];
    let tx = b[0] - a[0], ty = b[1] - a[1];
    const L = Math.hypot(tx, ty) || 1; tx /= L; ty /= L;
    const nx = -ty, ny = tx;
    out.push([p[0] + nx * thickness / 2, p[1] + ny * thickness / 2]);
    inn.push([p[0] - nx * thickness / 2, p[1] - ny * thickness / 2]);
  }
  return profile(out.concat(inn.reverse()), width, 0.005, 2);
}

const _A = new THREE.Vector3(), _B = new THREE.Vector3(), _D = new THREE.Vector3(), _Y = new THREE.Vector3(0, 1, 0);

/** Telescoping hydraulic cylinder between two marker objects. */
export class HydCylinder {
  constructor(parent, a, b, { rBarrel = 0.09, rRod = 0.05, barrel = 1.2, mat, rodMat } = {}) {
    const m = materials();
    this.parent = parent; this.a = a; this.b = b;
    this.group = new THREE.Group();
    parent.add(this.group);
    this.barrelLen = barrel;
    const bg = new THREE.CylinderGeometry(rBarrel, rBarrel, 1, 18); bg.translate(0, 0.5, 0);
    const rg = new THREE.CylinderGeometry(rRod, rRod, 1, 14); rg.translate(0, -0.5, 0);
    this.barrel = new THREE.Mesh(bg, mat || m.paint);
    this.rod = new THREE.Mesh(rg, rodMat || m.chrome);
    const eyeG = new THREE.CylinderGeometry(rBarrel * 1.05, rBarrel * 1.05, rBarrel * 2.2, 16); eyeG.rotateZ(Math.PI / 2);
    this.eyeA = new THREE.Mesh(eyeG, m.darkSteel);
    this.eyeB = new THREE.Mesh(eyeG, m.darkSteel);
    const gland = new THREE.Mesh(new THREE.CylinderGeometry(rBarrel * 1.12, rBarrel * 1.12, 0.08, 18), m.darkSteel);
    gland.position.y = 1;
    this.gland = gland;
    this.barrel.add(gland);
    this.group.add(this.barrel, this.rod, this.eyeA, this.eyeB);
    shadowAll(this.group);
  }
  update() {
    this.a.getWorldPosition(_A); this.b.getWorldPosition(_B);
    this.parent.worldToLocal(_A); this.parent.worldToLocal(_B);
    _D.subVectors(_B, _A);
    const L = _D.length();
    _D.divideScalar(L || 1);
    const q = new THREE.Quaternion().setFromUnitVectors(_Y, _D);
    const bl = Math.min(this.barrelLen, L - 0.1);
    this.barrel.position.copy(_A); this.barrel.quaternion.copy(q); this.barrel.scale.set(1, bl, 1);
    this.gland.scale.set(1, 1 / bl, 1);
    this.rod.position.copy(_B); this.rod.quaternion.copy(q); this.rod.scale.set(1, Math.max(0.05, L - 0.15), 1);
    this.eyeA.position.copy(_A); this.eyeB.position.copy(_B);
    // eyes align with pin axis (local x of parent) - keep default orientation
    this.length = L;
  }
}

/** Beam (box) stretched between two markers, e.g. dozer push arms, links. */
export class Strut {
  constructor(parent, a, b, w, h, mat) {
    this.parent = parent; this.a = a; this.b = b;
    const g = new THREE.BoxGeometry(w, h, 1); g.translate(0, 0, 0.5);
    this.mesh = new THREE.Mesh(g, mat);
    this.mesh.castShadow = this.mesh.receiveShadow = true;
    parent.add(this.mesh);
  }
  update() {
    this.a.getWorldPosition(_A); this.b.getWorldPosition(_B);
    this.parent.worldToLocal(_A); this.parent.worldToLocal(_B);
    this.mesh.position.copy(_A);
    this.mesh.lookAt(this.parent.localToWorld(_B.clone()));
    this.mesh.scale.z = _A.distanceTo(_B);
  }
}

export function marker(parent, x, y, z) {
  const o = new THREE.Object3D();
  o.position.set(x, y, z);
  parent.add(o);
  return o;
}

// ---------- crawler tracks ----------
/**
 * Build the closed belt path around a set of circles given in CCW order in the
 * (z, y) side plane. Returns {sample(s) -> {p:[z,y], t:[dz,dy]}, length}.
 */
export function beltPath(circles) {
  const n = circles.length;
  const segs = [];
  // tangent normals for each pair i -> i+1
  const normals = [];
  for (let i = 0; i < n; i++) {
    const A = circles[i], B = circles[(i + 1) % n];
    const dx = B.z - A.z, dy = B.y - A.y;
    const D = Math.hypot(dx, dy);
    const ux = dx / D, uy = dy / D;
    const rx = uy, ry = -ux; // right of travel (outward for CCW)
    const sb = (A.r - B.r) / D, cb = Math.sqrt(1 - sb * sb);
    normals.push([rx * cb + ux * sb, ry * cb + uy * sb]);
  }
  for (let i = 0; i < n; i++) {
    const A = circles[i], B = circles[(i + 1) % n];
    const nn = normals[i];
    const p0 = [A.z + nn[0] * A.r, A.y + nn[1] * A.r];
    const p1 = [B.z + nn[0] * B.r, B.y + nn[1] * B.r];
    segs.push({ type: 'line', p0, p1, len: Math.hypot(p1[0] - p0[0], p1[1] - p0[1]) });
    // arc around B from normals[i] to normals[i+1]
    const n2 = normals[(i + 1) % n];
    let a0 = Math.atan2(nn[1], nn[0]), a1 = Math.atan2(n2[1], n2[0]);
    while (a1 < a0) a1 += Math.PI * 2;
    segs.push({ type: 'arc', c: B, a0, a1, len: (a1 - a0) * B.r });
  }
  const length = segs.reduce((s, g) => s + g.len, 0);
  function sample(s) {
    s = ((s % length) + length) % length;
    for (const g of segs) {
      if (s <= g.len) {
        if (g.type === 'line') {
          const t = s / g.len;
          const dz = (g.p1[0] - g.p0[0]) / g.len, dy = (g.p1[1] - g.p0[1]) / g.len;
          return { z: g.p0[0] + (g.p1[0] - g.p0[0]) * t, y: g.p0[1] + (g.p1[1] - g.p0[1]) * t, tz: dz, ty: dy };
        }
        const a = g.a0 + s / g.c.r;
        return { z: g.c.z + Math.cos(a) * g.c.r, y: g.c.y + Math.sin(a) * g.c.r, tz: -Math.sin(a), ty: Math.cos(a) };
      }
      s -= g.len;
    }
    return { z: 0, y: 0, tz: 1, ty: 0 };
  }
  return { sample, length };
}

/** Instanced track shoes + links running around a belt path. */
export class TrackBelt {
  constructor(parent, circles, { x, shoeWidth, pitch = 0.19, thick = 0.03, grouser = 0.035 }) {
    const m = materials();
    this.path = beltPath(circles);
    this.count = Math.round(this.path.length / pitch);
    this.pitch = this.path.length / this.count;
    this.x = x;
    this.offset = 0;
    // shoe: plate + grouser bar + two link bars
    const shoeG = new THREE.BoxGeometry(shoeWidth, thick, this.pitch * 0.94);
    shoeG.translate(0, -thick / 2, 0);
    const grG = new THREE.BoxGeometry(shoeWidth, grouser, 0.03);
    grG.translate(0, -thick - grouser / 2, this.pitch * 0.15);
    const linkG = new THREE.BoxGeometry(0.06, 0.09, this.pitch * 1.02);
    const l1 = linkG.clone().translate(0.16, 0.045, 0), l2 = linkG.clone().translate(-0.16, 0.045, 0);
    this.shoes = new THREE.InstancedMesh(mergeGeos([shoeG, grG]), m.wornSteel, this.count);
    this.links = new THREE.InstancedMesh(mergeGeos([l1, l2]), m.darkSteel, this.count);
    this.shoes.castShadow = this.shoes.receiveShadow = true;
    this.links.castShadow = true;
    this.shoes.frustumCulled = this.links.frustumCulled = false;
    parent.add(this.shoes, this.links);
    this._m = new THREE.Matrix4(); this._q = new THREE.Quaternion(); this._e = new THREE.Euler();
    this.update(0);
  }
  update(dOffset) {
    this.offset += dOffset;
    const pos = new THREE.Vector3(), sc = new THREE.Vector3(1, 1, 1);
    for (let i = 0; i < this.count; i++) {
      const s = this.path.sample(i * this.pitch + this.offset);
      // outward normal = right of travel = (tz, ty) rotated -90 => (ty, -tz)
      const ang = Math.atan2(-s.ty, s.tz); // rotation about x: forward z towards tangent
      this._e.set(ang, 0, 0);
      this._q.setFromEuler(this._e);
      pos.set(this.x, s.y, s.z);
      this._m.compose(pos, this._q, sc);
      this.shoes.setMatrixAt(i, this._m);
      this.links.setMatrixAt(i, this._m);
    }
    this.shoes.instanceMatrix.needsUpdate = true;
    this.links.instanceMatrix.needsUpdate = true;
  }
}

export function mergeGeos(list) {
  // minimal non-indexed merge of position/normal/uv
  const geos = list.map((g) => (g.index ? g.toNonIndexed() : g));
  let n = 0;
  for (const g of geos) n += g.attributes.position.count;
  const pos = new Float32Array(n * 3), nor = new Float32Array(n * 3), uv = new Float32Array(n * 2);
  let o = 0;
  for (const g of geos) {
    pos.set(g.attributes.position.array, o * 3);
    nor.set(g.attributes.normal.array, o * 3);
    if (g.attributes.uv) uv.set(g.attributes.uv.array, o * 2);
    o += g.attributes.position.count;
  }
  const out = new THREE.BufferGeometry();
  out.setAttribute('position', new THREE.BufferAttribute(pos, 3));
  out.setAttribute('normal', new THREE.BufferAttribute(nor, 3));
  out.setAttribute('uv', new THREE.BufferAttribute(uv, 2));
  return out;
}

/** Sprocket with teeth (axis along x). */
export function sprocket(r, width, teeth, mat) {
  const g = new THREE.Group();
  cyl(r * 0.86, width, mat, 'x', 0, 0, 0, g, 28);
  const tg = new THREE.BoxGeometry(width * 0.9, r * 0.22, r * 0.22);
  const inst = new THREE.InstancedMesh(tg, mat, teeth);
  const m = new THREE.Matrix4();
  for (let i = 0; i < teeth; i++) {
    const a = (i / teeth) * Math.PI * 2;
    m.makeRotationX(a).setPosition(0, Math.sin(a) * r * 0.92, Math.cos(a) * r * 0.92);
    inst.setMatrixAt(i, m);
  }
  g.add(inst);
  const m2 = materials();
  cyl(r * 0.45, width * 1.3, m2.darkSteel, 'x', 0, 0, 0, g, 20);
  // bolt circle
  const bolts = new THREE.InstancedMesh(new THREE.CylinderGeometry(0.02, 0.02, width * 1.45, 6).rotateZ(Math.PI / 2), m2.steel, 10);
  for (let i = 0; i < 10; i++) {
    const a = (i / 10) * Math.PI * 2;
    m.makeTranslation(0, Math.sin(a) * r * 0.33, Math.cos(a) * r * 0.33);
    bolts.setMatrixAt(i, m);
  }
  g.add(bolts);
  return shadowAll(g);
}

/** Idler / roller wheel with flange (axis x). */
export function idler(r, width, mat) {
  const g = new THREE.Group();
  const pts = [];
  const hw = width / 2;
  pts.push(new THREE.Vector2(0.001, -hw), new THREE.Vector2(r * 0.98, -hw), new THREE.Vector2(r, -hw * 0.7),
    new THREE.Vector2(r * 0.93, -hw * 0.5), new THREE.Vector2(r * 0.93, hw * 0.5), new THREE.Vector2(r, hw * 0.7),
    new THREE.Vector2(r * 0.98, hw), new THREE.Vector2(0.001, hw));
  const lg = new THREE.LatheGeometry(pts, 28);
  lg.rotateZ(Math.PI / 2);
  g.add(new THREE.Mesh(lg, mat));
  const m2 = materials();
  cyl(r * 0.35, width * 1.15, m2.darkSteel, 'x', 0, 0, 0, g, 16);
  return shadowAll(g);
}
