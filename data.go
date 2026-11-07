// The members of the Ravenscroft Library, their strength and what they say,
// plus the game's colours and texts.
//
// (c) 2026 Ironite
package main

import (
	"time"

	"github.com/adamglassner/chess/engine"
)

var version = "2.0.0" // set by build.sh from CHANGELOG.md

type folk struct {
	ID, Name, Full, Rating, Tag string
	Level                       int
}

var folks = []folk{
	{"pip", "Pip", "Pip the Page", "Beginner", "Fetches the books. Learning the moves, and very keen.", 1},
	{"hollis", "Mrs. Hollis", "Mrs. Hollis", "Casual", "The librarian. Plays a friendly game between returns.", 2},
	{"ambrose", "Brother Ambrose", "Brother Ambrose", "Club player", "Keeps the archive. Patient, solid, hard to rattle.", 3},
	{"whitlock", "Judge Whitlock", "Judge Whitlock", "Strong", "Retired from the bench. Punishes every loose piece.", 4},
	{"vance", "Professor Vance", "Professor Vance", "Expert", "Wrote three books on the endgame. Mentions it often.", 5},
	{"master", "The Old Master", "The Old Master", "Master", "Nobody remembers him losing. Nobody remembers him arriving.", 6},
}

func folkByID(id string) *folk {
	for i := range folks {
		if folks[i].ID == id {
			return &folks[i]
		}
	}
	return nil
}

type level struct {
	ai       engine.Options
	book     float64 // chance of playing from the opening book
	minDelay time.Duration
	maxDelay time.Duration
}

var levels = map[string]level{
	"pip":      {engine.Options{MaxDepth: 1, Noise: 140, Blunder: .18}, .5, 700 * time.Millisecond, 1500 * time.Millisecond},
	"hollis":   {engine.Options{MaxDepth: 2, Noise: 60, Blunder: .06}, .7, 700 * time.Millisecond, 1500 * time.Millisecond},
	"ambrose":  {engine.Options{MaxDepth: 3, Noise: 22, Blunder: .015}, .9, 600 * time.Millisecond, 1300 * time.Millisecond},
	"whitlock": {engine.Options{Time: 450 * time.Millisecond, MaxDepth: 5}, 1, 500 * time.Millisecond, 900 * time.Millisecond},
	"vance":    {engine.Options{Time: 1300 * time.Millisecond, MaxDepth: 9}, 1, 300 * time.Millisecond, 700 * time.Millisecond},
	"master":   {engine.Options{Time: 3000 * time.Millisecond}, 1, 200 * time.Millisecond, 500 * time.Millisecond},
}

var quips = map[string]map[string][]string{
	"pip":      {"greet": {"Oh! A game? I know how the horsey moves!", "I've been practising. A little bit."}, "check": {"Check! Did I say that right?", "Check! I think."}, "win": {"I won? I actually won!"}, "lose": {"Good game! Can we play again?"}, "draw": {"A draw! That's half a win, right?"}, "accept": {"Okay! A draw sounds nice."}, "decline": {"No thank you, I think I'm winning!"}},
	"hollis":   {"greet": {"Quietly now. Your move, dear.", "Just one game before closing."}, "check": {"Check, dear.", "Shh. Check."}, "win": {"Overdue, I'm afraid. Checkmate."}, "lose": {"Well played. I'll stamp that one."}, "draw": {"A draw. Shall we call it even?"}, "accept": {"A draw it is. Tea?"}, "decline": {"Not just yet, dear."}},
	"ambrose":  {"greet": {"Patience is the whole of it.", "Let us begin."}, "check": {"Check.", "Check, I believe."}, "win": {"Slow and steady."}, "lose": {"A fine game. I shall reflect on it."}, "draw": {"Peace, then."}, "accept": {"Peace. I accept."}, "decline": {"I think there is more to play."}},
	"whitlock": {"greet": {"Court is in session.", "Proceed. And mind your pieces."}, "check": {"Check. Order in the court.", "Check."}, "win": {"The verdict is checkmate."}, "lose": {"Overruled by a better player. Well done."}, "draw": {"A hung jury. Draw."}, "accept": {"Motion granted. Draw."}, "decline": {"Motion denied."}},
	"vance":    {"greet": {"Have you read chapter four? No matter.", "An interesting opening awaits."}, "check": {"Check. Textbook.", "Check. Footnote nine."}, "win": {"As I wrote in chapter seven: checkmate."}, "lose": {"Remarkable. I may need a fourth book."}, "draw": {"A theoretical draw, correctly played."}, "accept": {"Agreed. The position is level."}, "decline": {"Decline. The position is richer than that."}},
	"master":   {"greet": {"Sit. Play.", "Mm. Begin."}, "check": {"Check."}, "win": {"Again, whenever you are ready."}, "lose": {"...Well. It has been a long time."}, "draw": {"A draw. You have improved."}, "accept": {"Very well."}, "decline": {"No."}},
}

// colours (from the original style sheet)
var (
	cInk      = hexc("#2a1a0e")
	cAccent   = hexc("#6e2316")
	cGold     = hexc("#e2bd6a")
	cBarText  = hexc("#f0dcb4")
	cBtnInk   = hexc("#2a170a")
	cBtnEdge  = hexc("#6a4520")
	cCream    = hexc("#fbe9c4")
	cVer      = hexc("#e9d2a6")
	cAlert    = hexc("#ffd27a")
	cReview   = hexc("#bfe0ff")
	cBorderGd = hexc("#c9a24c")
)

var boardCoord = map[string]string{"walnut": "#e7c77c", "marble": "#d9c690", "club": "#e3c983"}

var boardNames = [][2]string{{"walnut", "Walnut & Maple"}, {"marble", "Marble"}, {"club", "Club Green"}}

// rules and credits text. "**x**" is bold; lines starting "# " are headings,
// "1. " numbered items, "- " bullets, "~ " small print.
const rulesText = `# How to Play
Each side starts with sixteen pieces. White moves first, then the players take turns. The goal is **checkmate**: attack the enemy king so that it has no way to escape.
1. **Moving.** Click a piece and then a square, or drag it there. Dots show where it can go; rings mark captures.
1. **The pieces.** The **king** moves one square any way. The **queen** slides any distance in any direction. The **rook** slides along ranks and files, the **bishop** along diagonals. The **knight** jumps in an L shape and can leap over pieces. A **pawn** steps forward one square (two on its first move) and captures one square diagonally forward.
1. **Check.** When a king is attacked it is in check, and its side must get out of check straight away.
1. **Castling.** Once per game, if the king and a rook haven't moved, the squares between are empty and the king isn't in, through or into check, the king moves two squares toward the rook and the rook hops over it. Drag the king two squares to castle.
1. **En passant.** If a pawn moves two squares and lands beside an enemy pawn, that pawn may capture it as if it had moved one square, but only on the very next move.
1. **Promotion.** A pawn that reaches the far side becomes a queen, rook, bishop or knight.
1. **Draws.** Stalemate (no legal move but not in check), the same position three times, fifty moves each without a capture or pawn move, too little material to mate, or a draw offer that's accepted.
**Play the Computer:** six members of the Ravenscroft Library will give you a game, from the page boy who is just learning to the Old Master. Who sits down changes from game to game, but each chair keeps its strength, and your record at each strength is kept.
**Same Computer:** two players share one computer. Type your names on the title screen, and the board turns to face whoever is moving.
**Online:** one player chooses Host a Game and tells their friend the address shown; the friend chooses Join a Game and types it in. Offer a draw, ask to take back a move, or ask for a rematch with colours swapped. Over the internet the host's router must forward TCP port 47025.
An unfinished game against the computer or on the same computer is waiting for you when you come back.
~ Keys: Ctrl+Z undo · H hint · F flip board · Left/Right step through the moves · N new game · M music · F11 full screen`

var creditRows = [][2]string{
	{"Developed & Published by", "Ironite"},
	{"Game Design", "Ironite"},
	{"Art & Illustration", "Ironite"},
	{"Chess Engine", "Ironite"},
	{"Music", "Ironite"},
	{"", `"The Quiet Library", "Nocturne of the Long Diagonal", "Minuet in the Reading Room", "Endgame by Lamplight"`},
	{"Sound Effects", "Ironite"},
	{"Programming", "Ironite"},
	{"Built with", "Go and Ebitengine"},
	{"Typeface", "Lora (SIL Open Font License)"},
}
