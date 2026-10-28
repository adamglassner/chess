/* Chess - a small opening book. The computer picks from these lines so games
   start differently, and the score sheet names the opening being played.
   (c) 2026 Ironite */
'use strict';
const BOOK = (() => {
const LINES = [
  ['Italian Game', 'e2e4 e7e5 g1f3 b8c6 f1c4 f8c5 c2c3 g8f6 d2d3 d7d6 e1g1 e8g8'],
  ['Italian Game: Two Knights', 'e2e4 e7e5 g1f3 b8c6 f1c4 g8f6 d2d3 f8e7 e1g1 e8g8'],
  ['Ruy Lopez', 'e2e4 e7e5 g1f3 b8c6 f1b5 a7a6 b5a4 g8f6 e1g1 f8e7 f1e1 b7b5 a4b3 d7d6'],
  ['Ruy Lopez: Berlin Defence', 'e2e4 e7e5 g1f3 b8c6 f1b5 g8f6 e1g1 f6e4 d2d4 e4d6'],
  ['Scotch Game', 'e2e4 e7e5 g1f3 b8c6 d2d4 e5d4 f3d4 g8f6 d4c6 b7c6'],
  ['Four Knights Game', 'e2e4 e7e5 g1f3 b8c6 b1c3 g8f6 f1b5 f8b4 e1g1 e8g8'],
  ['Petrov Defence', 'e2e4 e7e5 g1f3 g8f6 f3e5 d7d6 e5f3 f6e4 d2d4 d6d5'],
  ['Sicilian Defence: Najdorf', 'e2e4 c7c5 g1f3 d7d6 d2d4 c5d4 f3d4 g8f6 b1c3 a7a6'],
  ['Sicilian Defence: Classical', 'e2e4 c7c5 g1f3 b8c6 d2d4 c5d4 f3d4 g8f6 b1c3 d7d6'],
  ['Sicilian Defence: Alapin', 'e2e4 c7c5 c2c3 g8f6 e4e5 f6d5 d2d4 c5d4 g1f3 b8c6'],
  ['French Defence', 'e2e4 e7e6 d2d4 d7d5 b1c3 g8f6 c1g5 f8e7 e4e5 f6d7'],
  ['French Defence: Advance', 'e2e4 e7e6 d2d4 d7d5 e4e5 c7c5 c2c3 b8c6 g1f3 d8b6'],
  ['Caro-Kann Defence', 'e2e4 c7c6 d2d4 d7d5 b1c3 d5e4 c3e4 c8f5 e4g3 f5g6'],
  ['Scandinavian Defence', 'e2e4 d7d5 e4d5 d8d5 b1c3 d5a5 d2d4 g8f6 g1f3 c8f5'],
  ['Pirc Defence', 'e2e4 d7d6 d2d4 g8f6 b1c3 g7g6 g1f3 f8g7 f1e2 e8g8'],
  ["Queen's Gambit Declined", 'd2d4 d7d5 c2c4 e7e6 b1c3 g8f6 c1g5 f8e7 e2e3 e8g8 g1f3 h7h6'],
  ["Queen's Gambit Accepted", 'd2d4 d7d5 c2c4 d5c4 g1f3 g8f6 e2e3 e7e6 f1c4 c7c5'],
  ['Slav Defence', 'd2d4 d7d5 c2c4 c7c6 g1f3 g8f6 b1c3 d5c4 a2a4 c8f5'],
  ['London System', 'd2d4 d7d5 c1f4 g8f6 e2e3 e7e6 g1f3 c7c5 c2c3 b8c6'],
  ["King's Indian Defence", 'd2d4 g8f6 c2c4 g7g6 b1c3 f8g7 e2e4 d7d6 g1f3 e8g8 f1e2 e7e5'],
  ['Nimzo-Indian Defence', 'd2d4 g8f6 c2c4 e7e6 b1c3 f8b4 e2e3 e8g8 f1d3 d7d5'],
  ["Queen's Indian Defence", 'd2d4 g8f6 c2c4 e7e6 g1f3 b7b6 g2g3 c8b7 f1g2 f8e7'],
  ['Grünfeld Defence', 'd2d4 g8f6 c2c4 g7g6 b1c3 d7d5 c4d5 f6d5 e2e4 d5c3 b2c3 f8g7'],
  ['Dutch Defence', 'd2d4 f7f5 g2g3 g8f6 f1g2 e7e6 g1f3 f8e7 e1g1 e8g8'],
  ['English Opening', 'c2c4 e7e5 b1c3 g8f6 g1f3 b8c6 g2g3 d7d5 c4d5 f6d5'],
  ['English Opening: Symmetrical', 'c2c4 c7c5 b1c3 b8c6 g2g3 g7g6 f1g2 f8g7 g1f3 g8f6'],
  ['Réti Opening', 'g1f3 d7d5 c2c4 e7e6 g2g3 g8f6 f1g2 f8e7 e1g1 e8g8'],
  ["King's Gambit", 'e2e4 e7e5 f2f4 e5f4 g1f3 g7g5 h2h4 g5g4 f3e5'],
  ['Vienna Game', 'e2e4 e7e5 b1c3 g8f6 g2g3 d7d5 e4d5 f6d5 f1g2 d5c3 b2c3'],
  ['Alekhine Defence', 'e2e4 g8f6 e4e5 f6d5 d2d4 d7d6 g1f3 c8g4 f1e2 e7e6'],
];
const lines = LINES.map(([name, s]) => ({ name, moves: s.split(' ') }));
// Book continuations after the given moves (UCI), each with the line's name.
function next(played) {
  const out = [];
  for (const l of lines) {
    if (played.length >= l.moves.length) continue;
    let ok = true;
    for (let i = 0; i < played.length; i++) if (l.moves[i] !== played[i]) { ok = false; break; }
    if (ok) out.push({ move: l.moves[played.length], name: l.name });
  }
  return out;
}
// The most specific opening name that matches the moves so far.
function name(played) {
  let best = null, bestLen = 0;
  for (const l of lines) {
    let n = 0;
    while (n < played.length && n < l.moves.length && l.moves[n] === played[n]) n++;
    if (n < 2 || !(n === played.length || n === l.moves.length) || n <= bestLen) continue;
    best = n >= 8 ? l.name : l.name.split(':')[0]; bestLen = n;
  }
  return best;
}
return { next, name, lines };
})();
