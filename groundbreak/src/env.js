// Sky, sun, image-based lighting, surrounding countryside and site furniture.
import * as THREE from 'three';
import { Sky } from '../vendor/Sky.js';
import { baseHeight, makeSoilMaterial } from './terrain.js';
import { makeChainLinkTexture, makeTextTexture } from './textures.js';
import { materials, box, cyl, rbox, shadowAll } from './parts.js';
import { rng } from './util.js';
import { SITE } from './config.js';

export function setupEnvironment(renderer, scene, detailTex) {
  // --- sky + sun ---
  const sky = new Sky();
  sky.scale.setScalar(4000);
  const u = sky.material.uniforms;
  u.turbidity.value = 5.5; u.rayleigh.value = 1.3; u.mieCoefficient.value = 0.004; u.mieDirectionalG.value = 0.82;
  const sunDir = new THREE.Vector3().setFromSphericalCoords(1, THREE.MathUtils.degToRad(90 - 36), THREE.MathUtils.degToRad(215));
  u.sunPosition.value.copy(sunDir);
  scene.add(sky);
  // environment map from the sky for reflections / ambient
  const pm = new THREE.PMREMGenerator(renderer);
  const skyScene = new THREE.Scene();
  const sky2 = new Sky(); sky2.scale.setScalar(4000);
  Object.assign(sky2.material.uniforms.sunPosition.value, sunDir);
  for (const k of ['turbidity', 'rayleigh', 'mieCoefficient', 'mieDirectionalG']) sky2.material.uniforms[k].value = u[k].value;
  skyScene.add(sky2);
  const ground = new THREE.Mesh(new THREE.CircleGeometry(3000, 16).rotateX(-Math.PI / 2), new THREE.MeshBasicMaterial({ color: 0x3d4a2a }));
  ground.position.y = -2; skyScene.add(ground);
  const envRT = pm.fromScene(skyScene, 0.02);
  scene.environment = envRT.texture;
  scene.environmentIntensity = 0.75;

  const fogColor = new THREE.Color(0xb9c6d2);
  scene.fog = new THREE.Fog(fogColor, 180, 1400);

  const sun = new THREE.DirectionalLight(0xfff1dc, 3.2);
  sun.position.copy(sunDir).multiplyScalar(120);
  sun.castShadow = true;
  sun.shadow.mapSize.set(4096, 4096);
  const sc = sun.shadow.camera;
  sc.left = -42; sc.right = 42; sc.top = 42; sc.bottom = -42; sc.near = 10; sc.far = 320;
  sun.shadow.bias = -0.0004;
  sun.shadow.normalBias = 0.03;
  scene.add(sun, sun.target);
  const hemi = new THREE.HemisphereLight(0xc4d6ff, 0x5c4a32, 0.35);
  scene.add(hemi);

  // --- surrounding countryside (exactly matches the dig grid at its border) ---
  const half = SITE.size / 2;
  const xs = new Set();
  for (let x = -1500; x <= 1500; x += 30) xs.add(x);
  for (let x = -300; x <= 300; x += 6) xs.add(x);
  for (let x = -90; x <= 90; x += 2) xs.add(x);
  xs.add(-half); xs.add(half);
  const coords = [...xs].sort((a, b) => a - b);
  const n = coords.length;
  const pos = new Float32Array(n * n * 3);
  for (let j = 0; j < n; j++) for (let i = 0; i < n; i++) {
    const x = coords[i], z = coords[j];
    const inside = Math.abs(x) < half - 0.01 && Math.abs(z) < half - 0.01;
    const k = (j * n + i) * 3;
    pos[k] = x; pos[k + 1] = inside ? baseHeight(x, z) - 3 : baseHeight(x, z); pos[k + 2] = z;
  }
  const idx = [];
  for (let j = 0; j < n - 1; j++) for (let i = 0; i < n - 1; i++) {
    const a = j * n + i, b = a + 1, c = a + n, d = c + 1;
    // skip cells fully inside the dig grid
    const cx = (coords[i] + coords[i + 1]) / 2, cz = (coords[j] + coords[j + 1]) / 2;
    if (Math.abs(cx) < half && Math.abs(cz) < half) continue;
    idx.push(a, c, b, b, c, d);
  }
  const g = new THREE.BufferGeometry();
  g.setAttribute('position', new THREE.BufferAttribute(pos, 3));
  g.setAttribute('soil', new THREE.BufferAttribute(new Float32Array(n * n * 4), 4));
  g.setAttribute('soil2', new THREE.BufferAttribute(new Float32Array(n * n * 4), 4));
  g.setIndex(idx);
  g.computeVertexNormals();
  const outer = new THREE.Mesh(g, makeSoilMaterial(detailTex));
  outer.receiveShadow = true;
  scene.add(outer);

  addTrees(scene);
  addFence(scene);
  addProps(scene);

  return { sun, sky, hemi };
}

function addTrees(scene) {
  const R = rng(1234);
  const N = 520;
  const trunkG = new THREE.CylinderGeometry(0.12, 0.22, 1, 6); trunkG.translate(0, 0.5, 0);
  const coneG = new THREE.ConeGeometry(1, 1, 8); coneG.translate(0, 0.5, 0);
  const blobG = new THREE.IcosahedronGeometry(1, 1);
  const trunks = new THREE.InstancedMesh(trunkG, new THREE.MeshStandardMaterial({ color: 0x4a3626, roughness: 1 }), N);
  const cones = new THREE.InstancedMesh(coneG, new THREE.MeshStandardMaterial({ color: 0x24401f, roughness: 0.95 }), N * 3);
  const blobs = new THREE.InstancedMesh(blobG, new THREE.MeshStandardMaterial({ color: 0x3d5a24, roughness: 0.95, flatShading: true }), N * 2);
  const m = new THREE.Matrix4(), q = new THREE.Quaternion(), s = new THREE.Vector3(), p = new THREE.Vector3();
  let ci = 0, bi = 0, ti = 0;
  const col = new THREE.Color();
  for (let i = 0; i < N; i++) {
    const a = R() * Math.PI * 2;
    const r = 62 + Math.pow(R(), 0.7) * 420;
    const x = Math.cos(a) * r, z = Math.sin(a) * r;
    if (Math.abs(z + 2) < 9 && x > 40) continue; // access road
    const y = baseHeight(x, z) - 0.2;
    const h = 7 + R() * 11;
    q.setFromAxisAngle(new THREE.Vector3(0, 1, 0), R() * 6.28);
    if (R() < 0.6) {
      m.compose(p.set(x, y, z), q, s.set(1, h * 0.35, 1)); trunks.setMatrixAt(ti++, m);
      for (let k = 0; k < 3; k++) {
        const w = (2.6 - k * 0.65) * (0.8 + R() * 0.4);
        m.compose(p.set(x, y + h * (0.22 + k * 0.24), z), q, s.set(w, h * 0.42, w));
        cones.setMatrixAt(ci, m); cones.setColorAt(ci++, col.setHSL(0.3 + R() * 0.05, 0.4, 0.17 + R() * 0.07));
      }
    } else {
      m.compose(p.set(x, y, z), q, s.set(1.3, h * 0.45, 1.3)); trunks.setMatrixAt(ti++, m);
      for (let k = 0; k < 2; k++) {
        const w = 2.8 + R() * 2;
        m.compose(p.set(x + (R() - 0.5) * 2, y + h * (0.55 + k * 0.2), z + (R() - 0.5) * 2), q, s.set(w, w * 0.8, w));
        blobs.setMatrixAt(bi, m); blobs.setColorAt(bi++, col.setHSL(0.2 + R() * 0.08, 0.45, 0.2 + R() * 0.1));
      }
    }
  }
  trunks.count = ti; cones.count = ci; blobs.count = bi;
  for (const o of [trunks, cones, blobs]) { o.castShadow = true; o.receiveShadow = true; scene.add(o); }
}

function addFence(scene) {
  const tex = makeChainLinkTexture();
  const L = 45.5;
  const mat = new THREE.MeshStandardMaterial({ map: tex, alphaMap: tex, transparent: false, alphaTest: 0.4, side: THREE.DoubleSide, metalness: 0.6, roughness: 0.5, color: 0xb0b4b8 });
  const posts = [];
  const sides = [
    [[-L, -L], [L, -L]], [[L, -L], [L, -6]], [[L, 6], [L, L]], [[L, L], [-L, L]], [[-L, L], [-L, -L]],
  ];
  const postMat = materials().steel;
  for (const [[x0, z0], [x1, z1]] of sides) {
    const len = Math.hypot(x1 - x0, z1 - z0);
    const segs = Math.round(len / 3);
    for (let i = 0; i < segs; i++) {
      const ax = x0 + (x1 - x0) * i / segs, az = z0 + (z1 - z0) * i / segs;
      const bx = x0 + (x1 - x0) * (i + 1) / segs, bz = z0 + (z1 - z0) * (i + 1) / segs;
      const ya = baseHeight(ax, az), yb = baseHeight(bx, bz);
      const sl = Math.hypot(bx - ax, bz - az);
      const pg = new THREE.PlaneGeometry(sl, 1.8);
      const pa = pg.attributes.position;
      for (let k = 0; k < pa.count; k++) {
        const t = (pa.getX(k) + sl / 2) / sl;
        pa.setY(k, pa.getY(k) + 0.95 + ya + (yb - ya) * t);
      }
      const uv = pg.attributes.uv;
      for (let k = 0; k < uv.count; k++) uv.setXY(k, uv.getX(k) * sl * 6, uv.getY(k) * 1.8 * 6);
      const pm = new THREE.Mesh(pg, mat);
      pm.position.set((ax + bx) / 2, 0, (az + bz) / 2);
      pm.rotation.y = -Math.atan2(bz - az, bx - ax);
      pm.castShadow = true;
      scene.add(pm);
      posts.push([ax, ya, az]);
    }
    posts.push([x1, baseHeight(x1, z1), z1]);
  }
  const pi = new THREE.InstancedMesh(new THREE.CylinderGeometry(0.03, 0.03, 2.0, 8).translate(0, 1.0, 0), postMat, posts.length);
  const m = new THREE.Matrix4();
  posts.forEach(([x, y, z], i) => { m.makeTranslation(x, y, z); pi.setMatrixAt(i, m); });
  pi.castShadow = true;
  scene.add(pi);
  // access road (gravel) heading east out of the gate
  const road = new THREE.Mesh(new THREE.PlaneGeometry(400, 8, 200, 4).rotateX(-Math.PI / 2), new THREE.MeshStandardMaterial({ color: 0x77706a, roughness: 1 }));
  const rp = road.geometry.attributes.position;
  for (let k = 0; k < rp.count; k++) { const x = rp.getX(k) + 248, z = rp.getZ(k); rp.setY(k, baseHeight(x, z) + 0.06); }
  road.geometry.computeVertexNormals();
  road.position.set(248, 0, 0);
  road.receiveShadow = true;
  scene.add(road);
}

function addProps(scene) {
  const M = materials();
  // site office trailer
  const office = new THREE.Group();
  rbox(10, 2.6, 3.0, 0.05, M.white, 0, 1.3 + 0.6, 0, office);
  box(10, 0.6, 2.6, M.darkSteel, 0, 0.3, 0, office);
  for (let i = 0; i < 4; i++) box(1.2, 0.9, 0.05, M.glass, -3.6 + i * 2.4, 2.2, 1.52, office);
  box(0.9, 2.0, 0.05, M.interior, 3.8, 1.6, 1.53, office);
  const sign = new THREE.Mesh(new THREE.PlaneGeometry(3.6, 0.9), new THREE.MeshStandardMaterial({ map: makeTextTexture('GROUNDBREAK SITE OFFICE', { w: 512, h: 128, bg: '#0f3f73', color: '#fff', font: 'bold 44px Arial' }) }));
  sign.position.set(-1, 3.0, 1.53); office.add(sign);
  for (let i = 0; i < 3; i++) box(1.0, 0.18 * (i + 1), 0.35, M.darkSteel, 3.8, 0.09 * (i + 1), 1.75 + (2 - i) * 0.33, office);
  office.position.set(32, baseHeight(32, 36), 37);
  office.rotation.y = Math.PI;
  scene.add(shadowAll(office));
  // portable toilet
  const pt = new THREE.Group();
  rbox(1.15, 2.3, 1.15, 0.05, new THREE.MeshStandardMaterial({ color: 0x1d5fb0, roughness: 0.6 }), 0, 1.15, 0, pt);
  pt.position.set(24, baseHeight(24, 38), 38.5); scene.add(shadowAll(pt));
  // concrete pipe stack
  const pipeG = new THREE.CylinderGeometry(0.6, 0.6, 2.4, 24, 1, true).rotateZ(Math.PI / 2);
  const pipeMat = new THREE.MeshStandardMaterial({ color: 0x9a9792, roughness: 0.9, side: THREE.DoubleSide });
  const pipes = new THREE.Group();
  [[0, 0.6], [1.25, 0.6], [2.5, 0.6], [0.62, 1.68], [1.87, 1.68]].forEach(([z, y]) => { const p = new THREE.Mesh(pipeG, pipeMat); p.position.set(0, y, z); pipes.add(p); });
  pipes.position.set(-38, baseHeight(-38, 30), 30); scene.add(shadowAll(pipes));
  // safety cones along the pit edge and stakes at offset positions
  const coneMat = new THREE.MeshStandardMaterial({ color: 0xff5a00, roughness: 0.5 });
  const coneG = new THREE.ConeGeometry(0.18, 0.7, 16).translate(0, 0.35, 0);
  const p = SITE.pit;
  const coneAt = (x, z) => { const c = new THREE.Mesh(coneG, coneMat); c.position.set(x, baseHeight(x, z), z); c.castShadow = true; scene.add(c); const b = box(0.4, 0.04, 0.4, M.black, x, baseHeight(x, z) + 0.02, z); scene.add(b); };
  for (let x = p.x0; x <= p.x1; x += 3) coneAt(x, p.z1 + 3.2);
  // offset stakes w/ flagging (2 m outside each corner)
  const stake = (x, z, color, text) => {
    const g = new THREE.Group();
    box(0.05, 1.2, 0.05, new THREE.MeshStandardMaterial({ color: 0xc8a070, roughness: 0.9 }), 0, 0.6, 0, g);
    box(0.06, 0.12, 0.06, new THREE.MeshStandardMaterial({ color, emissive: color, emissiveIntensity: 0.2 }), 0, 1.15, 0, g);
    const ribbon = new THREE.Mesh(new THREE.PlaneGeometry(0.05, 0.35), new THREE.MeshStandardMaterial({ color, side: THREE.DoubleSide }));
    ribbon.position.set(0.03, 1.0, 0.0); ribbon.rotation.z = 0.25; g.add(ribbon);
    if (text) {
      const lab = new THREE.Mesh(new THREE.PlaneGeometry(0.4, 0.2), new THREE.MeshBasicMaterial({ map: makeTextTexture(text, { w: 128, h: 64, color: '#111', bg: '#fff', font: 'bold 24px Arial' }) }));
      lab.position.set(0, 0.75, 0.03); g.add(lab);
    }
    g.position.set(x, baseHeight(x, z), z);
    scene.add(shadowAll(g));
  };
  const o = 2;
  stake(p.x0 - o, p.z0 - o, 0xff5500, 'C2.5'); stake(p.x1 + o, p.z0 - o, 0xff5500, 'C2.5');
  stake(p.x0 - o, p.z1 + o, 0xff5500, 'C2.5'); stake(p.x1 + o, p.z1 + o, 0xff5500, 'C2.5');
  const d = SITE.pad;
  stake(d.x0 - o, d.z0 - o, 0x1e5bff, 'F0.5'); stake(d.x1 + o, d.z0 - o, 0x1e5bff, 'F0.5');
  stake(d.x0 - o, d.z1 + o, 0x1e5bff, 'F0.5'); stake(d.x1 + o, d.z1 + o, 0x1e5bff, 'F0.5');
}
