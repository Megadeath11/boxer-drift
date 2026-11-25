// Procedural WebAudio sound: diesel engines, hydraulics, relief valve, track
// clatter, falling soil and the truck's reversing alarm. No audio files needed.
export class SoundSystem {
  constructor() {
    this.ctx = null;
    this.muted = false;
    this.engines = [];
  }
  start() {
    if (this.ctx) { if (this.ctx.state === 'suspended') this.ctx.resume(); return; }
    const AC = window.AudioContext || window.webkitAudioContext;
    if (!AC) return;
    this.ctx = new AC();
    const ctx = this.ctx;
    this.master = ctx.createGain(); this.master.gain.value = this.muted ? 0 : 0.55;
    const comp = ctx.createDynamicsCompressor();
    this.master.connect(comp); comp.connect(ctx.destination);
    // shared noise buffer
    const len = ctx.sampleRate * 2;
    const buf = ctx.createBuffer(1, len, ctx.sampleRate);
    const d = buf.getChannelData(0);
    let b = 0;
    for (let i = 0; i < len; i++) { const w = Math.random() * 2 - 1; b = 0.97 * b + 0.03 * w; d[i] = w * 0.5 + b * 4; }
    this.noiseBuf = buf;
    // soil pour / scrape
    this.pour = this.noiseVoice(500, 1.2);
    this.scrape = this.noiseVoice(1400, 2.5);
    // backup alarm
    this.alarm = ctx.createOscillator(); this.alarm.type = 'square'; this.alarm.frequency.value = 1150;
    this.alarmG = ctx.createGain(); this.alarmG.gain.value = 0;
    this.alarm.connect(this.alarmG); this.alarmG.connect(this.master); this.alarm.start();
    for (const e of this._pending || []) this._buildEngine(e);
  }
  noiseVoice(freq, q) {
    const ctx = this.ctx;
    const src = ctx.createBufferSource(); src.buffer = this.noiseBuf; src.loop = true;
    const f = ctx.createBiquadFilter(); f.type = 'bandpass'; f.frequency.value = freq; f.Q.value = q;
    const g = ctx.createGain(); g.gain.value = 0;
    src.connect(f); f.connect(g); g.connect(this.master); src.start();
    return g;
  }
  setMuted(m) { this.muted = m; if (this.master) this.master.gain.value = m ? 0 : 0.55; }

  addEngine(machine, cylinders) {
    const e = { machine, cylinders };
    this.engines.push(e);
    if (this.ctx) this._buildEngine(e); else (this._pending = this._pending || []).push(e);
  }
  _buildEngine(e) {
    const ctx = this.ctx;
    const out = ctx.createGain(); out.gain.value = 0;
    const pan = ctx.createStereoPanner ? ctx.createStereoPanner() : null;
    if (pan) { out.connect(pan); pan.connect(this.master); } else out.connect(this.master);
    const lp = ctx.createBiquadFilter(); lp.type = 'lowpass'; lp.frequency.value = 400; lp.Q.value = 2;
    lp.connect(out);
    const o1 = ctx.createOscillator(); o1.type = 'sawtooth';
    const o2 = ctx.createOscillator(); o2.type = 'square';
    const g1 = ctx.createGain(); g1.gain.value = 0.5;
    const g2 = ctx.createGain(); g2.gain.value = 0.25;
    o1.connect(g1); g1.connect(lp); o2.connect(g2); g2.connect(lp);
    // combustion roughness: noise amplitude-modulated
    const n = ctx.createBufferSource(); n.buffer = this.noiseBuf; n.loop = true;
    const nf = ctx.createBiquadFilter(); nf.type = 'lowpass'; nf.frequency.value = 260;
    const ng = ctx.createGain(); ng.gain.value = 0.4;
    n.connect(nf); nf.connect(ng); ng.connect(out);
    // turbo whistle
    const t = ctx.createOscillator(); t.type = 'sine';
    const tg = ctx.createGain(); tg.gain.value = 0;
    t.connect(tg); tg.connect(out);
    // hydraulic whine + relief squeal
    const h = ctx.createOscillator(); h.type = 'triangle'; h.frequency.value = 420;
    const hg = ctx.createGain(); hg.gain.value = 0;
    h.connect(hg); hg.connect(out);
    const r = ctx.createOscillator(); r.type = 'sawtooth'; r.frequency.value = 1700;
    const rf = ctx.createBiquadFilter(); rf.type = 'bandpass'; rf.frequency.value = 1700; rf.Q.value = 8;
    const rg = ctx.createGain(); rg.gain.value = 0;
    r.connect(rf); rf.connect(rg); rg.connect(out);
    // track clatter: noise gated by a square LFO
    const cn = ctx.createBufferSource(); cn.buffer = this.noiseBuf; cn.loop = true;
    const cf = ctx.createBiquadFilter(); cf.type = 'highpass'; cf.frequency.value = 900;
    const cg = ctx.createGain(); cg.gain.value = 0;
    const lfo = ctx.createOscillator(); lfo.type = 'square'; lfo.frequency.value = 4;
    const lfoG = ctx.createGain(); lfoG.gain.value = 0;
    lfo.connect(lfoG); lfoG.connect(cg.gain);
    const clat = ctx.createGain(); clat.gain.value = 0;
    cn.connect(cf); cf.connect(cg); cg.connect(clat); clat.connect(out);
    for (const s of [o1, o2, n, t, h, r, cn, lfo]) s.start();
    Object.assign(e, { out, pan, lp, o1, o2, t, tg, h, hg, r, rg, lfo, lfoG, clat, ng });
  }

  update(dt, listenerPos, listenerRight, active, particlesFlow) {
    if (!this.ctx || !this.engines[0]?.out) return;
    const now = this.ctx.currentTime;
    const sm = 0.08;
    for (const e of this.engines) {
      const m = e.machine;
      const d = m.body.pos.distanceTo(listenerPos);
      const att = 1 / (1 + Math.max(0, d - 4) * 0.12);
      const fire = (m.rpm / 60) * (e.cylinders / 2);
      e.o1.frequency.setTargetAtTime(fire, now, sm);
      e.o2.frequency.setTargetAtTime(fire * 0.5, now, sm);
      e.lp.frequency.setTargetAtTime(180 + m.engineLoad * 900 + m.rpm * 0.15, now, sm);
      e.t.frequency.setTargetAtTime(1800 + m.rpm * 2.2, now, sm);
      e.tg.gain.setTargetAtTime(0.015 * m.engineLoad, now, sm);
      e.ng.gain.setTargetAtTime(0.25 + m.engineLoad * 0.6, now, sm);
      e.out.gain.setTargetAtTime((0.22 + m.engineLoad * 0.35) * att * (m === active ? 1 : 0.8), now, sm);
      if (e.pan) {
        const dir = m.body.pos.clone().sub(listenerPos);
        const p = d > 0.5 ? Math.max(-1, Math.min(1, dir.normalize().dot(listenerRight))) : 0;
        e.pan.pan.setTargetAtTime(p * 0.8, now, 0.1);
      }
      // hydraulics
      const press = m.pressure ?? (m.hoistActive ? 0.5 : 0);
      const hydActive = (m.hydLoad ?? (m.hoistActive ? 0.6 : 0));
      e.h.frequency.setTargetAtTime(380 + press * 260 + m.rpm * 0.05, now, sm);
      e.hg.gain.setTargetAtTime(0.03 * Math.min(1, hydActive), now, sm);
      e.rg.gain.setTargetAtTime(m.stall > 0.5 ? 0.05 : 0, now, 0.05);
      // tracks
      const belt = Math.max(Math.abs(m.beltActualL || 0), Math.abs(m.beltActualR || 0));
      e.lfo.frequency.setTargetAtTime(Math.max(1, belt / 0.19), now, sm);
      e.lfoG.gain.setTargetAtTime(belt > 0.05 ? 0.5 : 0, now, sm);
      e.clat.gain.setTargetAtTime(Math.min(0.35, belt * 0.3), now, sm);
    }
    // soil sounds
    let scrape = 0;
    for (const e of this.engines) scrape += (e.machine.digRate || 0) * 0.6 + (e.machine.pile || 0) * Math.min(1, Math.abs(e.machine.vlong ?? e.machine.body.vel.length())) * 0.04;
    this.pour.gain.setTargetAtTime(Math.min(0.5, particlesFlow * 0.35), now, 0.06);
    this.scrape.gain.setTargetAtTime(Math.min(0.25, scrape * 0.4), now, 0.08);
    // reversing alarm (any truck in reverse and moving / ready)
    let alarm = false, ad = 99;
    for (const e of this.engines) if (e.machine.kind === 'truck' && e.machine.reverse) { alarm = true; ad = e.machine.body.pos.distanceTo(listenerPos); }
    const on = alarm && (Math.floor(now * 1.8) % 2 === 0);
    this.alarmG.gain.setTargetAtTime(on ? 0.06 / (1 + ad * 0.08) : 0, now, 0.01);
  }
}
