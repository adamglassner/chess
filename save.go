// The save file: %APPDATA%\Ironite\Chess\save.json, outside the install
// folder. The layout is the same as Chess 1.x, so an unfinished game, the
// record against each opponent and the settings carry over.
//
// (c) 2026 Ironite
package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

type Settings struct {
	Music bool    `json:"music"`
	Sfx   bool    `json:"sfx"`
	Vol   float64 `json:"vol"`
	Board string  `json:"board"` // walnut, marble, club
	Dots  string  `json:"dots"`  // on, off
	Anim  string  `json:"anim"`  // slow, normal, fast
	Promo string  `json:"promo"` // ask, queen
	Turn  string  `json:"turn"`  // on, off (two players: turn the board)
	// names remembered for Play Together
	Name string `json:"name"` // your name online
	Host string `json:"host"` // last address joined
	P1   string `json:"p1"`   // same computer: White's name
	P2   string `json:"p2"`   // same computer: Black's name
}

type Setup struct {
	Mode  string `json:"mode"`  // ai, pvp (same computer), online
	Opp   string `json:"opp"`   // opponent id
	Color string `json:"color"` // white, random, black
	Clock string `json:"clock"` // none, 5+3, 10, 15+10, 30
}

type ClockRec struct {
	W    float64 `json:"w"` // milliseconds left
	B    float64 `json:"b"`
	Inc  float64 `json:"inc"`
	Base string  `json:"base"`
}

type ResultRec struct {
	Result string `json:"result"` // 1-0, 0-1, 1/2-1/2
	Reason string `json:"reason"`
	Winner int    `json:"winner"` // 1 white, -1 black, 0 draw
}

type GameRec struct {
	StartFen string     `json:"startFen"`
	Moves    []string   `json:"moves"`
	Mode     string     `json:"mode"`
	Opp      string     `json:"opp"`
	YouWhite bool       `json:"youWhite"`
	Result   *ResultRec `json:"result"`
	Counted  bool       `json:"counted"`
	Clock    *ClockRec  `json:"clock"`
	Started  int64      `json:"started"`
	Flipped  bool       `json:"flipped"`
	OppShort string     `json:"oppShort,omitempty"` // the computer's name this game
	OppFull  string     `json:"oppFull,omitempty"`
	WName    string     `json:"wName,omitempty"` // two players: names
	BName    string     `json:"bName,omitempty"`
}

type Record struct {
	W int `json:"w"`
	D int `json:"d"`
	L int `json:"l"`
}

type Stats struct {
	Won    int                `json:"won"`
	Drawn  int                `json:"drawn"`
	Lost   int                `json:"lost"`
	Pvp    int                `json:"pvp"`
	Online Record             `json:"online"`
	Per    map[string]*Record `json:"per"`
}

type SaveData struct {
	Settings Settings `json:"settings"`
	Setup    Setup    `json:"setup"`
	Game     *GameRec `json:"game"`
	Stats    Stats    `json:"stats"`
}

func defaultSave() SaveData {
	return SaveData{
		Settings: Settings{Music: true, Sfx: true, Vol: 60, Board: "walnut", Dots: "on", Anim: "normal", Promo: "ask", Turn: "on", P1: "Player 1", P2: "Player 2"},
		Setup:    Setup{Mode: "ai", Opp: "pip", Color: "white", Clock: "none"},
		Stats:    Stats{Per: map[string]*Record{}},
	}
}

func saveDir() string {
	d, err := os.UserConfigDir()
	if err != nil {
		d = os.TempDir()
	}
	return filepath.Join(d, "Ironite", "Chess")
}

func loadSave() SaveData {
	sd := defaultSave()
	b, err := os.ReadFile(filepath.Join(saveDir(), "save.json"))
	if err == nil {
		if err := json.Unmarshal(b, &sd); err != nil {
			log.Printf("save file unreadable, starting fresh: %v", err)
			sd = defaultSave()
		}
	}
	if sd.Stats.Per == nil {
		sd.Stats.Per = map[string]*Record{}
	}
	if sd.Settings.Vol < 0 || sd.Settings.Vol > 100 {
		sd.Settings.Vol = 60
	}
	if _, ok := boardCoord[sd.Settings.Board]; !ok {
		sd.Settings.Board = "walnut"
	}
	if folkByID(sd.Setup.Opp) == nil {
		sd.Setup.Opp = "pip"
	}
	if sd.Game != nil && sd.Game.Moves == nil {
		sd.Game.Moves = []string{}
	}
	if sd.Game != nil && sd.Game.Mode == "online" {
		sd.Game = nil // online games can't be picked up again later
	}
	if sd.Setup.Mode != "ai" && sd.Setup.Mode != "pvp" && sd.Setup.Mode != "online" {
		sd.Setup.Mode = "ai"
	}
	return sd
}

func writeSave(sd *SaveData) {
	b, err := json.Marshal(sd)
	if err != nil {
		return
	}
	dir := saveDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("save: %v", err)
		return
	}
	p := filepath.Join(dir, "save.json")
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		log.Printf("save: %v", err)
		return
	}
	if err := os.Rename(tmp, p); err != nil {
		log.Printf("save: %v", err)
	}
}
