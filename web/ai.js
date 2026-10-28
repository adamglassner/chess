/* Chess - computer opponent. Runs in a Web Worker so the window never stalls.
   Alpha-beta (PVS) with iterative deepening, transposition table, quiescence,
   null-move pruning, late-move reductions, killer/history ordering and a
   tapered evaluation. Weaker levels search shallower and add noise.
   (c) 2026 Ironite */
'use strict';
importScripts('chess.js');
const C = self.Chess;
const { P, N, B, R, Q, K, WHITE, F_CAP, F_PROMO } = C;

/* ----------------------------------------------------------- evaluation */
const MG = [0, 100, 320, 330, 500, 950, 0], EG = [0, 120, 300, 315, 530, 950, 0];
const PHASE = [0, 0, 1, 1, 2, 4, 0];
// Tables from White's point of view, a8..h8 first.
const T = {
  P: [0, 0, 0, 0, 0, 0, 0, 0, 50, 50, 50, 50, 50, 50, 50, 50, 10, 10, 20, 30, 30, 20, 10, 10, 5, 5, 10, 25, 25, 10, 5, 5, 0, 0, 0, 20, 20, 0, 0, 0, 5, -5, -10, 0, 0, -10, -5, 5, 5, 10, 10, -20, -20, 10, 10, 5, 0, 0, 0, 0, 0, 0, 0, 0],
  N: [-50, -40, -30, -30, -30, -30, -40, -50, -40, -20, 0, 0, 0, 0, -20, -40, -30, 0, 10, 15, 15, 10, 0, -30, -30, 5, 15, 20, 20, 15, 5, -30, -30, 0, 15, 20, 20, 15, 0, -30, -30, 5, 10, 15, 15, 10, 5, -30, -40, -20, 0, 5, 5, 0, -20, -40, -50, -40, -30, -30, -30, -30, -40, -50],
  B: [-20, -10, -10, -10, -10, -10, -10, -20, -10, 0, 0, 0, 0, 0, 0, -10, -10, 0, 5, 10, 10, 5, 0, -10, -10, 5, 5, 10, 10, 5, 5, -10, -10, 0, 10, 10, 10, 10, 0, -10, -10, 10, 10, 10, 10, 10, 10, -10, -10, 5, 0, 0, 0, 0, 5, -10, -20, -10, -10, -10, -10, -10, -10, -20],
  R: [0, 0, 0, 0, 0, 0, 0, 0, 5, 10, 10, 10, 10, 10, 10, 5, -5, 0, 0, 0, 0, 0, 0, -5, -5, 0, 0, 0, 0, 0, 0, -5, -5, 0, 0, 0, 0, 0, 0, -5, -5, 0, 0, 0, 0, 0, 0, -5, -5, 0, 0, 0, 0, 0, 0, -5, 0, 0, 0, 5, 5, 0, 0, 0],
  Q: [-20, -10, -10, -5, -5, -10, -10, -20, -10, 0, 0, 0, 0, 0, 0, -10, -10, 0, 5, 5, 5, 5, 0, -10, -5, 0, 5, 5, 5, 5, 0, -5, 0, 0, 5, 5, 5, 5, 0, -5, -10, 5, 5, 5, 5, 5, 0, -10, -10, 0, 5, 0, 0, 0, 0, -10, -20, -10, -10, -5, -5, -10, -10, -20],
  KM: [-30, -40, -40, -50, -50, -40, -40, -30, -30, -40, -40, -50, -50, -40, -40, -30, -30, -40, -40, -50, -50, -40, -40, -30, -30, -40, -40, -50, -50, -40, -40, -30, -20, -30, -30, -40, -40, -30, -30, -20, -10, -20, -20, -20, -20, -20, -20, -10, 20, 20, 0, 0, 0, 0, 20, 20, 20, 30, 10, 0, 0, 10, 30, 20],
  KE: [-50, -40, -30, -20, -20, -30, -40, -50, -30, -20, -10, 0, 0, -10, -20, -30, -30, -10, 20, 30, 30, 20, -10, -30, -30, -10, 30, 40, 40, 30, -10, -30, -30, -10, 30, 40, 40, 30, -10, -30, -30, -10, 20, 30, 30, 20, -10, -30, -30, -30, 0, 0, 0, 0, -30, -30, -50, -30, -30, -30, -30, -30, -30, -50],
};
const PASSED = [0, 5, 10, 20, 35, 60, 100, 0];
const KEYS = [null, 'P', 'N', 'B', 'R', 'Q'];
const N_OFF = [33, 31, 18, 14, -33, -31, -18, -14], B_OFF = [17, 15, -17, -15], R_OFF = [16, 1, -16, -1];

function evaluate(pos) {
  const b = pos.board;
  let mg = 0, eg = 0, phase = 0;
  const pf = [new Int8Array(8), new Int8Array(8)]; // pawns per file
  const pMin = [new Int8Array(8).fill(9), new Int8Array(8).fill(9)], pMax = [new Int8Array(8).fill(-1), new Int8Array(8).fill(-1)];
  let bishops = [0, 0], nonPawn = [0, 0], mat = [0, 0];
  for (let s = 0; s < 128; s++) {
    if (s & 0x88) { s += 7; continue; }
    const p = b[s];
    if (!p || Math.abs(p) !== P) continue;
    const c = p > 0 ? 0 : 1, f = s & 7, r = s >> 4;
    pf[c][f]++; if (r < pMin[c][f]) pMin[c][f] = r; if (r > pMax[c][f]) pMax[c][f] = r;
  }
  for (let s = 0; s < 128; s++) {
    if (s & 0x88) { s += 7; continue; }
    const p = b[s];
    if (!p) continue;
    const t = Math.abs(p), white = p > 0, c = white ? 0 : 1, sg = white ? 1 : -1;
    const r = s >> 4, f = s & 7;
    const idx = white ? (7 - r) * 8 + f : r * 8 + f;
    let m = MG[t], e = EG[t];
    phase += PHASE[t];
    mat[c] += MG[t];
    if (t !== P && t !== K) nonPawn[c] += MG[t];
    if (t === K) { m += T.KM[idx]; e += T.KE[idx]; }
    else if (t === P) {
      m += T.P[idx]; e += T.P[idx] >> 1;
      const rel = white ? r : 7 - r;
      e += rel * 4;
      // passed pawn: no enemy pawn ahead on this or adjacent files
      const o = 1 - c; let passed = true;
      for (let ff = Math.max(0, f - 1); ff <= Math.min(7, f + 1) && passed; ff++) {
        if (!pf[o][ff]) continue;
        if (white ? pMax[o][ff] > r : pMin[o][ff] < r) passed = false;
      }
      if (passed) { m += PASSED[rel] >> 1; e += PASSED[rel]; }
      if (pf[c][f] > 1) { m -= 10; e -= 15; }
      if ((f === 0 || !pf[c][f - 1]) && (f === 7 || !pf[c][f + 1])) { m -= 12; e -= 10; }
    } else {
      const v = T[KEYS[t]][idx]; m += v; e += v;
      if (t === B) bishops[c]++;
      if (t === R) {
        if (!pf[c][f]) { const open = !pf[1 - c][f]; m += open ? 18 : 9; e += open ? 10 : 5; }
        if ((white && r === 6) || (!white && r === 1)) { m += 10; e += 15; }
      }
      // light mobility for minor pieces and rooks
      if (t === N) { let n = 0; for (const d of N_OFF) { const to = s + d; if (!(to & 0x88) && (b[to] === 0 || (b[to] > 0) !== white)) n++; } m += (n - 4) * 3; e += (n - 4) * 3; }
      else if (t === B || t === R) {
        let n = 0; for (const d of (t === B ? B_OFF : R_OFF)) { let to = s + d; while (!(to & 0x88)) { if (b[to]) { if ((b[to] > 0) !== white) n++; break; } n++; to += d; } }
        m += (n - 6) * 2; e += (n - 6) * 3;
      }
    }
    mg += sg * m; eg += sg * e;
  }
  if (bishops[0] >= 2) { mg += 30; eg += 45; }
  if (bishops[1] >= 2) { mg -= 30; eg -= 45; }
  if (phase > 24) phase = 24;
  let score = (mg * phase + eg * (24 - phase)) / 24;
  // Endgames: help the stronger side drive the lone king to the edge.
  const diff = mat[0] - mat[1];
  if (phase <= 8 && Math.abs(diff) >= 300) {
    const strong = diff > 0 ? 0 : 1, wk = pos.kings[strong], lk = pos.kings[1 - strong];
    const lf = lk & 7, lr = lk >> 4;
    const centre = Math.max(3 - lf, lf - 4) + Math.max(3 - lr, lr - 4);
    const dist = Math.abs((wk & 7) - lf) + Math.abs((wk >> 4) - lr);
    const bonus = (centre * 12 + (14 - dist) * 5) * (24 - phase) / 24;
    score += strong === 0 ? bonus : -bonus;
  }
  // Without pawns, a side up by only a minor piece can't win.
  if (!pf[0].some(Boolean) && !pf[1].some(Boolean) && Math.abs(nonPawn[0] - nonPawn[1]) < 400 && Math.max(nonPawn[0], nonPawn[1]) <= 330) score /= 8;
  score += 10 * pos.side; // tempo
  return Math.round(score) * pos.side;
}

/* ------------------------------------------------------------- search */
const MATE = 30000, INF = 32000;
const TT_BITS = 20, TT_SIZE = 1 << TT_BITS, TT_MASK = TT_SIZE - 1;
const ttKey = new Uint32Array(TT_SIZE), ttMove = new Int32Array(TT_SIZE), ttScore = new Int16Array(TT_SIZE), ttDepth = new Int8Array(TT_SIZE), ttFlag = new Uint8Array(TT_SIZE);
const EXACT = 1, LOWER = 2, UPPER = 3;
const enc = m => m.from | (m.to << 7) | ((m.promo ? Math.abs(m.promo) : 0) << 14);
const VAL = [0, 100, 320, 330, 500, 950, 2000];

let killers, history, nodes, stopAt, stopped, pos, rootBest;

function seeOrder(m) {
  if (m.captured) return 100000 + VAL[Math.abs(m.captured)] * 10 - VAL[Math.abs(m.piece)] / 10;
  return 0;
}

function orderMoves(moves, ply, tmove) {
  for (const m of moves) {
    let s;
    if (tmove && enc(m) === tmove) s = 1e9;
    else if (m.flags & F_PROMO) s = Math.abs(m.promo) === Q ? 200000 : -1000;
    else if (m.captured) s = seeOrder(m);
    else if (killers[ply][0] === enc(m)) s = 90000;
    else if (killers[ply][1] === enc(m)) s = 80000;
    else s = history[(m.piece > 0 ? 0 : 1) * 16384 + m.from * 128 + m.to];
    m.s = s;
  }
  moves.sort((a, b) => b.s - a.s);
}

function isDrawish() {
  if (pos.half >= 100) return true;
  const ks = pos.keys, n = ks.length - 1, k = ks[n];
  for (let i = n - 2; i >= 0 && i >= n - pos.half; i -= 2) if (ks[i] === k) return true;
  return false;
}

function hasNonPawn(side) {
  const b = pos.board;
  for (let s = 0; s < 128; s++) {
    if (s & 0x88) { s += 7; continue; }
    const p = b[s] * side;
    if (p > 1 && p < 6) return true;
  }
  return false;
}

function quiesce(alpha, beta, ply) {
  if ((++nodes & 2047) === 0 && Date.now() > stopAt) stopped = true;
  if (stopped) return 0;
  const stand = evaluate(pos);
  if (stand >= beta) return stand;
  if (stand > alpha) alpha = stand;
  if (ply > 60) return stand;
  const moves = pos.pseudo(true);
  for (const m of moves) m.s = m.captured ? seeOrder(m) : (Math.abs(m.promo) === Q ? 150000 : -1);
  moves.sort((a, b) => b.s - a.s);
  const us = pos.side;
  for (const m of moves) {
    if (m.s < 0) continue; // under-promotions
    if (m.captured && !(m.flags & F_PROMO) && stand + VAL[Math.abs(m.captured)] + 200 < alpha) continue; // delta pruning
    pos.make(m);
    if (pos.attacked(pos.kings[us === WHITE ? 0 : 1], -us)) { pos.unmake(); continue; }
    const sc = -quiesce(-beta, -alpha, ply + 1);
    pos.unmake();
    if (stopped) return 0;
    if (sc >= beta) return sc;
    if (sc > alpha) alpha = sc;
  }
  return alpha;
}

function search(depth, alpha, beta, ply, allowNull) {
  if ((++nodes & 2047) === 0 && Date.now() > stopAt) stopped = true;
  if (stopped) return 0;
  if (ply > 0 && isDrawish()) return 0;
  const us = pos.side, inCheck = pos.inCheck();
  if (inCheck) depth++;
  if (depth <= 0) return quiesce(alpha, beta, ply);

  // mate distance pruning
  if (ply > 0) { alpha = Math.max(alpha, -MATE + ply); beta = Math.min(beta, MATE - ply - 1); if (alpha >= beta) return alpha; }

  const slot = pos.h2 & TT_MASK;
  let tmove = 0;
  if (ttKey[slot] === pos.h1) {
    tmove = ttMove[slot];
    if (ply > 0 && ttDepth[slot] >= depth) {
      let sc = ttScore[slot];
      if (sc > MATE - 500) sc -= ply; else if (sc < -MATE + 500) sc += ply;
      const fl = ttFlag[slot];
      if (fl === EXACT || (fl === LOWER && sc >= beta) || (fl === UPPER && sc <= alpha)) return sc;
    }
  }
  const pvNode = beta - alpha > 1;

  // null move
  if (allowNull && !pvNode && !inCheck && depth >= 3 && ply > 0 && Math.abs(beta) < MATE - 500 && hasNonPawn(us) && evaluate(pos) >= beta) {
    pos.makeNull();
    const sc = -search(depth - 3, -beta, -beta + 1, ply + 1, false);
    pos.unmakeNull();
    if (stopped) return 0;
    if (sc >= beta) return beta;
  }

  const moves = pos.pseudo(false);
  orderMoves(moves, ply, tmove);
  let best = -INF, bestMove = null, legal = 0;
  const a0 = alpha;
  for (const m of moves) {
    pos.make(m);
    if (pos.attacked(pos.kings[us === WHITE ? 0 : 1], -us)) { pos.unmake(); continue; }
    legal++;
    let sc;
    const quiet = !m.captured && !m.promo;
    if (legal === 1) sc = -search(depth - 1, -beta, -alpha, ply + 1, true);
    else {
      let red = 0;
      if (depth >= 3 && legal > 3 && quiet && !inCheck && !pos.inCheck()) red = legal > 10 ? 2 : 1;
      sc = -search(depth - 1 - red, -alpha - 1, -alpha, ply + 1, true);
      if (sc > alpha && red) sc = -search(depth - 1, -alpha - 1, -alpha, ply + 1, true);
      if (sc > alpha && sc < beta) sc = -search(depth - 1, -beta, -alpha, ply + 1, true);
    }
    pos.unmake();
    if (stopped) return 0;
    if (sc > best) {
      best = sc; bestMove = m;
      if (ply === 0) rootBest = { m, sc };
      if (sc > alpha) {
        alpha = sc;
        if (sc >= beta) {
          if (quiet) {
            const e = enc(m);
            if (killers[ply][0] !== e) { killers[ply][1] = killers[ply][0]; killers[ply][0] = e; }
            history[(m.piece > 0 ? 0 : 1) * 16384 + m.from * 128 + m.to] += depth * depth;
          }
          break;
        }
      }
    }
  }
  if (!legal) return inCheck ? -MATE + ply : 0;
  // store
  let st = best;
  if (st > MATE - 500) st += ply; else if (st < -MATE + 500) st -= ply;
  ttKey[slot] = pos.h1; ttMove[slot] = bestMove ? enc(bestMove) : 0; ttScore[slot] = st; ttDepth[slot] = depth;
  ttFlag[slot] = best >= beta ? LOWER : best > a0 ? EXACT : UPPER;
  return best;
}

function think(opts) {
  // opts: {fen, moves (uci from fen), time (ms), maxDepth, noise (cp), blunder (0..1)}
  pos = new C.Position(opts.fen);
  for (const u of opts.moves || []) {
    const m = pos.legal().find(x => C.uci(x) === u);
    if (!m) break;
    pos.make(m);
  }
  const legal = pos.legal();
  if (!legal.length) return { move: null };
  if (legal.length === 1) return { move: C.uci(legal[0]), score: 0, depth: 0 };
  killers = Array.from({ length: 128 }, () => [0, 0]);
  history = new Int32Array(2 * 16384);
  nodes = 0; stopped = false;
  const t0 = Date.now();
  stopAt = t0 + (opts.time || 1000);
  const maxDepth = opts.maxDepth || 64;

  // Weaker players: score every root move at a shallow depth, then pick with noise.
  if (opts.noise) {
    stopAt = t0 + 20000;
    const depth = Math.max(1, Math.min(maxDepth, 3));
    const scored = [];
    const us = pos.side;
    for (const m of legal) {
      pos.make(m);
      let sc = -search(depth - 1, -INF, INF, 1, true);
      pos.unmake();
      if (m.captured && Math.random() < (opts.blunder || 0) * .5) sc -= 120; // sometimes misses a capture
      scored.push({ m, sc: sc + gauss() * opts.noise });
    }
    if (Math.random() < (opts.blunder || 0)) { // an outright careless move (never one that allows mate in one)
      const ok = scored.filter(x => x.sc > -MATE + 500);
      const pick = ok[Math.floor(Math.random() * ok.length)] || scored[0];
      return { move: C.uci(pick.m), score: Math.round(pick.sc), depth };
    }
    scored.sort((a, b) => b.sc - a.sc);
    void us;
    return { move: C.uci(scored[0].m), score: Math.round(scored[0].sc), depth };
  }

  let best = null, bestScore = 0, done = 0;
  for (let d = 1; d <= maxDepth; d++) {
    rootBest = null;
    const sc = search(d, -INF, INF, 0, true);
    if (stopped) { if (rootBest && best && rootBest.sc > bestScore + 30) best = rootBest.m; break; }
    if (rootBest) { best = rootBest.m; bestScore = sc; done = d; }
    if (Math.abs(sc) > MATE - 200) break; // found a forced mate
    if (Date.now() - t0 > (opts.time || 1000) * .55) break; // next ply won't finish
  }
  if (!best) best = legal[0];
  return { move: C.uci(best), score: bestScore, depth: done, nodes };
}

function gauss() { let u = 0, v = 0; while (!u) u = Math.random(); while (!v) v = Math.random(); return Math.sqrt(-2 * Math.log(u)) * Math.cos(2 * Math.PI * v); }

self.onmessage = (e) => {
  const d = e.data;
  try {
    const r = think(d);
    self.postMessage(Object.assign({ id: d.id }, r));
  } catch (err) {
    self.postMessage({ id: d.id, error: String(err && err.stack || err) });
  }
};
// expose for tests run outside a worker
self.ChessAI = { think, evaluate };
