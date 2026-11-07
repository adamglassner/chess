//go:build autotest

// Screenshot test build: go build -tags autotest. Walks through the screens
// and saves a PNG of each into $CHESS_SHOTS, then quits.
// (c) 2026 Ironite
package main

import (
	"image"
	"log"
	"os"
	"path/filepath"

	"github.com/adamglassner/chess/engine"
	"github.com/hajimehoshi/ebiten/v2"
)

type step struct {
	wait  int // frames to wait before running fn
	fn    func(a *App)
	shot  string
	until func(a *App) bool // optional: keep waiting until true
}

func init() {
	dir := os.Getenv("CHESS_SHOTS")
	if dir == "" {
		return
	}
	_ = os.MkdirAll(dir, 0o755)
	moves := func(a *App, us ...string) {
		for _, u := range us {
			if m, ok := findUCI(a.legal, u); ok {
				a.playMove(m)
			} else {
				log.Printf("autotest: illegal %s", u)
			}
		}
	}
	steps := []step{
		{wait: 90, shot: "01-title-computer"},
		{wait: 5, fn: func(a *App) { a.sd.Setup.Mode = "pvp" }, shot: "02-title-same-computer"},
		{wait: 5, fn: func(a *App) {
			a.sd.Setup.Mode = "online"
			a.sd.Settings.Name = "Adam"
			a.sd.Settings.Host = "192.168.1.20"
		}, shot: "03-title-online"},
		{wait: 5, fn: func(a *App) {
			a.sd.Setup = Setup{Mode: "ai", Opp: "whitlock", Color: "white", Clock: "none"}
			a.newGame(a.sd.Setup)
			moves(a, "e2e4")
		}},
		{wait: 10, until: func(a *App) bool { return len(a.hist) >= 2 && !a.animating() }},
		{wait: 5, fn: func(a *App) { moves(a, "g1f3") }},
		{wait: 10, until: func(a *App) bool { return len(a.hist) >= 4 && !a.animating() }},
		{wait: 5, fn: func(a *App) { moves(a, "f1c4") }},
		{wait: 10, until: func(a *App) bool { return len(a.hist) >= 6 && !a.animating() }},
		{wait: 40, fn: func(a *App) { a.selectSq(0x01) }, shot: "04-game-select"},
		{wait: 5, fn: func(a *App) { a.sel, a.selMoves = -1, nil; a.hint() }},
		{wait: 10, until: func(a *App) bool { return !a.busy }},
		{wait: 20, shot: "05-hint"},
		{wait: 5, fn: func(a *App) { a.ui.show(ovSettings) }, shot: "06-options"},
		{wait: 5, fn: func(a *App) { a.ui.hideAll(); a.ui.show(ovRules) }, shot: "07-rules"},
		{wait: 5, fn: func(a *App) { a.ui.hideAll(); a.ui.show(ovCredits) }, shot: "08-credits"},
		{wait: 5, fn: func(a *App) {
			a.ui.hideAll()
			a.sd.Settings.P1, a.sd.Settings.P2 = "Adam", "Grace"
			a.sd.Settings.Board = "marble"
			a.newGame(Setup{Mode: "pvp", Clock: "5+3"})
			moves(a, "e2e4", "d7d5", "e4d5", "c7c6", "d5c6", "d8b6", "c6b7", "b6b2", "g1f3", "b2a1", "f1c4", "e8d8")
		}},
		{wait: 60, fn: func(a *App) { a.selectSq(0x61); a.tryMove(0x61, 0x72) }, shot: "09-promotion-same-computer"},
		{wait: 5, fn: func(a *App) {
			a.promo = nil
			a.sd.Settings.Board = "club"
			a.newGame(Setup{Mode: "pvp", Clock: "none"})
			a.sd.Settings.Turn = "off"
			moves(a, "e2e4", "e7e5", "d1h5", "b8c6", "f1c4", "g8f6", "h5f7")
		}},
		{wait: 150, shot: "10-game-over"},
		{wait: 5, fn: func(a *App) {
			a.ui.hideAll()
			a.sd.Settings.Board = "walnut"
			a.sd.Settings.Turn = "on"
			a.newGameRec(&GameRec{Mode: "online", YouWhite: false, WName: "Grace", BName: "Adam", Clock: clockFor("10")})
			a.net = &onlineT{state: "playing", myName: "Adam", peerName: "Grace"}
			moves(a, "d2d4", "g8f6", "c2c4", "e7e6", "b1c3", "f8b4")
			a.ui.bubble("", 0)
		}},
		{wait: 60, shot: "11-online-game"},
		{wait: 5, fn: func(a *App) { a.net = nil; a.ui.onlinePanel = "host"; a.ui.show(ovOnline) }, shot: "12-online-hosting"},
		{wait: 5, fn: func(a *App) {
			a.ui.hideAll()
			a.ui.ask("Draw offered", "Grace offers a draw. Do you accept?", "Accept Draw", "Play On", nil, nil)
		}, shot: "13-online-draw-offer"},
		{wait: 5, fn: func(a *App) {
			a.ui.hideAll()
			a.newGame(Setup{Mode: "ai", Opp: "vance", Color: "black", Clock: "15+10"})
		}},
		{wait: 10, until: func(a *App) bool { return len(a.hist) >= 1 && !a.animating() }},
		{wait: 30, shot: "14-black-vs-professor"},
		{wait: 5, fn: func(a *App) { os.Exit(0) }},
	}
	_ = engine.White
	i, n := 0, 0
	shotNow := ""
	autotest = func(a *App) {
		if i >= len(steps) {
			return
		}
		s := steps[i]
		n++
		if n < s.wait {
			return
		}
		if s.until != nil && !s.until(a) {
			if n > 1200 {
				log.Printf("autotest: step %d timed out", i)
			} else {
				return
			}
		}
		if s.fn != nil {
			s.fn(a)
		}
		if s.shot != "" {
			shotNow = s.shot
		}
		i++
		n = 0
	}
	afterDraw = func(sc *ebiten.Image) {
		if shotNow == "" {
			return
		}
		b := sc.Bounds()
		px := make([]byte, 4*b.Dx()*b.Dy())
		sc.ReadPixels(px)
		im := &image.RGBA{Pix: px, Stride: 4 * b.Dx(), Rect: image.Rect(0, 0, b.Dx(), b.Dy())}
		_ = os.WriteFile(filepath.Join(dir, shotNow+".png"), pngBytes(im), 0o644)
		log.Printf("shot %s %dx%d", shotNow, b.Dx(), b.Dy())
		shotNow = ""
	}
}
