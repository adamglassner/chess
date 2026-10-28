const fs = require('fs'), vm = require('vm');
const ctx = { console, document: {}, window: {}, Math, setTimeout, localStorage: {} }; vm.createContext(ctx);
vm.runInContext(fs.readFileSync('web/core.js', 'utf8') + ';this.IR=IR;', ctx);
vm.runInContext(fs.readFileSync('web/music.js', 'utf8') + ';this.MUSIC=MUSIC;', ctx);
const { IR, MUSIC } = ctx;
for (const tune of MUSIC.TUNES) {
  for (const sec of ['A', 'B', 'T']) {
    const S = tune[sec]; if (!S) continue;
    const bars = S.mel.split('|').map(b => b.trim());
    if (bars.length !== S.ch.length) console.log(tune.name, sec, 'bar count', bars.length, 'vs chords', S.ch.length);
    bars.forEach((b, i) => {
      let t = 0; const ch = IR.chord(S.ch[i]);
      b.split(/\s+/).forEach(tok => { const [n, d] = tok.split(':'); if (n !== 'R') { const pc = IR.n2m(n) % 12; const strong = Math.abs(t % 1) < 1e-9 && (t === 0 || (tune.beats === 4 && t === 2)); if (strong && !ch.pcs.includes(pc)) console.log(`  clash ${tune.name} ${sec} bar ${i + 1} ${S.ch[i]} note ${n} at beat ${t}`); } t += +d; });
      if (Math.abs(t - tune.beats) > 1e-9) console.log(`BAD LENGTH ${tune.name} ${sec} bar ${i + 1}: ${t}`);
    });
  }
  const a = MUSIC.tunes().find(x => x.name === tune.name);
  console.log(tune.name, 'events', a.events.length, 'length', (a.len * 60 / a.bpm).toFixed(0) + 's', 'instruments', [...new Set(a.events.map(e => e.i))].join(','));
  for (const e of a.events) if (!MUSIC.instruments[e.i]) console.log('missing instrument', e.i);
}
