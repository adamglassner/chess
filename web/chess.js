/* Chess rules engine - (c) 2026 Ironite.
 * 0x88 board. Pieces: 1 pawn, 2 knight, 3 bishop, 4 rook, 5 queen, 6 king;
 * white positive, black negative. Square = rank * 16 + file (rank 0 = rank 1).
 * Works in the page and in the AI worker. */
(function (root) {
  'use strict';
  const P = 1, N = 2, B = 3, R = 4, Q = 5, K = 6;
  const WHITE = 1, BLACK = -1;
  const F_CAP = 1, F_EP = 2, F_CASTLE = 4, F_DOUBLE = 8, F_PROMO = 16;
  const N_OFF = [33, 31, 18, 14, -33, -31, -18, -14];
  const B_OFF = [17, 15, -17, -15];
  const R_OFF = [16, 1, -16, -1];
  const K_OFF = [17, 15, -17, -15, 16, 1, -16, -1];
  const LETTERS = ' PNBRQK';
  const START = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1';

  const sqName = (s) => 'abcdefgh'[s & 7] + ((s >> 4) + 1);
  const sqFrom = (n) => (n.charCodeAt(1) - 49) * 16 + (n.charCodeAt(0) - 97);

  // Zobrist keys: two 32-bit halves so repetition and the AI's table are reliable.
  let seed = 0x9e3779b9;
  const rnd = () => { seed ^= seed << 13; seed ^= seed >>> 17; seed ^= seed << 5; return seed >>> 0; };
  const ZP1 = new Uint32Array(13 * 128), ZP2 = new Uint32Array(13 * 128);
  for (let i = 0; i < ZP1.length; i++) { ZP1[i] = rnd(); ZP2[i] = rnd(); }
  const ZC1 = new Uint32Array(16), ZC2 = new Uint32Array(16);
  for (let i = 0; i < 16; i++) { ZC1[i] = rnd(); ZC2[i] = rnd(); }
  const ZE1 = new Uint32Array(8), ZE2 = new Uint32Array(8);
  for (let i = 0; i < 8; i++) { ZE1[i] = rnd(); ZE2[i] = rnd(); }
  const ZS1 = rnd(), ZS2 = rnd();
  const zi = (p, s) => (p + 6) * 128 + s;

  // Castling rights lost when a piece moves from / to these squares.
  const CASTLE_MASK = new Uint8Array(128).fill(15);
  CASTLE_MASK[0x04] = 15 & ~3; CASTLE_MASK[0x00] = 15 & ~2; CASTLE_MASK[0x07] = 15 & ~1;
  CASTLE_MASK[0x74] = 15 & ~12; CASTLE_MASK[0x70] = 15 & ~8; CASTLE_MASK[0x77] = 15 & ~4;

  class Position {
    constructor(fen) { this.load(fen || START); }

    load(fen) {
      const parts = fen.trim().split(/\s+/);
      if (parts.length < 4) throw new Error('bad FEN');
      this.board = new Int8Array(128);
      let r = 7, f = 0;
      for (const ch of parts[0]) {
        if (ch === '/') { r--; f = 0; continue; }
        if (ch >= '1' && ch <= '8') { f += +ch; continue; }
        const t = LETTERS.indexOf(ch.toUpperCase());
        if (t < 1) throw new Error('bad FEN piece');
        this.board[r * 16 + f] = ch === ch.toUpperCase() ? t : -t;
        f++;
      }
      this.side = parts[1] === 'b' ? BLACK : WHITE;
      this.castle = 0;
      if (parts[2].includes('K')) this.castle |= 1;
      if (parts[2].includes('Q')) this.castle |= 2;
      if (parts[2].includes('k')) this.castle |= 4;
      if (parts[2].includes('q')) this.castle |= 8;
      this.ep = parts[3] === '-' ? -1 : sqFrom(parts[3]);
      this.half = +(parts[4] || 0);
      this.full = +(parts[5] || 1);
      this.kings = [0, 0];
      for (let s = 0; s < 128; s++) {
        if (s & 0x88) { s += 7; continue; }
        if (this.board[s] === K) this.kings[0] = s;
        if (this.board[s] === -K) this.kings[1] = s;
      }
      this.stack = [];
      this.computeHash();
      this.keys = [this.key()];
    }

    computeHash() {
      let h1 = 0, h2 = 0;
      for (let s = 0; s < 128; s++) {
        if (s & 0x88) { s += 7; continue; }
        const p = this.board[s];
        if (p) { h1 ^= ZP1[zi(p, s)]; h2 ^= ZP2[zi(p, s)]; }
      }
      h1 ^= ZC1[this.castle]; h2 ^= ZC2[this.castle];
      if (this.ep >= 0) { h1 ^= ZE1[this.ep & 7]; h2 ^= ZE2[this.ep & 7]; }
      if (this.side === BLACK) { h1 ^= ZS1; h2 ^= ZS2; }
      this.h1 = h1 >>> 0; this.h2 = h2 >>> 0;
    }
    key() { return this.h1 * 4294967296 + this.h2; }

    fen() {
      let out = '';
      for (let r = 7; r >= 0; r--) {
        let e = 0;
        for (let f = 0; f < 8; f++) {
          const p = this.board[r * 16 + f];
          if (!p) { e++; continue; }
          if (e) { out += e; e = 0; }
          const c = LETTERS[Math.abs(p)];
          out += p > 0 ? c : c.toLowerCase();
        }
        if (e) out += e;
        if (r) out += '/';
      }
      let c = '';
      if (this.castle & 1) c += 'K';
      if (this.castle & 2) c += 'Q';
      if (this.castle & 4) c += 'k';
      if (this.castle & 8) c += 'q';
      return `${out} ${this.side === WHITE ? 'w' : 'b'} ${c || '-'} ${this.ep >= 0 ? sqName(this.ep) : '-'} ${this.half} ${this.full}`;
    }

    attacked(sq, by) { // is sq attacked by side `by`?
      const b = this.board;
      // pawns
      const pd = by === WHITE ? -16 : 16;
      for (const d of [pd - 1, pd + 1]) {
        const s = sq + d;
        if (!(s & 0x88) && b[s] === P * by) return true;
      }
      for (const d of N_OFF) {
        const s = sq + d;
        if (!(s & 0x88) && b[s] === N * by) return true;
      }
      for (const d of K_OFF) {
        const s = sq + d;
        if (!(s & 0x88) && b[s] === K * by) return true;
      }
      for (const d of B_OFF) {
        let s = sq + d;
        while (!(s & 0x88)) {
          const p = b[s];
          if (p) { if (p === B * by || p === Q * by) return true; break; }
          s += d;
        }
      }
      for (const d of R_OFF) {
        let s = sq + d;
        while (!(s & 0x88)) {
          const p = b[s];
          if (p) { if (p === R * by || p === Q * by) return true; break; }
          s += d;
        }
      }
      return false;
    }

    inCheck(side) {
      side = side || this.side;
      return this.attacked(this.kings[side === WHITE ? 0 : 1], -side);
    }

    // Pseudo-legal moves. capturesOnly for the AI's quiescence search.
    pseudo(capturesOnly) {
      const b = this.board, us = this.side, moves = [];
      const add = (from, to, piece, captured, flags, promo) => moves.push({ from, to, piece, captured, flags, promo: promo || 0 });
      for (let s = 0; s < 128; s++) {
        if (s & 0x88) { s += 7; continue; }
        const p = b[s];
        if (!p || (p > 0) !== (us > 0)) continue;
        const t = p * us;
        if (t === P) {
          const fwd = 16 * us, startRank = us === WHITE ? 1 : 6, lastRank = us === WHITE ? 7 : 0;
          const one = s + fwd;
          if (!(one & 0x88) && !b[one]) {
            if ((one >> 4) === lastRank) {
              for (const pr of [Q, R, B, N]) add(s, one, p, 0, F_PROMO, pr * us);
            } else if (!capturesOnly) {
              add(s, one, p, 0, 0);
              const two = one + fwd;
              if ((s >> 4) === startRank && !b[two]) add(s, two, p, 0, F_DOUBLE);
            }
          }
          for (const d of [fwd - 1, fwd + 1]) {
            const to = s + d;
            if (to & 0x88) continue;
            const c = b[to];
            if (c && (c > 0) !== (us > 0)) {
              if ((to >> 4) === lastRank) for (const pr of [Q, R, B, N]) add(s, to, p, c, F_CAP | F_PROMO, pr * us);
              else add(s, to, p, c, F_CAP);
            } else if (to === this.ep) add(s, to, p, -us * P, F_CAP | F_EP);
          }
          continue;
        }
        if (t === N || t === K) {
          for (const d of (t === N ? N_OFF : K_OFF)) {
            const to = s + d;
            if (to & 0x88) continue;
            const c = b[to];
            if (!c) { if (!capturesOnly) add(s, to, p, 0, 0); }
            else if ((c > 0) !== (us > 0)) add(s, to, p, c, F_CAP);
          }
          if (t === K && !capturesOnly) {
            const home = us === WHITE ? 0x04 : 0x74;
            if (s === home) {
              const kBit = us === WHITE ? 1 : 4, qBit = us === WHITE ? 2 : 8;
              if ((this.castle & kBit) && !b[s + 1] && !b[s + 2] && b[s + 3] === R * us &&
                !this.attacked(s, -us) && !this.attacked(s + 1, -us) && !this.attacked(s + 2, -us))
                add(s, s + 2, p, 0, F_CASTLE);
              if ((this.castle & qBit) && !b[s - 1] && !b[s - 2] && !b[s - 3] && b[s - 4] === R * us &&
                !this.attacked(s, -us) && !this.attacked(s - 1, -us) && !this.attacked(s - 2, -us))
                add(s, s - 2, p, 0, F_CASTLE);
            }
          }
          continue;
        }
        const dirs = t === B ? B_OFF : t === R ? R_OFF : K_OFF;
        for (const d of dirs) {
          let to = s + d;
          while (!(to & 0x88)) {
            const c = b[to];
            if (!c) { if (!capturesOnly) add(s, to, p, 0, 0); }
            else { if ((c > 0) !== (us > 0)) add(s, to, p, c, F_CAP); break; }
            to += d;
          }
        }
      }
      return moves;
    }

    make(m) {
      const b = this.board, us = this.side;
      this.stack.push({ m, castle: this.castle, ep: this.ep, half: this.half, h1: this.h1, h2: this.h2 });
      let h1 = this.h1 ^ ZC1[this.castle], h2 = this.h2 ^ ZC2[this.castle];
      if (this.ep >= 0) { h1 ^= ZE1[this.ep & 7]; h2 ^= ZE2[this.ep & 7]; }
      // remove mover
      h1 ^= ZP1[zi(m.piece, m.from)]; h2 ^= ZP2[zi(m.piece, m.from)];
      b[m.from] = 0;
      if (m.flags & F_EP) {
        const cs = m.to - 16 * us;
        h1 ^= ZP1[zi(b[cs], cs)]; h2 ^= ZP2[zi(b[cs], cs)];
        b[cs] = 0;
      } else if (m.captured) {
        h1 ^= ZP1[zi(m.captured, m.to)]; h2 ^= ZP2[zi(m.captured, m.to)];
      }
      const placed = m.promo || m.piece;
      b[m.to] = placed;
      h1 ^= ZP1[zi(placed, m.to)]; h2 ^= ZP2[zi(placed, m.to)];
      if (m.flags & F_CASTLE) {
        const rf = m.to > m.from ? m.from + 3 : m.from - 4, rt = m.to > m.from ? m.from + 1 : m.from - 1;
        const rook = b[rf];
        b[rf] = 0; b[rt] = rook;
        h1 ^= ZP1[zi(rook, rf)] ^ ZP1[zi(rook, rt)]; h2 ^= ZP2[zi(rook, rf)] ^ ZP2[zi(rook, rt)];
      }
      if (m.piece * us === K) this.kings[us === WHITE ? 0 : 1] = m.to;
      this.castle &= CASTLE_MASK[m.from] & CASTLE_MASK[m.to];
      this.ep = (m.flags & F_DOUBLE) ? m.from + 16 * us : -1;
      this.half = (m.piece * us === P || m.captured) ? 0 : this.half + 1;
      if (us === BLACK) this.full++;
      this.side = -us;
      h1 ^= ZC1[this.castle] ^ ZS1; h2 ^= ZC2[this.castle] ^ ZS2;
      if (this.ep >= 0) { h1 ^= ZE1[this.ep & 7]; h2 ^= ZE2[this.ep & 7]; }
      this.h1 = h1 >>> 0; this.h2 = h2 >>> 0;
      this.keys.push(this.key());
    }

    unmake() {
      const st = this.stack.pop(), m = st.m, b = this.board;
      this.keys.pop();
      this.side = -this.side;
      const us = this.side;
      if (us === BLACK) this.full--;
      b[m.from] = m.piece;
      if (m.flags & F_EP) { b[m.to] = 0; b[m.to - 16 * us] = m.captured; }
      else b[m.to] = m.captured;
      if (m.flags & F_CASTLE) {
        const rf = m.to > m.from ? m.from + 3 : m.from - 4, rt = m.to > m.from ? m.from + 1 : m.from - 1;
        b[rf] = b[rt]; b[rt] = 0;
      }
      if (m.piece * us === K) this.kings[us === WHITE ? 0 : 1] = m.from;
      this.castle = st.castle; this.ep = st.ep; this.half = st.half; this.h1 = st.h1; this.h2 = st.h2;
      return m;
    }

    // Null move for the AI.
    makeNull() {
      this.stack.push({ m: null, castle: this.castle, ep: this.ep, half: this.half, h1: this.h1, h2: this.h2 });
      let h1 = this.h1 ^ ZS1, h2 = this.h2 ^ ZS2;
      if (this.ep >= 0) { h1 ^= ZE1[this.ep & 7]; h2 ^= ZE2[this.ep & 7]; }
      this.ep = -1; this.side = -this.side; this.half++;
      this.h1 = h1 >>> 0; this.h2 = h2 >>> 0;
      this.keys.push(this.key());
    }
    unmakeNull() {
      const st = this.stack.pop();
      this.keys.pop();
      this.side = -this.side;
      this.castle = st.castle; this.ep = st.ep; this.half = st.half; this.h1 = st.h1; this.h2 = st.h2;
    }

    legal() {
      const out = [], us = this.side;
      for (const m of this.pseudo(false)) {
        this.make(m);
        if (!this.attacked(this.kings[us === WHITE ? 0 : 1], -us)) out.push(m);
        this.unmake();
      }
      return out;
    }

    repetitions() { // how many times the current position has occurred
      const k = this.keys[this.keys.length - 1];
      let n = 0;
      for (let i = this.keys.length - 1; i >= 0; i -= 2) if (this.keys[i] === k) n++;
      return n;
    }

    insufficient() {
      const minors = { w: [], b: [] };
      for (let s = 0; s < 128; s++) {
        if (s & 0x88) { s += 7; continue; }
        const p = this.board[s], t = Math.abs(p);
        if (!p || t === K) continue;
        if (t === P || t === R || t === Q) return false;
        minors[p > 0 ? 'w' : 'b'].push({ t, color: ((s >> 4) + (s & 7)) & 1 });
      }
      const all = minors.w.concat(minors.b);
      if (all.length <= 1) return true; // K v K, K+minor v K
      if (all.every(m => m.t === B) && all.every(m => m.color === all[0].color)) return true; // bishops on one colour
      return false;
    }

    // Game status for the current side: {over, result, reason}
    status() {
      const moves = this.legal();
      if (!moves.length) {
        if (this.inCheck()) return { over: true, result: this.side === WHITE ? '0-1' : '1-0', reason: 'checkmate', winner: -this.side };
        return { over: true, result: '1/2-1/2', reason: 'stalemate', winner: 0 };
      }
      if (this.insufficient()) return { over: true, result: '1/2-1/2', reason: 'insufficient material', winner: 0 };
      if (this.half >= 100) return { over: true, result: '1/2-1/2', reason: 'fifty-move rule', winner: 0 };
      if (this.repetitions() >= 3) return { over: true, result: '1/2-1/2', reason: 'threefold repetition', winner: 0 };
      return { over: false, moves };
    }

    // Standard algebraic notation for a legal move in the current position.
    san(m, legalMoves) {
      const t = Math.abs(m.piece);
      let s;
      if (m.flags & F_CASTLE) s = m.to > m.from ? 'O-O' : 'O-O-O';
      else {
        s = '';
        if (t !== P) {
          s = LETTERS[t];
          const others = (legalMoves || this.legal()).filter(o => o.to === m.to && o.piece === m.piece && o.from !== m.from);
          if (others.length) {
            const sameFile = others.some(o => (o.from & 7) === (m.from & 7));
            const sameRank = others.some(o => (o.from >> 4) === (m.from >> 4));
            if (!sameFile) s += 'abcdefgh'[m.from & 7];
            else if (!sameRank) s += (m.from >> 4) + 1;
            else s += sqName(m.from);
          }
        } else if (m.captured) s = 'abcdefgh'[m.from & 7];
        if (m.captured) s += 'x';
        s += sqName(m.to);
        if (m.promo) s += '=' + LETTERS[Math.abs(m.promo)];
      }
      this.make(m);
      if (this.inCheck()) s += this.legal().length ? '+' : '#';
      this.unmake();
      return s;
    }

    perft(d) {
      if (d === 0) return 1;
      let n = 0;
      const us = this.side;
      for (const m of this.pseudo(false)) {
        this.make(m);
        if (!this.attacked(this.kings[us === WHITE ? 0 : 1], -us)) n += d === 1 ? 1 : this.perft(d - 1);
        this.unmake();
      }
      return n;
    }
  }

  const sameMove = (a, b) => a && b && a.from === b.from && a.to === b.to && (a.promo || 0) === (b.promo || 0);
  const uci = (m) => sqName(m.from) + sqName(m.to) + (m.promo ? 'nbrq'[Math.abs(m.promo) - 2] : '');

  root.Chess = { Position, P, N, B, R, Q, K, WHITE, BLACK, F_CAP, F_EP, F_CASTLE, F_DOUBLE, F_PROMO, START, sqName, sqFrom, sameMove, uci, LETTERS };
})(typeof self !== 'undefined' ? self : this);
