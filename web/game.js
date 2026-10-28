/* Chess - Ravenscroft Library. Game flow, board, input, clocks and saving.
   (c) 2026 Ironite */
'use strict';
(async () => {
const VERSION = '1.0.0';
const $ = IR.$, C = Chess;
const VAL = [0, 1, 3, 3, 5, 9, 0];

/* ---------- the members of the library ---------- */
const LEVELS = {
  pip: { ai: { maxDepth: 1, noise: 140, blunder: .18 }, book: .5, delay: [700, 1500] },
  hollis: { ai: { maxDepth: 2, noise: 60, blunder: .06 }, book: .7, delay: [700, 1500] },
  ambrose: { ai: { maxDepth: 3, noise: 22, blunder: .015 }, book: .9, delay: [600, 1300] },
  whitlock: { ai: { time: 450, maxDepth: 5 }, book: 1, delay: [500, 900] },
  vance: { ai: { time: 1300 }, book: 1, delay: [300, 700] },
  master: { ai: { time: 3000 }, book: 1, delay: [200, 500] },
};
const QUIPS = {
  pip: { greet: ['Oh! A game? I know how the horsey moves!', "I've been practising. A little bit."], check: ['Check! Did I say that right?', 'Check! I think.'], win: ['I won? I actually won!'], lose: ['Good game! Can we play again?'], draw: ["A draw! That's half a win, right?"], accept: ['Okay! A draw sounds nice.'], decline: ["No thank you, I think I'm winning!"] },
  hollis: { greet: ['Quietly now. Your move, dear.', "Just one game before closing."], check: ['Check, dear.', "Shh. Check."], win: ["Overdue, I'm afraid. Checkmate."], lose: ["Well played. I'll stamp that one."], draw: ['A draw. Shall we call it even?'], accept: ['A draw it is. Tea?'], decline: ["Not just yet, dear."] },
  ambrose: { greet: ['Patience is the whole of it.', 'Let us begin.'], check: ['Check.', 'Check, I believe.'], win: ['Slow and steady.'], lose: ['A fine game. I shall reflect on it.'], draw: ['Peace, then.'], accept: ['Peace. I accept.'], decline: ['I think there is more to play.'] },
  whitlock: { greet: ['Court is in session.', 'Proceed. And mind your pieces.'], check: ['Check. Order in the court.', 'Check.'], win: ['The verdict is checkmate.'], lose: ['Overruled by a better player. Well done.'], draw: ['A hung jury. Draw.'], accept: ['Motion granted. Draw.'], decline: ['Motion denied.'] },
  vance: { greet: ['Have you read chapter four? No matter.', 'An interesting opening awaits.'], check: ['Check. Textbook.', 'Check. Footnote nine.'], win: ['As I wrote in chapter seven: checkmate.'], lose: ['Remarkable. I may need a fourth book.'], draw: ['A theoretical draw, correctly played.'], accept: ['Agreed. The position is level.'], decline: ['Decline. The position is richer than that.'] },
  master: { greet: ['Sit. Play.', 'Mm. Begin.'], check: ['Check.'], win: ['Again, whenever you are ready.'], lose: ['...Well. It has been a long time.'], draw: ['A draw. You have improved.'], accept: ['Very well.'], decline: ['No.'] },
};
const YOU = { id: 'you', full: 'You', look: { coat: ['#6a2a2a', '#3e1616'], hair: '#4a2e1c', skin: '#e8b892', collar: x => { IR.draw.poly(x, [[.42, 1.09], [.5, 1.18], [.58, 1.09]], '#e9dfc9'); } } };
const P2 = { id: 'p2', full: 'Black', look: { coat: ['#2f4a3a', '#1a2c22'], hair: '#1e1a17', skin: '#c99068', skinShade: '#a8704c', glasses: '#2a2a2a' } };

/* ---------- save ---------- */
const save = new IR.Save('chess.save', {
  settings: { music: true, sfx: true, vol: 60, board: 'walnut', dots: 'on', anim: 'normal', promo: 'ask', turn: 'on' },
  setup: { mode: 'ai', opp: 'pip', color: 'white', clock: 'none' },
  game: null,
  stats: { won: 0, drawn: 0, lost: 0, pvp: 0, per: {} },
});
await save.load();
const S = save.data.settings, SD = save.data, ST = SD.stats;

/* ---------- stage ---------- */
const W = 1600, H = 844, BX = 30, BY = 22, FR = 800, M = 40, SQ = 90;
const boardCv = $('#board'), bx = boardCv.getContext('2d');
const fx = new IR.Particles($('#fx'));
let res = 0, staticCv = null;
const dpr = () => window.devicePixelRatio || 1;
/* ---------- audio ---------- */
const audio = new IR.Audio(S);
audio.setTunes(MUSIC.tunes(), MUSIC.instruments);
const sfx = MUSIC.sfx(audio);
IR.wireAudioControls(audio, save);
IR.on('tune', n => { const e = $('#nowPlaying'); e.textContent = '♫ ' + n; e.classList.add('show'); setTimeout(() => e.classList.remove('show'), 5000); });
$('#ver').textContent = 'v' + VERSION + ' · © 2026 Ironite';

/* ---------- AI worker ---------- */
let worker = null, reqId = 0;
const pending = new Map();
function startWorker() {
  worker = new Worker('ai.js');
  worker.onmessage = e => { const p = pending.get(e.data.id); if (p) { pending.delete(e.data.id); p(e.data); } };
  worker.onerror = () => { for (const p of pending.values()) p({ error: 'worker' }); pending.clear(); worker.terminate(); startWorker(); };
}
startWorker();
const think = opts => new Promise(res => { const id = ++reqId; pending.set(id, res); worker.postMessage(Object.assign({ id }, opts)); });
// Stop a search that's no longer wanted (new game, undo...): restart the worker.
function cancelThinking() { if (!pending.size) return; worker.terminate(); for (const p of pending.values()) p({ cancelled: true }); pending.clear(); startWorker(); }

/* ---------- game state ---------- */
let G = null;          // the saved game record
let pos = null;        // live position
let hist = [];         // per ply: {m, san, uci}
let view = -1;         // -1 = live, else number of plies shown
let viewPos = null;
let flipped = false;
let sel = -1, selMoves = [], legalNow = [];
let drag = null;       // {from, x, y, moved}
let anims = [];        // [{piece, x0, y0, x1, y1, t0, dur, fade}]
let hidden = new Set(); // squares whose piece is animating
let marks = new Set(), arrows = [], hintArrow = null, rightFrom = -1;
let busy = false, gen = 0, openingName = '';
let drawOfferPly = -99;

const isAI = () => G && G.mode === 'ai';
const youWhite = () => !G || G.mode !== 'ai' || G.youWhite;
const aiSide = () => (G.youWhite ? C.BLACK : C.WHITE);
const yourTurn = () => G && !G.result && (G.mode === 'pvp' || pos.side === (G.youWhite ? C.WHITE : C.BLACK));
const live = () => view < 0;
const curPos = () => (live() ? pos : viewPos);
const oppInfo = () => ART.FOLKS.find(f => f.id === G.opp) || ART.FOLKS[0];
const animMs = () => ({ slow: 320, normal: 200, fast: 110 }[S.anim] || 200);
const sideName = s => (s === C.WHITE ? 'White' : 'Black');

function rebuild() {
  pos = new C.Position(G.startFen || C.START); hist = [];
  for (const u of G.moves) {
    const legal = pos.legal(), m = legal.find(x => C.uci(x) === u);
    if (!m) break;
    hist.push({ m, san: pos.san(m, legal), uci: u });
    pos.make(m);
  }
  G.moves = hist.map(h => h.uci);
  legalNow = pos.legal();
}

function persist() { SD.game = G; save.write(); }

/* ---------- geometry ---------- */
function sqXY(sq) { const f = sq & 7, r = sq >> 4; return flipped ? [M + (7 - f) * SQ, M + r * SQ] : [M + f * SQ, M + (7 - r) * SQ]; }
function xySq(x, y) {
  const f = Math.floor((x - M) / SQ), r = Math.floor((y - M) / SQ);
  if (f < 0 || f > 7 || r < 0 || r > 7) return -1;
  return flipped ? r * 16 + (7 - f) : (7 - r) * 16 + f;
}
function evXY(e) { const b = boardCv.getBoundingClientRect(); return [(e.clientX - b.left) / b.width * FR, (e.clientY - b.top) / b.height * FR]; }

/* ---------- drawing ---------- */
function buildStatic() {
  if (!res) return;
  const px = Math.round(FR * res), inner = Math.round(FR * res) - Math.round(M * res) * 2;
  const [c, x] = [document.createElement('canvas'), null];
  c.width = c.height = px;
  const cx = c.getContext('2d');
  cx.drawImage(ART.boardFrame(S.board, px, inner, flipped), 0, 0);
  const surf = ART.boardSurface(S.board, inner);
  const off = Math.round(M * res);
  if (flipped) { cx.save(); cx.translate(off + inner, off + inner); cx.rotate(Math.PI); cx.drawImage(surf, 0, 0); cx.restore(); }
  else cx.drawImage(surf, off, off);
  staticCv = c; void x;
}

let raf = 0;
function dirty() { if (!raf) raf = requestAnimationFrame(draw); }
function draw(now) {
  raf = 0; now = now || performance.now();
  if (!staticCv) return;
  const x = bx, p = curPos();
  x.setTransform(1, 0, 0, 1, 0, 0);
  x.drawImage(staticCv, 0, 0);
  x.setTransform(res, 0, 0, res, 0, 0);
  // last move
  const last = live() ? hist[hist.length - 1] : hist[view - 1];
  if (last) for (const s of [last.m.from, last.m.to]) { const [a, b] = sqXY(s); x.fillStyle = 'rgba(255,196,64,.30)'; x.fillRect(a, b, SQ, SQ); }
  // check
  if (p.inCheck()) {
    const k = p.kings[p.side === C.WHITE ? 0 : 1], [a, b] = sqXY(k);
    x.fillStyle = IR.draw.rad(x, a + SQ / 2, b + SQ / 2, 4, SQ * .62, ['rgba(255,40,20,.85)', 'rgba(230,30,10,.45)', 'rgba(200,0,0,0)']); x.fillRect(a, b, SQ, SQ);
  }
  // selection
  if (sel >= 0 && live()) { const [a, b] = sqXY(sel); x.fillStyle = 'rgba(255,214,90,.48)'; x.fillRect(a, b, SQ, SQ); }
  // marks
  for (const s of marks) { const [a, b] = sqXY(s); x.beginPath(); x.arc(a + SQ / 2, b + SQ / 2, SQ * .43, 0, Math.PI * 2); x.strokeStyle = 'rgba(40,140,70,.85)'; x.lineWidth = 5; x.stroke(); }
  // pieces
  const ps = Math.round(SQ * res);
  for (let s = 0; s < 128; s++) {
    if (s & 0x88) { s += 7; continue; }
    const pc = p.board[s];
    if (!pc || hidden.has(s) || (drag && drag.moved && drag.from === s)) continue;
    const [a, b] = sqXY(s);
    x.drawImage(ART.piece(pc, ps), a, b, SQ, SQ);
  }
  // legal destinations
  if (sel >= 0 && live() && S.dots === 'on') {
    for (const m of selMoves) {
      const [a, b] = sqXY(m.to);
      if (m.captured) { x.beginPath(); x.arc(a + SQ / 2, b + SQ / 2, SQ * .46, 0, Math.PI * 2); x.arc(a + SQ / 2, b + SQ / 2, SQ * .37, 0, Math.PI * 2, true); x.fillStyle = 'rgba(30,15,5,.38)'; x.fill(); }
      else { x.beginPath(); x.arc(a + SQ / 2, b + SQ / 2, SQ * .15, 0, Math.PI * 2); x.fillStyle = 'rgba(30,15,5,.34)'; x.fill(); }
    }
  }
  // animations
  let animating = false;
  anims = anims.filter(an => {
    const t = Math.min(1, (now - an.t0) / an.dur), e = t < .5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2;
    if (an.fade) { x.globalAlpha = 1 - t; x.drawImage(ART.piece(an.piece, ps), an.x0, an.y0, SQ, SQ); x.globalAlpha = 1; }
    else {
      const lift = Math.sin(t * Math.PI) * SQ * .06;
      x.drawImage(ART.piece(an.piece, ps), an.x0 + (an.x1 - an.x0) * e - lift / 2, an.y0 + (an.y1 - an.y0) * e - lift, SQ + lift, SQ + lift);
    }
    if (t < 1) { animating = true; return true; }
    if (an.done) an.done();
    return false;
  });
  // arrows
  for (const a of arrows) arrow(x, a[0], a[1], 'rgba(40,140,70,.8)');
  if (hintArrow) arrow(x, hintArrow[0], hintArrow[1], 'rgba(226,170,40,.88)');
  // dragged piece on top
  if (drag && drag.moved) {
    const pc = p.board[drag.from], sz = SQ * 1.12;
    x.save(); x.shadowColor = 'rgba(0,0,0,.45)'; x.shadowBlur = 14; x.shadowOffsetY = 8;
    x.drawImage(ART.piece(pc, Math.round(sz * res)), drag.x - sz / 2, drag.y - sz * .62, sz, sz); x.restore();
  }
  if (animating) dirty();
}
function arrow(x, from, to, col) {
  const [a, b] = sqXY(from), [c, d] = sqXY(to);
  const x0 = a + SQ / 2, y0 = b + SQ / 2, x1 = c + SQ / 2, y1 = d + SQ / 2;
  const ang = Math.atan2(y1 - y0, x1 - x0), len = Math.hypot(x1 - x0, y1 - y0) - SQ * .32;
  x.save(); x.translate(x0, y0); x.rotate(ang);
  x.beginPath(); x.moveTo(SQ * .18, -SQ * .085); x.lineTo(len - SQ * .26, -SQ * .085); x.lineTo(len - SQ * .26, -SQ * .2); x.lineTo(len, 0); x.lineTo(len - SQ * .26, SQ * .2); x.lineTo(len - SQ * .26, SQ * .085); x.lineTo(SQ * .18, SQ * .085); x.closePath();
  x.fillStyle = col; x.fill(); x.restore();
}
function animate(piece, from, to, done) {
  const [x0, y0] = sqXY(from), [x1, y1] = sqXY(to);
  hidden.add(to);
  anims.push({ piece, x0, y0, x1, y1, t0: performance.now(), dur: animMs(), done: () => { hidden.delete(to); done && done(); } });
  dirty();
}
function fadeOut(piece, sq) { const [x0, y0] = sqXY(sq); anims.push({ piece, x0, y0, fade: true, t0: performance.now(), dur: animMs() }); dirty(); }
const waitAnims = () => new Promise(r => { const chk = () => (anims.length ? setTimeout(chk, 30) : r()); chk(); });

/* ---------- side panel ---------- */
const av = {};
function avatarFor(f) { if (!av[f.id]) av[f.id] = ART.avatar(f, 200); return av[f.id]; }
function plaqueFor(side) {
  if (!G) return { f: YOU, nm: 'You', rt: '' };
  if (G.mode === 'pvp') return side === C.WHITE ? { f: YOU, nm: 'White', rt: 'Player one' } : { f: P2, nm: 'Black', rt: 'Player two' };
  const mine = side === (G.youWhite ? C.WHITE : C.BLACK);
  if (mine) { const r = ST.per[G.opp] || { w: 0, d: 0, l: 0 }; return { f: YOU, nm: 'You', rt: `Playing ${sideName(side)} · vs ${oppInfo().name}: ${r.w}–${r.d}–${r.l}` }; }
  const o = oppInfo(); return { f: o, nm: o.full, rt: `${o.rating} · ${sideName(side)}` };
}
function refreshPlaques() {
  if (!G) return;
  const topSide = flipped ? C.WHITE : C.BLACK;
  [['#pTop', topSide], ['#pBot', -topSide]].forEach(([id, side]) => {
    const el = $(id), info = plaqueFor(side);
    el.dataset.side = side;
    const cv = el.querySelector('.av'), cx = cv.getContext('2d'); cx.clearRect(0, 0, 200, 200); cx.drawImage(avatarFor(info.f), 0, 0);
    el.querySelector('.nm').textContent = info.nm;
    el.querySelector('.rt').textContent = info.rt;
    el.classList.toggle('turn', !G.result && pos.side === side);
  });
  refreshCaps(); refreshClocks();
}
function refreshCaps() {
  const p = curPos(), count = { 1: {}, '-1': {} };
  for (let s = 0; s < 128; s++) { if (s & 0x88) { s += 7; continue; } const pc = p.board[s]; if (pc) { const k = pc > 0 ? 1 : -1; count[k][Math.abs(pc)] = (count[k][Math.abs(pc)] || 0) + 1; } }
  const full = { 1: 8, 2: 2, 3: 2, 4: 2, 5: 1 };
  let mat = { 1: 0, '-1': 0 };
  for (const side of [1, -1]) for (let t = 1; t <= 5; t++) mat[side] += (count[side][t] || 0) * VAL[t];
  const ps = Math.round(30 * Math.max(1, res));
  for (const id of ['#pTop', '#pBot']) {
    const el = $(id), side = +el.dataset.side, box = el.querySelector('.caps');
    box.innerHTML = '';
    let first = true;
    for (const t of [5, 4, 3, 2, 1]) { // enemy pieces this side has taken
      const n = Math.max(0, full[t] - (count[-side][t] || 0));
      for (let i = 0; i < n; i++) { const c = document.createElement('canvas'); c.width = c.height = ps; c.getContext('2d').drawImage(ART.piece(-side * t, ps), 0, 0); if (i === 0 && !first) c.className = 'gap'; box.appendChild(c); first = false; }
    }
    const adv = mat[side] - mat[-side];
    if (adv > 0) IR.el('span', 'adv', box, '+' + adv);
  }
}
function fmtClock(ms) {
  ms = Math.max(0, ms);
  if (ms < 10000) return (ms / 1000).toFixed(1);
  const s = Math.ceil(ms / 1000), m = Math.floor(s / 60);
  return m + ':' + String(s % 60).padStart(2, '0');
}
function refreshClocks() {
  for (const id of ['#pTop', '#pBot']) {
    const el = $(id).querySelector('.clock'), side = +$(id).dataset.side;
    if (!G || !G.clock) { el.classList.add('hidden'); continue; }
    el.classList.remove('hidden');
    const ms = side === C.WHITE ? G.clock.w : G.clock.b;
    el.textContent = fmtClock(ms);
    el.classList.toggle('run', !G.result && pos.side === side && live());
    el.classList.toggle('low', ms < 20000);
  }
}
function setThinking(on) {
  for (const id of ['#pTop', '#pBot']) $(id).querySelector('.think').textContent = '';
  if (on && G && isAI()) { const id = +$('#pTop').dataset.side === aiSide() ? '#pTop' : '#pBot'; $(id).querySelector('.think').textContent = 'Thinking…'; }
}
function refreshMoves() {
  const box = $('#moveList'); box.innerHTML = '';
  const startNo = 1;
  for (let i = 0; i < hist.length; i += 2) {
    IR.el('div', 'no', box, (startNo + i / 2) + '.');
    for (let j = 0; j < 2; j++) {
      const k = i + j;
      if (k >= hist.length) { IR.el('div', '', box, ''); continue; }
      const e = IR.el('div', 'mv', box, hist[k].san.replace(/^([KQRBN])/, (m, p) => '<span class="fig">' + ({ K: '♔', Q: '♕', R: '♖', B: '♗', N: '♘' })[p] + '</span>'));
      const shown = live() ? hist.length : view;
      if (k === shown - 1) e.classList.add('cur');
      e.onclick = () => setView(k + 1);
    }
  }
  if (G && G.result) IR.el('div', 'res', box, `${G.result.result} · ${resultLine()}`);
  const cur = box.querySelector('.cur'); if (cur) cur.scrollIntoView({ block: 'nearest' }); else box.scrollTop = box.scrollHeight;
  // opening name
  if (!G || (G.startFen && G.startFen !== C.START)) openingName = '';
  else { const n = BOOK.name(G.moves.slice(0, Math.max(0, live() ? G.moves.length : view))); if (n) openingName = n; if (!G.moves.length) openingName = ''; }
  $('#opening').textContent = openingName;
  const n = hist.length, v = live() ? n : view;
  const nb = document.querySelectorAll('.nav button');
  nb[0].disabled = nb[1].disabled = v === 0; nb[2].disabled = nb[3].disabled = v === n;
}
function setStatus(t, cls = '') { $('#statusText').textContent = t; $('#status').className = cls; }
function refreshStatus() {
  if (!G) return setStatus(' ');
  if (!live()) return setStatus(`Reviewing move ${Math.ceil(view / 2) || 0}${view % 2 === 0 && view ? '…' : ''} · press End to return`, 'review');
  if (G.result) return setStatus(resultLine(), 'alert');
  const chk = pos.inCheck();
  if (G.mode === 'pvp') return setStatus(`${sideName(pos.side)} to move${chk ? ' · Check!' : ''}`, chk ? 'alert' : '');
  if (yourTurn()) return setStatus(chk ? 'Check! Your king is under attack.' : 'Your move', chk ? 'alert' : '');
  setStatus(`${oppInfo().name} is thinking…`);
}
function resultLine() {
  const r = G.result; if (!r) return '';
  const why = { checkmate: 'by checkmate', resignation: 'by resignation', time: 'on time', stalemate: 'Stalemate', 'insufficient material': 'Not enough material to mate', 'fifty-move rule': 'Fifty-move rule', 'threefold repetition': 'Threefold repetition', agreement: 'Draw agreed', 'time vs insufficient material': 'Flag fell, but no mating material' }[r.reason] || r.reason;
  if (!r.winner) return `Draw · ${why}`;
  return `${sideName(r.winner)} wins ${why}`;
}
function refreshButtons() {
  const ok = G && !G.result && live();
  $('#bUndo').disabled = !G || !!G.result || !hist.length || (isAI() && busy && hist.length < 1);
  $('#bHint').disabled = !ok || !yourTurn() || busy;
  $('#bDraw').disabled = !ok || !yourTurn() || hist.length < 2 || (isAI() && hist.length - drawOfferPly < 10);
  $('#bResign').disabled = !ok || hist.length < 1;
  $('#sWon').textContent = ST.won; $('#sDrawn').textContent = ST.drawn; $('#sLost').textContent = ST.lost;
}
function refreshAll() { refreshPlaques(); refreshMoves(); refreshStatus(); refreshButtons(); dirty(); }

/* ---------- reviewing ---------- */
function setView(v) {
  if (!G) return;
  if (v >= hist.length) { view = -1; viewPos = null; }
  else { view = Math.max(0, v); viewPos = new C.Position(G.startFen || C.START); for (let i = 0; i < view; i++) viewPos.make(hist[i].m); }
  sel = -1; selMoves = []; hintArrow = null;
  sfx.click();
  refreshAll();
}

/* ---------- playing moves ---------- */
async function playMove(m, opts = {}) {
  const legal = legalNow, san = pos.san(m, legal), mover = pos.side;
  // clock: the mover gets their increment
  if (G.clock && hist.length) { if (mover === C.WHITE) G.clock.w += G.clock.inc; else G.clock.b += G.clock.inc; }
  if (G.clock) G.clock.last = performance.now();
  hist.push({ m, san, uci: C.uci(m) }); G.moves.push(C.uci(m));
  sel = -1; selMoves = []; hintArrow = null; marks.clear(); arrows = [];
  // animation
  const piece = m.promo || m.piece;
  if (!opts.noAnim) {
    if (m.captured && !(m.flags & C.F_EP)) fadeOut(m.captured, m.to);
    if (m.flags & C.F_EP) fadeOut(m.captured, m.to - 16 * mover);
    animate(m.piece, m.from, m.to);
    if (m.flags & C.F_CASTLE) { const rf = m.to > m.from ? m.from + 3 : m.from - 4, rt = m.to > m.from ? m.from + 1 : m.from - 1; animate(C.R * mover, rf, rt); }
  }
  pos.make(m); legalNow = pos.legal();
  void piece;
  // sounds
  const chk = pos.inCheck();
  if (m.promo) sfx.promote(); else if (m.flags & C.F_CASTLE) sfx.castle(); else if (m.captured) sfx.capture(); else sfx.move();
  if (chk) setTimeout(() => sfx.check(), 120);
  const st = pos.status();
  if (st.over) { G.result = { result: st.result, reason: st.reason, winner: st.winner }; }
  persist();
  refreshAll();
  if (G.result) { await waitAnims(); return gameOver(); }
  if (isAI() && chk && mover === aiSide() && Math.random() < .5) quip('check');
  if (G.mode === 'pvp' && S.turn === 'on') { await waitAnims(); setTimeout(() => setFlip(pos.side === C.BLACK), 250); }
  if (isAI() && pos.side === aiSide()) aiTurn();
}

async function aiTurn() {
  const my = ++gen;
  busy = true; setThinking(true); refreshStatus(); refreshButtons();
  const L = LEVELS[G.opp] || LEVELS.pip;
  const t0 = performance.now();
  await waitAnims();
  let uci = null;
  // opening book
  if ((!G.startFen || G.startFen === C.START) && Math.random() < L.book) {
    const opts = BOOK.next(G.moves);
    if (opts.length) uci = opts[Math.floor(Math.random() * opts.length)].move;
  }
  if (!uci) {
    const ai = Object.assign({}, L.ai);
    if (G.clock) { const left = aiSide() === C.WHITE ? G.clock.w : G.clock.b; ai.time = Math.max(150, Math.min(ai.time || 600, left / 30 + G.clock.inc * .8)); }
    const r = await think(Object.assign({ fen: G.startFen || C.START, moves: G.moves }, ai));
    if (my !== gen || r.cancelled) return;
    if (r.error || !r.move) { console.error(r.error); uci = legalNow[0] && C.uci(legalNow[0]); }
    else uci = r.move;
  }
  const minWait = L.delay[0] + Math.random() * (L.delay[1] - L.delay[0]);
  const waited = performance.now() - t0;
  if (waited < minWait) await new Promise(r => setTimeout(r, minWait - waited));
  if (my !== gen || !G || G.result) return;
  const m = legalNow.find(x => C.uci(x) === uci);
  busy = false; setThinking(false);
  if (m) await playMove(m);
}

/* ---------- player input ---------- */
let promoPending = null;
function canMoveFrom(sq) {
  if (!G || G.result || !live() || busy || promoPending) return false;
  if (!yourTurn()) return false;
  const pc = pos.board[sq];
  return pc && (pc > 0) === (pos.side > 0);
}
function select(sq) { sel = sq; selMoves = legalNow.filter(m => m.from === sq); sfx.pick(); dirty(); }
function tryMove(from, to) {
  const ms = legalNow.filter(m => m.from === from && m.to === to);
  if (!ms.length) return false;
  if (ms.length > 1) { // promotion
    if (S.promo === 'queen') { playMove(ms.find(m => Math.abs(m.promo) === C.Q), { noAnim: false }); return true; }
    showPromo(ms); return true;
  }
  playMove(ms[0]); return true;
}
function showPromo(ms) {
  promoPending = ms;
  const to = ms[0].to, side = ms[0].piece > 0 ? 1 : -1, [x, y] = sqXY(to);
  const box = $('#promo'); box.innerHTML = '';
  const atTop = y < FR / 2;
  [C.Q, C.N, C.R, C.B].forEach(t => {
    const b = IR.el('button', '', box); const c = document.createElement('canvas'); const s = Math.round(90 * Math.max(1, res)); c.width = c.height = s;
    c.getContext('2d').drawImage(ART.piece(side * t, s), 0, 0); b.appendChild(c); b.title = ['', '', 'Knight', 'Bishop', 'Rook', 'Queen'][t];
    b.onclick = e => { e.stopPropagation(); const m = ms.find(mm => Math.abs(mm.promo) === t); hidePromo(); playMove(m); };
  });
  if (!atTop) box.style.flexDirection = 'column-reverse'; else box.style.flexDirection = 'column';
  box.style.left = (BX + x) + 'px'; box.style.top = (BY + (atTop ? y : y + SQ - 360)) + 'px';
  box.classList.remove('hidden');
}
function hidePromo() { promoPending = null; $('#promo').classList.add('hidden'); sel = -1; selMoves = []; dirty(); }

boardCv.addEventListener('contextmenu', e => e.preventDefault());
boardCv.addEventListener('pointerdown', e => {
  audio.unlock();
  const [x, y] = evXY(e), sq = xySq(x, y);
  if (promoPending) { hidePromo(); return; }
  if (e.button === 2) { rightFrom = sq; return; }
  if (e.button !== 0) return;
  if (marks.size || arrows.length) { marks.clear(); arrows = []; dirty(); }
  if (sq < 0) { sel = -1; selMoves = []; dirty(); return; }
  if (!live()) { setView(hist.length); return; }
  if (sel >= 0 && sel !== sq && selMoves.some(m => m.to === sq)) { tryMove(sel, sq); return; }
  if (canMoveFrom(sq)) {
    const wasSel = sel === sq;
    if (!wasSel) select(sq);
    drag = { from: sq, x, y, x0: x, y0: y, moved: false, wasSel };
    boardCv.setPointerCapture(e.pointerId);
    boardCv.classList.add('grab');
  } else {
    if (sel >= 0 && pos.board[sq] && sq !== sel) sfx.illegal();
    sel = -1; selMoves = []; dirty();
  }
});
boardCv.addEventListener('pointermove', e => {
  const [x, y] = evXY(e);
  if (drag) { drag.x = x; drag.y = y; if (!drag.moved && Math.hypot(x - drag.x0, y - drag.y0) > 6) drag.moved = true; dirty(); return; }
  const sq = xySq(x, y);
  boardCv.classList.toggle('hand', sq >= 0 && (canMoveFrom(sq) || (sel >= 0 && selMoves.some(m => m.to === sq))));
});
boardCv.addEventListener('pointerup', e => {
  const [x, y] = evXY(e), sq = xySq(x, y);
  if (e.button === 2) {
    if (rightFrom >= 0 && sq >= 0) {
      if (sq === rightFrom) { marks.has(sq) ? marks.delete(sq) : marks.add(sq); }
      else { const i = arrows.findIndex(a => a[0] === rightFrom && a[1] === sq); if (i >= 0) arrows.splice(i, 1); else arrows.push([rightFrom, sq]); }
      dirty();
    }
    rightFrom = -1; return;
  }
  if (!drag) return;
  const d = drag; drag = null; boardCv.classList.remove('grab');
  if (d.moved) {
    if (sq >= 0 && sq !== d.from && tryMove(d.from, sq)) return;
    if (sq >= 0 && sq !== d.from) sfx.illegal();
    dirty();
  } else if (d.wasSel) { sel = -1; selMoves = []; dirty(); }
});

/* ---------- flipping ---------- */
function setFlip(f) { if (f === flipped) return; flipped = f; if (G) G.flipped = f; buildStatic(); refreshPlaques(); dirty(); }

/* ---------- clocks ---------- */
let lastTick = performance.now(), lastTickSave = 0;
setInterval(() => {
  const now = performance.now(), dt = now - lastTick; lastTick = now;
  if (!G || !G.clock || G.result || !hist.length) return;
  if (!$('#title').classList.contains('hidden') || !$('#confirm').classList.contains('hidden')) return; // paused while a menu is open
  const k = pos.side === C.WHITE ? 'w' : 'b';
  const before = G.clock[k];
  G.clock[k] -= dt;
  if (before > 10000 && G.clock[k] <= 10000 && (G.mode === 'pvp' || pos.side !== aiSide())) sfx.tick();
  if (G.clock[k] <= 0) { G.clock[k] = 0; flag(pos.side); }
  refreshClocks();
  if (now - lastTickSave > 5000) { lastTickSave = now; persist(); }
}, 100);
function canMate(side) {
  let minors = 0;
  for (let s = 0; s < 128; s++) { if (s & 0x88) { s += 7; continue; } const pc = pos.board[s] * side; if (pc === 1 || pc === 4 || pc === 5) return true; if (pc === 2 || pc === 3) minors++; }
  return minors >= 2;
}
function flag(side) {
  cancelThinking(); gen++; busy = false; setThinking(false);
  G.result = canMate(-side) ? { result: side === C.WHITE ? '0-1' : '1-0', reason: 'time', winner: -side } : { result: '1/2-1/2', reason: 'time vs insufficient material', winner: 0 };
  persist(); refreshAll(); gameOver();
}

/* ---------- game over ---------- */
function quip(kind) {
  if (!isAI()) return;
  const list = (QUIPS[G.opp] || QUIPS.pip)[kind]; if (!list) return;
  const el = +$('#pTop').dataset.side === aiSide() ? $('#pTop') : $('#pBot');
  const r = el.getBoundingClientRect(), sr = $('#stage').getBoundingClientRect(), sc = IR.Stage.scale;
  const x = (r.left - sr.left) / sc + 64, y = (r.top - sr.top) / sc - 56 + (el.id === 'pTop' ? 140 : -12);
  IR.bubble($('#table'), list[Math.floor(Math.random() * list.length)], x + 200, y, el.id === 'pTop' ? 'down' : '', 3200);
}
function gameOver() {
  busy = false; setThinking(false);
  const r = G.result;
  let kick = '', title = '', text = resultLine(), face = YOU;
  if (isAI()) {
    face = oppInfo();
    const you = G.youWhite ? C.WHITE : C.BLACK;
    const rec = ST.per[G.opp] = ST.per[G.opp] || { w: 0, d: 0, l: 0 };
    if (!G.counted) {
      if (!r.winner) { ST.drawn++; rec.d++; } else if (r.winner === you) { ST.won++; rec.w++; } else { ST.lost++; rec.l++; }
      G.counted = true;
    }
    if (!r.winner) { title = 'A Draw'; kick = 'Honours even'; sfx.draw(); }
    else if (r.winner === you) { title = 'You Win!'; kick = r.reason === 'checkmate' ? 'Checkmate' : 'Victory'; sfx.win(); confetti(); }
    else { title = `${oppInfo().name} Wins`; kick = r.reason === 'checkmate' ? 'Checkmate' : 'Defeat'; sfx.lose(); }
    const q = (QUIPS[G.opp] || QUIPS.pip)[!r.winner ? 'draw' : r.winner === you ? 'lose' : 'win'];
    $('#overQuote').textContent = `“${q[Math.floor(Math.random() * q.length)]}” — ${oppInfo().full}`;
    text = `${resultLine()} after ${Math.ceil(hist.length / 2)} moves. Your record against ${oppInfo().name}: ${rec.w} won, ${rec.d} drawn, ${rec.l} lost.`;
  } else {
    if (!G.counted) { ST.pvp++; G.counted = true; }
    title = r.winner ? `${sideName(r.winner)} Wins` : 'A Draw'; kick = r.reason === 'checkmate' ? 'Checkmate' : 'Game over';
    face = r.winner === C.BLACK ? P2 : YOU;
    $('#overQuote').textContent = '';
    if (r.winner) { sfx.win(); confetti(); } else sfx.draw();
    text = `${resultLine()} after ${Math.ceil(hist.length / 2)} moves.`;
  }
  SD.game = G; save.write(true);
  $('#overKick').textContent = kick; $('#overTitle').textContent = title; $('#overText').textContent = text;
  const oc = $('#overAv').getContext('2d'); oc.clearRect(0, 0, 240, 240); oc.drawImage(avatarFor(face), 0, 0, 240, 240);
  refreshAll();
  setTimeout(() => IR.panel.show('over'), 900);
}
function confetti() { const cols = ['#e2bd6a', '#f6ead0', '#6e2316', '#2f5c3a', '#fff3c4']; fx.burst(BX + FR / 2, BY + FR * .45, 140, cols, { speed: 720, life: 2.6 }); }

/* ---------- commands ---------- */
function newGame(setup) {
  cancelThinking(); gen++; busy = false;
  const youW = setup.color === 'random' ? Math.random() < .5 : setup.color !== 'black';
  const clk = { none: null, '5+3': [5, 3], '10': [10, 0], '15+10': [15, 10], '30': [30, 0] }[setup.clock];
  G = { startFen: C.START, moves: [], mode: setup.mode, opp: setup.opp, youWhite: setup.mode === 'ai' ? youW : true, result: null, counted: false,
    clock: clk ? { w: clk[0] * 60000, b: clk[0] * 60000, inc: clk[1] * 1000, base: setup.clock } : null, started: Date.now() };
  drawOfferPly = -99; view = -1; viewPos = null; sel = -1; selMoves = []; marks.clear(); arrows = []; hintArrow = null; anims = []; hidden.clear(); openingName = '';
  rebuild();
  flipped = G.mode === 'ai' ? !G.youWhite : false; G.flipped = flipped;
  buildStatic(); persist(); refreshAll();
  IR.panel.hide('title'); IR.panel.hide('over');
  sfx.start();
  if (isAI()) { setTimeout(() => quip('greet'), 500); if (pos.side === aiSide()) aiTurn(); }
}
function resumeGame() {
  G = SD.game; rebuild();
  flipped = !!G.flipped; view = -1; viewPos = null; busy = false; sel = -1; selMoves = [];
  buildStatic(); refreshAll();
  IR.panel.hide('title');
  if (isAI() && !G.result && pos.side === aiSide()) aiTurn();
}
function undo() {
  if (!G || G.result || !hist.length) return;
  if (!live()) setView(hist.length);
  cancelThinking(); gen++; busy = false; setThinking(false);
  let n = 1;
  if (isAI()) { n = pos.side === aiSide() ? 1 : 2; if (hist.length < n) n = hist.length; if (hist.length - n < 0) return; if (G.youWhite === false && hist.length - n === 0) n = hist.length - 1; if (n <= 0) return; }
  const undone = hist.slice(-n);
  G.moves = G.moves.slice(0, -n); rebuild();
  const last = undone[0];
  animate(last.m.piece, last.m.to, last.m.from);
  sfx.move(); sel = -1; selMoves = []; hintArrow = null;
  if (G.mode === 'pvp' && S.turn === 'on') setFlip(pos.side === C.BLACK);
  persist(); refreshAll();
  if (isAI() && pos.side === aiSide()) aiTurn();
}
async function hint() {
  if (!G || !yourTurn() || !live() || busy) return;
  busy = true; refreshButtons(); setStatus('Considering the position…');
  const my = gen;
  const r = await think({ fen: G.startFen || C.START, moves: G.moves, time: 1200 });
  busy = false;
  if (my !== gen || r.cancelled || !r.move) { refreshAll(); return; }
  const m = legalNow.find(x => C.uci(x) === r.move);
  if (!m) { refreshAll(); return; }
  hintArrow = [m.from, m.to]; sfx.hint();
  refreshButtons(); setStatus('Hint: ' + pos.san(m, legalNow), 'alert'); dirty();
}
function ask(title, text, yes, no = 'No') {
  return new Promise(res => {
    $('#cTitle').textContent = title; $('#cText').textContent = text; $('#cYes').textContent = yes; $('#cNo').textContent = no;
    IR.panel.show('confirm');
    $('#cYes').onclick = () => { IR.panel.hide('confirm'); sfx.click(); res(true); };
    $('#cNo').onclick = () => { IR.panel.hide('confirm'); sfx.click(); res(false); };
  });
}
async function resign() {
  if (!G || G.result || !hist.length) return;
  const who = G.mode === 'pvp' ? sideName(pos.side) : 'you';
  if (!await ask('Resign?', G.mode === 'pvp' ? `${who} resigns and ${sideName(-pos.side)} wins the game.` : `${oppInfo().name} will be credited with the win.`, 'Resign', 'Keep Playing')) return;
  cancelThinking(); gen++; busy = false;
  const loser = G.mode === 'pvp' ? pos.side : (G.youWhite ? C.WHITE : C.BLACK);
  G.result = { result: loser === C.WHITE ? '0-1' : '1-0', reason: 'resignation', winner: -loser };
  persist(); gameOver();
}
async function offerDraw() {
  if (!G || G.result || !yourTurn()) return;
  if (G.mode === 'pvp') {
    if (!await ask('Draw offered', `${sideName(pos.side)} offers a draw. ${sideName(-pos.side)}, do you accept?`, 'Accept Draw', 'Play On')) return;
    G.result = { result: '1/2-1/2', reason: 'agreement', winner: 0 }; persist(); return gameOver();
  }
  drawOfferPly = hist.length; refreshButtons();
  busy = true; setStatus(`${oppInfo().name} considers your offer…`);
  const my = gen;
  const r = await think({ fen: G.startFen || C.START, moves: G.moves, time: 700 });
  busy = false;
  if (my !== gen || r.cancelled) return;
  const aiScore = -(r.score || 0); // r.score is from your side (you're to move)
  const accept = aiScore <= -80 || (aiScore <= 30 && hist.length >= 50) || (G.opp === 'pip' && aiScore <= 0);
  if (accept) { quip('accept'); G.result = { result: '1/2-1/2', reason: 'agreement', winner: 0 }; persist(); gameOver(); }
  else { quip('decline'); refreshAll(); }
}
function pgn() {
  const d = new Date(G.started || Date.now()), ds = `${d.getFullYear()}.${String(d.getMonth() + 1).padStart(2, '0')}.${String(d.getDate()).padStart(2, '0')}`;
  const wName = G.mode === 'pvp' ? 'White' : G.youWhite ? 'You' : oppInfo().full, bName = G.mode === 'pvp' ? 'Black' : G.youWhite ? oppInfo().full : 'You';
  const res = G.result ? G.result.result : '*';
  let s = `[Event "Ravenscroft Library"]\n[Site "Ironite Chess"]\n[Date "${ds}"]\n[White "${wName}"]\n[Black "${bName}"]\n[Result "${res}"]\n\n`;
  const parts = []; hist.forEach((h, i) => { if (i % 2 === 0) parts.push((i / 2 + 1) + '.'); parts.push(h.san); }); parts.push(res);
  let line = ''; for (const p of parts) { if ((line + ' ' + p).length > 78) { s += line.trim() + '\n'; line = ''; } line += ' ' + p; }
  return s + line.trim() + '\n';
}
async function copyPgn() {
  const t = pgn(); let ok = false;
  try { await navigator.clipboard.writeText(t); ok = true; } catch (e) { const ta = IR.el('textarea', '', document.body); ta.value = t; ta.select(); try { ok = document.execCommand('copy'); } catch (e2) { } ta.remove(); }
  banner(ok ? 'Moves copied' : 'Could not copy', ok ? 'Paste them into any chess program' : '');
}
function banner(t, sub) { const b = $('#banner'); b.innerHTML = t + (sub ? `<small>${sub}</small>` : ''); b.classList.add('show'); clearTimeout(banner.t); banner.t = setTimeout(() => b.classList.remove('show'), 1800); }

/* ---------- title screen ---------- */
const setup = SD.setup;
function chips(id, obj, key, after) {
  const box = $(id); const sync = () => box.querySelectorAll('button').forEach(b => b.classList.toggle('on', b.dataset.v === obj[key]));
  box.querySelectorAll('button').forEach(b => b.onclick = () => { obj[key] = b.dataset.v; save.write(); sync(); sfx.click(); after && after(); }); sync();
}
function buildFolks() {
  const box = $('#folks'); box.innerHTML = '';
  ART.FOLKS.forEach(f => {
    const r = ST.per[f.id] || { w: 0, d: 0, l: 0 };
    const d = IR.el('div', 'folk' + (setup.opp === f.id ? ' on' : ''), box);
    const c = document.createElement('canvas'); c.width = c.height = 200; c.getContext('2d').drawImage(avatarFor(f), 0, 0); d.appendChild(c);
    IR.el('b', '', d, f.full);
    IR.el('span', 'lv', d, `${f.rating} <span class="stars">${'♟'.repeat(f.level)}</span>`);
    IR.el('span', 'rec', d, r.w + r.d + r.l ? `You: ${r.w}W ${r.d}D ${r.l}L` : 'Not played yet');
    d.onclick = () => { setup.opp = f.id; save.write(); sfx.click(); buildFolks(); };
  });
  const f = ART.FOLKS.find(x => x.id === setup.opp) || ART.FOLKS[0];
  $('#folkTag').textContent = setup.mode === 'ai' ? `“${f.tag}”` : 'Pass and play on one computer. The board turns to face whoever is moving.';
  box.classList.toggle('off', setup.mode !== 'ai');
  $('#rowColor').style.visibility = setup.mode === 'ai' ? 'visible' : 'hidden';
}
chips('#optMode', setup, 'mode', buildFolks);
chips('#optColor', setup, 'color');
chips('#optClock', setup, 'clock');
chips('#optBoard', S, 'board', () => { buildStatic(); dirty(); });
chips('#optDots', S, 'dots', dirty);
chips('#optAnim', S, 'anim');
chips('#optPromo', S, 'promo');
chips('#optTurn', S, 'turn');
function showTitle() {
  buildFolks();
  const can = SD.game && !SD.game.result && SD.game.moves && SD.game.moves.length;
  $('#bContinue').classList.toggle('hidden', !can);
  IR.panel.hide('over');
  IR.panel.show('title');
}
$('#bStart').onclick = () => { audio.unlock(); newGame(setup); };
$('#bContinue').onclick = () => { audio.unlock(); sfx.click(); resumeGame(); };
$('#bNew').onclick = () => { sfx.click(); showTitle(); };
$('#bUndo').onclick = undo;
$('#bHint').onclick = hint;
$('#bFlip').onclick = () => { sfx.click(); setFlip(!flipped); persist(); };
$('#bDraw').onclick = offerDraw;
$('#bResign').onclick = resign;
$('#bRematch').onclick = () => { const s = Object.assign({}, setup, { mode: G.mode, opp: G.opp, color: G.mode === 'ai' ? (G.youWhite ? 'black' : 'white') : 'white', clock: G.clock ? G.clock.base : 'none' }); newGame(s); };
$('#bOverNew').onclick = () => { sfx.click(); showTitle(); };
$('#bReview').onclick = () => { IR.panel.hide('over'); setView(0); };
$('#bPgn').onclick = copyPgn;
$('#bSettings').onclick = $('#bSettings2').onclick = () => { sfx.click(); IR.panel.show('settings'); };
$('#bHow').onclick = () => { sfx.click(); IR.panel.show('rules'); };
$('#bCredits').onclick = $('#bCredits2').onclick = async () => { $('#changelog').innerHTML = IR.mdChangelog(await IR.loadChangelog()); IR.panel.show('credits'); };
document.querySelectorAll('.nav button').forEach(b => b.onclick = () => {
  const n = hist.length, v = live() ? n : view;
  setView({ start: 0, prev: v - 1, next: v + 1, end: n }[b.dataset.nav]);
});
IR.panel.wire();
addEventListener('keydown', e => {
  if (!$('#title').classList.contains('hidden') || !$('#confirm').classList.contains('hidden')) { if (e.key === 'Enter' && !$('#title').classList.contains('hidden')) $('#bStart').click(); return; }
  const k = e.key.toLowerCase();
  if ((e.ctrlKey && k === 'z') || k === 'backspace') { e.preventDefault(); undo(); }
  else if (k === 'h') hint();
  else if (k === 'f') $('#bFlip').click();
  else if (k === 'n') showTitle();
  else if (G && (k === 'arrowleft' || k === 'arrowright' || k === 'home' || k === 'end')) {
    e.preventDefault(); const n = hist.length, v = live() ? n : view;
    setView({ arrowleft: v - 1, arrowright: v + 1, home: 0, end: n }[k]);
  } else if (k === 'escape' && promoPending) hidePromo();
});

/* ---------- start ---------- */
IR.Stage.init($('#stage'), s => {
  const r = IR.clamp(Math.round(s * dpr() * 100) / 100, 1, 4);
  if (Math.abs(r - res) > .02) {
    res = r;
    ART.scene($('#scene'), W, H, Math.min(2.5, r), { lx: BX - 14, ly: BY - 12, lw: FR + 28, lh: FR + 24 });
    boardCv.width = Math.round(FR * res); boardCv.height = Math.round(FR * res);
    buildStatic(); dirty();
    refreshPlaques();
  }
  fx.resize(W, H, Math.min(2, s * dpr()));
});


if (SD.game && SD.game.moves) { G = SD.game; rebuild(); flipped = !!G.flipped; buildStatic(); refreshAll(); }
else { G = { startFen: C.START, moves: [], mode: 'ai', opp: setup.opp, youWhite: true, result: null, clock: null }; rebuild(); refreshAll(); G = null; refreshButtons(); }
showTitle();

// test hooks
window.__chess = { get G() { return G; }, get sel() { return sel; }, get drag() { return drag; }, get pos() { return pos; }, get busy() { return busy; }, get hist() { return hist; }, C, newGame, playUci: u => { const m = legalNow.find(x => C.uci(x) === u); if (m) playMove(m); return !!m; }, get flipped() { return flipped; }, undo, hint, setView, pgn, ST, SD, sqXY, resign: () => resign(), flag };
})();
