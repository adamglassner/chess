global.self = global; global.importScripts = f => require('../web/' + f); require('../web/ai.js');
const C = self.Chess, AI = self.ChessAI;
function play(fen, wOpts, bOpts, maxPly = 300, verbose) {
  const p = new C.Position(fen); const moves = [];
  for (let ply = 0; ply < maxPly; ply++) {
    const st = p.status(); if (st.over) return { result: st.result, reason: st.reason, ply, moves };
    const o = p.side === 1 ? wOpts : bOpts;
    const r = AI.think(Object.assign({ fen, moves }, o));
    const m = p.legal().find(x => C.uci(x) === r.move); if (!m) return { result: '?', reason: 'illegal ' + r.move };
    moves.push(r.move); p.make(m);
  }
  return { result: '*', reason: 'ply limit', ply: maxPly, moves };
}
const mode = process.argv[2];
if (mode === 'endgames') {
  for (const fen of ['8/8/8/4k3/8/8/8/4K2Q w - - 0 1', '8/8/8/4k3/8/8/8/4K2R w - - 0 1', '8/8/3k4/8/8/8/8/R3K2R w - - 0 1', '7k/8/8/8/8/8/8/KQ6 b - - 0 1']) {
    const t = Date.now(); const r = play(fen, { time: 250 }, { time: 250 }, 200);
    console.log(fen, '=>', r.result, r.reason, 'plies', r.ply, ((Date.now() - t) / 1000).toFixed(1) + 's');
  }
} else {
  const L = { 1: { maxDepth: 1, noise: 140, blunder: .18 }, 2: { maxDepth: 2, noise: 60, blunder: .06 }, 3: { maxDepth: 3, noise: 22, blunder: .015 }, 4: { time: 300, maxDepth: 5 }, 5: { time: 700 }, 6: { time: 1500 } };
  const [a, b, n] = process.argv.slice(3).map(Number);
  let score = { a: 0, b: 0, d: 0 };
  for (let g = 0; g < n; g++) {
    const aw = g % 2 === 0; const r = play(C.START, aw ? L[a] : L[b], aw ? L[b] : L[a], 400);
    const wWin = r.result === '1-0', bWin = r.result === '0-1';
    if (wWin) score[aw ? 'a' : 'b']++; else if (bWin) score[aw ? 'b' : 'a']++; else score.d++;
    console.log(`game ${g + 1}: L${aw ? a : b} (W) vs L${aw ? b : a} (B): ${r.result} ${r.reason} in ${r.ply} plies`);
  }
  console.log(`L${a} ${score.a}  L${b} ${score.b}  draws ${score.d}`);
}
