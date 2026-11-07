// Chess: Ravenscroft Library. A native Windows game written in Go with
// Ebitengine. Play six members of the library from beginner to master, a
// friend on the same computer, or a friend online.
//
// (c) 2026 Ironite
package main

import (
	"log"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/adamglassner/chess/engine"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type game struct {
	a        *App
	lastStep time.Time
	devW     int
	devH     int
}

func newApp() *App {
	sd := loadSave()
	a := &App{sd: &sd, view: -1, sel: -1, rightFrom: -1, marks: map[int]bool{}, drawOfferPly: -99, lastTick: time.Now()}
	a.names = drawNames()
	return a
}

func (g *game) Update() error {
	a := g.a
	if ebiten.IsWindowBeingClosed() {
		a.leaveOnline()
		if a.G != nil && a.G.Mode != "online" {
			a.persist()
		} else {
			writeSave(a.sd)
		}
		return ebiten.Termination
	}
	now := time.Now()
	dt := math.Min(.1, now.Sub(g.lastStep).Seconds())
	g.lastStep = now

	a.ui.runTimers()
	a.updateNet()
	a.updateJob()
	a.updateClock()
	a.updateAnims()
	a.ui.stepParticles(dt)
	updateMusic()
	updateSounds()
	a.input()
	if autotest != nil {
		autotest(a)
	}
	return nil
}

// input handles the mouse and keyboard.
func (a *App) input() {
	mx, my := ebiten.CursorPosition()
	lx, ly := toLogical(float64(mx), float64(my))

	// hover
	hoverID = ""
	if h := hitAt(lx, ly); h != nil && h.enabled && h.fn != nil {
		hoverID = h.id
	}
	cursor := ebiten.CursorShapeDefault
	if hoverID != "" {
		cursor = ebiten.CursorShapePointer
	}

	// volume slider drag
	if a.ui.dragVol {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			v := (lx - volX) / 88 * 100
			a.sd.Settings.Vol = math.Round(math.Max(0, math.Min(100, v)))
		} else {
			a.ui.dragVol = false
			writeSave(a.sd)
		}
	}

	// typing into a field
	if a.ui.focus != "" {
		a.typeText()
	}

	// wheel scrolling
	if _, wy := ebiten.Wheel(); wy != 0 {
		if h := hitAt(lx, ly); h != nil {
			switch h.id {
			case "sheet":
				a.ui.sheetScroll -= wy * 40
			case "scroll":
				a.ui.panelScroll -= wy * 50
			}
		}
	}

	onBoard := a.ui.top() == ovNone && a.promo == nil
	bxl, byl := lx-boardX, ly-boardY

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		audioUnlock()
		h := hitAt(lx, ly)
		if a.ui.focus != "" && (h == nil || h.id != "field:"+a.ui.focus) {
			a.ui.focus = ""
		}
		switch {
		case h != nil && h.id == "board" && onBoard:
			a.boardDown(bxl, byl)
		case h != nil && h.enabled && h.fn != nil:
			if h.id == "tb:vol" {
				h.fn()
				v := (lx - volX) / 88 * 100
				a.sd.Settings.Vol = math.Round(math.Max(0, math.Min(100, v)))
			} else {
				pressID = h.id
				h.fn()
			}
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) && onBoard {
		a.rightFrom = a.xySq(bxl, byl)
	}
	if a.drag != nil {
		a.drag.x, a.drag.y = bxl, byl
		if !a.drag.moved && math.Hypot(bxl-a.drag.x0, byl-a.drag.y0) > 6 {
			a.drag.moved = true
		}
		cursor = ebiten.CursorShapeMove
	} else if onBoard {
		if sq := a.xySq(bxl, byl); sq >= 0 && (a.canMoveFrom(sq) || (a.sel >= 0 && a.hasTarget(sq))) {
			cursor = ebiten.CursorShapePointer
		}
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		a.boardUp(bxl, byl)
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonRight) && onBoard {
		sq := a.xySq(bxl, byl)
		if a.rightFrom >= 0 && sq >= 0 {
			if sq == a.rightFrom {
				if a.marks[sq] {
					delete(a.marks, sq)
				} else {
					a.marks[sq] = true
				}
			} else {
				found := -1
				for i, ar := range a.arrows {
					if ar[0] == a.rightFrom && ar[1] == sq {
						found = i
					}
				}
				if found >= 0 {
					a.arrows = append(a.arrows[:found], a.arrows[found+1:]...)
				} else {
					a.arrows = append(a.arrows, [2]int{a.rightFrom, sq})
				}
			}
		}
		a.rightFrom = -1
	}
	ebiten.SetCursorShape(cursor)
	a.keys()
}

var volX = 1000.0 // left edge of the volume slider, set by drawToolbar

func (a *App) hasTarget(sq int) bool {
	for _, m := range a.selMoves {
		if m.To == sq {
			return true
		}
	}
	return false
}

func (a *App) boardDown(x, y float64) {
	sq := a.xySq(x, y)
	if len(a.marks) > 0 || len(a.arrows) > 0 {
		a.marks, a.arrows = map[int]bool{}, nil
	}
	if a.G == nil {
		return
	}
	if sq < 0 {
		a.sel, a.selMoves = -1, nil
		return
	}
	if !a.live() {
		a.setView(len(a.hist))
		return
	}
	if a.sel >= 0 && a.sel != sq && a.hasTarget(sq) {
		a.tryMove(a.sel, sq)
		return
	}
	if a.canMoveFrom(sq) {
		wasSel := a.sel == sq
		if !wasSel {
			a.selectSq(sq)
		}
		a.drag = &dragT{from: sq, x: x, y: y, x0: x, y0: y, wasSel: wasSel}
		return
	}
	if a.sel >= 0 && a.pos.Board[sq] != 0 && sq != a.sel {
		play("illegal")
	}
	a.sel, a.selMoves = -1, nil
}

func (a *App) boardUp(x, y float64) {
	d := a.drag
	if d == nil {
		return
	}
	a.drag = nil
	sq := a.xySq(x, y)
	if d.moved {
		if sq >= 0 && sq != d.from && a.tryMove(d.from, sq) {
			return
		}
		if sq >= 0 && sq != d.from {
			play("illegal")
		}
		return
	}
	if d.wasSel {
		a.sel, a.selMoves = -1, nil
	}
}

func (a *App) keys() {
	ctrl := ebiten.IsKeyPressed(ebiten.KeyControl)
	jp := inpututil.IsKeyJustPressed
	if jp(ebiten.KeyF11) || (jp(ebiten.KeyEnter) && ebiten.IsKeyPressed(ebiten.KeyAlt)) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
		return
	}
	if a.ui.focus != "" {
		return
	}
	top := a.ui.top()
	if jp(ebiten.KeyEscape) {
		switch {
		case a.promo != nil:
			a.promo, a.sel, a.selMoves = nil, -1, nil
		case top == ovSettings || top == ovRules || top == ovCredits || top == ovMsg:
			a.ui.hide(top)
			writeSave(a.sd)
		case top == ovOnline:
			a.leaveOnline()
			a.ui.hide(ovOnline)
		case top == ovNone && ebiten.IsFullscreen():
			ebiten.SetFullscreen(false)
		}
		return
	}
	if top == ovTitle {
		if jp(ebiten.KeyEnter) && a.sd.Setup.Mode != "online" {
			a.newGame(a.sd.Setup)
		}
		return
	}
	if top != ovNone {
		return
	}
	if jp(ebiten.KeyM) {
		a.sd.Settings.Music = !a.sd.Settings.Music
		writeSave(a.sd)
	}
	if a.G == nil {
		return
	}
	n := len(a.hist)
	v := n
	if !a.live() {
		v = a.view
	}
	switch {
	case (ctrl && jp(ebiten.KeyZ)) || jp(ebiten.KeyBackspace):
		a.undo()
	case jp(ebiten.KeyH):
		a.hint()
	case jp(ebiten.KeyF):
		play("click")
		a.setFlip(!a.flipped)
		a.persist()
	case jp(ebiten.KeyN):
		a.showTitle()
	case jp(ebiten.KeyArrowLeft) && v > 0:
		a.setView(v - 1)
	case jp(ebiten.KeyArrowRight) && v < n:
		a.setView(v + 1)
	case jp(ebiten.KeyHome) && v > 0:
		a.setView(0)
	case jp(ebiten.KeyEnd) && v < n:
		a.setView(n)
	}
}

// typeText edits the focused text field.
func (a *App) typeText() {
	v := fieldVals[a.ui.focus]
	if v == nil {
		a.ui.focus = ""
		return
	}
	limit := 14
	if a.ui.focus == "addr" {
		limit = 60
	}
	changed := false
	for _, r := range ebiten.AppendInputChars(nil) {
		if r >= ' ' && len([]rune(*v)) < limit {
			*v += string(r)
			changed = true
		}
	}
	if d := inpututil.KeyPressDuration(ebiten.KeyBackspace); d == 1 || (d > 25 && d%3 == 0) {
		if rs := []rune(*v); len(rs) > 0 {
			*v = string(rs[:len(rs)-1])
			changed = true
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyControl) && inpututil.IsKeyJustPressed(ebiten.KeyV) {
		if s, err := pasteText(); err == nil {
			for _, r := range s {
				if r >= ' ' && len([]rune(*v)) < limit {
					*v += string(r)
				}
			}
			changed = true
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyTab) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		a.ui.focus = ""
	}
	if changed {
		writeSave(a.sd)
	}
}

func (g *game) Draw(screen *ebiten.Image) {
	b := screen.Bounds()
	if st.fit(float64(b.Dx()), float64(b.Dy())) {
		textCache.clear()
		boxCache.clear()
	}
	g.a.draw(screen)
	if afterDraw != nil {
		afterDraw(screen)
	}
}

func (g *game) Layout(w, h int) (int, int) { return w, h } // LayoutF is used instead

func (g *game) LayoutF(w, h float64) (float64, float64) {
	s := ebiten.Monitor().DeviceScaleFactor()
	return math.Round(w * s), math.Round(h * s)
}

var (
	autotest  func(a *App)           // set by the screenshot test build
	afterDraw func(sc *ebiten.Image) // likewise
)

func main() {
	logPath := filepath.Join(os.TempDir(), "IroniteChess.log")
	if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644); err == nil {
		log.SetOutput(f)
	}
	log.Printf("Chess %s starting", version)
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "/version") {
		os.Stdout.WriteString(version + "\n")
		return
	}
	if err := loadAssets(); err != nil {
		log.Fatal(err)
	}
	a := newApp()
	initAudio(&a.sd.Settings, a.ui.nowPlaying)
	if a.sd.Game != nil && a.sd.Game.Moves != nil {
		a.G = a.sd.Game
		a.rebuild()
		a.flipped = a.G.Flipped
		a.refreshOpening()
	} else {
		a.G = &GameRec{StartFen: engine.StartFEN, Moves: []string{}, Mode: "ai", Opp: a.sd.Setup.Opp, YouWhite: true}
		a.rebuild()
		a.G = nil
	}
	a.showTitle()

	ebiten.SetWindowTitle("Chess")
	ebiten.SetWindowIcon(art.icons)
	mw, mh := ebiten.Monitor().Size()
	w, h := 1600, 900
	if mw > 0 && mh > 0 {
		s := math.Min(float64(mw)*.88/1600, float64(mh)*.85/900)
		w, h = int(1600*s), int(900*s)
	}
	ebiten.SetWindowSize(w, h)
	ebiten.SetWindowSizeLimits(640, 360, -1, -1)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowClosingHandled(true)
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetTPS(60)
	g := &game{a: a, lastStep: time.Now()}
	op := &ebiten.RunGameOptions{}
	if err := ebiten.RunGameWithOptions(g, op); err != nil {
		log.Fatal(err)
	}
}
