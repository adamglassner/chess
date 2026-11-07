// Game flow: moves, the computer's turn, clocks, undo, hints, draw offers,
// resigning, results and the player's record.
//
// (c) 2026 Ironite
package main

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/adamglassner/chess/engine"
)

type histEntry struct {
	m   engine.Move
	san string
	uci string
}

type anim struct {
	piece          int8
	x0, y0, x1, y1 float64 // board-relative logical coordinates
	t0             time.Time
	dur            time.Duration
	fade           bool
	hideSq         int // square kept empty until this animation ends (-1: none)
}

type aiJob struct {
	kind    string // "move", "hint", "draw"
	gen     int
	ch      chan engine.Result
	start   time.Time
	minWait time.Duration
	got     bool
	res     engine.Result
	book    string // a book move chosen without searching
	srch    *engine.Searcher
}

type App struct {
	sd *SaveData
	G  *GameRec

	pos     *engine.Position
	hist    []histEntry
	legal   []engine.Move
	view    int // -1: live position; else number of plies shown
	viewPos *engine.Position
	flipped bool

	sel       int
	selMoves  []engine.Move
	drag      *dragT
	anims     []anim
	marks     map[int]bool
	arrows    [][2]int
	hintArrow *[2]int
	rightFrom int
	promo     []engine.Move

	busy         bool
	gen          int
	job          *aiJob
	drawOfferPly int
	opening      string

	lastTick   time.Time
	lastSaveAt time.Time

	ui    uiState
	net   *onlineT
	names map[string]oppName // names drawn for the next game against each strength
}

type dragT struct {
	from          int
	x, y, x0, y0  float64
	moved, wasSel bool
}

var searchers = sync.Pool{New: func() any { return engine.NewSearcher() }}

func (a *App) isAI() bool { return a.G != nil && a.G.Mode == "ai" }
func (a *App) aiSide() int {
	if a.G.YouWhite {
		return engine.Black
	}
	return engine.White
}
func (a *App) yourTurn() bool {
	return a.G != nil && a.G.Result == nil && (a.G.Mode == "pvp" || a.pos.Side == a.youSide())
}
func (a *App) youSide() int {
	if a.G.YouWhite {
		return engine.White
	}
	return engine.Black
}
func (a *App) live() bool { return a.view < 0 }
func (a *App) curPos() *engine.Position {
	if a.live() {
		return a.pos
	}
	return a.viewPos
}
func (a *App) oppShort() string {
	if a.G.OppShort != "" {
		return a.G.OppShort
	}
	return a.opp().Name
}
func (a *App) oppFull() string {
	if a.G.OppFull != "" {
		return a.G.OppFull
	}
	return a.opp().Full
}

// sideLabel names a side: the player names in two-player games, else White/Black.
func (a *App) sideLabel(s int) string {
	if a.G != nil && a.G.Mode != "ai" {
		n := a.G.WName
		if s == engine.Black {
			n = a.G.BName
		}
		if n != "" {
			return n
		}
	}
	return sideName(s)
}

func (a *App) opp() *folk {
	if f := folkByID(a.G.Opp); f != nil {
		return f
	}
	return &folks[0]
}
func animDur(s *Settings) time.Duration {
	switch s.Anim {
	case "slow":
		return 320 * time.Millisecond
	case "fast":
		return 110 * time.Millisecond
	}
	return 200 * time.Millisecond
}
func sideName(s int) string {
	if s == engine.White {
		return "White"
	}
	return "Black"
}

func (a *App) startFen() string {
	if a.G.StartFen == "" {
		return engine.StartFEN
	}
	return a.G.StartFen
}

// rebuild replays the saved moves into the live position.
func (a *App) rebuild() {
	p, err := engine.NewPosition(a.startFen())
	if err != nil {
		p = engine.MustPosition(engine.StartFEN)
		a.G.StartFen = engine.StartFEN
	}
	a.pos, a.hist = p, nil
	for _, u := range a.G.Moves {
		legal := a.pos.Legal()
		m, ok := findUCI(legal, u)
		if !ok {
			break
		}
		a.hist = append(a.hist, histEntry{m, a.pos.SAN(m, legal), u})
		a.pos.Make(m)
	}
	a.G.Moves = a.G.Moves[:len(a.hist)]
	a.legal = a.pos.Legal()
}

func findUCI(legal []engine.Move, u string) (engine.Move, bool) {
	for _, m := range legal {
		if m.UCI() == u {
			return m, true
		}
	}
	return engine.Move{}, false
}

func (a *App) persist() {
	a.sd.Game = a.G
	writeSave(a.sd)
}

/* ------------------------------------------------------------ geometry */

const (
	boardX = 30.0 // frame origin on the stage
	boardY = 78.0
	frameM = 40.0 // frame width
	sqSize = 90.0
)

// sqXY gives a square's top-left corner relative to the frame origin.
func (a *App) sqXY(sq int) (float64, float64) {
	f, r := sq&7, sq>>4
	if a.flipped {
		return frameM + float64(7-f)*sqSize, frameM + float64(r)*sqSize
	}
	return frameM + float64(f)*sqSize, frameM + float64(7-r)*sqSize
}

// xySq gives the square under a board-relative point, or -1.
func (a *App) xySq(x, y float64) int {
	f, r := int(math.Floor((x-frameM)/sqSize)), int(math.Floor((y-frameM)/sqSize))
	if x < frameM || y < frameM || f < 0 || f > 7 || r < 0 || r > 7 {
		return -1
	}
	if a.flipped {
		return r*16 + (7 - f)
	}
	return (7-r)*16 + f
}

/* ------------------------------------------------------------ animation */

func (a *App) animate(piece int8, from, to int) {
	x0, y0 := a.sqXY(from)
	x1, y1 := a.sqXY(to)
	a.anims = append(a.anims, anim{piece: piece, x0: x0, y0: y0, x1: x1, y1: y1, t0: time.Now(), dur: animDur(&a.sd.Settings), hideSq: to})
}
func (a *App) fadeOut(piece int8, sq int) {
	x0, y0 := a.sqXY(sq)
	a.anims = append(a.anims, anim{piece: piece, x0: x0, y0: y0, t0: time.Now(), dur: animDur(&a.sd.Settings), fade: true, hideSq: -1})
}
func (a *App) animating() bool { return len(a.anims) > 0 }

func (a *App) updateAnims() {
	now := time.Now()
	keep := a.anims[:0]
	for _, an := range a.anims {
		if now.Sub(an.t0) < an.dur {
			keep = append(keep, an)
		}
	}
	a.anims = keep
}

func (a *App) hidden(sq int) bool {
	for _, an := range a.anims {
		if an.hideSq == sq {
			return true
		}
	}
	return false
}

/* ------------------------------------------------------------ moves */

func (a *App) playMove(m engine.Move) {
	san, mover := a.pos.SAN(m, a.legal), a.pos.Side
	if a.G.Clock != nil && len(a.hist) > 0 {
		if mover == engine.White {
			a.G.Clock.W += a.G.Clock.Inc
		} else {
			a.G.Clock.B += a.G.Clock.Inc
		}
	}
	a.hist = append(a.hist, histEntry{m, san, m.UCI()})
	a.G.Moves = append(a.G.Moves, m.UCI())
	if a.online() && mover == a.youSide() {
		a.sendMove(m.UCI())
	}
	a.sel, a.selMoves, a.hintArrow = -1, nil, nil
	a.marks, a.arrows = map[int]bool{}, nil

	if m.Captured != 0 && m.Flags&engine.FlagEnPassant == 0 {
		a.fadeOut(m.Captured, m.To)
	}
	if m.Flags&engine.FlagEnPassant != 0 {
		a.fadeOut(m.Captured, m.To-16*mover)
	}
	a.animate(m.Piece, m.From, m.To)
	if m.Flags&engine.FlagCastle != 0 {
		rf, rt := m.From-4, m.From-1
		if m.To > m.From {
			rf, rt = m.From+3, m.From+1
		}
		a.animate(int8(engine.Rook*mover), rf, rt)
	}
	a.pos.Make(m)
	a.legal = a.pos.Legal()

	chk := a.pos.InCheck()
	switch {
	case m.Promo != 0:
		play("promote")
	case m.Flags&engine.FlagCastle != 0:
		play("castle")
	case m.Captured != 0:
		play("capture")
	default:
		play("move")
	}
	if chk {
		playLater("check", 120*time.Millisecond)
	}
	if st := a.pos.Status(); st.Over {
		a.G.Result = &ResultRec{Result: st.Result, Reason: st.Reason, Winner: st.Winner}
	}
	a.persist()
	a.refreshOpening()
	a.ui.scrollToCurrent = true
	if a.G.Result != nil {
		a.recordResult()
		g := a.G
		a.ui.after(animDur(&a.sd.Settings), func() {
			if a.G == g {
				a.gameOver()
			}
		})
		return
	}
	if a.online() {
		a.ui.statusOverride = ""
	}
	if a.isAI() && chk && mover == a.aiSide() && rand.Float64() < .5 {
		a.quip("check")
	}
	if a.G.Mode == "pvp" && a.sd.Settings.Turn == "on" {
		flip, g := a.pos.Side == engine.Black, a.G
		a.ui.after(animDur(&a.sd.Settings)+250*time.Millisecond, func() {
			if a.G == g {
				a.setFlip(flip)
			}
		})
	}
	if a.isAI() && a.pos.Side == a.aiSide() {
		a.aiTurn()
	}
}

// aiTurn starts the computer thinking about its move.
func (a *App) aiTurn() {
	a.gen++
	a.busy = true
	L, ok := levels[a.G.Opp]
	if !ok {
		L = levels["pip"]
	}
	job := &aiJob{kind: "move", gen: a.gen, start: time.Now()}
	job.minWait = L.minDelay + time.Duration(rand.Int63n(int64(L.maxDelay-L.minDelay)+1))
	if a.startFen() == engine.StartFEN && rand.Float64() < L.book {
		if opts := engine.BookNext(a.G.Moves); len(opts) > 0 {
			job.book = opts[rand.Intn(len(opts))].UCI
		}
	}
	if job.book == "" {
		o := L.ai
		if a.G.Clock != nil {
			left := a.G.Clock.W
			if a.aiSide() == engine.Black {
				left = a.G.Clock.B
			}
			t := time.Duration(left/30+a.G.Clock.Inc*.8) * time.Millisecond
			if o.Time == 0 {
				o.Time = 600 * time.Millisecond
			}
			o.Time = max(150*time.Millisecond, min(o.Time, t))
		}
		a.startSearch(job, o)
	}
	a.job = job
}

func (a *App) startSearch(job *aiJob, o engine.Options) {
	job.ch = make(chan engine.Result, 1)
	s := searchers.Get().(*engine.Searcher)
	job.srch = s
	p := a.pos.Clone()
	go func() {
		r := s.Think(p, o)
		searchers.Put(s)
		job.ch <- r
	}()
}

// cancelThinking abandons any search in progress.
func (a *App) cancelThinking() {
	if a.job != nil && a.job.srch != nil {
		a.job.srch.Stop()
	}
	a.job = nil
	a.gen++
	a.busy = false
}

// updateJob collects the computer's answer when it's ready.
func (a *App) updateJob() {
	j := a.job
	if j == nil || j.gen != a.gen {
		return
	}
	if !j.got {
		if j.book != "" {
			j.got = true
		} else {
			select {
			case r := <-j.ch:
				j.res, j.got = r, true
			default:
				return
			}
		}
	}
	switch j.kind {
	case "move":
		if time.Since(j.start) < j.minWait || a.animating() || a.G == nil || a.G.Result != nil {
			if a.G == nil || a.G.Result != nil {
				a.job, a.busy = nil, false
			}
			return
		}
		a.job, a.busy = nil, false
		var m engine.Move
		ok := false
		if j.book != "" {
			m, ok = findUCI(a.legal, j.book)
		} else if j.res.OK {
			m, ok = findUCI(a.legal, j.res.Move.UCI())
		}
		if !ok && len(a.legal) > 0 {
			m, ok = a.legal[0], true
		}
		if ok {
			a.playMove(m)
		}
	case "hint":
		a.job, a.busy = nil, false
		if j.res.OK {
			if m, ok := findUCI(a.legal, j.res.Move.UCI()); ok {
				a.hintArrow = &[2]int{m.From, m.To}
				play("hint")
				a.ui.statusOverride = "Hint: " + a.pos.SAN(m, a.legal)
			}
		}
	case "draw":
		a.job, a.busy = nil, false
		aiScore := -j.res.Score // the search scored the position for you
		accept := aiScore <= -80 || (aiScore <= 30 && len(a.hist) >= 50) || (a.G.Opp == "pip" && aiScore <= 0)
		if accept {
			a.quip("accept")
			a.G.Result = &ResultRec{Result: "1/2-1/2", Reason: "agreement"}
			a.persist()
			a.gameOver()
		} else {
			a.quip("decline")
		}
	}
}

/* ------------------------------------------------------------ input on the board */

func (a *App) canMoveFrom(sq int) bool {
	if a.G == nil || a.G.Result != nil || !a.live() || a.busy || a.promo != nil || !a.yourTurn() {
		return false
	}
	pc := a.pos.Board[sq]
	return pc != 0 && (pc > 0) == (a.pos.Side > 0)
}

func (a *App) selectSq(sq int) {
	a.sel = sq
	a.selMoves = a.selMoves[:0]
	for _, m := range a.legal {
		if m.From == sq {
			a.selMoves = append(a.selMoves, m)
		}
	}
	play("pick")
}

func (a *App) tryMove(from, to int) bool {
	var ms []engine.Move
	for _, m := range a.legal {
		if m.From == from && m.To == to {
			ms = append(ms, m)
		}
	}
	if len(ms) == 0 {
		return false
	}
	if len(ms) > 1 { // promotion
		if a.sd.Settings.Promo == "queen" {
			for _, m := range ms {
				if absI8(m.Promo) == engine.Queen {
					a.playMove(m)
				}
			}
			return true
		}
		a.promo = ms
		return true
	}
	a.playMove(ms[0])
	return true
}

func (a *App) setFlip(f bool) {
	if f == a.flipped {
		return
	}
	a.flipped = f
	if a.G != nil {
		a.G.Flipped = f
	}
}

/* ------------------------------------------------------------ clocks */

func (a *App) updateClock() {
	now := time.Now()
	dt := float64(now.Sub(a.lastTick)) / float64(time.Millisecond)
	a.lastTick = now
	if a.G == nil || a.G.Clock == nil || a.G.Result != nil || len(a.hist) == 0 || (!a.online() && a.ui.paused()) {
		return
	}
	c := a.G.Clock
	left := &c.W
	if a.pos.Side == engine.Black {
		left = &c.B
	}
	before := *left
	*left -= dt
	if before > 10000 && *left <= 10000 && (a.G.Mode == "pvp" || a.pos.Side != a.aiSide()) {
		play("tick")
	}
	if *left <= 0 {
		*left = 0
		if !a.online() || a.net == nil || a.net.host { // online, the host's clock decides
			a.flag(a.pos.Side)
		}
	}
	if now.Sub(a.lastSaveAt) > 5*time.Second {
		a.lastSaveAt = now
		a.persist()
	}
}

func (a *App) canMate(side int) bool {
	minors := 0
	for s := 0; s < 128; s++ {
		if s&0x88 != 0 {
			s += 7
			continue
		}
		pc := int(a.pos.Board[s]) * side
		if pc == engine.Pawn || pc == engine.Rook || pc == engine.Queen {
			return true
		}
		if pc == engine.Knight || pc == engine.Bishop {
			minors++
		}
	}
	return minors >= 2
}

func (a *App) flag(side int) {
	a.cancelThinking()
	if a.canMate(-side) {
		res := "1-0"
		if side == engine.White {
			res = "0-1"
		}
		a.G.Result = &ResultRec{Result: res, Reason: "time", Winner: -side}
	} else {
		a.G.Result = &ResultRec{Result: "1/2-1/2", Reason: "time vs insufficient material"}
	}
	if a.online() {
		a.hostFlag(a.G.Result)
	}
	a.persist()
	a.gameOver()
}

/* ------------------------------------------------------------ results */

func (a *App) resultLine() string {
	r := a.G.Result
	if r == nil {
		return ""
	}
	why := map[string]string{"checkmate": "by checkmate", "resignation": "by resignation", "time": "on time", "stalemate": "Stalemate",
		"insufficient material": "Not enough material to mate", "fifty-move rule": "Fifty-move rule", "threefold repetition": "Threefold repetition",
		"agreement": "Draw agreed", "time vs insufficient material": "Flag fell, but no mating material"}[r.Reason]
	if why == "" {
		why = r.Reason
	}
	if r.Reason == "left" {
		return "Game stopped"
	}
	if r.Winner == 0 {
		return "Draw · " + why
	}
	return a.sideLabel(r.Winner) + " wins " + why
}

// recordResult adds a finished game to the player's record (once).
func (a *App) recordResult() {
	r := a.G.Result
	if r == nil || a.G.Counted || r.Reason == "left" {
		return
	}
	a.G.Counted = true
	switch a.G.Mode {
	case "ai":
		rec := a.sd.Stats.Per[a.G.Opp]
		if rec == nil {
			rec = &Record{}
			a.sd.Stats.Per[a.G.Opp] = rec
		}
		switch {
		case r.Winner == 0:
			a.sd.Stats.Drawn++
			rec.D++
		case r.Winner == a.youSide():
			a.sd.Stats.Won++
			rec.W++
		default:
			a.sd.Stats.Lost++
			rec.L++
		}
	case "online":
		rec := &a.sd.Stats.Online
		switch {
		case r.Winner == 0:
			rec.D++
		case r.Winner == a.youSide():
			rec.W++
		default:
			rec.L++
		}
	default:
		a.sd.Stats.Pvp++
	}
	writeSave(a.sd)
}

func (a *App) gameOver() {
	if a.G == nil || a.G.Result == nil {
		return
	}
	a.busy = false
	r := a.G.Result
	o := &a.ui.over
	o.face, o.quote = "you", ""
	moves := (len(a.hist) + 1) / 2
	if a.isAI() {
		o.face = a.G.Opp
		a.recordResult()
		rec := a.sd.Stats.Per[a.G.Opp]
		you := a.youSide()
		kind := "win"
		switch {
		case r.Winner == 0:
			o.title, o.kick, kind = "A Draw", "Honours even", "draw"
			play("draw")
		case r.Winner == you:
			o.title, o.kick, kind = "You Win!", "Victory", "lose"
			if r.Reason == "checkmate" {
				o.kick = "Checkmate"
			}
			play("win")
			a.ui.confetti()
		default:
			o.title, o.kick = a.oppShort()+" Wins", "Defeat"
			if r.Reason == "checkmate" {
				o.kick = "Checkmate"
			}
			play("lose")
		}
		q := quips[a.G.Opp][kind]
		o.quote = fmt.Sprintf("“%s” — %s", q[rand.Intn(len(q))], a.oppFull())
		o.text = fmt.Sprintf("%s after %d moves. Your record against %s players: %d won, %d drawn, %d lost.", a.resultLine(), moves, a.opp().Rating, rec.W, rec.D, rec.L)
	} else if a.online() {
		a.recordResult()
		rec := &a.sd.Stats.Online
		you := a.youSide()
		o.kick = "Game over"
		if r.Reason == "checkmate" {
			o.kick = "Checkmate"
		}
		o.face = "p2"
		switch {
		case r.Winner == 0:
			o.title = "A Draw"
			play("draw")
		case r.Winner == you:
			o.title, o.face = "You Win!", "you"
			play("win")
			a.ui.confetti()
		default:
			o.title = a.sideLabel(r.Winner) + " Wins"
			play("lose")
		}
		o.text = fmt.Sprintf("%s after %d moves. Your online record: %d won, %d drawn, %d lost.", a.resultLine(), moves, rec.W, rec.D, rec.L)
	} else {
		a.recordResult()
		o.kick = "Game over"
		if r.Reason == "checkmate" {
			o.kick = "Checkmate"
		}
		if r.Winner != 0 {
			o.title = a.sideLabel(r.Winner) + " Wins"
			play("win")
			a.ui.confetti()
		} else {
			o.title = "A Draw"
			play("draw")
		}
		if r.Winner == engine.Black {
			o.face = "p2"
		}
		o.text = fmt.Sprintf("%s after %d moves.", a.resultLine(), moves)
	}
	if a.online() && a.net != nil {
		a.net.pending = ""
	}
	a.ui.statusOverride = ""
	a.persist()
	g := a.G
	a.ui.after(900*time.Millisecond, func() {
		if a.G == g && a.G.Result != nil {
			a.ui.showUnder(ovOver)
		}
	})
}

func (a *App) quip(kind string) {
	if !a.isAI() {
		return
	}
	list := quips[a.G.Opp][kind]
	if len(list) == 0 {
		return
	}
	a.ui.bubble(list[rand.Intn(len(list))], a.aiSide())
}

/* ------------------------------------------------------------ commands */

func clockFor(base string) *ClockRec {
	spec := map[string][2]float64{"5+3": {5, 3}, "10": {10, 0}, "15+10": {15, 10}, "30": {30, 0}}[base]
	if spec[0] == 0 {
		return nil
	}
	return &ClockRec{W: spec[0] * 60000, B: spec[0] * 60000, Inc: spec[1] * 1000, Base: base}
}

func (a *App) newGame(s Setup) {
	youW := s.Color != "black"
	if s.Color == "random" {
		youW = rand.Intn(2) == 0
	}
	g := &GameRec{Mode: s.Mode, Opp: s.Opp, YouWhite: youW, Clock: clockFor(s.Clock)}
	if s.Mode == "ai" {
		n, ok := a.names[s.Opp]
		if !ok {
			n = randomName(s.Opp, "")
		}
		g.OppShort, g.OppFull = n.Short, n.Full
		a.names[s.Opp] = randomName(s.Opp, lastWord(n.Short)) // a new name next time
	} else {
		g.YouWhite = true
		g.WName = cleanName(a.sd.Settings.P1, "Player 1")
		g.BName = cleanName(a.sd.Settings.P2, "Player 2")
	}
	a.leaveOnline()
	a.newGameRec(g)
}

// newGameRec starts the game described by g.
func (a *App) newGameRec(g *GameRec) {
	a.cancelThinking()
	g.StartFen, g.Moves, g.Started = engine.StartFEN, []string{}, time.Now().UnixMilli()
	a.G = g
	a.drawOfferPly = -99
	a.view, a.viewPos, a.sel, a.selMoves = -1, nil, -1, nil
	a.marks, a.arrows, a.hintArrow, a.anims, a.promo = map[int]bool{}, nil, nil, nil, nil
	a.ui.statusOverride = ""
	a.rebuild()
	a.flipped = a.G.Mode != "pvp" && !a.G.YouWhite
	a.G.Flipped = a.flipped
	a.refreshOpening()
	a.persist()
	a.ui.hideAll()
	play("start")
	if a.isAI() {
		g := a.G
		a.ui.after(500*time.Millisecond, func() {
			if a.G == g {
				a.quip("greet")
			}
		})
		if a.pos.Side == a.aiSide() {
			a.aiTurn()
		}
	}
}

func (a *App) resume() {
	a.G = a.sd.Game
	a.rebuild()
	a.flipped = a.G.Flipped
	a.view, a.viewPos, a.busy, a.sel, a.selMoves = -1, nil, false, -1, nil
	a.refreshOpening()
	a.ui.hideAll()
	if a.isAI() && a.G.Result == nil && a.pos.Side == a.aiSide() {
		a.aiTurn()
	}
}

func (a *App) undo() {
	if a.G == nil || a.G.Result != nil || len(a.hist) == 0 {
		return
	}
	if a.online() {
		// ask the friend; only a move of yours can be taken back
		if len(a.hist) == 1 && a.pos.Side == a.youSide() {
			return
		}
		a.askPeer("undo")
		return
	}
	if !a.live() {
		a.setView(len(a.hist))
	}
	a.cancelThinking()
	n := 1
	if a.isAI() {
		if a.pos.Side != a.aiSide() {
			n = 2
		}
		if n > len(a.hist) {
			n = len(a.hist)
		}
		if !a.G.YouWhite && len(a.hist)-n == 0 {
			n = len(a.hist) - 1 // keep the computer's first move when you play Black
		}
		if n <= 0 {
			return
		}
	}
	a.takeBack(n)
	if a.isAI() && a.pos.Side == a.aiSide() {
		a.aiTurn()
	}
}

// takeBack removes the last n moves.
func (a *App) takeBack(n int) {
	if n <= 0 || n > len(a.hist) {
		return
	}
	if !a.live() {
		a.view, a.viewPos = -1, nil
	}
	last := a.hist[len(a.hist)-n]
	a.G.Moves = a.G.Moves[:len(a.G.Moves)-n]
	a.G.Result = nil
	a.rebuild()
	a.animate(last.m.Piece, last.m.To, last.m.From)
	play("move")
	a.sel, a.selMoves, a.hintArrow = -1, nil, nil
	if a.G.Mode == "pvp" && a.sd.Settings.Turn == "on" {
		a.setFlip(a.pos.Side == engine.Black)
	}
	a.refreshOpening()
	a.persist()
}

func (a *App) hint() {
	if a.G == nil || a.online() || !a.yourTurn() || !a.live() || a.busy {
		return
	}
	a.gen++
	a.busy = true
	a.ui.statusOverride = "Considering the position…"
	job := &aiJob{kind: "hint", gen: a.gen, start: time.Now()}
	a.startSearch(job, engine.Options{Time: 1200 * time.Millisecond})
	a.job = job
}

func (a *App) resign() {
	if a.G == nil || a.G.Result != nil || len(a.hist) == 0 {
		return
	}
	text := a.oppFull() + " will be credited with the win."
	switch a.G.Mode {
	case "pvp":
		text = fmt.Sprintf("%s resigns and %s wins the game.", a.sideLabel(a.pos.Side), a.sideLabel(-a.pos.Side))
	case "online":
		text = a.sideLabel(-a.youSide()) + " will be credited with the win."
	}
	a.ui.ask("Resign?", text, "Resign", "Keep Playing", func() {
		if a.G.Result != nil {
			return
		}
		a.cancelThinking()
		loser := a.pos.Side
		if a.G.Mode != "pvp" {
			loser = a.youSide()
		}
		if a.online() && a.net != nil {
			a.net.peer.send(Msg{T: "resign"})
		}
		a.G.Result = &ResultRec{Result: resultFor(loser), Reason: "resignation", Winner: -loser}
		a.persist()
		a.gameOver()
	}, nil)
}

func (a *App) offerDraw() {
	if a.G == nil || a.G.Result != nil || !a.yourTurn() {
		return
	}
	if a.online() {
		a.askPeer("draw")
		return
	}
	if a.G.Mode == "pvp" {
		a.ui.ask("Draw offered", fmt.Sprintf("%s offers a draw. %s, do you accept?", a.sideLabel(a.pos.Side), a.sideLabel(-a.pos.Side)), "Accept Draw", "Play On", func() {
			a.G.Result = &ResultRec{Result: "1/2-1/2", Reason: "agreement"}
			a.persist()
			a.gameOver()
		}, nil)
		return
	}
	a.drawOfferPly = len(a.hist)
	a.gen++
	a.busy = true
	a.ui.statusOverride = a.oppShort() + " considers your offer…"
	job := &aiJob{kind: "draw", gen: a.gen, start: time.Now()}
	a.startSearch(job, engine.Options{Time: 700 * time.Millisecond})
	a.job = job
}

func (a *App) setView(v int) {
	if a.G == nil {
		return
	}
	if v >= len(a.hist) {
		a.view, a.viewPos = -1, nil
	} else {
		if v < 0 {
			v = 0
		}
		a.view = v
		p, _ := engine.NewPosition(a.startFen())
		for i := 0; i < v; i++ {
			p.Make(a.hist[i].m)
		}
		a.viewPos = p
	}
	a.sel, a.selMoves, a.hintArrow = -1, nil, nil
	a.refreshOpening()
	a.ui.scrollToCurrent = true
	play("click")
}

func (a *App) refreshOpening() {
	if a.G == nil || a.startFen() != engine.StartFEN || len(a.G.Moves) == 0 {
		a.opening = ""
		return
	}
	n := len(a.G.Moves)
	if !a.live() {
		n = a.view
	}
	if name := engine.OpeningName(a.G.Moves[:n]); name != "" {
		a.opening = name
	}
}

func (a *App) pgn() string {
	d := time.UnixMilli(a.G.Started)
	if a.G.Started == 0 {
		d = time.Now()
	}
	wName, bName := a.sideLabel(engine.White), a.sideLabel(engine.Black)
	if a.isAI() {
		if a.G.YouWhite {
			wName, bName = "You", a.oppFull()
		} else {
			wName, bName = a.oppFull(), "You"
		}
	}
	res := "*"
	if a.G.Result != nil {
		res = a.G.Result.Result
	}
	var b strings.Builder
	fmt.Fprintf(&b, "[Event \"Ravenscroft Library\"]\n[Site \"Ironite Chess\"]\n[Date \"%s\"]\n[White \"%s\"]\n[Black \"%s\"]\n[Result \"%s\"]\n\n", d.Format("2006.01.02"), wName, bName, res)
	var parts []string
	for i, h := range a.hist {
		if i%2 == 0 {
			parts = append(parts, fmt.Sprintf("%d.", i/2+1))
		}
		parts = append(parts, h.san)
	}
	parts = append(parts, res)
	line := ""
	for _, p := range parts {
		if len(line)+1+len(p) > 78 {
			b.WriteString(strings.TrimSpace(line) + "\n")
			line = ""
		}
		line += " " + p
	}
	b.WriteString(strings.TrimSpace(line) + "\n")
	return b.String()
}

func (a *App) rematch() {
	if a.online() {
		if a.net == nil || a.net.peer == nil {
			a.ui.message("Rematch", "Your friend has left, so there can't be a rematch.")
			return
		}
		a.askPeer("rematch")
		return
	}
	s := a.sd.Setup
	s.Mode, s.Opp = a.G.Mode, a.G.Opp
	s.Color = "white"
	if a.G.Mode == "ai" && a.G.YouWhite {
		s.Color = "black"
	}
	s.Clock = "none"
	if a.G.Clock != nil {
		s.Clock = a.G.Clock.Base
	}
	a.newGame(s)
}

func lastWord(s string) string {
	if i := strings.LastIndex(s, " "); i >= 0 {
		return s[i+1:]
	}
	return s
}
