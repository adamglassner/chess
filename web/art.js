/* Chess - art. Every piece, board, the library desk, the opponents' portraits
   and the game's icon are drawn here in code at whatever resolution the
   screen needs, so everything stays sharp from 720p up to 4K.
   (c) 2026 Ironite */
'use strict';
const ART = (() => {
const D = IR.draw;

/* ------------------------------------------------------------ helpers */
const rng = (seed) => { let s = seed >>> 0 || 1; return () => { s ^= s << 13; s >>>= 0; s ^= s >> 17; s ^= s << 5; s >>>= 0; return s / 4294967296; }; };
const canvas = (w, h) => { const c = document.createElement('canvas'); c.width = Math.max(1, Math.ceil(w)); c.height = Math.max(1, Math.ceil(h)); return [c, c.getContext('2d')]; };

// Smooth closed path through points; points flagged as corners stay sharp.
function spline(x, pts, closed = true) {
  const n = pts.length, get = i => pts[(i + n) % n];
  x.moveTo(pts[0][0], pts[0][1]);
  const last = closed ? n : n - 1;
  for (let i = 0; i < last; i++) {
    const p0 = get(i - 1), p1 = get(i), p2 = get(i + 1), p3 = get(i + 2);
    const a = p1[2] ? p1 : p0, d = p2[2] ? p2 : p3;
    const t = 1 / 6;
    x.bezierCurveTo(p1[0] + (p2[0] - a[0]) * t, p1[1] + (p2[1] - a[1]) * t, p2[0] - (d[0] - p1[0]) * t, p2[1] - (d[1] - p1[1]) * t, p2[0], p2[1]);
  }
  if (closed) x.closePath();
}

/* ------------------------------------------------------------- pieces
   Unit square: x 0..1, y 0..1 (1 = bottom of the square). Turned pieces are
   given as a right-hand profile [y, halfWidth, corner, ring] from the base up. */
const BASE = [[.905, .3, 1], [.885, .318], [.86, .318], [.84, .3, 1, 1], [.825, .262, 1], [.81, .256], [.79, .232, 1, 1]];
const PROFILES = {
  P: { body: BASE.concat([[.775, .2, 1], [.74, .17], [.66, .125], [.58, .1, 1], [.565, .158, 1, 1], [.545, .17], [.525, .158, 1], [.512, .1, 1]]), ball: [.5, .405, .118] },
  R: { body: BASE.concat([[.775, .228, 1], [.72, .206], [.6, .178], [.48, .174], [.4, .19], [.355, .218, 1, 1], [.335, .238, 1], [.305, .238, 1, 1], [.29, .246, 1]]), top: 'rook' },
  B: { body: BASE.concat([[.775, .205, 1], [.7, .158], [.6, .113], [.53, .094, 1], [.518, .158, 1, 1], [.5, .166], [.48, .15, 1], [.468, .094, 1], [.45, .118], [.39, .153], [.32, .156], [.25, .128], [.195, .082], [.16, .026], [.15, 0, 1]]), ball: [.5, .128, .036], slit: true },
  Q: { body: BASE.concat([[.775, .222, 1], [.7, .176], [.58, .124], [.46, .106], [.415, .106, 1], [.402, .166, 1, 1], [.385, .174], [.366, .156, 1], [.354, .118, 1], [.31, .14], [.25, .2], [.215, .228, 1]]), top: 'queen' },
  K: { body: BASE.concat([[.775, .222, 1], [.7, .178], [.58, .13], [.44, .116], [.395, .116, 1], [.382, .176, 1, 1], [.365, .184], [.346, .164, 1], [.334, .126, 1], [.29, .146], [.24, .184], [.215, .198, 1]]), top: 'king' },
};

function bodyPoints(prof) {
  const right = prof.body.map(([y, w, c]) => [.5 + w, y, c]);
  const top = topPoints(prof);
  const left = prof.body.slice().reverse().map(([y, w, c]) => [.5 - w, y, c]);
  // flat bottom: centre bottom point keeps the base edge straight
  return right.concat(top, left, [[.5, .905, 1]]);
}

function topPoints(prof) {
  const last = prof.body[prof.body.length - 1], y = last[0], w = last[1];
  if (prof.top === 'rook') {
    // four merlons with three notches
    const t = .17, nd = .228, pts = [];
    const xs = [.5 + w, .5 + w * .62, .5 + w * .32, .5 - w * .32, .5 - w * .62, .5 - w];
    pts.push([xs[0], t, 1]);
    for (let i = 1; i < 5; i++) {
      const x0 = xs[i] + (i <= 2 ? .022 : -.022), x1 = xs[i] - (i <= 2 ? .022 : -.022);
      pts.push([x0, t, 1], [x0, nd, 1], [x1, nd, 1], [x1, t, 1]);
    }
    pts.push([xs[5], t, 1]);
    return pts;
  }
  if (prof.top === 'queen') {
    // coronet with seven points
    const pts = [], n = 7, top = .148;
    for (let i = 0; i <= n * 2; i++) {
      const u = i / (n * 2), xx = .5 + w - u * w * 2;
      const tip = i % 2 === 0, edge = i === 0 || i === n * 2;
      pts.push([xx, edge ? top + .006 : tip ? top - .012 * Math.sin(u * Math.PI) : top + .062, 1]);
    }
    return pts;
  }
  if (prof.top === 'king') {
    return [[.5 + w, y - .012, 1], [.5 + w * .7, y - .052], [.5, y - .064], [.5 - w * .7, y - .052], [.5 - w, y - .012, 1]];
  }
  return [];
}

const PALETTES = {
  ivory: { edge: '#a7865d', mid: '#e9d8b7', light: '#f8eedb', hi: '#fffdf6', line: '#6b5134', ring: 'rgba(110,80,45,.5)', ringHi: 'rgba(255,255,255,.75)', shadow: 'rgba(30,15,5,.55)', grainA: 'rgba(160,120,70,.10)' },
  ebony: { edge: '#070403', mid: '#2b1d16', light: '#4a362a', hi: '#a88a72', line: '#020101', ring: 'rgba(0,0,0,.7)', ringHi: 'rgba(230,190,150,.35)', shadow: 'rgba(0,0,0,.7)', grainA: 'rgba(120,80,50,.10)' },
};

function shadeFill(x, pal, l = .2, r = .8) {
  const g = x.createLinearGradient(l, 0, r, 0);
  g.addColorStop(0, pal.edge); g.addColorStop(.18, pal.mid); g.addColorStop(.36, pal.light); g.addColorStop(.46, pal.hi);
  g.addColorStop(.58, pal.light); g.addColorStop(.82, pal.mid); g.addColorStop(1, pal.edge);
  x.fillStyle = g; x.fill();
  // darker towards the base, lifted at the top (lamp is above)
  const v = x.createLinearGradient(0, .05, 0, .92);
  v.addColorStop(0, 'rgba(255,240,215,.16)'); v.addColorStop(.6, 'rgba(0,0,0,0)'); v.addColorStop(1, 'rgba(0,0,0,.22)');
  x.fillStyle = v; x.fill();
}

function ringLines(x, prof, pal, lw) {
  for (const [y, w, , ring] of prof.body) {
    if (!ring) continue;
    x.beginPath(); x.ellipse(.5, y - .004, w * .98, .014, 0, .15, Math.PI - .15); x.strokeStyle = pal.ring; x.lineWidth = lw * 1.1; x.stroke();
    x.beginPath(); x.ellipse(.5, y - .014, w * .9, .012, 0, .35, Math.PI - .35); x.strokeStyle = pal.ringHi; x.lineWidth = lw * .8; x.stroke();
  }
}

function sphere(x, cx, cy, r, pal, lw) {
  x.beginPath(); x.arc(cx, cy, r, 0, Math.PI * 2);
  const g = x.createRadialGradient(cx - r * .38, cy - r * .42, r * .08, cx, cy, r * 1.05);
  g.addColorStop(0, pal.hi); g.addColorStop(.35, pal.light); g.addColorStop(.75, pal.mid); g.addColorStop(1, pal.edge);
  x.fillStyle = g; x.fill();
  x.strokeStyle = pal.line; x.lineWidth = lw; x.stroke();
}

function drawTurned(x, t, pal, lw) {
  const prof = PROFILES[t];
  x.beginPath(); spline(x, bodyPoints(prof)); shadeFill(x, pal);
  x.save(); x.clip();
  // fine turned-wood grain
  const r = rng(t.charCodeAt(0) * 99);
  for (let i = 0; i < 26; i++) { const yy = .1 + r() * .8; x.beginPath(); x.ellipse(.5, yy, .4, .01 + r() * .01, 0, 0, Math.PI); x.strokeStyle = pal.grainA; x.lineWidth = lw * .6; x.stroke(); }
  ringLines(x, prof, pal, lw);
  // specular streak
  const sp = x.createLinearGradient(.36, 0, .47, 0); sp.addColorStop(0, 'rgba(255,255,255,0)'); sp.addColorStop(.5, 'rgba(255,255,255,.22)'); sp.addColorStop(1, 'rgba(255,255,255,0)');
  x.fillStyle = sp; x.fillRect(.36, 0, .11, .9);
  x.restore();
  x.beginPath(); spline(x, bodyPoints(prof)); x.strokeStyle = pal.line; x.lineWidth = lw; x.lineJoin = 'round'; x.stroke();
  if (prof.slit) { // bishop's mitre cut
    x.beginPath(); x.moveTo(.565, .235); x.quadraticCurveTo(.52, .27, .455, .318);
    x.strokeStyle = pal.line; x.lineWidth = lw * 2.4; x.lineCap = 'round'; x.stroke();
    x.beginPath(); x.moveTo(.57, .245); x.quadraticCurveTo(.525, .282, .462, .33);
    x.strokeStyle = pal.ringHi; x.lineWidth = lw * .9; x.stroke();
  }
  if (prof.ball) sphere(x, prof.ball[0], prof.ball[1], prof.ball[2], pal, lw);
  if (prof.top === 'queen') {
    const w = prof.body[prof.body.length - 1][1];
    for (let i = 0; i <= 7; i++) { const u = i / 7; sphere(x, .5 + w - u * w * 2, .148 - .012 * Math.sin(u * Math.PI) - .012, .021, pal, lw * .8); }
    // little dome and finial
    x.beginPath(); x.ellipse(.5, .2, .085, .05, 0, Math.PI, 0); x.closePath(); shadeFill(x, pal, .4, .6); x.strokeStyle = pal.line; x.lineWidth = lw; x.stroke();
    sphere(x, .5, .12, .036, pal, lw);
  }
  if (prof.top === 'king') {
    // cross: vertical then bar, both bevelled
    const bar = (x0, y0, w0, h0) => {
      D.rr(x, x0, y0, w0, h0, Math.min(w0, h0) * .25);
      const g = x.createLinearGradient(x0, 0, x0 + w0, 0);
      g.addColorStop(0, pal.mid); g.addColorStop(.4, pal.hi); g.addColorStop(1, pal.edge);
      x.fillStyle = g; x.fill(); x.strokeStyle = pal.line; x.lineWidth = lw; x.stroke();
    };
    bar(.468, .015, .064, .155);
    bar(.415, .055, .17, .05);
    x.beginPath(); x.ellipse(.5, .158, .06, .022, 0, 0, Math.PI * 2); shadeFill(x, pal, .44, .56); x.strokeStyle = pal.line; x.lineWidth = lw; x.stroke();
  }
}

// The knight: a carved horse's head on the turned base, facing left.
const KNIGHT = [[.775, .79, 1], [.79, .7], [.79, .6], [.765, .48], [.715, .37], [.66, .275], [.635, .2], [.632, .135, 1], [.6, .175], [.572, .2, 1], [.555, .12, 1], [.52, .16], [.49, .19], [.42, .215], [.35, .265], [.275, .345], [.215, .42], [.17, .485, 1], [.165, .53], [.19, .565, 1], [.25, .575], [.32, .545], [.375, .52], [.42, .51, 1], [.44, .545], [.4, .6], [.33, .67], [.27, .735], [.245, .79, 1]];
function drawKnight(x, pal, lw) {
  // base
  const base = BASE.map(([y, w, c]) => [.5 + w, y, c]).concat(BASE.slice().reverse().map(([y, w, c]) => [.5 - w, y, c]), [[.5, .905, 1]]);
  x.beginPath(); spline(x, base); shadeFill(x, pal); x.strokeStyle = pal.line; x.lineWidth = lw; x.stroke();
  x.save(); x.beginPath(); spline(x, base); x.clip(); ringLines(x, { body: BASE }, pal, lw); x.restore();
  // head
  x.beginPath(); spline(x, KNIGHT); shadeFill(x, pal, .15, .85);
  x.save(); x.clip();
  // cheek and jaw modelling
  x.fillStyle = D.rad(x, .5, .44, .01, .15, ['rgba(60,30,10,.12)', 'rgba(60,30,10,0)']); x.fillRect(.3, .28, .4, .32);
  x.fillStyle = D.rad(x, .42, .36, .01, .12, ['rgba(255,250,235,.16)', 'rgba(255,250,235,0)']); x.fillRect(.28, .22, .3, .3);
  const sp = x.createLinearGradient(.3, .25, .55, .4); sp.addColorStop(0, 'rgba(255,255,255,0)'); sp.addColorStop(.5, 'rgba(255,255,255,.2)'); sp.addColorStop(1, 'rgba(255,255,255,0)');
  x.fillStyle = sp; x.beginPath(); x.moveTo(.27, .36); x.quadraticCurveTo(.4, .24, .55, .2); x.lineTo(.58, .24); x.quadraticCurveTo(.42, .29, .3, .4); x.fill();
  // mane: carved ridges along the back of the neck
  for (let i = 0; i < 9; i++) {
    const u = i / 8, mx = .735 - Math.sin(u * 1.25) * .015 - u * .1, my = .74 - u * .5;
    x.beginPath(); x.moveTo(mx + .05, my + .02); x.quadraticCurveTo(mx - .01, my - .015, mx - .045, my - .055);
    x.strokeStyle = pal.ring; x.lineWidth = lw * 1.3; x.lineCap = 'round'; x.stroke();
    x.beginPath(); x.moveTo(mx + .055, my + .008); x.quadraticCurveTo(mx, my - .025, mx - .035, my - .062);
    x.strokeStyle = pal.ringHi; x.lineWidth = lw * .7; x.stroke();
  }
  x.restore();
  x.beginPath(); spline(x, KNIGHT); x.strokeStyle = pal.line; x.lineWidth = lw; x.lineJoin = 'round'; x.stroke();
  // ear inner, eye, nostril, mouth
  x.beginPath(); x.moveTo(.575, .195); x.quadraticCurveTo(.578, .16, .59, .14); x.strokeStyle = pal.ring; x.lineWidth = lw * 1.2; x.stroke();
  D.ell(x, .435, .3, .03, .019, pal.line, -.35);
  D.ell(x, .43, .296, .01, .007, pal.ringHi);
  x.beginPath(); x.moveTo(.385, .272); x.quadraticCurveTo(.44, .262, .475, .285); x.strokeStyle = pal.ring; x.lineWidth = lw; x.stroke();
  D.ell(x, .208, .5, .017, .022, pal.line, .5);
  x.beginPath(); x.moveTo(.19, .548); x.quadraticCurveTo(.24, .538, .28, .55); x.strokeStyle = pal.line; x.lineWidth = lw * 1.1; x.stroke();
  // bridle-less noseband line for definition
  x.beginPath(); x.moveTo(.255, .4); x.quadraticCurveTo(.3, .44, .36, .43); x.strokeStyle = pal.ring; x.lineWidth = lw * .9; x.stroke();
}

const pieceCache = new Map();
const LET = ' PNBRQK';
// Returns a canvas of a piece drawn for a square `size` device pixels across.
function piece(code, size) {
  size = Math.max(8, Math.round(size));
  const key = code + size;
  if (pieceCache.has(key)) return pieceCache.get(key);
  const [c, x] = canvas(size, size);
  const t = LET[Math.abs(code)], pal = code > 0 ? PALETTES.ivory : PALETTES.ebony;
  x.scale(size, size);
  const lw = Math.max(.006, 1.1 / size);
  // soft contact shadow
  x.save(); x.filter = `blur(${Math.max(1, size * .02)}px)`;
  D.ell(x, .5, .9, .33, .055, pal === PALETTES.ivory ? 'rgba(20,10,0,.45)' : 'rgba(0,0,0,.6)'); x.restore();
  if (t === 'N') drawKnight(x, pal, lw); else drawTurned(x, t, pal, lw);
  pieceCache.set(key, c);
  if (pieceCache.size > 200) pieceCache.delete(pieceCache.keys().next().value);
  return c;
}

/* -------------------------------------------------------------- boards */
const BOARDS = {
  walnut: { name: 'Walnut & Maple', light: ['#e6c896', '#d9b47c', '#c99f66'], dark: ['#7b4a2a', '#663a1f', '#4f2b16'], frame: ['#4a2414', '#2e140a'], inlay: '#d4a94e', coord: '#e7c77c', grain: true },
  marble: { name: 'Marble', light: ['#efebe3', '#e2ddd2', '#d3cdc1'], dark: ['#4f6b5f', '#3f574c', '#2f4239'], frame: ['#3a3633', '#1f1c1a'], inlay: '#c9b27a', coord: '#d9c690', marble: true },
  club: { name: 'Club Green', light: ['#ece4c8', '#e2d8b6', '#d5c9a2'], dark: ['#557a4c', '#4a6b42', '#3c5735'], frame: ['#3b2a1c', '#21160e'], inlay: '#c9a24c', coord: '#e3c983' },
};

function woodSquare(x, s, cols, r, vertical) {
  const g = x.createLinearGradient(0, 0, vertical ? s : 0, vertical ? 0 : s);
  g.addColorStop(0, cols[0]); g.addColorStop(.5 + (r() - .5) * .3, cols[1]); g.addColorStop(1, cols[2]);
  x.fillStyle = g; x.fillRect(0, 0, s, s);
  const lines = 26;
  for (let i = 0; i < lines; i++) {
    const p = (i + r()) / lines * s, amp = s * (.01 + r() * .03), ph = r() * 6, fr = 1 + r() * 2.5;
    x.beginPath();
    for (let k = 0; k <= 24; k++) {
      const u = k / 24 * s, w = Math.sin(u / s * Math.PI * fr + ph) * amp;
      if (vertical) (k ? x.lineTo(p + w, u) : x.moveTo(p + w, u)); else (k ? x.lineTo(u, p + w) : x.moveTo(u, p + w));
    }
    x.strokeStyle = r() < .5 ? `rgba(60,30,10,${.05 + r() * .1})` : `rgba(255,235,200,${.03 + r() * .06})`;
    x.lineWidth = s * (.003 + r() * .008); x.stroke();
  }
  // a knot now and then
  if (r() < .18) { const kx = r() * s, ky = r() * s; for (let k = 4; k > 0; k--) { x.beginPath(); x.ellipse(kx, ky, s * .02 * k, s * .008 * k, vertical ? Math.PI / 2 : 0, 0, Math.PI * 2); x.strokeStyle = `rgba(60,30,10,${.06})`; x.lineWidth = s * .004; x.stroke(); } }
}
function marbleSquare(x, s, cols, r) {
  const g = x.createLinearGradient(0, 0, s, s); g.addColorStop(0, cols[0]); g.addColorStop(.6, cols[1]); g.addColorStop(1, cols[2]);
  x.fillStyle = g; x.fillRect(0, 0, s, s);
  for (let i = 0; i < 7; i++) {
    let px = r() * s, py = r() * s, a = r() * Math.PI * 2;
    x.beginPath(); x.moveTo(px, py);
    for (let k = 0; k < 18; k++) { a += (r() - .5) * .9; px += Math.cos(a) * s * .07; py += Math.sin(a) * s * .07; x.lineTo(px, py); }
    x.strokeStyle = r() < .5 ? `rgba(255,255,255,${.12 + r() * .2})` : `rgba(0,0,0,${.05 + r() * .1})`; x.lineWidth = s * (.002 + r() * .01); x.stroke();
  }
}
function feltSquare(x, s, cols, r) {
  x.fillStyle = D.lin(x, 0, 0, s, s, cols); x.fillRect(0, 0, s, s);
  for (let i = 0; i < s * 2; i++) { x.fillStyle = r() < .5 ? 'rgba(255,255,255,.04)' : 'rgba(0,0,0,.05)'; x.fillRect(r() * s, r() * s, 1, 1); }
}

const boardCache = {};
// The 8x8 playing surface (no frame), `px` device pixels across. White at the bottom unless flipped.
function boardSurface(theme, px) {
  const key = theme + px;
  if (boardCache[key]) return boardCache[key];
  const T = BOARDS[theme] || BOARDS.walnut, s = px / 8;
  const [c, x] = canvas(px, px);
  const r = rng(theme === 'walnut' ? 1234 : theme === 'marble' ? 777 : 4242);
  for (let rank = 0; rank < 8; rank++) for (let file = 0; file < 8; file++) {
    const light = (rank + file) % 2 === 1; // a1 is dark
    const [sq, sx] = canvas(s, s);
    if (T.grain) woodSquare(sx, s, light ? T.light : T.dark, r, light ? (file % 2 === 0) : (rank % 2 === 0));
    else if (T.marble) marbleSquare(sx, s, light ? T.light : T.dark, r);
    else feltSquare(sx, s, light ? T.light : T.dark, r);
    x.drawImage(sq, file * s, (7 - rank) * s);
  }
  // inlay seams
  x.strokeStyle = 'rgba(0,0,0,.18)'; x.lineWidth = Math.max(1, px / 900);
  for (let i = 1; i < 8; i++) { x.beginPath(); x.moveTo(i * s, 0); x.lineTo(i * s, px); x.stroke(); x.beginPath(); x.moveTo(0, i * s); x.lineTo(px, i * s); x.stroke(); }
  // lacquer and lamp light
  x.fillStyle = D.rad(x, px * .22, px * .12, px * .02, px * 1.05, ['rgba(255,236,190,.22)', 'rgba(255,220,160,.05)', 'rgba(0,0,0,.18)']); x.fillRect(0, 0, px, px);
  x.fillStyle = D.lin(x, 0, 0, px, px, ['rgba(255,255,255,.07)', 'rgba(255,255,255,0)', 'rgba(255,255,255,.03)']); x.fillRect(0, 0, px, px);
  boardCache[key] = c;
  return c;
}

// Frame around the board, drawn at `px` = frame outer size; returns canvas.
function boardFrame(theme, px, inner, flipped) {
  const T = BOARDS[theme] || BOARDS.walnut;
  const [c, x] = canvas(px, px);
  const m = (px - inner) / 2;
  // drop shadow
  x.save(); x.shadowColor = 'rgba(0,0,0,.65)'; x.shadowBlur = m * .9; x.shadowOffsetY = m * .25;
  D.rr(x, m * .12, m * .12, px - m * .24, px - m * .24, m * .35); x.fillStyle = T.frame[0]; x.fill(); x.restore();
  // mitred wood frame
  D.rr(x, m * .12, m * .12, px - m * .24, px - m * .24, m * .35);
  x.fillStyle = D.lin(x, 0, 0, px, px, [T.frame[0], T.frame[1], T.frame[0]]); x.fill();
  const r = rng(99);
  x.save(); D.rr(x, m * .12, m * .12, px - m * .24, px - m * .24, m * .35); x.clip();
  for (let i = 0; i < 260; i++) { const p = r() * px; const horiz = r() < .5; x.beginPath(); if (horiz) { x.moveTo(0, p); x.bezierCurveTo(px * .3, p + (r() - .5) * 8, px * .6, p + (r() - .5) * 8, px, p); } else { x.moveTo(p, 0); x.bezierCurveTo(p + (r() - .5) * 8, px * .3, p + (r() - .5) * 8, px * .6, p, px); } x.strokeStyle = `rgba(${r() < .5 ? '0,0,0' : '255,210,160'},${.03 + r() * .05})`; x.lineWidth = 1 + r() * 2; x.stroke(); }
  x.restore();
  // bevel highlight
  D.rr(x, m * .16, m * .16, px - m * .32, px - m * .32, m * .3); x.strokeStyle = 'rgba(255,220,170,.22)'; x.lineWidth = m * .04; x.stroke();
  // brass inlay line
  x.strokeStyle = T.inlay; x.lineWidth = Math.max(1.5, m * .06);
  x.strokeRect(m * .78, m * .78, px - m * 1.56, px - m * 1.56);
  x.strokeStyle = 'rgba(255,245,200,.5)'; x.lineWidth = Math.max(.6, m * .015); x.strokeRect(m * .78 - m * .02, m * .78 - m * .02, px - m * 1.56 + m * .04, px - m * 1.56 + m * .04);
  // recessed edge around the squares
  x.fillStyle = 'rgba(0,0,0,.55)'; x.fillRect(m - m * .08, m - m * .08, inner + m * .16, inner + m * .16);
  // brass corner plates
  [[m * .12, m * .12, 0], [px - m * .12, m * .12, 1], [px - m * .12, px - m * .12, 2], [m * .12, px - m * .12, 3]].forEach(([cx, cy, k]) => {
    x.save(); x.translate(cx, cy); x.rotate(k * Math.PI / 2);
    x.beginPath(); x.moveTo(m * .18, m * .18); x.lineTo(m * 1.3, m * .18); x.quadraticCurveTo(m * .7, m * .45, m * .62, m * .62); x.quadraticCurveTo(m * .45, m * .7, m * .18, m * 1.3); x.closePath();
    x.fillStyle = D.lin(x, 0, 0, m * 1.2, m * 1.2, ['#f4dc93', '#b58a35', '#e5c46e', '#8c6526']); x.fill();
    x.strokeStyle = 'rgba(70,45,10,.7)'; x.lineWidth = Math.max(1, m * .03); x.stroke();
    D.ell(x, m * .42, m * .42, m * .07, m * .07, '#7a5520'); D.ell(x, m * .405, m * .405, m * .03, m * .03, '#ffeab0');
    x.restore();
  });
  // coordinates
  const s = inner / 8;
  x.fillStyle = T.coord; x.textAlign = 'center'; x.textBaseline = 'middle';
  x.font = `600 ${Math.round(m * .38)}px Lora, Georgia, serif`;
  x.shadowColor = 'rgba(0,0,0,.6)'; x.shadowBlur = m * .06; x.shadowOffsetY = m * .02;
  for (let i = 0; i < 8; i++) {
    const file = 'abcdefgh'[flipped ? 7 - i : i], rank = String(flipped ? i + 1 : 8 - i);
    x.fillText(file, m + s * (i + .5), px - m * .42);
    x.fillText(file, m + s * (i + .5), m * .44);
    x.fillText(rank, m * .44, m + s * (i + .5));
    x.fillText(rank, px - m * .42, m + s * (i + .5));
  }
  return c;
}

/* ---------------------------------------------------------- the desk scene */
function scene(cv, W, H, res, layout) {
  cv.width = Math.round(W * res); cv.height = Math.round(H * res);
  cv.style.width = W + 'px'; cv.style.height = H + 'px';
  const x = cv.getContext('2d'); x.scale(res, res);
  const r = rng(2026);
  // mahogany desk top: long planks
  x.fillStyle = D.lin(x, 0, 0, 0, H, ['#3d1b0f', '#2c130a', '#1f0d06']); x.fillRect(0, 0, W, H);
  const plank = 168;
  for (let py = 0; py < H; py += plank) {
    const tone = r();
    x.fillStyle = `rgba(${tone < .5 ? '255,180,120' : '0,0,0'},${.02 + r() * .04})`; x.fillRect(0, py, W, plank);
    for (let i = 0; i < 70; i++) {
      const y0 = py + r() * plank, a = 2 + r() * 6, f = .002 + r() * .006, ph = r() * 9;
      x.beginPath(); for (let px = 0; px <= W; px += 20) { const yy = y0 + Math.sin(px * f + ph) * a; px ? x.lineTo(px, yy) : x.moveTo(px, yy); }
      x.strokeStyle = r() < .6 ? `rgba(0,0,0,${.06 + r() * .1})` : `rgba(255,190,130,${.03 + r() * .05})`; x.lineWidth = .6 + r() * 1.8; x.stroke();
    }
    x.fillStyle = 'rgba(0,0,0,.45)'; x.fillRect(0, py - 1.5, W, 3);
    x.fillStyle = 'rgba(255,200,150,.07)'; x.fillRect(0, py + 1.5, W, 1);
  }
  // tooled green leather writing surface under the board
  if (layout) {
    const { lx, ly, lw, lh } = layout;
    x.save(); x.shadowColor = 'rgba(0,0,0,.6)'; x.shadowBlur = 18; D.rr(x, lx, ly, lw, lh, 10); x.fillStyle = '#1f3a2a'; x.fill(); x.restore();
    D.rr(x, lx, ly, lw, lh, 10); x.fillStyle = D.rad(x, lx + lw * .4, ly + lh * .35, 20, lw * .8, ['#3a6247', '#24432f', '#152a1d']); x.fill();
    x.save(); D.rr(x, lx, ly, lw, lh, 10); x.clip();
    for (let i = 0; i < 2500; i++) { x.fillStyle = r() < .5 ? 'rgba(255,255,255,.025)' : 'rgba(0,0,0,.05)'; x.fillRect(lx + r() * lw, ly + r() * lh, 2, 2); }
    x.restore();
    x.strokeStyle = '#c9a24c'; x.lineWidth = 2; x.strokeRect(lx + 14, ly + 14, lw - 28, lh - 28);
    x.strokeStyle = 'rgba(201,162,76,.55)'; x.lineWidth = 1; x.strokeRect(lx + 20, ly + 20, lw - 40, lh - 40);
    // gilt corner flourishes
    [[lx + 14, ly + 14, 1, 1], [lx + lw - 14, ly + 14, -1, 1], [lx + lw - 14, ly + lh - 14, -1, -1], [lx + 14, ly + lh - 14, 1, -1]].forEach(([cx, cy, sx, sy]) => {
      x.beginPath(); x.moveTo(cx, cy + sy * 40); x.quadraticCurveTo(cx + sx * 16, cy + sy * 16, cx + sx * 40, cy); x.strokeStyle = '#c9a24c'; x.lineWidth = 2; x.stroke();
      D.ell(x, cx + sx * 12, cy + sy * 12, 4, 4, '#c9a24c');
    });
  }
  // warm lamp pool from the upper left, vignette at the edges
  x.fillStyle = D.rad(x, W * .28, H * .1, 40, W * .85, ['rgba(255,196,120,.20)', 'rgba(255,170,90,.06)', 'rgba(0,0,0,.42)']); x.fillRect(0, 0, W, H);
}

/* -------------------------------------------------------- the opponents */
const libraryBg = (a, b) => x => {
  x.fillStyle = D.lin(x, 0, 0, 0, IR.RH, [a, b]); x.fillRect(0, 0, 1, IR.RH);
  // shelves of books behind
  const r = rng(a.length * 31 + b.charCodeAt(2));
  for (let row = 0; row < 3; row++) {
    const y0 = .12 + row * .42;
    x.fillStyle = 'rgba(40,20,8,.55)'; x.fillRect(0, y0 + .32, 1, .035);
    let bx = 0;
    while (bx < 1) { const w = .04 + r() * .05, h = .22 + r() * .09; const hue = [[120, 40, 30], [40, 60, 90], [50, 80, 50], [110, 80, 40], [70, 30, 60]][Math.floor(r() * 5)];
      x.fillStyle = `rgba(${hue.join(',')},.55)`; x.fillRect(bx, y0 + .32 - h, w - .006, h);
      x.fillStyle = 'rgba(220,180,90,.35)'; x.fillRect(bx + w * .15, y0 + .32 - h * .82, w * .6, .008); bx += w; }
  }
  x.fillStyle = D.rad(x, .5, .9, .05, .9, ['rgba(255,210,140,.35)', 'rgba(0,0,0,.25)']); x.fillRect(0, 0, 1, IR.RH);
};
const FOLKS = [
  { id: 'pip', level: 1, name: 'Pip', full: 'Pip the Page', rating: 'Beginner', tag: 'Fetches the books. Learning the moves, and very keen.',
    look: { coat: ['#2f5c8a', '#1b3a5c'], hair: '#c27a35', blush: true, skin: '#f2c8a4', hat: { type: 'cap', color: '#2f5c8a', brim: '#244a70', button: '#e8c66a' } } },
  { id: 'hollis', level: 2, name: 'Mrs. Hollis', full: 'Mrs. Hollis', rating: 'Casual', tag: 'The librarian. Plays a friendly game between returns.',
    look: { coat: ['#7a3048', '#4a1a2c'], hair: '#b9b2ad', bun: true, glasses: '#8a6a2a', lips: '#a84a5a', lashes: true, skin: '#efc9ab', earring: '#e8c66a',
      collar: x => { D.poly(x, [[.4, 1.09], [.5, 1.2], [.6, 1.09]], '#f4ecdc'); } } },
  { id: 'ambrose', level: 3, name: 'Brother Ambrose', full: 'Brother Ambrose', rating: 'Club player', tag: 'Keeps the archive. Patient, solid, hard to rattle.',
    look: { coat: ['#5a4a36', '#3a2e20'], hair: '#7a6450', skin: '#e2b08a', hairTop: false, beard: '#7a6450', beardLen: -.06,
      body: x => { x.beginPath(); x.moveTo(.3, 1.09); x.quadraticCurveTo(.5, 1.32, .7, 1.09); x.lineTo(.74, 1.12); x.quadraticCurveTo(.5, 1.4, .26, 1.12); x.fillStyle = '#4a3c2a'; x.fill(); } } },
  { id: 'whitlock', level: 4, name: 'Judge Whitlock', full: 'Judge Whitlock', rating: 'Strong', tag: 'Retired from the bench. Punishes every loose piece.',
    look: { coat: ['#1d1d22', '#0c0c10'], hair: '#e8e6e2', brows: '#cfcac4', mustache: '#e6e3dd', skin: '#e7b996', glasses: '#c9a24c',
      collar: x => { D.poly(x, [[.42, 1.09], [.5, 1.16], [.58, 1.09]], '#f8f6f0'); D.poly(x, [[.47, 1.12], [.5, 1.22], [.53, 1.12]], '#8a1f22'); } } },
  { id: 'vance', level: 5, name: 'Professor Vance', full: 'Professor Vance', rating: 'Expert', tag: 'Wrote three books on the endgame. Mentions it often.',
    look: { coat: ['#5b4a2f', '#3a2f1d'], hair: '#3a2a1e', longHair: true, lips: '#9a4a40', lashes: true, skin: '#c98d62', skinShade: '#a8704c', earring: '#d9d2c8', glasses: '#2a2a2a',
      collar: x => { D.poly(x, [[.4, 1.09], [.5, 1.22], [.6, 1.09]], '#e9dfc9'); } } },
  { id: 'master', level: 6, name: 'The Old Master', full: 'The Old Master', rating: 'Master', tag: 'Nobody remembers him losing. Nobody remembers him arriving.',
    look: { coat: ['#2a1f3a', '#140f1f'], hair: '#f2efe8', brows: '#e6e2da', beard: '#f2efe8', beardLen: .08, mustache: '#f6f3ec', skin: '#e2b493', hairTop: true } },
];
const BGS = [['#5a4630', '#2a1f14'], ['#4a3a2a', '#24180e'], ['#3e3a2c', '#1e1a10'], ['#3a2a22', '#1c120c'], ['#2e3440', '#161a22'], ['#2a2236', '#120e1a']];
function avatar(f, size) {
  const [c, x] = canvas(size, size), s = size;
  x.save(); x.beginPath(); x.arc(s / 2, s / 2, s / 2, 0, Math.PI * 2); x.clip();
  x.translate(s * -.12, s * -.44); x.scale(s * 1.24, s * 1.24);
  const i = Math.max(0, FOLKS.indexOf(f));
  IR.portrait(x, Object.assign({ bg: libraryBg(BGS[i][0], BGS[i][1]) }, f.look || {}));
  x.restore();
  return c;
}

/* --------------------------------------------------------------- icon */
function icon(size) {
  const [c, x] = canvas(size, size), s = size;
  x.scale(s / 1024, s / 1024);
  // rounded tile: small checkerboard of walnut and maple
  D.rr(x, 40, 40, 944, 944, 190);
  x.save(); x.clip();
  const board = boardSurface('walnut', 1100);
  x.drawImage(board, -38, -38, 1100, 1100);
  x.fillStyle = D.rad(x, 360, 300, 50, 900, ['rgba(255,220,160,.25)', 'rgba(0,0,0,.05)', 'rgba(0,0,0,.6)']); x.fillRect(0, 0, 1024, 1024);
  x.restore();
  D.rr(x, 40, 40, 944, 944, 190); x.strokeStyle = '#d4a94e'; x.lineWidth = 26; x.stroke();
  D.rr(x, 62, 62, 900, 900, 172); x.strokeStyle = 'rgba(60,30,10,.7)'; x.lineWidth = 6; x.stroke();
  // ivory king in front, ebony knight behind
  x.drawImage(piece(-2, 820), 250, 120, 820, 820);
  x.drawImage(piece(6, 900), -40, 80, 900, 900);
  return c;
}

return { piece, boardSurface, boardFrame, BOARDS, scene, FOLKS, avatar, icon, PALETTES };
})();
