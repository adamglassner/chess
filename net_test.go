package main

import (
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/adamglassner/chess/engine"
)

// Online tests: a host and a guest over real TCP on this computer.

func testApp(t *testing.T) *App {
	sd := defaultSave()
	a := &App{sd: &sd, view: -1, sel: -1, rightFrom: -1, marks: map[int]bool{}, drawOfferPly: -99, lastTick: time.Now()}
	a.names = drawNames()
	return a
}

// pump runs both games' network and timers until cond holds.
func pump(t *testing.T, cond func() bool, apps ...*App) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("timed out")
		}
		for _, a := range apps {
			a.updateNet()
			a.ui.runTimers()
			a.updateClock()
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func connect(t *testing.T, color, clock string) (*App, *App) {
	host, guest := testApp(t), testApp(t)
	host.sd.Setup = Setup{Mode: "online", Color: color, Clock: clock}
	if err := host.startHosting("Hal"); err != nil {
		t.Fatal(err)
	}
	guest.join("Gus", "127.0.0.1")
	t.Cleanup(func() { host.leaveOnline(); guest.leaveOnline() })
	pump(t, func() bool { return host.online() && guest.online() }, host, guest)
	return host, guest
}

// answer clicks Yes or No on the question showing, as the buttons do.
func answer(a *App, yes bool) {
	a.ui.hide(ovConfirm)
	if yes && a.ui.askT.onYes != nil {
		a.ui.askT.onYes()
	}
	if !yes && a.ui.askT.onNo != nil {
		a.ui.askT.onNo()
	}
}

func same(t *testing.T, a, b *App) {
	t.Helper()
	if a.pos.FEN() != b.pos.FEN() || len(a.hist) != len(b.hist) {
		t.Fatalf("out of step:\n host  %s (%d)\n guest %s (%d)", a.pos.FEN(), len(a.hist), b.pos.FEN(), len(b.hist))
	}
}

func TestMain(m *testing.M) {
	d, _ := os.MkdirTemp("", "chesstest")
	os.Setenv("APPDATA", d)
	os.Setenv("XDG_CONFIG_HOME", d)
	os.Exit(m.Run())
}

func TestOnlineFullGames(t *testing.T) {
	host, guest := connect(t, "random", "none")
	if host.G.YouWhite == guest.G.YouWhite {
		t.Fatal("both players got the same colour")
	}
	if host.G.WName == "" || host.G.WName != guest.G.WName || host.G.BName != guest.G.BName {
		t.Fatalf("names differ: %q/%q vs %q/%q", host.G.WName, host.G.BName, guest.G.WName, guest.G.BName)
	}
	games := 0
	for games < 3 {
		for ply := 0; ply < 400 && host.G.Result == nil; ply++ {
			mover := host
			if host.pos.Side != host.youSide() {
				mover = guest
			}
			m := mover.legal[rand.Intn(len(mover.legal))]
			// prefer captures so games finish
			for _, x := range mover.legal {
				if x.Captured != 0 && rand.Intn(2) == 0 {
					m = x
					break
				}
			}
			n := len(mover.hist)
			mover.playMove(m)
			pump(t, func() bool { return len(host.hist) == n+1 && len(guest.hist) == n+1 }, host, guest)
			same(t, host, guest)
		}
		if host.G.Result == nil {
			// play-out limit reached: agree a draw
			host.askPeer("draw")
			pump(t, func() bool { return guest.ui.visible(ovConfirm) }, host, guest)
			answer(guest, true)
			pump(t, func() bool { return host.G.Result != nil }, host, guest)
		}
		if guest.G.Result == nil || *host.G.Result != *guest.G.Result {
			t.Fatalf("results differ: %+v vs %+v", host.G.Result, guest.G.Result)
		}
		t.Logf("game %d: %s %s after %d plies", games+1, host.G.Result.Result, host.G.Result.Reason, len(host.hist))
		games++
		// rematch with colours swapped
		wasWhite := host.G.YouWhite
		guest.rematch()
		pump(t, func() bool { return host.ui.visible(ovConfirm) }, host, guest)
		answer(host, true)
		pump(t, func() bool {
			return guest.G.Result == nil && host.G.Result == nil && len(guest.hist) == 0 && guest.net.pending == ""
		}, host, guest)
		if host.G.YouWhite == wasWhite || guest.G.YouWhite == host.G.YouWhite {
			t.Fatal("rematch didn't swap colours")
		}
	}
	if r := host.sd.Stats.Online; r.W+r.D+r.L != 3 {
		t.Fatalf("host online record %+v, want 3 games", r)
	}
	host.leaveOnline()
	guest.leaveOnline()
}

func TestOnlineDrawUndoResign(t *testing.T) {
	host, guest := connect(t, "white", "none")
	mv := func(a *App, u string) {
		n := len(a.hist)
		m, ok := findUCI(a.legal, u)
		if !ok {
			t.Fatalf("illegal %s", u)
		}
		a.playMove(m)
		pump(t, func() bool { return len(host.hist) == n+1 && len(guest.hist) == n+1 }, host, guest)
	}
	mv(host, "e2e4")
	mv(guest, "e7e5")
	mv(host, "g1f3")
	// guest asks to take back... their own last move is e5, and it's host's turn? no: after Nf3 it's guest's turn
	mv(guest, "b8c6")
	// host's turn; guest regrets Nc6
	guest.undo()
	pump(t, func() bool { return host.ui.visible(ovConfirm) }, host, guest)
	answer(host, true)
	pump(t, func() bool { return len(host.hist) == 3 && len(guest.hist) == 3 }, host, guest)
	same(t, host, guest)
	if host.pos.Side != engine.Black {
		t.Fatal("after the take-back it should be Black's move")
	}
	// a refused draw
	guest.offerDraw()
	pump(t, func() bool { return host.ui.visible(ovConfirm) }, host, guest)
	answer(host, false)
	pump(t, func() bool { return guest.net.pending == "" }, host, guest)
	if host.G.Result != nil || guest.G.Result != nil {
		t.Fatal("a refused draw ended the game")
	}
	// resign
	guest.resign()
	answer(guest, true)
	pump(t, func() bool { return host.G.Result != nil }, host, guest)
	if host.G.Result.Winner != engine.White || guest.G.Result.Winner != engine.White || host.G.Result.Reason != "resignation" {
		t.Fatalf("resign: %+v / %+v", host.G.Result, guest.G.Result)
	}
	host.leaveOnline()
	guest.leaveOnline()
}

func TestOnlineClockAndFlag(t *testing.T) {
	host, guest := connect(t, "black", "5+3")
	if host.G.YouWhite || !guest.G.YouWhite {
		t.Fatal("host asked to play Black")
	}
	m, _ := findUCI(guest.legal, "d2d4")
	guest.playMove(m)
	pump(t, func() bool { return len(host.hist) == 1 }, host, guest)
	m, _ = findUCI(host.legal, "d7d5")
	host.playMove(m)
	pump(t, func() bool { return len(guest.hist) == 2 }, host, guest)
	if d := host.G.Clock.B - guest.G.Clock.B; d > 50 || d < -50 {
		t.Fatalf("clocks disagree: %v vs %v", host.G.Clock.B, guest.G.Clock.B)
	}
	if host.G.Clock.B <= 5*60000 { // got the 3 second increment
		t.Logf("black clock %v", host.G.Clock.B)
	}
	// White (the guest) is on move: run their clock out on the host
	host.G.Clock.W = 50
	pump(t, func() bool { return guest.G.Result != nil }, host, guest)
	if host.G.Result.Reason != "time" || guest.G.Result.Winner != engine.Black {
		t.Fatalf("flag: %+v / %+v", host.G.Result, guest.G.Result)
	}
	host.leaveOnline()
	guest.leaveOnline()
}

func TestOnlinePeerLeaves(t *testing.T) {
	host, guest := connect(t, "white", "none")
	m, _ := findUCI(host.legal, "e2e4")
	host.playMove(m)
	pump(t, func() bool { return len(guest.hist) == 1 }, host, guest)
	guest.leaveOnline()
	pump(t, func() bool { return host.ui.visible(ovMsg) }, host)
	if host.G.Result == nil || host.net != nil {
		t.Fatal("host didn't notice the guest leaving")
	}
	// a second guest can't join a full game; a refused version mismatch
	h2 := testApp(t)
	h2.sd.Setup = Setup{Mode: "online", Color: "white", Clock: "none"}
	if err := h2.startHosting("Hal"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h2.leaveOnline)
	g2, g3 := testApp(t), testApp(t)
	g2.join("A", "127.0.0.1")
	pump(t, func() bool { return h2.online() && g2.online() }, h2, g2)
	g3.join("B", "127.0.0.1")
	pump(t, func() bool { return g3.ui.visible(ovMsg) }, h2, g2, g3)
	if g3.online() || g3.net != nil {
		t.Fatal("a third player got in")
	}
	h2.leaveOnline()
	g2.leaveOnline()
}

func TestRandomNames(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		for _, f := range folks {
			n := randomName(f.ID, "")
			if n.Short == "" || n.Full == "" {
				t.Fatal("empty name")
			}
			seen[n.Short] = true
		}
	}
	if len(seen) < 30 {
		t.Fatalf("only %d different names", len(seen))
	}
	a := testApp(t)
	a.sd.Setup = Setup{Mode: "ai", Opp: "hollis", Color: "white", Clock: "none"}
	before := a.names["hollis"]
	a.newGame(a.sd.Setup)
	if a.G.OppShort != before.Short || a.oppFull() == "" {
		t.Fatalf("game didn't use the name on the title screen: %q vs %q", a.G.OppShort, before.Short)
	}
	if a.names["hollis"].Short == before.Short {
		t.Fatal("next game would have the same name")
	}
	if a.G.OppShort[:4] != "Mrs." {
		t.Fatalf("librarian name %q", a.G.OppShort)
	}
}

func TestComputerGameAndResume(t *testing.T) {
	for _, id := range []string{"pip", "ambrose", "whitlock"} {
		L := levels[id]
		L.minDelay, L.maxDelay = 0, 0
		levels[id] = L
	}
	a := testApp(t)
	a.newGame(Setup{Mode: "ai", Opp: "pip", Color: "black", Clock: "none"})
	step := func() {
		deadline := time.Now().Add(20 * time.Second)
		for a.busy || a.animating() {
			a.updateJob()
			a.updateAnims()
			a.ui.runTimers()
			time.Sleep(time.Millisecond)
			if time.Now().After(deadline) {
				t.Fatal("computer never moved")
			}
		}
	}
	step()
	if len(a.hist) != 1 {
		t.Fatalf("computer (White) should have moved first: %d plies", len(a.hist))
	}
	for ply := 0; ply < 300 && a.G.Result == nil; ply++ {
		if !a.yourTurn() {
			t.Fatalf("not your turn at ply %d", len(a.hist))
		}
		a.playMove(a.legal[rand.Intn(len(a.legal))])
		step()
		if ply == 6 { // take back: your move and the reply
			n := len(a.hist)
			a.undo()
			step()
			if len(a.hist) != n-2 || !a.yourTurn() {
				t.Fatalf("undo: %d -> %d plies", n, len(a.hist))
			}
		}
	}
	t.Logf("vs %s: %v after %d plies", a.G.OppShort, a.G.Result, len(a.hist))
	if a.G.Result != nil && a.sd.Stats.Won+a.sd.Stats.Drawn+a.sd.Stats.Lost != 1 {
		t.Fatal("result not recorded")
	}

	// an unfinished game comes back after a restart
	a.newGame(Setup{Mode: "ai", Opp: "whitlock", Color: "white", Clock: "10"})
	for i := 0; i < 4; i++ {
		a.playMove(a.legal[0])
		step()
	}
	fen, name := a.pos.FEN(), a.G.OppShort
	b := testApp(t)
	*b.sd = loadSave()
	if b.sd.Game == nil {
		t.Fatal("no saved game")
	}
	b.resume()
	if b.pos.FEN() != fen || b.G.OppShort != name || b.G.Clock == nil {
		t.Fatalf("resumed %s %q, want %s %q", b.pos.FEN(), b.G.OppShort, fen, name)
	}
}
