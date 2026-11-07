package engine

// The computer opponent: alpha-beta (PVS) with iterative deepening, a
// transposition table, quiescence search, null-move pruning, late-move
// reductions and killer/history move ordering, over a tapered evaluation.
// Weaker players search shallower and add noise to their choices.

import (
	"math"
	"math/rand"
	"sort"
	"sync/atomic"
	"time"
)

var (
	mgVal = [7]int{0, 100, 320, 330, 500, 950, 0}
	egVal = [7]int{0, 120, 300, 315, 530, 950, 0}
	phase = [7]int{0, 0, 1, 1, 2, 4, 0}
	val   = [7]int{0, 100, 320, 330, 500, 950, 2000}
)

// Piece-square tables from White's side, a8..h8 first.
var (
	pstP     = [64]int{0, 0, 0, 0, 0, 0, 0, 0, 50, 50, 50, 50, 50, 50, 50, 50, 10, 10, 20, 30, 30, 20, 10, 10, 5, 5, 10, 25, 25, 10, 5, 5, 0, 0, 0, 20, 20, 0, 0, 0, 5, -5, -10, 0, 0, -10, -5, 5, 5, 10, 10, -20, -20, 10, 10, 5, 0, 0, 0, 0, 0, 0, 0, 0}
	pstN     = [64]int{-50, -40, -30, -30, -30, -30, -40, -50, -40, -20, 0, 0, 0, 0, -20, -40, -30, 0, 10, 15, 15, 10, 0, -30, -30, 5, 15, 20, 20, 15, 5, -30, -30, 0, 15, 20, 20, 15, 0, -30, -30, 5, 10, 15, 15, 10, 5, -30, -40, -20, 0, 5, 5, 0, -20, -40, -50, -40, -30, -30, -30, -30, -40, -50}
	pstB     = [64]int{-20, -10, -10, -10, -10, -10, -10, -20, -10, 0, 0, 0, 0, 0, 0, -10, -10, 0, 5, 10, 10, 5, 0, -10, -10, 5, 5, 10, 10, 5, 5, -10, -10, 0, 10, 10, 10, 10, 0, -10, -10, 10, 10, 10, 10, 10, 10, -10, -10, 5, 0, 0, 0, 0, 5, -10, -20, -10, -10, -10, -10, -10, -10, -20}
	pstR     = [64]int{0, 0, 0, 0, 0, 0, 0, 0, 5, 10, 10, 10, 10, 10, 10, 5, -5, 0, 0, 0, 0, 0, 0, -5, -5, 0, 0, 0, 0, 0, 0, -5, -5, 0, 0, 0, 0, 0, 0, -5, -5, 0, 0, 0, 0, 0, 0, -5, -5, 0, 0, 0, 0, 0, 0, -5, 0, 0, 0, 5, 5, 0, 0, 0}
	pstQ     = [64]int{-20, -10, -10, -5, -5, -10, -10, -20, -10, 0, 0, 0, 0, 0, 0, -10, -10, 0, 5, 5, 5, 5, 0, -10, -5, 0, 5, 5, 5, 5, 0, -5, 0, 0, 5, 5, 5, 5, 0, -5, -10, 5, 5, 5, 5, 5, 0, -10, -10, 0, 5, 0, 0, 0, 0, -10, -20, -10, -10, -5, -5, -10, -10, -20}
	pstKM    = [64]int{-30, -40, -40, -50, -50, -40, -40, -30, -30, -40, -40, -50, -50, -40, -40, -30, -30, -40, -40, -50, -50, -40, -40, -30, -30, -40, -40, -50, -50, -40, -40, -30, -20, -30, -30, -40, -40, -30, -30, -20, -10, -20, -20, -20, -20, -20, -20, -10, 20, 20, 0, 0, 0, 0, 20, 20, 20, 30, 10, 0, 0, 10, 30, 20}
	pstKE    = [64]int{-50, -40, -30, -20, -20, -30, -40, -50, -30, -20, -10, 0, 0, -10, -20, -30, -30, -10, 20, 30, 30, 20, -10, -30, -30, -10, 30, 40, 40, 30, -10, -30, -30, -10, 30, 40, 40, 30, -10, -30, -30, -10, 20, 30, 30, 20, -10, -30, -30, -30, 0, 0, 0, 0, -30, -30, -50, -30, -30, -30, -30, -30, -30, -50}
	pstMinor = [7]*[64]int{nil, nil, &pstN, &pstB, &pstR, &pstQ, nil}
	passed   = [8]int{0, 5, 10, 20, 35, 60, 100, 0}
)

// Evaluate scores the position for the side to move, in centipawns.
func Evaluate(p *Position) int {
	b := &p.Board
	mg, eg, ph := 0, 0, 0
	var pf [2][8]int
	var pMin, pMax [2][8]int
	for c := 0; c < 2; c++ {
		for f := 0; f < 8; f++ {
			pMin[c][f], pMax[c][f] = 9, -1
		}
	}
	var bishops, nonPawn, mat [2]int
	anyPawn := false
	for s := 0; s < 128; s++ {
		if s&0x88 != 0 {
			s += 7
			continue
		}
		if v := b[s]; v == Pawn || v == -Pawn {
			c, f, r := 0, s&7, s>>4
			if v < 0 {
				c = 1
			}
			pf[c][f]++
			anyPawn = true
			if r < pMin[c][f] {
				pMin[c][f] = r
			}
			if r > pMax[c][f] {
				pMax[c][f] = r
			}
		}
	}
	for s := 0; s < 128; s++ {
		if s&0x88 != 0 {
			s += 7
			continue
		}
		v := b[s]
		if v == 0 {
			continue
		}
		t := abs8(v)
		white := v > 0
		c, sg := 0, 1
		if !white {
			c, sg = 1, -1
		}
		r, f := s>>4, s&7
		idx := r*8 + f
		if white {
			idx = (7-r)*8 + f
		}
		m, e := mgVal[t], egVal[t]
		ph += phase[t]
		mat[c] += mgVal[t]
		if t != Pawn && t != King {
			nonPawn[c] += mgVal[t]
		}
		switch t {
		case King:
			m += pstKM[idx]
			e += pstKE[idx]
		case Pawn:
			m += pstP[idx]
			e += pstP[idx] >> 1
			rel := r
			if !white {
				rel = 7 - r
			}
			e += rel * 4
			o := 1 - c
			isPassed := true
			for ff := max(0, f-1); ff <= min(7, f+1) && isPassed; ff++ {
				if pf[o][ff] == 0 {
					continue
				}
				if (white && pMax[o][ff] > r) || (!white && pMin[o][ff] < r) {
					isPassed = false
				}
			}
			if isPassed {
				m += passed[rel] >> 1
				e += passed[rel]
			}
			if pf[c][f] > 1 {
				m -= 10
				e -= 15
			}
			if (f == 0 || pf[c][f-1] == 0) && (f == 7 || pf[c][f+1] == 0) {
				m -= 12
				e -= 10
			}
		default:
			pv := pstMinor[t][idx]
			m += pv
			e += pv
			if t == Bishop {
				bishops[c]++
			}
			if t == Rook {
				if pf[c][f] == 0 {
					if pf[1-c][f] == 0 {
						m += 18
						e += 10
					} else {
						m += 9
						e += 5
					}
				}
				if (white && r == 6) || (!white && r == 1) {
					m += 10
					e += 15
				}
			}
			switch t {
			case Knight:
				n := 0
				for _, d := range knightOff {
					to := s + d
					if to&0x88 == 0 && (b[to] == 0 || (b[to] > 0) != white) {
						n++
					}
				}
				m += (n - 4) * 3
				e += (n - 4) * 3
			case Bishop, Rook:
				dirs := bishopOff
				if t == Rook {
					dirs = rookOff
				}
				n := 0
				for _, d := range dirs {
					for to := s + d; to&0x88 == 0; to += d {
						if b[to] != 0 {
							if (b[to] > 0) != white {
								n++
							}
							break
						}
						n++
					}
				}
				m += (n - 6) * 2
				e += (n - 6) * 3
			}
		}
		mg += sg * m
		eg += sg * e
	}
	if bishops[0] >= 2 {
		mg += 30
		eg += 45
	}
	if bishops[1] >= 2 {
		mg -= 30
		eg -= 45
	}
	if ph > 24 {
		ph = 24
	}
	score := float64(mg*ph+eg*(24-ph)) / 24
	// Endgames: help the stronger side drive the lone king to the edge.
	diff := mat[0] - mat[1]
	if ph <= 8 && (diff >= 300 || diff <= -300) {
		strong := 0
		if diff < 0 {
			strong = 1
		}
		wk, lk := p.Kings[strong], p.Kings[1-strong]
		lf, lr := lk&7, lk>>4
		centre := max(3-lf, lf-4) + max(3-lr, lr-4)
		dist := absI(wk&7-lf) + absI(wk>>4-lr)
		bonus := float64(centre*12+(14-dist)*5) * float64(24-ph) / 24
		if strong == 0 {
			score += bonus
		} else {
			score -= bonus
		}
	}
	// Without pawns, a side up by only a minor piece can't win.
	if !anyPawn && absI(nonPawn[0]-nonPawn[1]) < 400 && max(nonPawn[0], nonPawn[1]) <= 330 {
		score /= 8
	}
	score += float64(10 * p.Side)
	return int(math.Round(score)) * p.Side
}

func absI(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

const (
	mate   = 30000
	inf    = 32000
	ttBits = 20
	ttSize = 1 << ttBits
	ttMask = ttSize - 1
	exact  = 1
	lower  = 2
	upper  = 3
)

type ttEntry struct {
	key   uint64
	move  int32
	score int16
	depth int8
	flag  uint8
}

func enc(m Move) int32 { return int32(m.From | m.To<<7 | abs8(m.Promo)<<14) }

// Options set how the computer plays.
type Options struct {
	Time     time.Duration // thinking time for full-strength play
	MaxDepth int           // 0 = no limit
	Noise    float64       // weaker players: random noise added to move scores (centipawns)
	Blunder  float64       // weaker players: chance of a careless move
}

// Result is the computer's choice.
type Result struct {
	Move  Move
	OK    bool
	Score int // for the side to move
	Depth int
	Nodes int
}

// Searcher thinks about positions. One search at a time; Stop cancels it.
type Searcher struct {
	tt       []ttEntry
	killers  [128][2]int32
	history  [2 * 16384]int32
	nodes    int
	stopAt   time.Time
	stopped  bool
	cancel   atomic.Bool
	pos      *Position
	rootBest Move
	rootHas  bool
	rootSc   int
	rnd      *rand.Rand
}

// NewSearcher makes a searcher with its own transposition table.
func NewSearcher() *Searcher {
	return &Searcher{tt: make([]ttEntry, ttSize), rnd: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

// Stop asks a running search to finish at once (its result is then not useful).
func (s *Searcher) Stop() { s.cancel.Store(true) }

func (s *Searcher) orderMoves(moves []Move, ply int, tmove int32) {
	for i := range moves {
		m := &moves[i]
		var sc int32
		switch {
		case tmove != 0 && enc(*m) == tmove:
			sc = 1e9
		case m.Flags&FlagPromo != 0:
			if abs8(m.Promo) == Queen {
				sc = 200000
			} else {
				sc = -1000
			}
		case m.Captured != 0:
			sc = int32(100000 + val[abs8(m.Captured)]*10 - val[abs8(m.Piece)]/10)
		case s.killers[ply][0] == enc(*m):
			sc = 90000
		case s.killers[ply][1] == enc(*m):
			sc = 80000
		default:
			sc = s.history[histIdx(*m)]
		}
		m.score = sc
	}
	sort.SliceStable(moves, func(a, b int) bool { return moves[a].score > moves[b].score })
}

func histIdx(m Move) int {
	c := 0
	if m.Piece < 0 {
		c = 1
	}
	return c*16384 + m.From*128 + m.To
}

func (s *Searcher) drawish() bool {
	p := s.pos
	if p.Half >= 100 {
		return true
	}
	ks := p.keys
	n := len(ks) - 1
	k := ks[n]
	for i := n - 2; i >= 0 && i >= n-p.Half; i -= 2 {
		if ks[i] == k {
			return true
		}
	}
	return false
}

func (s *Searcher) hasNonPawn(side int) bool {
	b := &s.pos.Board
	for sq := 0; sq < 128; sq++ {
		if sq&0x88 != 0 {
			sq += 7
			continue
		}
		v := int(b[sq]) * side
		if v > 1 && v < 6 {
			return true
		}
	}
	return false
}

func (s *Searcher) tick() bool {
	s.nodes++
	if s.nodes&2047 == 0 && (time.Now().After(s.stopAt) || s.cancel.Load()) {
		s.stopped = true
	}
	return s.stopped
}

func (s *Searcher) quiesce(alpha, beta, ply int) int {
	if s.tick() {
		return 0
	}
	p := s.pos
	stand := Evaluate(p)
	if stand >= beta {
		return stand
	}
	if stand > alpha {
		alpha = stand
	}
	if ply > 60 {
		return stand
	}
	moves := p.Pseudo(true, make([]Move, 0, 16))
	for i := range moves {
		m := &moves[i]
		switch {
		case m.Captured != 0:
			m.score = int32(100000 + val[abs8(m.Captured)]*10 - val[abs8(m.Piece)]/10)
		case abs8(m.Promo) == Queen:
			m.score = 150000
		default:
			m.score = -1
		}
	}
	sort.SliceStable(moves, func(a, b int) bool { return moves[a].score > moves[b].score })
	us := p.Side
	for _, m := range moves {
		if m.score < 0 {
			continue // under-promotions
		}
		if m.Captured != 0 && m.Flags&FlagPromo == 0 && stand+val[abs8(m.Captured)]+200 < alpha {
			continue // delta pruning
		}
		p.Make(m)
		if p.Attacked(p.Kings[kingIdx(us)], -us) {
			p.Unmake()
			continue
		}
		sc := -s.quiesce(-beta, -alpha, ply+1)
		p.Unmake()
		if s.stopped {
			return 0
		}
		if sc >= beta {
			return sc
		}
		if sc > alpha {
			alpha = sc
		}
	}
	return alpha
}

func (s *Searcher) search(depth, alpha, beta, ply int, allowNull bool) int {
	if s.tick() {
		return 0
	}
	p := s.pos
	if ply > 0 && s.drawish() {
		return 0
	}
	us := p.Side
	inCheck := p.InCheck()
	if inCheck {
		depth++
	}
	if depth <= 0 {
		return s.quiesce(alpha, beta, ply)
	}
	if ply > 0 {
		alpha = max(alpha, -mate+ply)
		beta = min(beta, mate-ply-1)
		if alpha >= beta {
			return alpha
		}
	}
	slot := &s.tt[p.Hash&ttMask]
	var tmove int32
	if slot.key == p.Hash {
		tmove = slot.move
		if ply > 0 && int(slot.depth) >= depth {
			sc := int(slot.score)
			if sc > mate-500 {
				sc -= ply
			} else if sc < -mate+500 {
				sc += ply
			}
			if slot.flag == exact || (slot.flag == lower && sc >= beta) || (slot.flag == upper && sc <= alpha) {
				return sc
			}
		}
	}
	pvNode := beta-alpha > 1
	if allowNull && !pvNode && !inCheck && depth >= 3 && ply > 0 && absI(beta) < mate-500 && s.hasNonPawn(us) && Evaluate(p) >= beta {
		p.makeNull()
		sc := -s.search(depth-3, -beta, -beta+1, ply+1, false)
		p.unmakeNull()
		if s.stopped {
			return 0
		}
		if sc >= beta {
			return beta
		}
	}
	moves := p.Pseudo(false, make([]Move, 0, 64))
	s.orderMoves(moves, ply, tmove)
	best, legal := -inf, 0
	var bestMove Move
	hasBest := false
	a0 := alpha
	for _, m := range moves {
		p.Make(m)
		if p.Attacked(p.Kings[kingIdx(us)], -us) {
			p.Unmake()
			continue
		}
		legal++
		quiet := m.Captured == 0 && m.Promo == 0
		var sc int
		if legal == 1 {
			sc = -s.search(depth-1, -beta, -alpha, ply+1, true)
		} else {
			red := 0
			if depth >= 3 && legal > 3 && quiet && !inCheck && !p.InCheck() {
				red = 1
				if legal > 10 {
					red = 2
				}
			}
			sc = -s.search(depth-1-red, -alpha-1, -alpha, ply+1, true)
			if sc > alpha && red > 0 {
				sc = -s.search(depth-1, -alpha-1, -alpha, ply+1, true)
			}
			if sc > alpha && sc < beta {
				sc = -s.search(depth-1, -beta, -alpha, ply+1, true)
			}
		}
		p.Unmake()
		if s.stopped {
			return 0
		}
		if sc > best {
			best, bestMove, hasBest = sc, m, true
			if ply == 0 {
				s.rootBest, s.rootHas, s.rootSc = m, true, sc
			}
			if sc > alpha {
				alpha = sc
				if sc >= beta {
					if quiet {
						e := enc(m)
						if s.killers[ply][0] != e {
							s.killers[ply][1] = s.killers[ply][0]
							s.killers[ply][0] = e
						}
						s.history[histIdx(m)] += int32(depth * depth)
					}
					break
				}
			}
		}
	}
	if legal == 0 {
		if inCheck {
			return -mate + ply
		}
		return 0
	}
	st := best
	if st > mate-500 {
		st += ply
	} else if st < -mate+500 {
		st -= ply
	}
	slot.key = p.Hash
	slot.move = 0
	if hasBest {
		slot.move = enc(bestMove)
	}
	slot.score = int16(st)
	slot.depth = int8(depth)
	switch {
	case best >= beta:
		slot.flag = lower
	case best > a0:
		slot.flag = exact
	default:
		slot.flag = upper
	}
	return best
}

func (s *Searcher) gauss() float64 { return s.rnd.NormFloat64() }

// Think picks a move for the side to move in pos (pos is not changed).
func (s *Searcher) Think(pos *Position, o Options) Result {
	s.pos = pos.Clone()
	s.cancel.Store(false)
	legal := s.pos.Legal()
	if len(legal) == 0 {
		return Result{}
	}
	if len(legal) == 1 {
		return Result{Move: legal[0], OK: true}
	}
	s.killers = [128][2]int32{}
	s.history = [2 * 16384]int32{}
	s.nodes = 0
	s.stopped = false
	if o.Time <= 0 {
		o.Time = time.Second
	}
	t0 := time.Now()
	s.stopAt = t0.Add(o.Time)
	maxDepth := o.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 64
	}

	// Weaker players: score every root move at a shallow depth, then pick with noise.
	if o.Noise > 0 {
		s.stopAt = t0.Add(20 * time.Second)
		depth := max(1, min(maxDepth, 3))
		type scored struct {
			m  Move
			sc float64
		}
		list := make([]scored, 0, len(legal))
		for _, m := range legal {
			s.pos.Make(m)
			sc := float64(-s.search(depth-1, -inf, inf, 1, true))
			s.pos.Unmake()
			if s.stopped {
				return Result{}
			}
			if m.Captured != 0 && s.rnd.Float64() < o.Blunder*.5 {
				sc -= 120 // sometimes misses a capture
			}
			list = append(list, scored{m, sc + s.gauss()*o.Noise})
		}
		if s.rnd.Float64() < o.Blunder { // an outright careless move, never one that allows mate
			var ok []scored
			for _, x := range list {
				if x.sc > -mate+500 {
					ok = append(ok, x)
				}
			}
			pick := list[0]
			if len(ok) > 0 {
				pick = ok[s.rnd.Intn(len(ok))]
			}
			return Result{Move: pick.m, OK: true, Score: int(pick.sc), Depth: depth}
		}
		sort.SliceStable(list, func(a, b int) bool { return list[a].sc > list[b].sc })
		return Result{Move: list[0].m, OK: true, Score: int(list[0].sc), Depth: depth}
	}

	var best Move
	hasBest, bestScore, done := false, 0, 0
	for d := 1; d <= maxDepth; d++ {
		s.rootHas = false
		sc := s.search(d, -inf, inf, 0, true)
		if s.stopped {
			if s.rootHas && hasBest && s.rootSc > bestScore+30 {
				best = s.rootBest
			}
			break
		}
		if s.rootHas {
			best, hasBest, bestScore, done = s.rootBest, true, sc, d
		}
		if absI(sc) > mate-200 {
			break // found a forced mate
		}
		if time.Since(t0) > o.Time*55/100 {
			break // the next ply won't finish
		}
	}
	if !hasBest {
		if s.cancel.Load() {
			return Result{}
		}
		best = legal[0]
	}
	if s.cancel.Load() {
		return Result{}
	}
	return Result{Move: best, OK: true, Score: bestScore, Depth: done, Nodes: s.nodes}
}
