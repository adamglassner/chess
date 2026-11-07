// Interface state that the game logic talks to: which panel is open, timed
// actions, questions, messages, speech bubbles, banners and confetti.
//
// (c) 2026 Ironite
package main

import (
	"math"
	"math/rand"
	"time"
)

type ovKind int

const (
	ovNone ovKind = iota
	ovTitle
	ovOver
	ovConfirm
	ovSettings
	ovRules
	ovCredits
	ovOnline
	ovMsg
)

type timerT struct {
	at time.Time
	fn func()
}

type bubbleT struct {
	text string
	side int
	at   time.Time
}

type particle struct {
	x, y, vx, vy, rot, vr, w, h float64
	clr                         [3]uint8
	born                        time.Time
	life                        float64
}

type uiState struct {
	stack           []ovKind
	timers          []timerT
	statusOverride  string
	scrollToCurrent bool

	over struct{ face, kick, title, text, quote string }
	askT struct {
		title, text, yes, no string
		onYes, onNo          func()
	}
	msgT struct{ title, text string }

	bubbles   []bubbleT
	particles []particle
	toast     string
	toastAt   time.Time
	bannerT   string
	bannerSub string
	bannerAt  time.Time

	sheetScroll  float64
	panelScroll  float64
	focus        string // text field being typed in
	onlinePanel  string // host, join
	dragVol      bool
	titleMessage string
}

func (u *uiState) after(d time.Duration, fn func()) {
	u.timers = append(u.timers, timerT{time.Now().Add(d), fn})
}

func (u *uiState) runTimers() {
	now := time.Now()
	var due []func()
	keep := u.timers[:0]
	for _, t := range u.timers {
		if now.After(t.at) {
			due = append(due, t.fn)
		} else {
			keep = append(keep, t)
		}
	}
	u.timers = keep
	for _, fn := range due {
		fn()
	}
}

func (u *uiState) top() ovKind {
	if len(u.stack) == 0 {
		return ovNone
	}
	return u.stack[len(u.stack)-1]
}

func (u *uiState) show(k ovKind) {
	u.hide(k)
	u.stack = append(u.stack, k)
	u.panelScroll = 0
	u.focus = ""
}

// showUnder opens k beneath any question or message already showing.
func (u *uiState) showUnder(k ovKind) {
	t := u.top()
	if t != ovConfirm && t != ovMsg {
		u.show(k)
		return
	}
	u.hide(k)
	n := len(u.stack)
	u.stack = append(u.stack[:n-1], k, t)
}

func (u *uiState) hide(k ovKind) {
	out := u.stack[:0]
	for _, s := range u.stack {
		if s != k {
			out = append(out, s)
		}
	}
	u.stack = out
	u.focus = ""
}

func (u *uiState) hideAll() { u.stack = u.stack[:0]; u.focus = "" }

func (u *uiState) visible(k ovKind) bool {
	for _, s := range u.stack {
		if s == k {
			return true
		}
	}
	return false
}

// paused: the clocks stop while the title or a question is up (not online).
func (u *uiState) paused() bool { return u.visible(ovTitle) || u.visible(ovConfirm) }

func (u *uiState) message(title, text string) {
	u.msgT.title, u.msgT.text = title, text
	u.show(ovMsg)
}

func (u *uiState) ask(title, text, yes, no string, onYes, onNo func()) {
	u.askT.title, u.askT.text, u.askT.yes, u.askT.no = title, text, yes, no
	u.askT.onYes, u.askT.onNo = onYes, onNo
	u.show(ovConfirm)
}

func (u *uiState) bubble(text string, side int) {
	u.bubbles = []bubbleT{{text, side, time.Now()}}
}

func (u *uiState) banner(t, sub string) {
	u.bannerT, u.bannerSub, u.bannerAt = t, sub, time.Now()
}

func (u *uiState) nowPlaying(name string) {
	u.toast, u.toastAt = "♫ "+name, time.Now()
}

// confetti bursts from the middle of the board.
func (u *uiState) confetti() {
	cols := [][3]uint8{{0xe2, 0xbd, 0x6a}, {0xf6, 0xea, 0xd0}, {0x6e, 0x23, 0x16}, {0x2f, 0x5c, 0x3a}, {0xff, 0xf3, 0xc4}}
	now := time.Now()
	cx, cy := boardX+400.0, boardY+360.0
	for i := 0; i < 140; i++ {
		a := rand.Float64() * 2 * math.Pi
		sp := 200 + rand.Float64()*520
		u.particles = append(u.particles, particle{
			x: cx, y: cy, vx: math.Cos(a) * sp, vy: math.Sin(a)*sp - 260,
			rot: rand.Float64() * 6, vr: (rand.Float64() - .5) * 14,
			w: 6 + rand.Float64()*7, h: 3 + rand.Float64()*4,
			clr: cols[rand.Intn(len(cols))], born: now, life: 1.8 + rand.Float64()*.9,
		})
	}
}

func (u *uiState) stepParticles(dt float64) {
	now := time.Now()
	keep := u.particles[:0]
	for _, p := range u.particles {
		if now.Sub(p.born).Seconds() > p.life {
			continue
		}
		p.vy += 900 * dt
		p.vx *= math.Pow(.4, dt)
		p.vy *= math.Pow(.7, dt)
		p.x += p.vx * dt
		p.y += p.vy * dt
		p.rot += p.vr * dt
		keep = append(keep, p)
	}
	u.particles = keep
}
