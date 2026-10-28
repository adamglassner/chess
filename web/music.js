/* Chess music and sound effects. Four original chamber pieces played live by
   the game (piano, cello, flute, harp, pizzicato bass and a soft string
   section), plus the sounds of wooden pieces on a wooden board.
   (c) 2026 Ironite */
'use strict';
const MUSIC = (() => {
const parseMel = (s, at) => { const ev = []; let t = at; s.split(/\s+/).filter(x => x && x !== '|').forEach(tok => { const [n, d] = tok.split(':'); if (n !== 'R') ev.push({ n, t, d: +d }); t += +d; }); return ev; };

function accomp(style, bars, at, B, soft) {
  const ev = [], v = soft ? .75 : 1;
  bars.forEach((sym, bi) => {
    const c = IR.chord(sym), t0 = at + bi * B;
    const root = IR.pcAbove(c.root, 38), fifth = IR.pcAbove((c.root + 7) % 12, root);
    const mid = c.pcs.map(pc => IR.pcAbove(pc, 55)).sort((a, b) => a - b);
    if (style === 'waltz') {
      ev.push({ t: t0, i: 'pLow', m: root, d: 1.2, v: .8 * v });
      ev.push({ t: t0 + 1, i: 'pChord', ms: mid, d: .9, v: .5 * v }); ev.push({ t: t0 + 2, i: 'pChord', ms: mid, d: .9, v: .42 * v });
    } else if (style === 'arp') {
      const third = IR.pcAbove(c.pcs[1], root + 12);
      const seq = [root, fifth, root + 12, third, root + 12, fifth, third, fifth + 12];
      seq.forEach((m, k) => ev.push({ t: t0 + k * .5, i: 'pLow', m, d: 1.6, v: (k === 0 ? .7 : .42) * v }));
    } else if (style === 'minuet') {
      ev.push({ t: t0, i: 'pizz', m: root - 12 < 31 ? root : root - 12, v: .9 * v });
      ev.push({ t: t0 + 1, i: 'harpChord', ms: mid, v: .55 * v }); ev.push({ t: t0 + 2, i: 'harpChord', ms: mid, v: .45 * v });
    } else if (style === 'block') {
      ev.push({ t: t0, i: 'pLow', m: root, d: B * .9, v: .65 * v });
      ev.push({ t: t0 + .5, i: 'pChord', ms: mid, d: 1.4, v: .38 * v }); ev.push({ t: t0 + 2.5, i: 'pChord', ms: mid, d: 1.3, v: .32 * v });
    }
    if (!soft && (style === 'block' || style === 'arp')) ev.push({ t: t0, i: 'strings', ms: [IR.pcAbove(c.root, 48)].concat(mid), d: B, v: .55 });
  });
  return ev;
}

function arrange(tune) {
  const ev = [], B = tune.beats;
  let t = 0;
  if (tune.intro) { ev.push(...accomp(tune.style, tune.intro, t, B, true)); t += tune.intro.length * B; }
  for (const [sec, inst, shift = 0] of tune.form) {
    const S = tune[sec];
    parseMel(S.mel, t).forEach(e => ev.push({ t: e.t, i: inst, m: IR.n2m(e.n) + shift, d: e.d, v: 1 }));
    ev.push(...accomp(S.style || tune.style, S.ch, t, B, false));
    t += S.ch.length * B;
  }
  if (tune.tag) { ev.push(...accomp(tune.style, tune.tag, t, B, true)); ev.push({ t: t + tune.tag.length * B - B, i: 'harp', m: IR.pcAbove(IR.chord(tune.tag[tune.tag.length - 1]).root, 74), v: .7 }); t += tune.tag.length * B; }
  ev.sort((a, b) => a.t - b.t);
  return { name: tune.name, bpm: tune.bpm, events: ev, len: t, gap: 4 };
}

const TUNES = [
  { name: 'The Quiet Library', bpm: 96, beats: 3, style: 'waltz',
    intro: ['D', 'A', 'D', 'D'],
    A: { ch: ['D', 'A', 'Bm', 'F#m', 'G', 'D', 'Em', 'A7', 'D', 'D7', 'G', 'Gm', 'D', 'A7', 'D', 'D'],
      mel: 'F#5:2 E5:.5 D5:.5 | E5:2 C#5:1 | D5:1.5 C#5:.5 B4:1 | A4:3 | B4:1 D5:1 G5:1 | F#5:2 A5:1 | G5:1.5 F#5:.5 E5:1 | E5:3 | F#5:2 E5:.5 D5:.5 | C5:2 A4:1 | B4:1.5 D5:.5 G5:1 | Bb4:2 D5:1 | D5:1.5 F#5:.5 A5:1 | G5:1 E5:1 C#5:1 | D5:3 | R:3' },
    B: { ch: ['Bm', 'G', 'D', 'A', 'Bm', 'Em', 'F#', 'F#', 'G', 'A', 'F#m', 'Bm', 'Em', 'A7', 'D', 'D'],
      mel: 'B5:1.5 A5:.5 F#5:1 | G5:1 B4:1 D5:1 | F#5:1.5 E5:.5 D5:1 | C#5:2 E5:1 | D5:1 F#5:1 B5:1 | B5:1.5 G5:.5 E5:1 | A#4:1 C#5:1 F#5:1 | F#5:3 | G5:1.5 F#5:.5 E5:1 | E5:1 C#5:1 A4:1 | A4:1 C#5:1 F#5:1 | F#5:2 D5:1 | E5:1.5 G5:.5 B5:1 | A5:1 G5:1 C#5:1 | D5:3 | R:3' },
    form: [['A', 'piano'], ['B', 'cello', -12], ['A', 'piano']], tag: ['G', 'A7', 'D'] },
  { name: 'Nocturne of the Long Diagonal', bpm: 64, beats: 4, style: 'arp',
    intro: ['Am', 'Am'],
    A: { ch: ['Am', 'Dm', 'G', 'C', 'F', 'Dm', 'E', 'E7', 'Am', 'Dm', 'G', 'C', 'F', 'E7', 'Am', 'Am'],
      mel: 'E5:3 C5:1 | D5:2 F5:1 A5:1 | G5:3 D5:1 | E5:4 | F5:2 A5:1 C6:1 | A5:2 F5:1 D5:1 | G#5:2 E5:1 B4:1 | D5:2 B4:2 | C5:2 E5:1 A5:1 | A5:1.5 G5:.5 F5:2 | D5:1.5 G5:.5 B5:2 | G5:2 E5:2 | A5:2 F5:1 C5:1 | B4:2 G#4:2 | A4:4 | R:4' },
    B: { ch: ['C', 'G', 'Am', 'Em', 'F', 'C', 'Dm', 'E'],
      mel: 'G5:2 E5:1 G5:1 | B5:2 D5:2 | C6:2 A5:1 E5:1 | G5:2 B5:2 | A5:1.5 C6:.5 F5:2 | E5:2 G5:2 | F5:1 E5:1 D5:1 A5:1 | G#5:4' },
    form: [['A', 'piano'], ['B', 'flute'], ['A', 'piano']], tag: ['Dm', 'Am'] },
  { name: 'Minuet in the Reading Room', bpm: 112, beats: 3, style: 'minuet',
    intro: ['G', 'D', 'G'],
    A: { ch: ['G', 'D', 'G', 'C', 'G', 'Am', 'D', 'D', 'G', 'Em', 'C', 'D', 'G', 'C', 'D', 'G'],
      mel: 'G5:.5 A5:.5 B5:1 G5:1 | A5:1 F#5:1 D5:1 | B5:.5 C6:.5 D6:1 B5:1 | C6:1 G5:1 E5:1 | D5:1 G5:1 B5:1 | C6:1 A5:1 E5:1 | F#5:1 A5:1 D6:1 | D6:2 R:1 | B5:1 D6:.5 B5:.5 G5:1 | G5:1 E5:1 B4:1 | E5:1 G5:1 C6:1 | A5:1.5 G5:.5 F#5:1 | G5:1 B5:1 D6:1 | E6:1 C6:1 G5:1 | A5:1 F#5:1 D5:1 | G5:3' },
    T: { ch: ['Em', 'B7', 'Em', 'Am', 'D', 'G', 'C', 'D'],
      mel: 'B5:1.5 A5:.5 G5:1 | F#5:1 D#5:1 B4:1 | E5:1 G5:1 B5:1 | C6:1.5 B5:.5 A5:1 | F#5:1 A5:1 D6:1 | D6:1 B5:1 G5:1 | E5:1 G5:1 C6:1 | A5:3' },
    form: [['A', 'flute'], ['T', 'harpsi'], ['A', 'harpsi', 0]], tag: ['C', 'D', 'G'] },
  { name: 'Endgame by Lamplight', bpm: 70, beats: 4, style: 'block',
    intro: ['Em', 'C'],
    A: { ch: ['Em', 'C', 'G', 'D', 'Em', 'Am', 'B7', 'B7', 'Em', 'C', 'G', 'D', 'C', 'Am', 'B7', 'Em'],
      mel: 'B3:3 G3:1 | E4:2 C4:2 | D4:3 B3:1 | A3:4 | G3:2 B3:1 E4:1 | E4:2 C4:1 A3:1 | D#4:2 F#4:2 | A3:2 B3:2 | G4:2 E4:1 B3:1 | E4:2 G4:2 | D4:1.5 B3:.5 G3:2 | F#4:2 A4:2 | G4:2 E4:2 | C4:2 E4:1 A3:1 | B3:2 D#4:2 | E4:4' },
    form: [['A', 'cello'], ['A', 'piano', 12]], tag: ['Am', 'B7', 'Em'] },
];

const instruments = {
  piano(t, e, d, v) { this.piano(t, e.m, 1.7 * v, d); },
  pLow(t, e, d, v) { this.piano(t, e.m, 1 * v, d); },
  pChord(t, e, d, v) { e.ms.forEach((m, i) => this.piano(t + i * .006, m, .58 * v, d)); },
  harp(t, e, d, v) { this.pluck(t, e.m, .2 * v, { decay: .9992, bright: .7, dur: 3, lp: 5000 }); },
  harpChord(t, e, d, v) { e.ms.forEach((m, i) => this.pluck(t + i * .03, m, .13 * v, { decay: .999, bright: .65, dur: 2, lp: 4200 })); },
  pizz(t, e, d, v) { this.pluck(t, e.m, .34 * v, { decay: .993, bright: .35, dur: .9, lp: 1300, body: 190 }); },
  harpsi(t, e, d, v) { this.pluck(t, e.m, .2 * v, { decay: .9975, bright: .92, dur: 1.6, lp: 6500 }); this.pluck(t + .004, e.m + 12, .06 * v, { decay: .996, bright: .9, dur: 1, lp: 7000 }); },
  flute(t, e, d, v) { this.flute(t, e.m, .9 * v, d * .95); },
  strings(t, e, d, v) { this.pad(t, e.ms, .9 * v, d * .95); },
  cello(t, e, d, v) {
    const ac = this.ac, f = IR.mtof(e.m), dur = d * .96;
    const o1 = ac.createOscillator(), o2 = ac.createOscillator(); o1.type = o2.type = 'sawtooth'; o1.frequency.value = f; o2.frequency.value = f * 1.0025;
    const lfo = ac.createOscillator(), lg = ac.createGain(); lfo.frequency.value = 5; lg.gain.setValueAtTime(0, t); lg.gain.linearRampToValueAtTime(f * .006, t + Math.min(.6, dur * .6)); lfo.connect(lg); lg.connect(o1.frequency); lg.connect(o2.frequency);
    const lp = ac.createBiquadFilter(); lp.type = 'lowpass'; lp.frequency.value = Math.min(2000, f * 6); lp.Q.value = .4;
    const body = ac.createBiquadFilter(); body.type = 'peaking'; body.frequency.value = 260; body.Q.value = 1; body.gain.value = 5;
    const g = ac.createGain(); this.env(g, t, .14, v * .085, .5, .85, .22, t + dur);
    o1.connect(lp); o2.connect(lp); lp.connect(body); body.connect(g); g.connect(this.bus);
    [o1, o2, lfo].forEach(x => { x.start(t); x.stop(t + dur + 1.2); });
  },
};

/* ---------------------------------------------------------------- sounds */
function sfx(audio) {
  // A boxwood piece set down on a lacquered board: a short woody knock with a little ring.
  const clack = function (t, o, v = 1, pitch = 1) {
    v *= 3;
    this.noiseHit(t, .55 * v, { type: 'lowpass', freq: 520 * pitch, decay: .05, out: o });
    this.noiseHit(t, .35 * v, { freq: 1500 * pitch, q: 3.5, decay: .045, out: o });
    this.noiseHit(t + .002, .18 * v, { freq: 3400 * pitch, q: 5, decay: .02, out: o });
    const ac = this.ac, os = ac.createOscillator(), g = ac.createGain(); os.type = 'sine'; os.frequency.setValueAtTime(820 * pitch, t); os.frequency.exponentialRampToValueAtTime(700 * pitch, t + .06);
    g.gain.setValueAtTime(.08 * v, t); g.gain.exponentialRampToValueAtTime(.0005, t + .09); os.connect(g); g.connect(o); os.start(t); os.stop(t + .1);
  };
  return {
    pick() { audio.sfx(function (t, o) { this.noiseHit(t, .35, { freq: 2800, q: 2, decay: .025, out: o }); }); },
    move() { audio.sfx(function (t, o) { clack.call(this, t, o, 1, .95 + Math.random() * .1); }); },
    capture() { audio.sfx(function (t, o) { clack.call(this, t, o, 1.15, .85); clack.call(this, t + .07, o, .6, 1.1); this.noiseHit(t + .1, .12, { freq: 1800, q: .8, attack: .02, decay: .12, sweep: 900, out: o }); }); },
    castle() { audio.sfx(function (t, o) { clack.call(this, t, o, 1, 1); clack.call(this, t + .16, o, .9, .9); }); },
    check() { audio.sfx(function (t, o) { this.bell(t, 81, .8, o, 1.6); this.bell(t + .1, 76, .6, o, 1.4); }); },
    promote() { audio.sfx(function (t, o) { clack.call(this, t, o, 1, 1); [72, 76, 79, 84, 88].forEach((m, i) => this.bell(t + .08 + i * .07, m, .75, o, 1.8)); }); },
    illegal() { audio.sfx(function (t, o) { this.noiseHit(t, .7, { type: 'lowpass', freq: 240, decay: .06, out: o }); }); },
    click() { audio.sfx(function (t, o) { this.noiseHit(t, .45, { freq: 3800, q: 2.5, decay: .018, out: o }); }); },
    hint() { audio.sfx(function (t, o) { this.bell(t, 84, .45, o, 1.2); this.bell(t + .09, 88, .35, o, 1.2); }); },
    tick() { audio.sfx(function (t, o) { this.noiseHit(t, .6, { freq: 5200, q: 6, decay: .012, out: o }); }); },
    start() { audio.sfx(function (t, o) { [62, 69, 74].forEach((m, i) => this.pluck(t + i * .06, m, .22, { decay: .999, bright: .7, dur: 2.5, lp: 5000, out: o })); }); },
    win() { audio.sfx(function (t, o) { [67, 71, 74, 79].forEach((m, i) => this.pluck(t + i * .09, m, .25, { decay: .9992, bright: .7, dur: 3, lp: 5000, out: o })); [79, 83, 86, 91].forEach((m, i) => this.bell(t + .4 + i * .1, m, .7, o, 2.2)); this.pad(t + .35, [55, 59, 62, 67], 1.6, 2.2, o); }); },
    lose() { audio.sfx(function (t, o) { [64, 60, 57, 52].forEach((m, i) => this.pluck(t + i * .22, m, .22, { decay: .9985, bright: .45, dur: 2.5, lp: 2600, out: o })); }); },
    draw() { audio.sfx(function (t, o) { [67, 72].forEach((m, i) => this.bell(t + i * .18, m, .6, o, 1.8)); }); },
  };
}

return { tunes: () => TUNES.map(arrange), instruments, sfx, TUNES };
})();
