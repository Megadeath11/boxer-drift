import * as THREE from 'three';
import { Terrain } from './terrain.js';
import { makeDetailTexture } from './textures.js';
import { setupEnvironment } from './env.js';
import { SoilParticles } from './particles.js';
import { Clods } from './clods.js';
import { SoilChunks } from './chunks.js';
import { Excavator } from './excavator.js';
import { Dozer } from './dozer.js';
import { Truck } from './truck.js';
import { collideMachines } from './physics.js';
import { Input, CONTROLS } from './input.js';
import { SoundSystem } from './audio.js';
import { SITE, SOIL } from './config.js';
import { clamp } from './util.js';

const loading = document.getElementById('loading');
const setStatus = (t) => { if (loading) loading.querySelector('.status').textContent = t; };

async function frame() { await new Promise((r) => requestAnimationFrame(() => r())); }

async function init() {
  const canvas = document.getElementById('c');
  const renderer = new THREE.WebGLRenderer({ canvas, antialias: true, powerPreference: 'high-performance' });
  renderer.setPixelRatio(Math.min(devicePixelRatio, 2));
  renderer.setSize(innerWidth, innerHeight);
  renderer.toneMapping = THREE.ACESFilmicToneMapping;
  renderer.toneMappingExposure = 0.95;
  renderer.shadowMap.enabled = true;
  renderer.shadowMap.type = THREE.PCFSoftShadowMap;
  const scene = new THREE.Scene();
  const camera = new THREE.PerspectiveCamera(55, innerWidth / innerHeight, 0.05, 5000);

  setStatus('Generating soil textures…'); await frame();
  const detail = makeDetailTexture(512);
  setStatus('Building terrain & strata…'); await frame();
  const terrain = new Terrain();
  scene.add(terrain.buildMesh(detail));
  setStatus('Lighting & environment…'); await frame();
  const env = setupEnvironment(renderer, scene, detail);
  const particles = new SoilParticles(scene, terrain);
  const clods = new Clods(scene, terrain);
  const focus = new THREE.Vector3();

  setStatus('Assembling machines…'); await frame();
  const ex = new Excavator(scene, terrain, particles);
  const dz = new Dozer(scene, terrain, particles);
  const tr = new Truck(scene, terrain, particles);
  ex.place(8, -1.5, 0);
  tr.place(13.6, -4.0, Math.PI, 1.15);
  dz.place(-12, -8, Math.PI + 0.6);
  const machines = [ex, tr, dz];
  const chunks = new SoilChunks(scene, terrain);
  ex.chunks = chunks;
  // ---- clod colliders: bucket shell, truck body, machine bodies, track/tyre crush zones ----
  const boxColliders = [];
  const mkBox = (machine, rec, crush, localC, localH) => {
    const c = { kind: 'box', machine, crush, M: new THREE.Matrix4(), inv: new THREE.Matrix4(), h: localH || rec.h, aabb: new Float32Array(6), moving: true, mu: 0.6 };
    const tmp = new THREE.Vector3(), pv = new THREE.Vector3(), bm = new THREE.Matrix4(), one = new THREE.Vector3(1, 1, 1), tr0 = new THREE.Matrix4();
    c.update = (dt) => {
      if (rec) c.M.copy(rec.matrixWorld);
      else { bm.compose(machine.body.pos, machine.body.quat, one); c.M.multiplyMatrices(bm, tr0.makeTranslation(localC.x, localC.y, localC.z)); }
      c.inv.copy(c.M).invert();
      const o = tmp.setFromMatrixPosition(c.M), R = Math.hypot(c.h.x, c.h.y, c.h.z) + 0.2;
      c.aabb.set([o.x - R, o.y - R, o.z - R, o.x + R, o.y + R, o.z + R]);
      c.moving = machine.body.vel.lengthSq() > 1e-4 || machine.body.angVel.lengthSq() > 1e-5;
      c._dt = dt;
    };
    c.motionAt = (x, y, z) => machine.body.pointVel(tmp.set(x, y, z), pv).multiplyScalar(c._dt);
    boxColliders.push(c);
    return c;
  };
  for (const m of machines) for (const b of m.boxes) {
    if (b === ex.bucketBox || b === tr.bedBox) continue;
    mkBox(m, b, false);
  }
  const V = (x, y, z) => new THREE.Vector3(x, y, z);
  for (const s of [1, -1]) {
    mkBox(ex, null, true, V(s * 1.19, -0.55, 0), V(0.33, 0.45, 2.25));
    mkBox(dz, null, true, V(s * 0.99, -0.5, 0), V(0.3, 0.5, 1.7));
  }
  for (const w of tr.wheelDefs) mkBox(tr, null, true, V(w.x, -1.1 + 0.54, w.z), V(w.dual ? 0.34 : 0.17, 0.54, 0.45));
  let chunkPhase = 0;
  function chunkStep(dt) {
    for (const m of machines) { m._chunkF.set(0, 0, 0); m._chunkT.set(0, 0, 0); }
    const cols = [ex.bucketCollider(), tr.bedCollider()];
    for (const m of machines) m.getCollisionBoxes();
    for (const b of boxColliders) { b.update(dt); cols.push(b); }
    chunks.step(dt, cols);
    // what the clods are sitting in
    const bk = chunks.volumeWhere(1); ex.load = bk.vol; ex.loadType = bk.type;
    const bd = chunks.volumeWhere(2); tr.payloadVol = bd.vol; tr.payloadType = bd.type;
  }
  function physicsSubstep() {
    for (let i = 0; i < machines.length; i++) for (let j = i + 1; j < machines.length; j++) collideMachines(machines[i], machines[j]);
    for (const m of machines) {
      bound(m);
      // soil clods pressing on the bucket / truck body / blade
      const f = m._chunkF, t = m._chunkT;
      const cap = m.body.mass * 30;
      if (f.length() < cap) { m.body.force.add(f); m.body.torque.add(t); }
      m.step(DT); m.guard();
    }
    if ((chunkPhase++ & 1) === 1) chunkStep(2 * DT);
  }
  for (const m of machines) { m.updateVisuals(0); m.refreshCollisionLocal(); }

  // design grades
  const avgOrig = (r) => {
    let s = 0, n = 0;
    for (let z = r.z0; z <= r.z1; z += 0.5) for (let x = r.x0; x <= r.x1; x += 0.5) { const k = terrain.nearestIdx(x, z); s += terrain.orig[k]; n++; }
    return s / n;
  };
  const pitElev = avgOrig(SITE.pit) - SITE.pit.depth;
  const padElev = avgOrig(SITE.pad) + SITE.pad.lift;
  terrain.material.userData.uniforms.uPitElev.value = pitElev;
  terrain.material.userData.uniforms.uPadElev.value = padElev;
  ex.gradeTarget = pitElev;

  const input = new Input();
  const sound = new SoundSystem();
  sound.addEngine(ex, 4); sound.addEngine(tr, 6); sound.addEngine(dz, 6);
  let active = 0;
  let camMode = 'orbit';
  const cam = { yaw: Math.PI * 0.85, pitch: 0.38, dist: 17, target: new THREE.Vector3(), lookYaw: 0, lookPitch: 0, lastDrag: -10 };
  cam.target.copy(ex.body.pos);

  // ---- mouse camera ----
  let dragging = false, lx = 0, ly = 0;
  canvas.addEventListener('pointerdown', (e) => { dragging = true; lx = e.clientX; ly = e.clientY; canvas.setPointerCapture(e.pointerId); sound.start(); hideClickHint(); });
  canvas.addEventListener('pointerup', () => { dragging = false; });
  canvas.addEventListener('pointermove', (e) => {
    if (!dragging) return;
    const dx = e.clientX - lx, dy = e.clientY - ly; lx = e.clientX; ly = e.clientY;
    if (camMode === 'orbit') { cam.yaw -= dx * 0.005; cam.pitch = clamp(cam.pitch + dy * 0.004, -0.15, 1.45); cam.lastDrag = performance.now() / 1000; }
    else { cam.lookYaw -= dx * 0.004; cam.lookPitch = clamp(cam.lookPitch - dy * 0.004, -1.2, 1.0); }
  });
  canvas.addEventListener('wheel', (e) => { cam.dist = clamp(cam.dist * Math.exp(e.deltaY * 0.001), 4, 80); e.preventDefault(); }, { passive: false });
  addEventListener('keydown', () => { sound.start(); hideClickHint(); }, { once: false });

  // ---- HUD ----
  const hud = document.getElementById('hud');
  const help = document.getElementById('help');
  const job = document.getElementById('job');
  const gradeEl = document.getElementById('grade');
  const btns = document.querySelectorAll('#machines button');
  btns.forEach((b, i) => b.addEventListener('click', () => { select(i); sound.start(); }));
  let showHelp = true;
  function hideClickHint() { document.getElementById('clickhint')?.remove(); }
  function select(i) {
    active = (i + machines.length) % machines.length;
    btns.forEach((b, k) => b.classList.toggle('on', k === active));
    renderHelp();
  }
  function renderHelp() {
    const m = machines[active];
    const rows = CONTROLS[m.kind].concat([['', '']]).concat(CONTROLS.global);
    help.innerHTML = `<div class="ttl">Controls — ${m.label}</div>` + rows.map(([k, v]) => k ? `<div class="row"><kbd>${k}</kbd><span>${v}</span></div>` : '<div class="sep"></div>').join('');
    help.style.display = showHelp ? 'block' : 'none';
  }
  select(0);
  // desktop build: audio is allowed without a click
  if (navigator.userAgent.includes('Electron')) { sound.start(); hideClickHint(); }

  // ---- job progress ----
  const jobState = { pitPct: 0, pitVol: 0, over: 0, floorPct: 0, padPct: 0, padVol: 0, hauled: 0 };
  function updateJob() {
    const T = terrain;
    let need = 0, done = 0, over = 0, floorOk = 0, nPit = 0;
    const p = SITE.pit;
    for (let z = p.z0 + 0.125; z < p.z1; z += 0.25) for (let x = p.x0 + 0.125; x < p.x1; x += 0.25) {
      const k = T.nearestIdx(x, z);
      const o = T.orig[k], H = T.bank[k] + T.loose[k];
      const req = o - pitElev;
      need += req; done += clamp(o - H, 0, req);
      if (H < pitElev - SITE.tolerance) over++;
      if (Math.abs(H - pitElev) <= SITE.tolerance) floorOk++;
      nPit++;
    }
    const d = SITE.pad;
    let padOk = 0, nPad = 0, padVol = 0;
    for (let z = d.z0 + 0.125; z < d.z1; z += 0.25) for (let x = d.x0 + 0.125; x < d.x1; x += 0.25) {
      const k = T.nearestIdx(x, z);
      const H = T.bank[k] + T.loose[k];
      if (Math.abs(H - padElev) <= SITE.tolerance) padOk++;
      padVol += clamp(H - T.orig[k], 0, padElev - T.orig[k]) * T.cellA;
      nPad++;
    }
    jobState.pitPct = done / need; jobState.pitVol = done * T.cellA;
    jobState.over = over * T.cellA; jobState.floorPct = floorOk / nPit;
    jobState.padPct = padOk / nPad; jobState.padVol = padVol;
    jobState.hauled = chunks.padDumped;
    const bar = (f) => `<div class="bar"><i style="width:${(clamp(f, 0, 1) * 100).toFixed(1)}%"></i></div>`;
    const pitTotal = need * T.cellA;
    job.innerHTML = `
      <div class="ttl">Work order #2611 — Lot 14</div>
      <div class="obj"><b>1. Excavate basement</b> 12 × 9 m to −${SITE.pit.depth} m (EL ${pitElev.toFixed(2)})
        ${bar(jobState.pitPct)}<div class="sm">${jobState.pitVol.toFixed(1)} / ${pitTotal.toFixed(0)} m³ bank · floor on grade ${(jobState.floorPct * 100).toFixed(0)}%${jobState.over > 0.2 ? ` · <span class="warn">over-dig ${jobState.over.toFixed(1)} m²</span>` : ''}</div></div>
      <div class="obj"><b>2. Haul spoil to fill pad</b>
        <div class="sm">${jobState.hauled.toFixed(1)} m³ loose dumped on pad · total excavated ${T.stats.excavatedBank.toFixed(1)} m³ bank</div></div>
      <div class="obj"><b>3. Spread & grade pad</b> to +${SITE.pad.lift} m (EL ${padElev.toFixed(2)}) ±${(SITE.tolerance * 100).toFixed(0)} cm
        ${bar(jobState.padPct)}<div class="sm">${(jobState.padPct * 100).toFixed(0)}% on grade · ${jobState.padVol.toFixed(1)} m³ placed</div></div>`;
  }
  updateJob();
  let jobT = 0;

  // ---- loop ----
  const DT = 1 / 240;
  let acc = 0, last = performance.now(), fpsT = 0, frames = 0, fps = 60;
  let paused = false;
  const tmpV = new THREE.Vector3(), tmpQ = new THREE.Quaternion();
  if (loading) loading.remove();

  function bound(m) {
    const L = SITE.size / 2 - 3.5;
    const p = m.body.pos;
    for (const ax of ['x', 'z']) {
      if (Math.abs(p[ax]) > L) {
        const pen = Math.abs(p[ax]) - L;
        const f = new THREE.Vector3(); f[ax] = -Math.sign(p[ax]) * pen * m.body.mass * 30 - m.body.vel[ax] * m.body.mass * 2;
        m.body.addForce(f);
      }
    }
  }

  function tick() {
    const now = performance.now();
    let dt = (now - last) / 1000; last = now;
    if (dt > 0.1) dt = 0.1;
    frames++; fpsT += dt; if (fpsT > 0.5) { fps = frames / fpsT; frames = 0; fpsT = 0; }

    // global keys
    if (input.hit('Digit1')) select(0);
    if (input.hit('Digit2')) select(1);
    if (input.hit('Digit3')) select(2);
    if (input.hit('Tab') || input.padHit(9)) select(active + 1);
    if (input.hit('KeyC') || input.padHit(8)) { camMode = camMode === 'orbit' ? 'cab' : 'orbit'; cam.lookYaw = 0; cam.lookPitch = 0; }
    if (input.hit('KeyH')) { showHelp = !showHelp; help.style.display = showHelp ? 'block' : 'none'; }
    if (input.hit('KeyO')) { const u = terrain.material.userData.uniforms.uGrade; u.value = u.value > 0.5 ? 0 : 1; }
    if (input.hit('KeyM')) sound.setMuted(!sound.muted);
    if (input.hit('KeyP')) paused = !paused;

    const am = machines[active];
    for (const m of machines) {
      let inp = {};
      if (m === am) inp = m.kind === 'excavator' ? input.forExcavator() : m.kind === 'dozer' ? input.forDozer() : input.forTruck();
      m.control(inp, dt);
    }
    input.endFrame();

    if (!paused) {
      acc += dt;
      let steps = 0;
      while (acc >= DT && steps < 30) {
        physicsSubstep();
        acc -= DT; steps++;
        // consume one-shot toggles after first substep
        for (const m of machines) if (m.inp) { m.inp.toggleSpeed = false; m.inp.park = false; }
      }
      if (steps >= 30) acc = 0;
      terrain.relax(2);
      particles.update(dt);
    }
    for (const m of machines) { m.updateVisuals(dt); m.refreshCollisionLocal(); }
    chunks.render();
    terrain.updateMesh();

    // ---- camera ----
    const tgt = tmpV.copy(am.body.pos);
    tgt.y += am.kind === 'truck' ? 1.6 : 1.4;
    cam.target.lerp(tgt, 1 - Math.exp(-dt * 8));
    if (camMode === 'free') {
      // positioned externally (screenshot / photo hook)
    } else if (camMode === 'orbit') {
      // ease behind a moving machine when the user isn't steering the camera
      const idle = now / 1000 - cam.lastDrag > 4;
      const sp = am.body.vel.length();
      if (idle && sp > 1.2 && am.kind !== 'excavator') {
        const f = new THREE.Vector3(0, 0, 1).applyQuaternion(am.body.quat);
        const want = Math.atan2(-f.x, -f.z);
        let d = want - cam.yaw; d = Math.atan2(Math.sin(d), Math.cos(d));
        cam.yaw += d * (1 - Math.exp(-dt * 1.2));
      }
      const cp = Math.cos(cam.pitch);
      camera.position.set(cam.target.x + Math.sin(cam.yaw) * cp * cam.dist, cam.target.y + Math.sin(cam.pitch) * cam.dist, cam.target.z + Math.cos(cam.yaw) * cp * cam.dist);
      const gy = terrain.heightAt(camera.position.x, camera.position.z) + 0.6;
      if (camera.position.y < gy) camera.position.y = gy;
      camera.lookAt(cam.target);
      camera.near = 0.1;
    } else {
      am.cabCam.getWorldPosition(camera.position);
      am.cabCam.getWorldQuaternion(tmpQ);
      camera.quaternion.copy(tmpQ);
      camera.rotateY(Math.PI + cam.lookYaw); // markers look down -z by default; machines face +z
      camera.rotateX(cam.lookPitch - (am.kind === 'excavator' ? 0.25 : 0.12));
      camera.near = 0.05;
    }
    camera.updateProjectionMatrix();
    // clods are scattered around the point the camera is looking at
    focus.set(0, 0, -1).applyQuaternion(camera.quaternion).setY(0).normalize().multiplyScalar(camMode === 'cab' ? 9 : 4).add(camera.position);
    if (camMode === 'orbit') focus.lerp(cam.target, 0.7);
    clods.update(dt, focus);
    // shadows follow the action
    env.sun.target.position.copy(cam.target);
    env.sun.position.copy(cam.target).add(env.sun.userData.off || (env.sun.userData.off = env.sun.position.clone()));
    env.sun.shadow.camera.updateProjectionMatrix();

    // ---- HUD ----
    const h = am.hud();
    hud.innerHTML = `<div class="ttl">${h.title}</div><div class="sub">${h.sub}</div>` +
      h.rows.map(([k, v]) => `<div class="row"><span>${k}</span><b>${v}</b></div>`).join('') +
      `<div class="gauge ${h.stall ? 'hot' : ''}"><i style="width:${(clamp(h.gauge, 0, 1) * 100).toFixed(0)}%"></i></div>` +
      (h.stall ? `<div class="warn">${am.kind === 'excavator' ? 'RELIEF — hydraulics at max pressure' : am.kind === 'dozer' ? 'TRACK SLIP — reduce cut depth' : 'OVER GVWR'}</div>` : '') +
      `<div class="fps">${fps.toFixed(0)} fps · ${camMode === 'cab' ? 'cab view' : 'orbit view'}</div>`;
    if (h.grade !== null && h.grade !== undefined) {
      gradeEl.style.display = 'block';
      const g = h.grade;
      const pos = clamp(50 - g * 25, 2, 98);
      gradeEl.innerHTML = `<div class="lbl">GRADE</div><div class="scale"><i class="tgt"></i><i class="tip" style="top:${pos}%"></i></div><div class="val ${Math.abs(g) < SITE.tolerance ? 'ok' : g < 0 ? 'low' : ''}">${g >= 0 ? '+' : ''}${g.toFixed(2)} m</div><div class="lbl">${g > SITE.tolerance ? 'cut' : g < -SITE.tolerance ? 'over' : 'on grade'}</div>`;
    } else gradeEl.style.display = 'none';
    jobT += dt;
    if (jobT > 0.5) { jobT = 0; updateJob(); }

    // ---- sound ----
    const right = new THREE.Vector3(1, 0, 0).applyQuaternion(camera.quaternion);
    sound.update(dt, camera.position, right, am, particles.flowRate);

    renderer.render(scene, camera);
    requestAnimationFrame(tick);
  }

  addEventListener('resize', () => {
    renderer.setSize(innerWidth, innerHeight);
    camera.aspect = innerWidth / innerHeight;
    camera.updateProjectionMatrix();
  });
  // deterministic stepping hook (used by automated tests)
  function simulate(seconds, inpFor) {
    const n = Math.round(seconds * 240);
    for (let s = 0; s < n; s++) {
      if (s % 4 === 0) for (const m of machines) m.control(inpFor ? (inpFor(m, s / 240) || {}) : {}, 1 / 60);
      physicsSubstep();
      if (s % 4 === 3) { terrain.relax(1); particles.update(1 / 60); for (const m of machines) { m.updateVisuals(1 / 60); m.refreshCollisionLocal(); } }
    }
  }
  window.__gb = { simulate, chunks, simTime: () => 0, terrain, machines, camera, scene, renderer, select, SOIL, cam, setCamMode: (m) => { camMode = m; },
    photo: (px, py, pz, lx, ly, lz, fov = 55) => { camMode = 'free'; camera.position.set(px, py, pz); camera.lookAt(lx, ly, lz); camera.fov = fov; camera.updateProjectionMatrix(); }, help };
  requestAnimationFrame(tick);
}

init().catch((e) => {
  console.error(e);
  setStatus('Error: ' + e.message);
});
