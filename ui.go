// The interface: the library desk and board, the side panel, the toolbar and
// every screen (title, options, rules, credits, online, game over). Laid out
// on the same 1600x900 stage as Chess 1.x, with the same art and colours.
//
// (c) 2026 Ironite
package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/adamglassner/chess/engine"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

/* ------------------------------------------------------------ hit areas */

type hitT struct {
	id      string
	r       rect
	fn      func()
	enabled bool
}

var (
	hits      []hitT
	modalFrom = -1 // hits from this index belong to the open panel
	hoverID   string
	pressID   string
)

func addHit(id string, r rect, enabled bool, fn func()) {
	hits = append(hits, hitT{id, r, fn, enabled})
}

// hitAt finds the topmost active area under a stage point.
func hitAt(x, y float64) *hitT {
	from := 0
	if modalFrom >= 0 {
		from = modalFrom
	}
	for i := len(hits) - 1; i >= from; i-- {
		if hits[i].r.has(x, y) {
			return &hits[i]
		}
	}
	return nil
}

/* ------------------------------------------------------------ styles */

func straight(s string, a float64) color.RGBA { c := hexc(s); c.A = uint8(255 * a); return c }

var (
	stBarBtn = boxStyle{key: "barbtn", radius: 7, fill: []stop{{0, hexc("#f1dfb4")}, {1, hexc("#c69d5c")}}, border: straight("#6a4520", 1), borderW: 1,
		shadow: straight("#000000", .35), shadowBlur: 4, shadowY: 2, innerTop: straight("#ffffff", .6)}
	stBarBtnHot = boxStyle{key: "barbtnhot", radius: 7, fill: []stop{{0, hexc("#f8ead0")}, {1, hexc("#d6ad6a")}}, border: straight("#6a4520", 1), borderW: 1,
		shadow: straight("#000000", .4), shadowBlur: 6, shadowY: 2, innerTop: straight("#ffffff", .7)}
	stPlaque = boxStyle{key: "plaque", radius: 14, fill: []stop{{0, hexc("#3b2114")}, {1, hexc("#24130a")}}, border: straight("#e2bd6a", .35), borderW: 1,
		shadow: straight("#000000", .5), shadowBlur: 22, shadowY: 8, innerTop: straight("#ffdcaa", .12)}
	stPlaqueTurn = boxStyle{key: "plaqueturn", radius: 14, fill: []stop{{0, hexc("#3b2114")}, {1, hexc("#24130a")}}, border: straight("#f0cd78", .95), borderW: 1,
		shadow: straight("#000000", .5), shadowBlur: 22, shadowY: 8, innerTop: straight("#ffdcaa", .15),
		ring: straight("#e2bd6a", .35), ringW: 2, glow: straight("#f0be5a", .28), glowBlur: 26}
	stSheet = boxStyle{key: "sheet", radius: 12, fill: []stop{{0, hexc("#f6ead0")}, {1, hexc("#e8d5ab")}}, border: straight("#6e461e", .35), borderW: 1,
		shadow: straight("#000000", .55), shadowBlur: 26, shadowY: 10}
	stStatus = boxStyle{key: "status", radius: 10, fill: []stop{{0, straight("#140a04", .55)}}, border: straight("#e2bd6a", .25), borderW: 1}
	stPaper  = boxStyle{key: "paper", radius: 18, fill: []stop{{0, hexc("#f7ebd2")}, {1, hexc("#e8d3a8")}}, border: straight("#ffffff", .5), borderW: 1,
		shadow: straight("#000000", .6), shadowBlur: 40, shadowY: 16}
	stBig = boxStyle{key: "big", radius: 10, fill: []stop{{0, hexc("#f6e3b4")}, {1, hexc("#c99f5c")}}, border: straight("#6a4520", 1), borderW: 1,
		shadow: straight("#3a1e08", .45), shadowBlur: 10, shadowY: 4, innerTop: straight("#ffffff", .7)}
	stBigHot = boxStyle{key: "bighot", radius: 10, fill: []stop{{0, hexc("#fcefcc")}, {1, hexc("#d8b06a")}}, border: straight("#6a4520", 1), borderW: 1,
		shadow: straight("#3a1e08", .5), shadowBlur: 12, shadowY: 4, innerTop: straight("#ffffff", .8)}
	stGhost    = boxStyle{key: "ghost", radius: 10, fill: []stop{{0, straight("#ffffff", .25)}}, border: straight("#6e2316", .8), borderW: 1.5}
	stGhostHot = boxStyle{key: "ghosthot", radius: 10, fill: []stop{{0, straight("#ffffff", .5)}}, border: straight("#6e2316", 1), borderW: 1.5}
	stNav      = boxStyle{key: "nav", radius: 7, fill: []stop{{0, hexc("#f1dfb4")}, {1, hexc("#c69d5c")}}, border: straight("#6a4520", 1), borderW: 1, innerTop: straight("#ffffff", .5)}
	stClock    = boxStyle{key: "clock", radius: 10, fill: []stop{{0, hexc("#120904")}}, border: straight("#000000", .6), borderW: 1}
	stClockRun = boxStyle{key: "clockrun", radius: 10, fill: []stop{{0, hexc("#120904")}}, border: straight("#e2bd6a", .6), borderW: 2}
	stFolkOn   = boxStyle{key: "folkon", radius: 12, fill: []stop{{0, straight("#6e2316", .08)}}, border: straight("#6e2316", 1), borderW: 2}
	stFolkHot  = boxStyle{key: "folkhot", radius: 12, fill: []stop{{0, straight("#6e3c14", .08)}}}
	stField    = boxStyle{key: "field", radius: 8, fill: []stop{{0, hexc("#fffdf6")}}, border: straight("#6a4426", .6), borderW: 1.5}
	stFieldOn  = boxStyle{key: "fieldon", radius: 8, fill: []stop{{0, hexc("#fffdf6")}}, border: straight("#6e2316", 1), borderW: 2}
	stBubble   = boxStyle{key: "bubble", radius: 14, fill: []stop{{0, hexc("#fffaf0")}, {1, hexc("#f1e2c2")}}, border: straight("#6e461e", .45), borderW: 1,
		shadow: straight("#000000", .45), shadowBlur: 14, shadowY: 5}
	stPromo = boxStyle{key: "promo", radius: 10, fill: []stop{{0, hexc("#f6ead0")}, {1, hexc("#e3cfa4")}}, ring: straight("#c9a24c", 1), ringW: 3,
		shadow: straight("#000000", .7), shadowBlur: 30, shadowY: 10}
	stBar = boxStyle{key: "bar", radius: 0, fill: []stop{{0, hexc("#4a2616")}, {1, hexc("#22100a")}}, shadow: straight("#000000", .55), shadowBlur: 10, shadowY: 3}
)

var (
	tsBarBtn  = textStyle{f: fRegular, size: 16}
	tsPlaqueN = textStyle{f: fSemi, size: 26, spacing: .5}
	tsPlaqueR = textStyle{f: fItalic, size: 15}
	tsSheetH  = textStyle{f: fSemi, size: 15, spacing: 3, upper: true}
	tsMove    = textStyle{f: fRegular, size: 19}
	tsBody    = textStyle{f: fRegular, size: 17}
	tsBodyB   = textStyle{f: fSemi, size: 17}
	tsBodyI   = textStyle{f: fItalic, size: 17}
	tsSmall   = textStyle{f: fRegular, size: 14}
	tsKick    = textStyle{f: fRegular, size: 14, spacing: 4, upper: true}
	tsBig     = textStyle{f: fRegular, size: 21}
)

/* ------------------------------------------------------------ widgets */

// barButton is a toolbar button; returns its width.
func barButton(dst *ebiten.Image, id, label string, x float64, enabled bool, fn func()) float64 {
	w := measure(tsBarBtn, label) + 26
	r := rect{x, 11, w, 34}
	st := stBarBtn
	if enabled && hoverID == id {
		st = stBarBtnHot
	}
	a := 1.0
	if !enabled {
		a = .45
	}
	box(dst, st, r, a)
	text(dst, tsBarBtn, label, r.cx(), r.y+23, alignCenter, withA(cBtnInk, a))
	addHit(id, r, enabled, fn)
	return w
}

// bigButton is a large button on a panel (ghost = outlined).
func bigButton(dst *ebiten.Image, id, label string, cx, y float64, ghost bool, fn func()) rect {
	ts := tsBig
	w := measure(ts, label) + 64
	r := rect{cx - w/2, y, w, 52}
	st := stBig
	switch {
	case ghost && hoverID == id:
		st = stGhostHot
	case ghost:
		st = stGhost
	case hoverID == id:
		st = stBigHot
	}
	box(dst, st, r, 1)
	clr := cBtnInk
	if ghost {
		clr = cAccent
	}
	text(dst, ts, label, r.cx(), r.y+33, alignCenter, clr)
	addHit(id, r, true, fn)
	return r
}

// buttonRow centres big buttons; the first is solid, the rest outlined.
func buttonRow(dst *ebiten.Image, cx, y float64, labels []string, fns []func()) {
	ws := make([]float64, len(labels))
	total := -16.0
	for i, l := range labels {
		ws[i] = measure(tsBig, l) + 64
		total += ws[i] + 16
	}
	x := cx - total/2
	for i, l := range labels {
		bigButton(dst, "btn:"+l, l, x+ws[i]/2, y, i > 0, fns[i])
		x += ws[i] + 16
	}
}

// link draws underlined text that can be clicked; returns its width.
func linkText(dst *ebiten.Image, id, label string, x, y float64, al align, fn func()) float64 {
	w := measure(tsBody, label)
	switch al {
	case alignCenter:
		x -= w / 2
	case alignRight:
		x -= w
	}
	clr := cAccent
	if hoverID == id {
		clr = hexc("#a83a22")
	}
	text(dst, tsBody, label, x, y, alignLeft, clr)
	fillRect(dst, x, y+4, w, 1.2, clr)
	addHit(id, rect{x - 4, y - 18, w + 8, 26}, true, fn)
	return w
}

// chips is a segmented choice; returns its width.
func chips(dst *ebiten.Image, id string, x, y float64, opts [][2]string, cur string, set func(string)) float64 {
	ts := textStyle{f: fRegular, size: 17}
	ws := make([]float64, len(opts))
	total := 0.0
	for i, o := range opts {
		ws[i] = measure(ts, o[1]) + 30
		total += ws[i]
	}
	h := 40.0
	outer := boxStyle{key: "chips", radius: 8, fill: []stop{{0, straight("#ffffff", .3)}}, border: straight("#6e2316", .7), borderW: 1.2}
	box(dst, outer, rect{x, y, total, h}, 1)
	cx := x
	for i, o := range opts {
		r := rect{cx, y, ws[i], h}
		on := o[0] == cur
		if on {
			box(dst, boxStyle{key: "chipon", radius: 8, fill: []stop{{0, hexc("#7a2a1a")}, {1, hexc("#5e1d12")}}}, rect{r.x + 1, r.y + 1, r.w - 2, r.h - 2}, 1)
		} else if hoverID == id+":"+o[0] {
			fillRect(dst, r.x+1, r.y+1, r.w-2, r.h-2, withA(hexc("#6e3c14"), .08))
		}
		if i > 0 && !on && opts[i-1][0] != cur {
			fillRect(dst, r.x, r.y+1, 1, h-2, withA(cAccent, .35))
		}
		clr := cInk
		if on {
			clr = hexc("#fff4dc")
		}
		text(dst, ts, o[1], r.cx(), y+26, alignCenter, clr)
		v := o[0]
		addHit(id+":"+v, r, true, func() { play("click"); set(v) })
		cx += ws[i]
	}
	return total
}

// field is a one-line text box.
func field(dst *ebiten.Image, ui *uiState, id string, r rect, val *string, placeholder string) {
	on := ui.focus == id
	st := stField
	if on {
		st = stFieldOn
	}
	box(dst, st, r, 1)
	ty := r.y + r.h/2 + 6
	if *val == "" && !on {
		text(dst, tsBody, placeholder, r.x+12, ty, alignLeft, withA(cInk, .4))
	} else {
		w := text(dst, tsBody, *val, r.x+12, ty, alignLeft, cInk)
		if on && time.Now().UnixMilli()/530%2 == 0 {
			fillRect(dst, r.x+13+w, r.y+9, 1.5, r.h-18, cInk)
		}
	}
	fieldVals[id] = val
	addHit("field:"+id, r, true, func() { ui.focus = id })
}

var fieldVals = map[string]*string{}

// avatar draws a round portrait with a gold rim.
func avatar(dst *ebiten.Image, face string, cx, cy, r, rim float64) {
	img := art.people[face]
	if img == nil {
		img = art.people["you"]
	}
	fillCircle(dst, cx, cy, r+rim, cBorderGd)
	drawImage(dst, img, cx-r, cy-r, 2*r, 2*r, 1)
}

// pawnIcons draws n small pawns (difficulty).
func pawnIcons(dst *ebiten.Image, n int, x, cy, size float64) float64 {
	img := pieceImg(1, size)
	for i := 0; i < n; i++ {
		drawImageTinted(dst, img, x+float64(i)*size*.62, cy-size*.62, size, size, color.RGBA{0xc9, 0x93, 0x3a, 0xff})
	}
	return float64(n) * size * .62
}

// noteIcon is a small pair of quavers (the music button).
func noteIcon(dst *ebiten.Image, x, y float64, clr color.Color) {
	fillPath(dst, func(p *vector.Path) {
		p.MoveTo(float32(x+4), float32(y+1))
		p.LineTo(float32(x+13), float32(y-1))
		p.LineTo(float32(x+13), float32(y+11))
		p.LineTo(float32(x+11.6), float32(y+11))
		p.LineTo(float32(x+11.6), float32(y+2.4))
		p.LineTo(float32(x+5.4), float32(y+3.6))
		p.LineTo(float32(x+5.4), float32(y+13))
		p.LineTo(float32(x+4), float32(y+13))
		p.Close()
	}, clr, ebiten.FillRuleNonZero)
	for _, c := range [][2]float64{{x + 2.4, y + 13}, {x + 10, y + 11}} {
		vector.DrawFilledCircle(dst, float32(dx(c[0])), float32(dy(c[1])), float32(dv(2.6)), clr, true)
	}
}

/* ------------------------------------------------------------ the frame */

func (a *App) draw(dst *ebiten.Image) {
	hits = hits[:0]
	modalFrom = -1
	theme := a.sd.Settings.Board
	dst.Fill(color.RGBA{0x14, 0x0a, 0x05, 0xff})
	drawImage(dst, art.desk[theme], 0, 56, stageW, 844, 1)
	a.drawToolbar(dst)
	a.drawBoard(dst)
	a.drawSide(dst)
	a.drawExtras(dst)
	if k := a.ui.top(); k != ovNone {
		a.drawOverlay(dst, k)
	}
	a.drawConfetti(dst)
	st.frame++
	if st.frame%600 == 0 {
		textCache.sweep(300)
		boxCache.sweep(1200)
	}
}

func (a *App) drawToolbar(dst *ebiten.Image) {
	box(dst, stBar, rect{0, 0, stageW, 56}, 1)
	fillRect(dst, 0, 55, stageW, 1, withA(cGold, .25))
	tw := text(dst, textStyle{f: fSemi, size: 28, spacing: 3}, "Chess", 16, 38, alignLeft, cBarText)
	text(dst, textStyle{f: fRegular, size: 11, spacing: 2.2, upper: true}, "Ravenscroft Library", 16+tw+8, 37, alignLeft, withA(cBarText, .85))
	G := a.G
	ok := G != nil && G.Result == nil && a.live()
	pending := a.net != nil && a.net.pending != ""
	x := 310.0
	x += barButton(dst, "tb:new", "New Game", x, true, func() { play("click"); a.showTitle() }) + 10
	x += barButton(dst, "tb:undo", "Undo", x, G != nil && G.Result == nil && len(a.hist) > 0 && !pending, a.undo) + 10
	x += barButton(dst, "tb:hint", "Hint", x, ok && !a.online() && a.yourTurn() && !a.busy, a.hint) + 10
	x += barButton(dst, "tb:flip", "Flip Board", x, true, func() { play("click"); a.setFlip(!a.flipped); a.persist() }) + 10
	drawOK := ok && a.yourTurn() && len(a.hist) >= 2 && !a.busy && !pending && (!a.isAI() || len(a.hist)-a.drawOfferPly >= 10)
	x += barButton(dst, "tb:draw", "Offer Draw", x, drawOK, a.offerDraw) + 10
	x += barButton(dst, "tb:resign", "Resign", x, ok && len(a.hist) >= 1, a.resign) + 26

	// music: a note icon and the label
	s := &a.sd.Settings
	mw := measure(tsBarBtn, "Music") + 46
	mr := rect{x, 11, mw, 34}
	stb := stBarBtn
	if hoverID == "tb:music" {
		stb = stBarBtnHot
	}
	ma := 1.0
	if !s.Music {
		ma = .55
	}
	box(dst, stb, mr, ma)
	noteIcon(dst, x+12, 20, withA(cBtnInk, ma))
	text(dst, tsBarBtn, "Music", x+33, 34, alignLeft, withA(cBtnInk, ma))
	addHit("tb:music", mr, true, func() { s.Music = !s.Music; play("click"); writeSave(a.sd) })
	x += mw + 12
	// volume slider
	vr := rect{x, 18, 88, 20}
	volX = vr.x
	fillRect(dst, vr.x, vr.y+7, vr.w, 6, withA(hexc("#120904"), .9))
	fillRect(dst, vr.x, vr.y+7, vr.w*s.Vol/100, 6, cGold)
	kx := vr.x + vr.w*s.Vol/100
	fillCircle(dst, kx, vr.y+10, 8, hexc("#f1dfb4"))
	strokeCircle(dst, kx, vr.y+10, 8, 1, cBtnEdge)
	addHit("tb:vol", rect{vr.x - 8, vr.y - 4, vr.w + 16, 28}, true, func() { a.ui.dragVol = true })
	x += vr.w + 14
	sa := 1.0
	if !s.Sfx {
		sa = .55
	}
	sw := measure(tsBarBtn, "Sounds") + 26
	sr := rect{x, 11, sw, 34}
	stb = stBarBtn
	if hoverID == "tb:sfx" {
		stb = stBarBtnHot
	}
	box(dst, stb, sr, sa)
	text(dst, tsBarBtn, "Sounds", sr.cx(), 34, alignCenter, withA(cBtnInk, sa))
	addHit("tb:sfx", sr, true, func() { s.Sfx = !s.Sfx; play("click"); writeSave(a.sd) })
	x += sw + 10
	x += barButton(dst, "tb:opts", "Options", x, true, func() { play("click"); a.ui.show(ovSettings) }) + 10
	barButton(dst, "tb:credits", "Credits", x, true, func() { play("click"); a.ui.show(ovCredits) })

	// record against the computer
	ts := textStyle{f: fRegular, size: 16}
	tb := textStyle{f: fSemi, size: 16}
	rx := stageW - 16.0
	for _, kv := range [][2]string{{"Lost", fmt.Sprint(a.sd.Stats.Lost)}, {"Drawn", fmt.Sprint(a.sd.Stats.Drawn)}, {"Won", fmt.Sprint(a.sd.Stats.Won)}} {
		rx -= text(dst, tb, kv[1], rx, 34, alignRight, cCream) + 5
		rx -= text(dst, ts, kv[0], rx, 34, alignRight, cBarText) + 18
	}
}

/* ------------------------------------------------------------ the board */

var checkGlow *ebiten.Image

func (a *App) drawBoard(dst *ebiten.Image) {
	theme := a.sd.Settings.Board
	p := a.curPos()
	bx, by := boardX, boardY
	inner := 8 * sqSize
	// the squares (rotated when the board is flipped)
	surf := art.surface[theme]
	b := surf.Bounds()
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	sc := dv(inner) / float64(b.Dx())
	if a.flipped {
		op.GeoM.Translate(-float64(b.Dx()), -float64(b.Dy()))
		op.GeoM.Rotate(math.Pi)
	}
	op.GeoM.Scale(sc, sc)
	op.GeoM.Translate(dx(bx+frameM), dy(by+frameM))
	dst.DrawImage(surf, op)

	// coordinates on the frame
	coord := hexc(boardCoord[theme])
	ts := textStyle{f: fSemi, size: 15}
	for i := 0; i < 8; i++ {
		file := string("abcdefgh"[i])
		rank := fmt.Sprint(8 - i)
		if a.flipped {
			file, rank = string("abcdefgh"[7-i]), fmt.Sprint(i+1)
		}
		cx := bx + frameM + sqSize*(float64(i)+.5)
		for _, pt := range [][3]float64{{cx, by + 800 - frameM*.42, 0}, {cx, by + frameM*.44, 0}} {
			textMid(dst, ts, file, pt[0]+.6, pt[1]+.8, alignCenter, withA(hexc("#000000"), .55))
			textMid(dst, ts, file, pt[0], pt[1], alignCenter, coord)
		}
		cy := by + frameM + sqSize*(float64(i)+.5)
		for _, xx := range []float64{bx + frameM*.44, bx + 800 - frameM*.42} {
			textMid(dst, ts, rank, xx+.6, cy+.8, alignCenter, withA(hexc("#000000"), .55))
			textMid(dst, ts, rank, xx, cy, alignCenter, coord)
		}
	}

	sq := func(s int) (float64, float64) { x, y := a.sqXY(s); return bx + x, by + y }
	// last move
	var last *histEntry
	if a.live() && len(a.hist) > 0 {
		last = &a.hist[len(a.hist)-1]
	} else if !a.live() && a.view > 0 {
		last = &a.hist[a.view-1]
	}
	if last != nil {
		for _, s := range []int{last.m.From, last.m.To} {
			x, y := sq(s)
			fillRect(dst, x, y, sqSize, sqSize, withA(color.RGBA{255, 196, 64, 255}, .30))
		}
	}
	// check
	if p.InCheck() {
		k := p.Kings[0]
		if p.Side == engine.Black {
			k = p.Kings[1]
		}
		if checkGlow == nil {
			checkGlow = radialImage(128, 128, .5, .5, .69, func(t float64) [4]float64 {
				switch {
				case t < .55:
					return [4]float64{255, 40 + (230-255)*t/.55*0, 20, .85 - .4*t/.55}
				case t < 1:
					return [4]float64{230, 30, 10, .45 * (1 - (t-.55)/.45)}
				}
				return [4]float64{0, 0, 0, 0}
			})
		}
		x, y := sq(k)
		drawImage(dst, checkGlow, x, y, sqSize, sqSize, 1)
	}
	// selection and marks
	if a.sel >= 0 && a.live() {
		x, y := sq(a.sel)
		fillRect(dst, x, y, sqSize, sqSize, withA(color.RGBA{255, 214, 90, 255}, .48))
	}
	for s := range a.marks {
		x, y := sq(s)
		strokeCircle(dst, x+sqSize/2, y+sqSize/2, sqSize*.43, 5, withA(color.RGBA{40, 140, 70, 255}, .85))
	}
	// pieces
	for s := 0; s < 128; s++ {
		if s&0x88 != 0 {
			s += 7
			continue
		}
		pc := p.Board[s]
		if pc == 0 || (a.live() && a.hidden(s)) || (a.drag != nil && a.drag.moved && a.drag.from == s) {
			continue
		}
		x, y := sq(s)
		drawImage(dst, pieceImg(pc, sqSize), x, y, sqSize, sqSize, 1)
	}
	// legal moves
	if a.sel >= 0 && a.live() && a.sd.Settings.Dots == "on" {
		for _, m := range a.selMoves {
			x, y := sq(m.To)
			if m.Captured != 0 {
				strokeCircle(dst, x+sqSize/2, y+sqSize/2, sqSize*.415, sqSize*.09, withA(color.RGBA{30, 15, 5, 255}, .38))
			} else {
				fillCircle(dst, x+sqSize/2, y+sqSize/2, sqSize*.15, withA(color.RGBA{30, 15, 5, 255}, .34))
			}
		}
	}
	// moving pieces
	now := time.Now()
	for _, an := range a.anims {
		t := math.Min(1, float64(now.Sub(an.t0))/float64(an.dur))
		img := pieceImg(an.piece, sqSize*1.1)
		if an.fade {
			drawImage(dst, img, bx+an.x0, by+an.y0, sqSize, sqSize, 1-t)
			continue
		}
		e := 2 * t * t
		if t >= .5 {
			e = 1 - math.Pow(-2*t+2, 2)/2
		}
		lift := math.Sin(t*math.Pi) * sqSize * .06
		x := an.x0 + (an.x1-an.x0)*e - lift/2
		y := an.y0 + (an.y1-an.y0)*e - lift
		drawImage(dst, img, bx+x, by+y, sqSize+lift, sqSize+lift, 1)
	}
	// arrows
	for _, ar := range a.arrows {
		a.drawArrow(dst, ar[0], ar[1], withA(color.RGBA{40, 140, 70, 255}, .8))
	}
	if a.hintArrow != nil {
		a.drawArrow(dst, a.hintArrow[0], a.hintArrow[1], withA(color.RGBA{226, 170, 40, 255}, .88))
	}
	// the piece being dragged, lifted with a shadow
	if a.drag != nil && a.drag.moved {
		pc := a.pos.Board[a.drag.from]
		if pc != 0 {
			sz := sqSize * 1.12
			x, y := bx+a.drag.x-sz/2, by+a.drag.y-sz*.62
			drawImageTinted(dst, art.shadow[pc], x+2, y+9, sz, sz, color.RGBA{0, 0, 0, 115})
			drawImage(dst, pieceImg(pc, sz), x, y, sz, sz, 1)
		}
	}
	// board clicks (when no panel is open)
	addHit("board", rect{bx, by, 800, 800}, true, nil)
	if a.promo != nil {
		a.drawPromo(dst)
	}
}

func (a *App) drawArrow(dst *ebiten.Image, from, to int, clr color.Color) {
	ax, ay := a.sqXY(from)
	cx, cy := a.sqXY(to)
	x0, y0 := boardX+ax+sqSize/2, boardY+ay+sqSize/2
	x1, y1 := boardX+cx+sqSize/2, boardY+cy+sqSize/2
	ang := math.Atan2(y1-y0, x1-x0)
	l := math.Hypot(x1-x0, y1-y0) - sqSize*.32
	S := sqSize
	pts := [][2]float64{{S * .18, -S * .085}, {l - S*.26, -S * .085}, {l - S*.26, -S * .2}, {l, 0}, {l - S*.26, S * .2}, {l - S*.26, S * .085}, {S * .18, S * .085}}
	ca, sa := math.Cos(ang), math.Sin(ang)
	fillPath(dst, func(p *vector.Path) {
		for i, pt := range pts {
			x, y := x0+pt[0]*ca-pt[1]*sa, y0+pt[0]*sa+pt[1]*ca
			if i == 0 {
				p.MoveTo(float32(x), float32(y))
			} else {
				p.LineTo(float32(x), float32(y))
			}
		}
		p.Close()
	}, clr, ebiten.FillRuleNonZero)
}

func (a *App) promoRect() rect {
	x, y := a.sqXY(a.promo[0].To)
	top := y < 400
	if top {
		return rect{boardX + x, boardY + y, 90, 360}
	}
	return rect{boardX + x, boardY + y + sqSize - 360, 90, 360}
}

func (a *App) drawPromo(dst *ebiten.Image) {
	r := a.promoRect()
	modalFrom = len(hits)
	addHit("promo:outside", rect{0, 0, stageW, stageH}, true, func() { a.promo = nil; a.sel, a.selMoves = -1, nil })
	box(dst, stPromo, r, 1)
	side := int8(1)
	if a.promo[0].Piece < 0 {
		side = -1
	}
	order := []int{engine.Queen, engine.Knight, engine.Rook, engine.Bishop}
	_, y := a.sqXY(a.promo[0].To)
	for i, t := range order {
		yy := r.y + float64(i)*90
		if y >= 400 {
			yy = r.y + r.h - float64(i+1)*90
		}
		cell := rect{r.x, yy, 90, 90}
		id := fmt.Sprint("promo:", t)
		if hoverID == id {
			fillRect(dst, cell.x, cell.y, cell.w, cell.h, withA(cBorderGd, .35))
		}
		drawImage(dst, pieceImg(side*int8(t), 90), cell.x, cell.y, 90, 90, 1)
		tt := t
		addHit(id, cell, true, func() {
			for _, m := range a.promo {
				if absI8(m.Promo) == tt {
					a.promo = nil
					a.playMove(m)
					return
				}
			}
		})
	}
}

/* ------------------------------------------------------------ side panel */

const (
	sideX = 872.0
	sideW = 700.0
)

func (a *App) plaqueRect(side int) rect {
	topSide := engine.Black
	if a.flipped {
		topSide = engine.White
	}
	if side == topSide {
		return rect{sideX, boardY, sideW, 128}
	}
	return rect{sideX, boardY + 800 - 128, sideW, 128}
}

type plaqueInfo struct{ face, name, rt string }

func (a *App) plaqueFor(side int) plaqueInfo {
	G := a.G
	if G == nil {
		return plaqueInfo{"you", "You", ""}
	}
	switch G.Mode {
	case "pvp":
		face := "you"
		if side == engine.Black {
			face = "p2"
		}
		return plaqueInfo{face, a.sideLabel(side), sideName(side) + " · Sharing this computer"}
	case "online":
		if side == a.youSide() {
			return plaqueInfo{"you", a.sideLabel(side), sideName(side) + " · You"}
		}
		return plaqueInfo{"p2", a.sideLabel(side), sideName(side) + " · Playing online"}
	}
	if side == a.youSide() {
		r := a.sd.Stats.Per[G.Opp]
		if r == nil {
			r = &Record{}
		}
		return plaqueInfo{"you", "You", fmt.Sprintf("Playing %s · vs %s players: %d–%d–%d", sideName(side), a.opp().Rating, r.W, r.D, r.L)}
	}
	return plaqueInfo{G.Opp, a.oppFull(), a.opp().Rating + " · " + sideName(side)}
}

func (a *App) drawSide(dst *ebiten.Image) {
	if a.G == nil {
		return
	}
	for _, side := range []int{engine.White, engine.Black} {
		a.drawPlaque(dst, side)
	}
	a.drawSheet(dst)
	// status line
	sr := rect{sideX, boardY + 128 + 14 + 456 + 14, sideW, 50}
	box(dst, stStatus, sr, 1)
	msg, clr := a.statusText()
	textMid(dst, tsBig, msg, sr.cx(), sr.cy(), alignCenter, clr)
}

func (a *App) statusText() (string, color.Color) {
	G := a.G
	if !a.live() {
		n := (a.view + 1) / 2
		return fmt.Sprintf("Reviewing move %d · press End to return", n), cReview
	}
	if a.ui.statusOverride != "" {
		return a.ui.statusOverride, cAlert
	}
	if G.Result != nil {
		return a.resultLine(), cAlert
	}
	chk := a.pos.InCheck()
	switch G.Mode {
	case "pvp":
		s := a.sideLabel(a.pos.Side) + " to move"
		if chk {
			return s + " · Check!", cAlert
		}
		return s, cCream
	case "online":
		if a.net == nil {
			return "Not connected", cAlert
		}
		if a.yourTurn() {
			if chk {
				return "Check! Your king is under attack.", cAlert
			}
			return "Your move", cCream
		}
		return "Waiting for " + a.sideLabel(a.pos.Side) + "…", cCream
	}
	if a.yourTurn() {
		if chk {
			return "Check! Your king is under attack.", cAlert
		}
		return "Your move", cCream
	}
	return a.oppShort() + " is thinking…", cCream
}

func (a *App) drawPlaque(dst *ebiten.Image, side int) {
	r := a.plaqueRect(side)
	turn := a.G.Result == nil && a.pos.Side == side
	st := stPlaque
	if turn {
		st = stPlaqueTurn
	}
	box(dst, st, r, 1)
	info := a.plaqueFor(side)
	avatar(dst, info.face, r.x+12+52, r.cy(), 52, 3)
	x := r.x + 12 + 112 + 16
	text(dst, tsPlaqueN, info.name, x, r.y+40, alignLeft, cCream)
	text(dst, tsPlaqueR, info.rt, x, r.y+64, alignLeft, withA(cBarText, .78))
	if a.busy && a.job != nil && a.job.kind == "move" && a.isAI() && side == a.aiSide() {
		text(dst, textStyle{f: fRegular, size: 15}, "Thinking…", x, r.y+86, alignLeft, cGold)
	}
	a.drawCaps(dst, side, x, r.y+104)
	// clock
	if a.G.Clock != nil {
		ms := a.G.Clock.W
		if side == engine.Black {
			ms = a.G.Clock.B
		}
		run := turn && a.live() && len(a.hist) > 0
		cr := rect{r.x + r.w - 18 - 160, r.cy() - 30, 160, 60}
		cs := stClock
		if run {
			cs = stClockRun
		}
		box(dst, cs, cr, 1)
		clr := hexc("#f3e2c0")
		if run {
			clr = hexc("#ffe39a")
		}
		if ms < 20000 {
			clr = hexc("#ff9c86")
		}
		textMid(dst, textStyle{f: fSemi, size: 40}, fmtClock(ms), cr.cx(), cr.cy(), alignCenter, clr)
	}
}

func fmtClock(ms float64) string {
	ms = math.Max(0, ms)
	if ms < 10000 {
		return fmt.Sprintf("%.1f", math.Floor(ms/100)/10)
	}
	s := int(math.Ceil(ms / 1000))
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

// drawCaps shows the pieces this side has taken and its material lead.
func (a *App) drawCaps(dst *ebiten.Image, side int, x, cy float64) {
	p := a.curPos()
	var count [2][7]int
	for s := 0; s < 128; s++ {
		if s&0x88 != 0 {
			s += 7
			continue
		}
		pc := p.Board[s]
		if pc > 0 {
			count[0][pc]++
		} else if pc < 0 {
			count[1][-pc]++
		}
	}
	full := [7]int{0, 8, 2, 2, 2, 1, 0}
	val := [7]int{0, 1, 3, 3, 5, 9, 0}
	me, them := 0, 1
	if side == engine.Black {
		me, them = 1, 0
	}
	mat := [2]int{}
	for t := 1; t <= 5; t++ {
		mat[0] += count[0][t] * val[t]
		mat[1] += count[1][t] * val[t]
	}
	first := true
	for _, t := range []int{5, 4, 3, 2, 1} {
		n := full[t] - count[them][t]
		for i := 0; i < n; i++ {
			if i == 0 && !first {
				x += 10
			}
			drawImage(dst, pieceImg(int8(-side*t), 30), x, cy-17, 30, 30, 1)
			x += 21
			first = false
		}
	}
	if adv := mat[me] - mat[them]; adv > 0 {
		text(dst, textStyle{f: fSemi, size: 16}, fmt.Sprint("+", adv), x+18, cy+5, alignLeft, cGold)
	}
}

// the score sheet
const rowH = 31.0

func (a *App) sheetRect() rect { return rect{sideX, boardY + 128 + 14, sideW, 456} }

func (a *App) drawSheet(dst *ebiten.Image) {
	r := a.sheetRect()
	box(dst, stSheet, r, 1)
	text(dst, tsSheetH, "Score Sheet", r.x+20, r.y+30, alignLeft, cAccent)
	if a.opening != "" {
		text(dst, textStyle{f: fItalic, size: 15}, a.opening, r.x+r.w-20, r.y+30, alignRight, withA(cInk, .8))
	}
	fillRect(dst, r.x, r.y+44, r.w, 1, withA(hexc("#6e461e"), .25))
	list := rect{r.x, r.y + 45, r.w, r.h - 45 - 56}
	fillRect(dst, r.x, r.y+r.h-56, r.w, 1, withA(hexc("#6e461e"), .25))

	rows := (len(a.hist) + 1) / 2
	extra := 0.0
	if a.G.Result != nil {
		extra = rowH + 8
	}
	content := float64(rows)*rowH + 16 + extra
	maxScroll := math.Max(0, content-list.h)
	shown := len(a.hist)
	if !a.live() {
		shown = a.view
	}
	if a.ui.scrollToCurrent {
		a.ui.scrollToCurrent = false
		if a.live() && a.G.Result != nil {
			a.ui.sheetScroll = maxScroll
		} else if shown > 0 {
			ry := float64((shown-1)/2)*rowH + 8
			if ry < a.ui.sheetScroll {
				a.ui.sheetScroll = ry - 8
			}
			if ry+rowH > a.ui.sheetScroll+list.h {
				a.ui.sheetScroll = ry + rowH - list.h + 8
			}
		} else {
			a.ui.sheetScroll = 0
		}
	}
	a.ui.sheetScroll = math.Max(0, math.Min(maxScroll, a.ui.sheetScroll))

	clip := dst.SubImage(image.Rect(int(dx(list.x)), int(dy(list.y)), int(math.Ceil(dx(list.x+list.w))), int(math.Ceil(dy(list.y+list.h))))).(*ebiten.Image)
	y0 := list.y + 8 - a.ui.sheetScroll
	colX := [3]float64{r.x + 20, r.x + 90, r.x + 260}
	for i := 0; i < rows; i++ {
		y := y0 + float64(i)*rowH
		if y+rowH < list.y || y > list.y+list.h {
			continue
		}
		text(clip, tsMove, fmt.Sprintf("%d.", i+1), colX[0]+50, y+22, alignRight, withA(cInk, .55))
		for j := 0; j < 2; j++ {
			k := i*2 + j
			if k >= len(a.hist) {
				break
			}
			cell := rect{colX[j+1], y + 1, 150, rowH - 2}
			cur := k == shown-1
			id := fmt.Sprint("mv:", k)
			if cur {
				box(clip, boxStyle{key: "mvcur", radius: 6, fill: []stop{{0, cAccent}}}, cell, 1)
			} else if hoverID == id {
				box(clip, boxStyle{key: "mvhot", radius: 6, fill: []stop{{0, straight("#6e3c14", .1)}}}, cell, 1)
			}
			clr := cInk
			if cur {
				clr = hexc("#fff4dc")
			}
			san := a.hist[k].san
			tx := cell.x + 10
			if strings.ContainsAny(san[:1], "KQRBN") { // figurine
				t := strings.Index(" PNBRQK", san[:1])
				code := int8(t)
				if k%2 == 1 {
					code = -code
				}
				drawImage(clip, pieceImg(code, 27), tx-3, y+1, 27, 27, 1)
				tx += 22
				san = san[1:]
			}
			text(clip, tsMove, san, tx, y+22, alignLeft, clr)
			if cell.y+cell.h > list.y && cell.y < list.y+list.h {
				kk := k
				vis := cell
				vis.y = math.Max(cell.y, list.y)
				vis.h = math.Min(cell.y+cell.h, list.y+list.h) - vis.y
				addHit(id, vis, true, func() { a.setView(kk + 1) })
			}
		}
	}
	if a.G.Result != nil {
		y := y0 + float64(rows)*rowH + 4
		text(clip, textStyle{f: fSemi, size: 17}, a.G.Result.Result+" · "+a.resultLine(), r.cx(), y+22, alignCenter, cAccent)
	}
	addHit("sheet", list, true, nil)

	// navigation
	n := len(a.hist)
	v := n
	if !a.live() {
		v = a.view
	}
	navs := []struct {
		id   string
		ok   bool
		to   int
		icon int
	}{{"nav:start", v > 0, 0, 0}, {"nav:prev", v > 0, v - 1, 1}, {"nav:next", v < n, v + 1, 2}, {"nav:end", v < n, n, 3}}
	bx := r.cx() - (4*64+3*8)/2
	for i, nb := range navs {
		br := rect{bx + float64(i)*72, r.y + r.h - 46, 64, 36}
		al := 1.0
		if !nb.ok {
			al = .4
		}
		box(dst, stNav, br, al)
		navIcon(dst, nb.icon, br.cx(), br.cy(), withA(cBtnInk, al))
		to := nb.to
		addHit(nb.id, br, nb.ok, func() { a.setView(to) })
	}
}

// navIcon: 0 first, 1 back, 2 forward, 3 last.
func navIcon(dst *ebiten.Image, kind int, cx, cy float64, clr color.Color) {
	tri := func(x float64, left bool) {
		fillPath(dst, func(p *vector.Path) {
			if left {
				p.MoveTo(float32(x), float32(cy))
				p.LineTo(float32(x+9), float32(cy-7))
				p.LineTo(float32(x+9), float32(cy+7))
			} else {
				p.MoveTo(float32(x+9), float32(cy))
				p.LineTo(float32(x), float32(cy-7))
				p.LineTo(float32(x), float32(cy+7))
			}
			p.Close()
		}, clr, ebiten.FillRuleNonZero)
	}
	switch kind {
	case 0:
		fillRect(dst, cx-9, cy-7, 2.2, 14, clr)
		tri(cx-6, true)
		tri(cx+1, true)
	case 1:
		tri(cx-5, true)
	case 2:
		tri(cx-4, false)
	case 3:
		tri(cx-10, false)
		tri(cx-3, false)
		fillRect(dst, cx+7, cy-7, 2.2, 14, clr)
	}
}

/* ------------------------------------------------------------ extras */

func (a *App) drawExtras(dst *ebiten.Image) {
	text(dst, textStyle{f: fRegular, size: 12, spacing: 1.5, upper: true}, "v"+version+" · © 2026 Ironite", stageW-16, stageH-7, alignRight, withA(cVer, .55))
	now := time.Now()
	// now playing
	if a.ui.toast != "" {
		t := now.Sub(a.ui.toastAt).Seconds()
		al := math.Min(1, t/.4) * math.Min(1, math.Max(0, (5-t)/.6))
		if al > 0 {
			ts := textStyle{f: fItalic, size: 15}
			label := strings.TrimPrefix(a.ui.toast, "♫ ")
			w := measure(ts, label) + 22
			x := boardX + 400 - w/2
			noteIcon(dst, x, stageH-21, withA(cVer, al))
			text(dst, ts, label, x+22, stageH-8, alignLeft, withA(cVer, al))
		}
	}
	// banner
	if a.ui.bannerT != "" {
		t := now.Sub(a.ui.bannerAt).Seconds()
		al := math.Min(1, t/.25) * math.Min(1, math.Max(0, (1.8-t)/.4))
		if al > 0 {
			r := rect{boardX + 400 - 260, boardY + 340, 520, 120}
			box(dst, boxStyle{key: "banner", radius: 16, fill: []stop{{0, straight("#1e0f07", .92)}}, border: straight("#e2bd6a", .6), borderW: 1.5,
				shadow: straight("#000000", .5), shadowBlur: 24, shadowY: 8}, r, al)
			textMid(dst, textStyle{f: fSemi, size: 34, spacing: 2}, a.ui.bannerT, r.cx(), r.y+48, alignCenter, withA(cCream, al))
			if a.ui.bannerSub != "" {
				textMid(dst, textStyle{f: fItalic, size: 17}, a.ui.bannerSub, r.cx(), r.y+88, alignCenter, withA(cBarText, al))
			}
		}
	}
	// speech bubble by the computer's plaque
	for _, b := range a.ui.bubbles {
		if a.G == nil {
			break
		}
		t := now.Sub(b.at).Seconds()
		al := math.Min(1, t/.25) * math.Min(1, math.Max(0, (3.4-t)/.4))
		if al <= 0 {
			continue
		}
		pr := a.plaqueRect(b.side)
		ts := textStyle{f: fItalic, size: 18}
		lines := wrap(ts, b.text, 400)
		w := 0.0
		for _, l := range lines {
			w = math.Max(w, measure(ts, l))
		}
		w += 40
		h := float64(len(lines))*24 + 24
		down := pr.y < 400
		x := pr.x + 150
		y := pr.y + pr.h + 14
		if !down {
			y = pr.y - 14 - h
		}
		br := rect{x, y, w, h}
		box(dst, stBubble, br, al)
		// tail
		tx := x + 40
		fillPath(dst, func(p *vector.Path) {
			if down {
				p.MoveTo(float32(tx), float32(y+1))
				p.LineTo(float32(tx+10), float32(y-12))
				p.LineTo(float32(tx+22), float32(y+1))
			} else {
				p.MoveTo(float32(tx), float32(y+h-1))
				p.LineTo(float32(tx+10), float32(y+h+12))
				p.LineTo(float32(tx+22), float32(y+h-1))
			}
			p.Close()
		}, withA(hexc("#f8ecd4"), al), ebiten.FillRuleNonZero)
		for i, l := range lines {
			text(dst, ts, l, br.cx(), y+31+float64(i)*24, alignCenter, withA(cInk, al))
		}
	}
}

func (a *App) drawConfetti(dst *ebiten.Image) {
	now := time.Now()
	for _, p := range a.ui.particles {
		t := now.Sub(p.born).Seconds()
		al := math.Min(1, (p.life-t)/.5)
		if al <= 0 {
			continue
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-.5, -.5)
		op.GeoM.Scale(dv(p.w), dv(p.h)*math.Abs(math.Cos(p.rot*1.7)))
		op.GeoM.Rotate(p.rot)
		op.GeoM.Translate(dx(p.x), dy(p.y))
		op.ColorScale.ScaleWithColor(withA(color.RGBA{p.clr[0], p.clr[1], p.clr[2], 255}, al))
		dst.DrawImage(whiteSub, op)
	}
}

/* ------------------------------------------------------------ panels */

func drawVeil(dst *ebiten.Image) {
	fillRect(dst, -10, -10, stageW+20, stageH+20, withA(color.RGBA{10, 5, 2, 255}, .62))
}

// panel opens a paper box w x h in the middle of the stage.
func panel(dst *ebiten.Image, w, h float64, closable func()) rect {
	modalFrom = len(hits)
	drawVeil(dst)
	r := rect{(stageW - w) / 2, (stageH - h) / 2, w, h}
	addHit("veil", rect{-10, -10, stageW + 20, stageH + 20}, closable != nil, closable)
	box(dst, stPaper, r, 1)
	addHit("paper", r, true, nil)
	return r
}

func heading(dst *ebiten.Image, s string, cx, y, size float64) {
	text(dst, textStyle{f: fSemi, size: size, spacing: 1}, s, cx, y, alignCenter, cAccent)
}

func paragraph(dst *ebiten.Image, ts textStyle, s string, cx, y, w float64, clr color.Color) float64 {
	lh := ts.size * 1.45
	for _, l := range wrap(ts, s, w) {
		text(dst, ts, l, cx, y, alignCenter, clr)
		y += lh
	}
	return y
}

func (a *App) drawOverlay(dst *ebiten.Image, k ovKind) {
	switch k {
	case ovTitle:
		a.drawTitle(dst)
	case ovOver:
		a.drawOver(dst)
	case ovConfirm:
		r := panel(dst, 600, 270, nil)
		heading(dst, a.ui.askT.title, r.cx(), r.y+64, 34)
		paragraph(dst, tsBody, a.ui.askT.text, r.cx(), r.y+110, 500, cInk)
		yes, no := a.ui.askT.onYes, a.ui.askT.onNo
		buttonRow(dst, r.cx(), r.y+r.h-82, []string{a.ui.askT.yes, a.ui.askT.no}, []func(){
			func() {
				a.ui.hide(ovConfirm)
				play("click")
				if yes != nil {
					yes()
				}
			},
			func() {
				a.ui.hide(ovConfirm)
				play("click")
				if no != nil {
					no()
				}
			},
		})
	case ovMsg:
		r := panel(dst, 600, 260, func() { a.ui.hide(ovMsg) })
		heading(dst, a.ui.msgT.title, r.cx(), r.y+64, 34)
		paragraph(dst, tsBody, a.ui.msgT.text, r.cx(), r.y+110, 500, cInk)
		bigButton(dst, "msg:ok", "OK", r.cx(), r.y+r.h-82, false, func() { a.ui.hide(ovMsg); play("click") })
	case ovSettings:
		a.drawSettings(dst)
	case ovRules:
		a.drawRules(dst)
	case ovCredits:
		a.drawCredits(dst)
	case ovOnline:
		a.drawOnline(dst)
	}
}

/* ------------------------------------------------------------ title */

func (a *App) showTitle() {
	a.ui.show(ovTitle)
	a.ui.titleMessage = ""
}

func (a *App) drawTitle(dst *ebiten.Image) {
	r := panel(dst, 1060, 840, nil)
	cx := r.cx()
	text(dst, tsKick, "An evening at the", cx+2, r.y+46, alignCenter, withA(cInk, .7))
	text(dst, textStyle{f: fSemi, size: 84, spacing: 6}, "Chess", cx+3, r.y+136, alignCenter, cAccent)
	text(dst, textStyle{f: fRegular, size: 17, spacing: 5, upper: true}, "Ravenscroft Library", cx+2, r.y+176, alignCenter, cInk)
	su := &a.sd.Setup
	modes := [][2]string{{"ai", "Play the Computer"}, {"pvp", "Same Computer"}, {"online", "Online"}}
	mw := 0.0
	for _, m := range modes {
		mw += measure(textStyle{f: fRegular, size: 17}, m[1]) + 30
	}
	chips(dst, "mode", cx-mw/2, r.y+200, modes, su.Mode, func(v string) { su.Mode = v; writeSave(a.sd) })

	y := r.y + 262
	switch su.Mode {
	case "ai":
		y = a.drawFolks(dst, r, y)
	case "pvp":
		y = a.drawPvpSetup(dst, r, y)
	case "online":
		y = a.drawOnlineSetup(dst, r, y)
	}

	// colour and clock
	optRow := func(label string, yy float64, id string, opts [][2]string, cur string, set func(string)) {
		ts := textStyle{f: fRegular, size: 17}
		w := 0.0
		for _, o := range opts {
			w += measure(ts, o[1]) + 30
		}
		lw := measure(tsBody, label)
		x0 := cx - (lw+16+w)/2
		text(dst, tsBody, label, x0, yy+26, alignLeft, cInk)
		chips(dst, id, x0+lw+16, yy, opts, cur, set)
	}
	if su.Mode != "pvp" {
		lbl := "Play as"
		if su.Mode == "online" {
			lbl = "Host plays"
		}
		optRow(lbl, y, "color", [][2]string{{"white", "White"}, {"random", "Random"}, {"black", "Black"}}, su.Color, func(v string) { su.Color = v; writeSave(a.sd) })
		y += 50
	}
	clockOpts := [][2]string{{"none", "No clock"}, {"5+3", "5 + 3"}, {"10", "10 min"}, {"15+10", "15 + 10"}, {"30", "30 min"}}
	optRow("Clock", y, "clock", clockOpts, su.Clock, func(v string) { su.Clock = v; writeSave(a.sd) })
	y += 66

	switch su.Mode {
	case "online":
		buttonRow(dst, cx, y, []string{"Host a Game", "Join a Game"}, []func(){
			func() { play("click"); a.hostClicked() },
			func() { play("click"); a.joinClicked() },
		})
	default:
		can := a.sd.Game != nil && a.sd.Game.Result == nil && len(a.sd.Game.Moves) > 0 && a.sd.Game.Mode != "online"
		if can {
			buttonRow(dst, cx, y, []string{"Start Game", "Continue Game"}, []func(){
				func() { a.newGame(*su) },
				func() { play("click"); a.resume() },
			})
		} else {
			bigButton(dst, "start", "Start Game", cx, y, false, func() { a.newGame(*su) })
		}
	}
	y += 84
	// links
	parts := []struct {
		l  string
		fn func()
	}{{"How to Play", func() { play("click"); a.ui.show(ovRules) }}, {"Options", func() { play("click"); a.ui.show(ovSettings) }}, {"Credits & Changelog", func() { play("click"); a.ui.show(ovCredits) }}}
	total := 0.0
	for i, p := range parts {
		total += measure(tsBody, p.l)
		if i > 0 {
			total += 34
		}
	}
	lx := cx - total/2
	for i, p := range parts {
		if i > 0 {
			text(dst, tsBody, "·", lx+17, y, alignCenter, withA(cInk, .6))
			lx += 34
		}
		lx += linkText(dst, "lnk:"+p.l, p.l, lx, y, alignLeft, p.fn)
	}
	// an Ironite game
	ly := r.y + r.h - 34
	lb := art.logo.Bounds()
	lh := 22.0
	lw := lh * float64(lb.Dx()) / float64(lb.Dy())
	t1 := textStyle{f: fRegular, size: 13, spacing: 3, upper: true}
	w1, w2 := measure(t1, "An"), measure(t1, "game")
	sx := cx - (w1+12+lw+12+w2)/2
	text(dst, t1, "An", sx, ly+5, alignLeft, withA(cInk, .7))
	box(dst, boxStyle{key: "logopill", radius: 6, fill: []stop{{0, hexc("#1f4d3d")}, {1, hexc("#163a2e")}}}, rect{sx + w1 + 4, ly - lh/2 - 4, lw + 16, lh + 8}, 1)
	drawImage(dst, art.logo, sx+w1+12, ly-lh/2, lw, lh, 1)
	text(dst, t1, "game", sx+w1+12+lw+12, ly+5, alignLeft, withA(cInk, .7))
}

func (a *App) drawFolks(dst *ebiten.Image, r rect, y float64) float64 {
	su := &a.sd.Setup
	cw := 150.0
	gap := 12.0
	x0 := r.cx() - (6*cw+5*gap)/2
	var tag string
	for i, f := range folks {
		cr := rect{x0 + float64(i)*(cw+gap), y, cw, 236}
		id := "folk:" + f.ID
		if su.Opp == f.ID {
			box(dst, stFolkOn, cr, 1)
			tag = f.Tag
		} else if hoverID == id {
			box(dst, stFolkHot, cr, 1)
		}
		avatar(dst, f.ID, cr.cx(), cr.y+66, 56, 3)
		n := a.names[f.ID]
		lines := wrap(textStyle{f: fSemi, size: 17}, n.Full, cw-12)
		yy := cr.y + 150
		for _, l := range lines {
			text(dst, textStyle{f: fSemi, size: 17}, l, cr.cx(), yy, alignCenter, cInk)
			yy += 20
		}
		rt := textStyle{f: fRegular, size: 14, spacing: 1}
		text(dst, rt, f.Rating, cr.cx(), yy+2, alignCenter, cAccent)
		pw := float64(f.Level) * 14 * .62
		pawnIcons(dst, f.Level, cr.cx()-pw/2-2, yy+20, 14)
		rec := a.sd.Stats.Per[f.ID]
		recS := "Not played yet"
		if rec != nil && rec.W+rec.D+rec.L > 0 {
			recS = fmt.Sprintf("You: %dW %dD %dL", rec.W, rec.D, rec.L)
		}
		text(dst, textStyle{f: fRegular, size: 13}, recS, cr.cx(), yy+40, alignCenter, withA(cInk, .7))
		fid := f.ID
		addHit(id, cr, true, func() { su.Opp = fid; play("click"); writeSave(a.sd) })
	}
	y += 252
	if tag != "" {
		text(dst, textStyle{f: fItalic, size: 17}, "“"+tag+"”", r.cx(), y+6, alignCenter, withA(cInk, .85))
	}
	return y + 28
}

func (a *App) drawPvpSetup(dst *ebiten.Image, r rect, y float64) float64 {
	s := &a.sd.Settings
	paragraph(dst, tsBodyI, "Pass and play on one computer. The board turns to face whoever is moving (you can switch that off in Options).", r.cx(), y+14, 700, withA(cInk, .85))
	y += 56
	for i, side := range []struct {
		face, label string
		val         *string
	}{{"you", "White", &s.P1}, {"p2", "Black", &s.P2}} {
		cx := r.cx() - 190 + float64(i)*380
		avatar(dst, side.face, cx, y+62, 56, 3)
		text(dst, tsSheetH, side.label, cx, y+146, alignCenter, cAccent)
		field(dst, &a.ui, "p"+fmt.Sprint(i+1), rect{cx - 130, y + 158, 260, 40}, side.val, "Player "+fmt.Sprint(i+1))
	}
	y += 220
	if a.sd.Stats.Pvp > 0 {
		text(dst, textStyle{f: fRegular, size: 14}, fmt.Sprintf("Games played on this computer: %d", a.sd.Stats.Pvp), r.cx(), y+6, alignCenter, withA(cInk, .7))
	}
	return y + 28
}

func (a *App) drawOnlineSetup(dst *ebiten.Image, r rect, y float64) float64 {
	s := &a.sd.Settings
	paragraph(dst, tsBodyI, "Play a friend over the internet or on your home network. One of you hosts, the other joins with the host's address.", r.cx(), y+14, 720, withA(cInk, .85))
	y += 50
	avatar(dst, "you", r.cx()-250, y+60, 50, 3)
	text(dst, tsSheetH, "Your name", r.cx()-170, y+44, alignLeft, cAccent)
	field(dst, &a.ui, "name", rect{r.cx() - 170, y + 58, 260, 40}, &s.Name, "Your name")
	text(dst, tsSheetH, "Host's address", r.cx()+110, y+44, alignLeft, cAccent)
	field(dst, &a.ui, "addr", rect{r.cx() + 110, y + 58, 240, 40}, &s.Host, "e.g. 192.168.1.20")
	y += 124
	rec := a.sd.Stats.Online
	if rec.W+rec.D+rec.L > 0 {
		text(dst, textStyle{f: fRegular, size: 14}, fmt.Sprintf("Your online record: %d won, %d drawn, %d lost", rec.W, rec.D, rec.L), r.cx(), y, alignCenter, withA(cInk, .7))
	}
	return y + 26
}

func (a *App) hostClicked() {
	name := cleanName(a.sd.Settings.Name, "Host")
	a.sd.Settings.Name = name
	writeSave(a.sd)
	if err := a.startHosting(name); err != nil {
		a.ui.message("Online Game", "Couldn't open the game for a friend: "+err.Error())
		return
	}
	a.ui.onlinePanel = "host"
	a.ui.show(ovOnline)
}

func (a *App) joinClicked() {
	s := &a.sd.Settings
	name := cleanName(s.Name, "Guest")
	s.Name = name
	if strings.TrimSpace(s.Host) == "" {
		a.ui.message("Join a Game", "Type the host's address first. They'll see it on their screen when they choose Host a Game.")
		return
	}
	writeSave(a.sd)
	a.join(name, s.Host)
	a.ui.onlinePanel = "join"
	a.ui.show(ovOnline)
}

func (a *App) drawOnline(dst *ebiten.Image) {
	cancel := func() { play("click"); a.leaveOnline(); a.ui.hide(ovOnline) }
	r := panel(dst, 720, 420, nil)
	cx := r.cx()
	if a.ui.onlinePanel == "host" {
		heading(dst, "Waiting for a friend", cx, r.y+70, 38)
		y := paragraph(dst, tsBody, "Give your friend one of these addresses. They choose Online, then Join a Game, and type it in.", cx, r.y+118, 600, cInk)
		ips := localIPs()
		if len(ips) == 0 {
			ips = []string{"(no network found)"}
		}
		for _, ip := range ips {
			text(dst, textStyle{f: fSemi, size: 30, spacing: 1}, ip, cx, y+30, alignCenter, cAccent)
			y += 42
		}
		paragraph(dst, textStyle{f: fItalic, size: 14}, fmt.Sprintf("Over the internet, your friend uses your public address and your router must forward TCP port %d to this computer.", netPort), cx, y+18, 600, withA(cInk, .7))
	} else {
		heading(dst, "Joining", cx, r.y+70, 38)
		msg := "Connecting to " + a.sd.Settings.Host + "…"
		if a.net != nil && a.net.state == "waiting" {
			msg = "Connected. Waiting for the game to start…"
		}
		paragraph(dst, tsBody, msg, cx, r.y+150, 600, cInk)
	}
	dots := strings.Repeat("•", int(time.Now().UnixMilli()/400%4))
	text(dst, textStyle{f: fSemi, size: 28}, dots, cx, r.y+r.h-110, alignCenter, withA(cAccent, .6))
	bigButton(dst, "online:cancel", "Cancel", cx, r.y+r.h-84, true, cancel)
}

/* ------------------------------------------------------------ game over */

func (a *App) drawOver(dst *ebiten.Image) {
	o := &a.ui.over
	ph := 560.0
	if o.quote == "" {
		ph = 490
	}
	r := panel(dst, 660, ph, nil)
	cx := r.cx()
	avatar(dst, o.face, cx, r.y+92, 60, 4)
	text(dst, tsKick, o.kick, cx+2, r.y+186, alignCenter, withA(cInk, .7))
	text(dst, textStyle{f: fSemi, size: 44}, o.title, cx, r.y+236, alignCenter, cAccent)
	y := paragraph(dst, textStyle{f: fRegular, size: 18}, o.text, cx, r.y+278, 560, cInk)
	if o.quote != "" {
		paragraph(dst, textStyle{f: fItalic, size: 18}, o.quote, cx, y+14, 560, withA(cInk, .85))
	}
	buttonRow(dst, cx, r.y+r.h-136, []string{"Rematch", "New Game"}, []func(){
		func() { play("click"); a.ui.hide(ovOver); a.rematch() },
		func() { play("click"); a.leaveOnline(); a.showTitle() },
	})
	ly := r.y + r.h - 46
	w1, w2 := measure(tsBody, "Review the game"), measure(tsBody, "Copy moves (PGN)")
	lx := cx - (w1+34+w2)/2
	linkText(dst, "over:review", "Review the game", lx, ly, alignLeft, func() { a.ui.hide(ovOver); a.setView(0) })
	text(dst, tsBody, "·", lx+w1+17, ly, alignCenter, withA(cInk, .6))
	linkText(dst, "over:pgn", "Copy moves (PGN)", lx+w1+34, ly, alignLeft, a.copyPGN)
}

func (a *App) copyPGN() {
	if err := copyText(a.pgn()); err != nil {
		a.ui.banner("Could not copy", "")
		return
	}
	a.ui.banner("Moves copied", "Paste them into any chess program")
}

/* ------------------------------------------------------------ options, rules, credits */

func (a *App) drawSettings(dst *ebiten.Image) {
	s := &a.sd.Settings
	done := func() { play("click"); writeSave(a.sd); a.ui.hide(ovSettings) }
	r := panel(dst, 860, 560, done)
	heading(dst, "Options", r.cx(), r.y+66, 38)
	rows := []struct {
		label string
		id    string
		opts  [][2]string
		cur   *string
	}{
		{"Board", "o:board", [][2]string{{"walnut", "Walnut & Maple"}, {"marble", "Marble"}, {"club", "Club Green"}}, &s.Board},
		{"Show legal moves", "o:dots", [][2]string{{"on", "On"}, {"off", "Off"}}, &s.Dots},
		{"Move speed", "o:anim", [][2]string{{"slow", "Relaxed"}, {"normal", "Normal"}, {"fast", "Brisk"}}, &s.Anim},
		{"Promotion", "o:promo", [][2]string{{"ask", "Ask me"}, {"queen", "Always a queen"}}, &s.Promo},
		{"Same computer", "o:turn", [][2]string{{"on", "Turn the board each move"}, {"off", "Keep it still"}}, &s.Turn},
	}
	y := r.y + 110
	for _, row := range rows {
		text(dst, tsBody, row.label, r.x+70, y+26, alignLeft, cInk)
		cur := row.cur
		chips(dst, row.id, r.x+270, y, row.opts, *cur, func(v string) { *cur = v; writeSave(a.sd) })
		y += 58
	}
	paragraph(dst, textStyle{f: fItalic, size: 15}, "Right-click a square to mark it, or right-drag to draw an arrow while you plan. Click anywhere on the board to clear them.", r.cx(), y+16, 720, withA(cInk, .75))
	bigButton(dst, "set:done", "Done", r.cx(), r.y+r.h-80, false, done)
}

// richText draws the simple markup used by the rules text; returns the height used.
func richText(dst *ebiten.Image, src string, x, y, w float64, measureOnly bool) float64 {
	y0 := y
	for _, line := range strings.Split(src, "\n") {
		switch {
		case strings.HasPrefix(line, "# "):
			if !measureOnly {
				heading(dst, line[2:], x+w/2, y+40, 38)
			}
			y += 66
		case strings.HasPrefix(line, "~ "):
			ts := textStyle{f: fItalic, size: 15}
			for _, l := range wrap(ts, line[2:], w) {
				if !measureOnly {
					text(dst, ts, l, x+w/2, y+18, alignCenter, withA(cInk, .7))
				}
				y += 22
			}
			y += 6
		default:
			indent := 0.0
			num := ""
			if strings.HasPrefix(line, "1. ") {
				line = line[3:]
				indent = 30
				num = "•"
			}
			y = richPara(dst, line, x+indent, y, w-indent, num, measureOnly) + 8
		}
	}
	return y - y0
}

// richPara draws a paragraph with **bold** runs, wrapping words.
func richPara(dst *ebiten.Image, s string, x, y, w float64, bullet string, measureOnly bool) float64 {
	type word struct {
		t    string
		bold bool
		glue bool // no space before (punctuation right after a bold run)
	}
	var words []word
	bold := false
	prev := ""
	for i, part := range strings.Split(s, "**") {
		bold = i%2 == 1
		for j, wd := range strings.Fields(part) {
			// glue to the previous word when nothing separates them ("**checkmate**:")
			g := j == 0 && len(words) > 0 && part[0] != ' ' && prev != "" && !strings.HasSuffix(prev, " ")
			words = append(words, word{wd, bold, g})
		}
		prev = part
	}
	reg, bld := textStyle{f: fRegular, size: 17}, textStyle{f: fSemi, size: 17}
	space := measure(reg, " ")
	lh := 25.0
	cx := x
	y += 18
	if bullet != "" && !measureOnly {
		text(dst, bld, bullet, x-18, y, alignLeft, cAccent)
	}
	for i, wd := range words {
		ts := reg
		if wd.bold {
			ts = bld
		}
		ww := measure(ts, wd.t)
		if i > 0 && !wd.glue {
			cx += space
		}
		if cx > x && cx+ww > x+w && !wd.glue {
			cx = x
			y += lh
		}
		if !measureOnly {
			text(dst, ts, wd.t, cx, y, alignLeft, cInk)
		}
		cx += ww
	}
	return y + lh - 18
}

func (a *App) scrollPanel(dst *ebiten.Image, r rect, contentH float64, drawFn func(clip *ebiten.Image, y float64)) {
	maxS := math.Max(0, contentH-r.h)
	a.ui.panelScroll = math.Max(0, math.Min(maxS, a.ui.panelScroll))
	clip := dst.SubImage(image.Rect(int(dx(r.x)), int(dy(r.y)), int(math.Ceil(dx(r.x+r.w))), int(math.Ceil(dy(r.y+r.h))))).(*ebiten.Image)
	drawFn(clip, r.y-a.ui.panelScroll)
	if maxS > 0 { // scroll bar
		th := r.h * r.h / contentH
		ty := r.y + (r.h-th)*a.ui.panelScroll/maxS
		fillRect(dst, r.x+r.w+6, ty, 5, th, withA(hexc("#6e461e"), .45))
	}
	addHit("scroll", r, true, nil)
}

func (a *App) drawRules(dst *ebiten.Image) {
	closeFn := func() { play("click"); a.ui.hide(ovRules) }
	r := panel(dst, 980, 800, closeFn)
	area := rect{r.x + 60, r.y + 20, r.w - 120, r.h - 120}
	h := richText(dst, rulesText, area.x, 0, area.w, true)
	a.scrollPanel(dst, area, h+10, func(clip *ebiten.Image, y float64) { richText(clip, rulesText, area.x, y, area.w, false) })
	bigButton(dst, "rules:back", "Back to the Board", r.cx(), r.y+r.h-82, false, closeFn)
}

func (a *App) drawCredits(dst *ebiten.Image) {
	closeFn := func() { play("click"); a.ui.hide(ovCredits) }
	r := panel(dst, 900, 800, closeFn)
	area := rect{r.x + 50, r.y + 20, r.w - 100, r.h - 120}
	body := func(clip *ebiten.Image, y float64, measureOnly bool) float64 {
		y0 := y
		if !measureOnly {
			heading(clip, "Credits", area.x+area.w/2, y+48, 38)
		}
		y += 76
		for _, row := range creditRows {
			if row[0] == "" {
				ts := textStyle{f: fItalic, size: 15}
				for _, l := range wrap(ts, row[1], area.w-80) {
					if !measureOnly {
						text(clip, ts, l, area.x+area.w/2, y+16, alignCenter, withA(cInk, .8))
					}
					y += 21
				}
				y += 8
				continue
			}
			if !measureOnly {
				text(clip, tsBody, row[0], area.x+area.w/2-14, y+20, alignRight, withA(cInk, .75))
				text(clip, tsBodyB, row[1], area.x+area.w/2+14, y+20, alignLeft, cInk)
			}
			y += 30
		}
		if !measureOnly {
			text(clip, textStyle{f: fItalic, size: 15}, "Copyright © 2026 Ironite. All rights reserved.", area.x+area.w/2, y+26, alignCenter, withA(cInk, .7))
			heading(clip, "Changelog", area.x+area.w/2, y+100, 38)
		}
		y += 130
		for _, line := range strings.Split(changelogMD, "\n") {
			switch {
			case strings.HasPrefix(line, "## "):
				if !measureOnly {
					text(clip, textStyle{f: fSemi, size: 21}, strings.NewReplacer("[", "", "]", "").Replace(line[3:]), area.x, y+30, alignLeft, cAccent)
				}
				y += 42
			case strings.HasPrefix(line, "### "):
				if !measureOnly {
					text(clip, textStyle{f: fSemi, size: 15, spacing: 2, upper: true}, line[4:], area.x, y+22, alignLeft, withA(cInk, .8))
				}
				y += 30
			case strings.HasPrefix(line, "- "):
				y = richPara(clip, line[2:], area.x+22, y, area.w-22, "•", measureOnly) + 4
			case strings.HasPrefix(line, "# "), strings.TrimSpace(line) == "":
			default:
				y = richPara(clip, line, area.x, y, area.w, "", measureOnly) + 4
			}
		}
		return y - y0
	}
	h := body(nil, 0, true)
	a.scrollPanel(dst, area, h+20, func(clip *ebiten.Image, y float64) { body(clip, y, false) })
	bigButton(dst, "credits:close", "Close", r.cx(), r.y+r.h-82, false, closeFn)
}
