global.self = global; global.importScripts = f => require('../web/' + f); require('../web/ai.js');
const C = self.Chess, AI = self.ChessAI;
const puzzles = [
  ['6k1/5ppp/8/8/8/8/5PPP/3R2K1 w - - 0 1', 'd1d8', 'back-rank mate in 1'],
  ['r1bqkb1r/pppp1ppp/2n2n2/4p2Q/2B1P3/8/PPPP1PPP/RNB1K1NR w KQkq - 4 4', 'h5f7', 'scholar mate'],
  ['2r3k1/5ppp/8/8/8/8/5PPP/1Q4K1 w - - 0 1', 'b1b8', 'queen trade into mate? (Qxc8 not possible) '],
  ['k7/8/1K6/8/8/8/8/7R w - - 0 1', 'h1h8', 'rook mate in 1'],
  ['r5k1/5ppp/8/8/8/8/1Q3PPP/6K1 w - - 0 1', null, 'mate in 2 Qb8? '],
  ['4k3/8/8/8/8/8/3q4/4K3 b - - 0 1', null, 'black up queen'],
];
for (const [fen, want, name] of puzzles) {
  const t = Date.now(); const r = AI.think({ fen, time: 1500 });
  console.log(name.padEnd(40), r.move, want ? (r.move === want ? 'OK' : 'WANT ' + want) : '', 'score', r.score, 'd', r.depth, 'nodes', r.nodes, Date.now() - t + 'ms');
}
// speed from start position
for (const ms of [500, 2000, 4000]) { const r = AI.think({ fen: C.START, time: ms }); console.log('start', ms, r.move, r.depth, r.nodes, r.score); }
