// Keyboard + gamepad input, mapped to each machine's real control layout.
// Excavator uses the ISO (CAT/Komatsu standard) joystick pattern on a gamepad:
//   left stick  : fore/aft = stick out/in, left/right = swing
//   right stick : fore/aft = boom down/up, left/right = bucket curl/dump
export class Input {
  constructor() {
    this.keys = new Set();
    this.pressed = new Set();
    addEventListener('keydown', (e) => {
      if (e.code === 'Tab' || e.code === 'Space' || e.code.startsWith('Arrow')) e.preventDefault();
      if (!this.keys.has(e.code)) this.pressed.add(e.code);
      this.keys.add(e.code);
    });
    addEventListener('keyup', (e) => this.keys.delete(e.code));
    addEventListener('blur', () => this.keys.clear());
    this.padPrev = [];
  }
  k(code) { return this.keys.has(code) ? 1 : 0; }
  hit(code) { return this.pressed.has(code); }
  endFrame() { this.pressed.clear(); }

  pad() {
    const pads = navigator.getGamepads ? navigator.getGamepads() : [];
    for (const p of pads) if (p && p.connected) return p;
    return null;
  }
  padHit(i) {
    const p = this.pad();
    if (!p) return false;
    const now = !!p.buttons[i]?.pressed;
    const was = !!this.padPrev[i];
    this.padPrev[i] = now;
    return now && !was;
  }

  axes() {
    const p = this.pad();
    const dz = (v) => (Math.abs(v) < 0.12 ? 0 : (v - Math.sign(v) * 0.12) / 0.88);
    if (!p) return { lx: 0, ly: 0, rx: 0, ry: 0, lt: 0, rt: 0, lb: 0, rb: 0, du: 0, dd: 0, dl: 0, dr: 0 };
    const b = (i) => (p.buttons[i] ? p.buttons[i].value : 0);
    return { lx: dz(p.axes[0] || 0), ly: dz(p.axes[1] || 0), rx: dz(p.axes[2] || 0), ry: dz(p.axes[3] || 0),
      lt: b(6), rt: b(7), lb: b(4), rb: b(5), du: b(12), dd: b(13), dl: b(14), dr: b(15) };
  }

  forExcavator() {
    const a = this.axes();
    const fwd = this.k('ArrowUp') - this.k('ArrowDown') + a.du - a.dd;
    const turn = this.k('ArrowLeft') - this.k('ArrowRight');
    let trackL = fwd - turn, trackR = fwd + turn;
    trackL += a.lt - a.lb; trackR += a.rt - a.rb;
    return {
      stick: this.k('KeyW') - this.k('KeyS') - a.ly,
      swing: this.k('KeyA') - this.k('KeyD') - a.lx,
      boom: this.k('KeyK') - this.k('KeyI') + a.ry,
      bucket: this.k('KeyL') - this.k('KeyJ') + a.rx,
      trackL, trackR,
      toggleSpeed: this.hit('KeyT') || this.padHit(3),
    };
  }
  forDozer() {
    const a = this.axes();
    return {
      drive: this.k('KeyW') - this.k('KeyS') - a.ly + a.rt - a.lt,
      steer: this.k('KeyA') - this.k('KeyD') - a.lx,
      blade: this.k('KeyR') - this.k('KeyF') - a.ry,
      tilt: this.k('KeyQ') - this.k('KeyE') + a.rx,
      ripper: this.k('KeyG') - this.k('KeyT') + a.dd - a.du,
      slow: this.k('ShiftLeft') || this.k('ShiftRight'),
    };
  }
  forTruck() {
    const a = this.axes();
    return {
      throttle: Math.min(1, this.k('KeyW') + a.rt + Math.max(0, -a.ly)),
      brake: Math.min(1, this.k('KeyS') + a.lt + Math.max(0, a.ly)),
      steer: this.k('KeyA') - this.k('KeyD') - a.lx,
      bed: this.k('KeyR') - this.k('KeyF') - a.ry,
      park: this.hit('Space') || this.padHit(1),
    };
  }
}

export const CONTROLS = {
  excavator: [
    ['W / S', 'Stick out / in'],
    ['A / D', 'Swing left / right'],
    ['I / K', 'Boom down / up'],
    ['J / L', 'Bucket curl / dump'],
    ['↑ ↓', 'Travel forward / reverse'],
    ['← →', 'Counter-rotate (pivot) tracks'],
    ['T', 'Travel speed Hi / Lo'],
    ['Gamepad', 'ISO pattern · LT/RT tracks'],
  ],
  dozer: [
    ['W / S', 'Forward / reverse'],
    ['A / D', 'Steer left / right'],
    ['R / F', 'Blade raise / lower'],
    ['Q / E', 'Blade tilt'],
    ['G / T', 'Ripper down / up'],
    ['Shift', 'Inching (slow)'],
  ],
  truck: [
    ['W', 'Throttle'],
    ['S', 'Brake (hold when stopped = reverse)'],
    ['A / D', 'Steer'],
    ['R / F', 'Raise / lower dump body'],
    ['Space', 'Park brake'],
  ],
  global: [
    ['1 2 3 / Tab', 'Switch machine'],
    ['C', 'Cab / orbit camera'],
    ['Mouse drag / wheel', 'Look / zoom'],
    ['O', 'Grade-control overlay (cut / fill)'],
    ['M', 'Mute'],
    ['H', 'Hide help'],
  ],
};
