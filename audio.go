// Music and sound effects. The four chamber pieces and every sound are the
// game's original ones, recorded from its synthesiser (see tools/assets).
//
// (c) 2026 Ironite
package main

import (
	"bytes"
	"io"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

const sampleRate = 44100

var tuneNames = []string{"The Quiet Library", "Nocturne of the Long Diagonal", "Minuet in the Reading Room", "Endgame by Lamplight"}

type soundT struct {
	ctx      *audio.Context
	sfx      map[string][]byte
	music    *audio.Player
	idx      int
	gapUntil time.Time
	paused   bool
	ok       bool
	set      *Settings
	onTune   func(name string)
}

var snd soundT

func initAudio(set *Settings, onTune func(string)) {
	snd.set, snd.onTune = set, onTune
	defer func() {
		if r := recover(); r != nil {
			log.Printf("audio disabled: %v", r)
			snd.ok = false
		}
	}()
	snd.ctx = audio.NewContext(sampleRate)
	snd.sfx = map[string][]byte{}
	for _, n := range []string{"move", "move2", "move3", "move4", "capture", "castle", "check", "promote", "win", "lose", "draw", "click", "pick", "hint", "start", "illegal", "tick"} {
		b, err := assetFS.ReadFile("assets/audio/sfx_" + n + ".ogg")
		if err != nil {
			continue
		}
		st, err := vorbis.DecodeWithSampleRate(sampleRate, bytes.NewReader(b))
		if err != nil {
			log.Printf("sound %s: %v", n, err)
			continue
		}
		pcm, err := io.ReadAll(st)
		if err == nil {
			snd.sfx[n] = pcm
		}
	}
	snd.idx = rand.Intn(len(tuneNames))
	snd.ok = true
}

func musicVol() float64 { return math.Pow(snd.set.Vol/100, 1.6) }
func sfxVol() float64   { return math.Max(.35, math.Pow(snd.set.Vol/100, .6)) }

// updateMusic keeps the playlist going; call every frame.
func updateMusic() {
	if !snd.ok {
		return
	}
	if !snd.set.Music {
		if snd.music != nil && snd.music.IsPlaying() {
			snd.music.Pause()
			snd.paused = true
		}
		return
	}
	if snd.music != nil {
		snd.music.SetVolume(musicVol())
		if snd.music.IsPlaying() {
			return
		}
		if snd.paused { // switched back on: carry on where it left off
			snd.paused = false
			snd.music.Play()
			return
		}
		// finished: short pause, then the next piece
		_ = snd.music.Close()
		snd.music = nil
		snd.idx = (snd.idx + 1) % len(tuneNames)
		snd.gapUntil = time.Now().Add(1500 * time.Millisecond)
		return
	}
	if time.Now().Before(snd.gapUntil) {
		return
	}
	b, err := assetFS.ReadFile("assets/audio/music" + string(rune('1'+snd.idx)) + ".ogg")
	if err != nil {
		return
	}
	stm, err := vorbis.DecodeWithSampleRate(sampleRate, bytes.NewReader(b))
	if err != nil {
		log.Printf("music: %v", err)
		snd.ok = false
		return
	}
	p, err := snd.ctx.NewPlayer(stm)
	if err != nil {
		log.Printf("music: %v", err)
		snd.ok = false
		return
	}
	snd.music = p
	p.SetVolume(musicVol())
	p.Play()
	if snd.onTune != nil {
		snd.onTune(tuneNames[snd.idx])
	}
}

// play starts a sound effect.
func play(name string) {
	if !snd.ok || !snd.set.Sfx {
		return
	}
	if name == "move" {
		name = []string{"move", "move2", "move3", "move4"}[rand.Intn(4)]
	}
	pcm := snd.sfx[name]
	if pcm == nil {
		return
	}
	p := snd.ctx.NewPlayerFromBytes(pcm)
	p.SetVolume(sfxVol())
	p.Play()
}

// playLater plays a sound after a short delay (e.g. the check bell after the move).
func playLater(name string, d time.Duration) {
	pendingSounds = append(pendingSounds, pendingSound{name, time.Now().Add(d)})
}

type pendingSound struct {
	name string
	at   time.Time
}

var pendingSounds []pendingSound

func updateSounds() {
	now := time.Now()
	keep := pendingSounds[:0]
	for _, p := range pendingSounds {
		if now.After(p.at) {
			play(p.name)
		} else {
			keep = append(keep, p)
		}
	}
	pendingSounds = keep
}
