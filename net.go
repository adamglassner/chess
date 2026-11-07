// Online play, the same way as Texas Hold'em, Go Fish and Euchre: one
// player hosts, a friend joins by IP address (TCP port 47025). The host's
// game is the referee: it deals the colours, keeps the clocks and decides
// when a flag falls. Both sides check every move against the rules. Messages
// are JSON, one per line.
//
// (c) 2026 Ironite
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"sort"
	"strings"
	"time"
)

const (
	netPort  = 47025
	protoVer = 1
)

type Msg struct {
	T     string     `json:"t"`               // hello, welcome, refuse, start, move, ask, answer, resign, flag, bye, ping
	Name  string     `json:"name,omitempty"`  // hello / welcome
	Ver   int        `json:"ver,omitempty"`   //
	Why   string     `json:"why,omitempty"`   // refuse
	White bool       `json:"white,omitempty"` // start: the receiver plays White
	Clock string     `json:"clock,omitempty"` // start: clock setting
	UCI   string     `json:"uci,omitempty"`   // move
	Ply   int        `json:"ply,omitempty"`   // move: ply number (sanity check)
	CW    float64    `json:"cw,omitempty"`    // move: clocks after the move (ms)
	CB    float64    `json:"cb,omitempty"`    //
	What  string     `json:"what,omitempty"`  // ask / answer: draw, undo, rematch
	Yes   bool       `json:"yes,omitempty"`   // answer
	Res   *ResultRec `json:"res,omitempty"`   // flag: the result the host decided
}

type netEvent struct {
	m   Msg
	err error // connection closed
}

// link is one connection: a reader goroutine feeding events and a writer
// goroutine draining a queue.
type link struct {
	c      net.Conn
	out    chan []byte
	in     chan netEvent
	closed bool
}

func newLink(c net.Conn) *link {
	l := &link{c: c, out: make(chan []byte, 256), in: make(chan netEvent, 256)}
	go func() {
		for b := range l.out {
			_ = c.SetWriteDeadline(time.Now().Add(15 * time.Second))
			if _, err := c.Write(b); err != nil {
				_ = c.Close()
				for range l.out {
				}
				return
			}
		}
		_ = c.Close()
	}()
	go func() {
		r := bufio.NewReader(c)
		for {
			_ = c.SetReadDeadline(time.Now().Add(30 * time.Second))
			line, err := r.ReadBytes('\n')
			if err != nil {
				l.in <- netEvent{err: err}
				return
			}
			var m Msg
			if json.Unmarshal(line, &m) != nil {
				continue
			}
			if m.T == "ping" {
				continue
			}
			l.in <- netEvent{m: m}
		}
	}()
	return l
}

func (l *link) send(m Msg) {
	if l == nil || l.closed {
		return
	}
	b, err := json.Marshal(m)
	if err != nil {
		return
	}
	select {
	case l.out <- append(b, '\n'):
	default:
		l.close()
	}
}

func (l *link) close() {
	if l != nil && !l.closed {
		l.closed = true
		close(l.out)
	}
}

/* ------------------------------------------------------------- session */

type onlineT struct {
	host     bool
	ln       net.Listener
	accepted chan net.Conn
	peer     *link
	dialing  chan dialResult
	myName   string
	peerName string
	state    string // "", hosting, dialing, waiting (guest waiting for start), playing, gone
	lastPing time.Time
	pending  string // question we asked and are waiting on (draw, undo, rematch)
}

type dialResult struct {
	c   net.Conn
	err error
}

func (a *App) online() bool { return a.G != nil && a.G.Mode == "online" }

// startHosting opens the port and waits for a friend.
func (a *App) startHosting(name string) error {
	a.leaveOnline()
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", netPort))
	if err != nil {
		return err
	}
	o := &onlineT{host: true, ln: ln, accepted: make(chan net.Conn, 4), myName: name, state: "hosting"}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			o.accepted <- c
		}
	}()
	a.net = o
	return nil
}

// join connects to a host.
func (a *App) join(name, addr string) {
	a.leaveOnline()
	addr = strings.TrimSpace(addr)
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = net.JoinHostPort(strings.Trim(addr, "[]"), fmt.Sprint(netPort))
	}
	o := &onlineT{myName: name, state: "dialing", dialing: make(chan dialResult, 1)}
	go func() {
		c, err := net.DialTimeout("tcp", addr, 8*time.Second)
		o.dialing <- dialResult{c, err}
	}()
	a.net = o
}

// leaveOnline says goodbye and closes everything.
func (a *App) leaveOnline() {
	o := a.net
	if o == nil {
		return
	}
	if o.peer != nil {
		o.peer.send(Msg{T: "bye"})
		o.peer.close()
	}
	if o.ln != nil {
		_ = o.ln.Close()
	}
	a.net = nil
}

// localIPs lists this computer's network addresses, for the host to share.
func localIPs() []string {
	var out []string
	ifs, _ := net.Interfaces()
	for _, i := range ifs {
		if i.Flags&net.FlagUp == 0 || i.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := i.Addrs()
		for _, ad := range addrs {
			if ip, ok := ad.(*net.IPNet); ok && ip.IP.To4() != nil && !ip.IP.IsLinkLocalUnicast() {
				out = append(out, ip.IP.String())
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { // home-network addresses first
		pi, pj := strings.HasPrefix(out[i], "192.168."), strings.HasPrefix(out[j], "192.168.")
		return pi && !pj
	})
	if len(out) > 3 {
		out = out[:3]
	}
	return out
}

// updateNet handles connections and messages; called every frame.
func (a *App) updateNet() {
	o := a.net
	if o == nil {
		return
	}
	// a friend knocks
	if o.accepted != nil {
		select {
		case c := <-o.accepted:
			if o.peer != nil {
				l := newLink(c)
				l.send(Msg{T: "refuse", Why: "This game already has two players."})
				l.close()
			} else {
				o.peer = newLink(c)
				o.lastPing = time.Now()
			}
		default:
		}
	}
	if o.dialing != nil {
		select {
		case r := <-o.dialing:
			o.dialing = nil
			if r.err != nil {
				a.netFail("Couldn't reach that address. Check it with your friend, and that their game is waiting in Host a Game.")
				return
			}
			o.peer = newLink(r.c)
			o.peer.send(Msg{T: "hello", Name: o.myName, Ver: protoVer})
			o.state = "waiting"
			o.lastPing = time.Now()
		default:
		}
	}
	if o.peer == nil {
		return
	}
	if time.Since(o.lastPing) > 5*time.Second {
		o.lastPing = time.Now()
		o.peer.send(Msg{T: "ping"})
	}
	for i := 0; i < 32; i++ {
		select {
		case ev := <-o.peer.in:
			if ev.err != nil {
				a.peerGone()
				return
			}
			a.handleMsg(ev.m)
			if a.net == nil || a.net.peer == nil {
				return
			}
		default:
			return
		}
	}
}

func (a *App) netFail(why string) {
	a.leaveOnline()
	a.ui.message("Online Game", why)
}

func (a *App) peerGone() {
	o := a.net
	name := "Your friend"
	if o != nil && o.peerName != "" {
		name = o.peerName
	}
	if o != nil && o.host && o.state == "hosting" {
		// a knock that never said hello: keep waiting
		o.peer.close()
		o.peer = nil
		return
	}
	playing := a.online() && a.G.Result == nil && o != nil && o.state == "playing"
	a.leaveOnline()
	a.busy = false
	if playing {
		a.ui.message("Game Over", name+" left the game. It can't be finished online, but you can review the moves.")
		if a.G != nil {
			a.G.Result = &ResultRec{Result: "*", Reason: "left"}
		}
	} else if a.ui.top() == ovOnline {
		a.ui.message("Online Game", name+" left.")
	}
}

func (a *App) handleMsg(m Msg) {
	o := a.net
	switch m.T {
	case "hello": // host: a guest says who they are
		if !o.host || o.state != "hosting" {
			return
		}
		if m.Ver != protoVer {
			o.peer.send(Msg{T: "refuse", Why: "Your friend has a different version of Chess. Both players need the same version."})
			o.peer.close()
			o.peer = nil
			return
		}
		o.peerName = cleanName(m.Name, "Friend")
		o.peer.send(Msg{T: "welcome", Name: o.myName, Ver: protoVer})
		a.startOnlineGame()
	case "welcome":
		o.peerName = cleanName(m.Name, "Friend")
		if m.Ver != protoVer {
			a.netFail("Your friend has a different version of Chess. Both players need the same version.")
		}
	case "refuse":
		a.netFail(m.Why)
	case "start": // guest: the host has started a game
		if o.host {
			return
		}
		a.beginOnline(m.White, m.Clock)
	case "move":
		if !a.online() || a.G.Result != nil {
			return
		}
		if m.Ply != len(a.hist) {
			log.Printf("net: move for ply %d at ply %d", m.Ply, len(a.hist))
			a.netFail("The two games got out of step, so the online game has stopped.")
			return
		}
		mv, ok := findUCI(a.legal, m.UCI)
		if !ok || a.pos.Side == a.youSide() {
			a.netFail("Your friend's game sent a move this game doesn't accept, so the online game has stopped.")
			return
		}
		if !a.live() {
			a.setView(len(a.hist))
		}
		a.playMove(mv)
		if a.G.Clock != nil && (m.CW > 0 || m.CB > 0) {
			a.G.Clock.W, a.G.Clock.B = m.CW, m.CB
		}
	case "ask":
		a.answerPeer(m.What)
	case "answer":
		a.gotAnswer(m.What, m.Yes)
	case "resign":
		if a.online() && a.G.Result == nil {
			loser := -a.youSide()
			a.G.Result = &ResultRec{Result: resultFor(loser), Reason: "resignation", Winner: -loser}
			a.gameOver()
		}
	case "flag":
		if a.online() && m.Res != nil && a.G.Result == nil {
			a.G.Result = m.Res
			a.gameOver()
		}
	case "bye":
		a.peerGone()
	}
}

func resultFor(loser int) string {
	if loser > 0 {
		return "0-1"
	}
	return "1-0"
}

// startOnlineGame is the host dealing out the colours.
func (a *App) startOnlineGame() {
	o := a.net
	youW := true
	switch a.sd.Setup.Color {
	case "black":
		youW = false
	case "random":
		youW = time.Now().UnixNano()%2 == 0
	}
	o.peer.send(Msg{T: "start", White: !youW, Clock: a.sd.Setup.Clock})
	a.beginOnline(youW, a.sd.Setup.Clock)
}

func (a *App) beginOnline(youWhite bool, clock string) {
	o := a.net
	o.state, o.pending = "playing", ""
	wn, bn := o.myName, o.peerName
	if !youWhite {
		wn, bn = bn, wn
	}
	a.newGameRec(&GameRec{Mode: "online", YouWhite: youWhite, WName: wn, BName: bn, Clock: clockFor(clock)})
}

// sendMove tells the friend about a move made here.
func (a *App) sendMove(uci string) {
	if !a.online() || a.net == nil {
		return
	}
	m := Msg{T: "move", UCI: uci, Ply: len(a.hist) - 1}
	if a.G.Clock != nil {
		m.CW, m.CB = a.G.Clock.W, a.G.Clock.B
	}
	a.net.peer.send(m)
}

// askPeer sends a question (draw, undo, rematch) and waits for the answer.
func (a *App) askPeer(what string) {
	o := a.net
	if o == nil || o.peer == nil || o.pending != "" {
		return
	}
	o.pending = what
	o.peer.send(Msg{T: "ask", What: what})
	a.ui.statusOverride = map[string]string{"draw": "Draw offered. Waiting for " + o.peerName + "…", "undo": "Asked to take back a move. Waiting for " + o.peerName + "…", "rematch": "Rematch offered. Waiting for " + o.peerName + "…"}[what]
	if what == "rematch" {
		a.ui.message("Rematch", "Waiting for "+o.peerName+" to answer…")
	}
}

func (a *App) answerPeer(what string) {
	o := a.net
	reply := func(yes bool) func() {
		return func() {
			if a.net == nil || a.net.peer == nil {
				return
			}
			a.net.peer.send(Msg{T: "answer", What: what, Yes: yes})
			if yes {
				a.applyAgreed(what, false)
			}
		}
	}
	switch what {
	case "draw":
		a.ui.ask("Draw offered", o.peerName+" offers a draw. Do you accept?", "Accept Draw", "Play On", reply(true), reply(false))
	case "undo":
		a.ui.ask("Take back?", o.peerName+" would like to take back their last move. Allow it?", "Allow", "No", reply(true), reply(false))
	case "rematch":
		a.ui.ask("Rematch?", o.peerName+" would like a rematch, with colours swapped.", "Play Again", "No Thanks", reply(true), reply(false))
	}
	play("hint")
}

func (a *App) gotAnswer(what string, yes bool) {
	o := a.net
	if o.pending != what {
		return
	}
	o.pending = ""
	a.ui.statusOverride = ""
	if !yes {
		a.ui.hide(ovMsg)
		a.ui.banner(o.peerName+" said no", "")
		return
	}
	a.applyAgreed(what, true)
}

// applyAgreed carries out something both players agreed to. asker is true
// on the side that asked.
func (a *App) applyAgreed(what string, asker bool) {
	switch what {
	case "draw":
		if a.G.Result == nil {
			a.G.Result = &ResultRec{Result: "1/2-1/2", Reason: "agreement"}
			a.gameOver()
		}
	case "undo":
		// take back the asker's last move (and the reply to it, if any)
		askerSide := a.youSide()
		if !asker {
			askerSide = -askerSide
		}
		n := 1
		if a.pos.Side == askerSide {
			n = 2
		}
		if n > len(a.hist) {
			n = len(a.hist)
		}
		a.takeBack(n)
	case "rematch":
		a.ui.hide(ovMsg)
		if a.net.host {
			a.sd.Setup.Color = "white"
			if a.G.YouWhite {
				a.sd.Setup.Color = "black"
			}
			if a.G.Clock != nil {
				a.sd.Setup.Clock = a.G.Clock.Base
			} else {
				a.sd.Setup.Clock = "none"
			}
			a.startOnlineGame()
		}
	}
}

// hostFlag: on the host, a flag decides the game for both players.
func (a *App) hostFlag(res *ResultRec) {
	if a.net != nil && a.net.peer != nil {
		a.net.peer.send(Msg{T: "flag", Res: res})
	}
}

func cleanName(s, def string) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) > 14 {
		s = string(r[:14])
	}
	if s == "" {
		return def
	}
	return s
}

var errNoNet = errors.New("not online")
