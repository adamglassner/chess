# Chess: Ravenscroft Library

![Chess](art/hero.jpg)

Chess at a lamplit library desk, by Ironite. Written in Go with
[Ebitengine](https://ebitengine.org). Play six members of the library from
beginner to master, share one computer with a friend, or play a friend
online. It ships on its own and inside the Ironite Launcher.

![Game](art/screens/game.png)

## How to play

- Click a piece and then a square, or drag it. Dots show legal moves and rings show captures. Drag the king two squares to castle; a picker comes up when a pawn promotes.
- Full rules: castling, en passant, promotion, check, checkmate and stalemate, plus draws by threefold repetition, the fifty-move rule and insufficient material.
- Undo, Hint, Flip Board, Offer Draw and Resign; optional clocks (5 + 3, 10 minutes, 15 + 10, 30 minutes). Right-click marks squares, right-drag draws arrows.
- The score sheet names the opening, steps back through the game and copies it as PGN.
- Keys: `Ctrl+Z` undo, `H` hint, `F` flip, `N` new game, `M` music, `Left`/`Right` step through the moves, `F11` full screen.

### Ways to play

- **Play the Computer**: six strengths, from Beginner to Master. Each chair has its own portrait, and the person sitting in it gets a new name every game (the page boy, Mrs. ..., Brother ..., Judge ..., Professor ..., the Old Master ...). Your record is kept per strength.
- **Same Computer**: two players share one PC, with their names on the plaques. The board turns to face whoever is moving (can be switched off in Options).
- **Online**: one player hosts, the friend joins by IP address (TCP port 47025), on a home network or over the internet. The host's game is the referee: it deals the colours and keeps the clocks, and both games check every move. Offer draws, ask to take back a move, and ask for a rematch with colours swapped. If a friend leaves, the game stops and you can review it.

Your unfinished game, record and settings are saved in
`%APPDATA%\Ironite\Chess\save.json` (the same file as 1.x), outside the
install folder.

## Code

| File | What it is |
|---|---|
| `main.go` | Window setup, the frame loop, mouse, keyboard and text entry. |
| `game.go` | The game: moves, the computer's turn, clocks, undo, hints, draw offers, results and records. |
| `net.go` | Online play: host/guest protocol (JSON lines over TCP). |
| `names.go` | Random names for the computer opponents, matched to each portrait. |
| `ui.go`, `uistate.go` | The desk, board, side panel, toolbar and every screen. |
| `gfx.go` | Drawing at the screen's real resolution: text, panels, shapes. |
| `engine/` | Rules (0x88 move generator), the computer player (alpha-beta search) and the opening book, with tests. |
| `audio.go`, `save.go`, `assets.go`, `data.go` | Music and sound, the save file, embedded assets, opponents and texts. |
| `clip_windows.go` | Clipboard, for copying PGN and pasting an address. |
| `assets/` | Pieces, boards, the desk, portraits, fonts, music and sound effects (the same art and audio as 1.x, pre-rendered up to 4K). |
| `tools/mkres/` | Writes the Windows icon and version info into the .exe. |
| `installer/` | Inno Setup scripts for the standalone `Chess-Setup-v<version>.exe`. |

`autotest.go` (`-tags autotest`) walks through every screen and saves
screenshots into `$CHESS_SHOTS`. `net_test.go` runs a host and a guest over
real TCP: full games, rematches, take-backs, draw offers, resigning, clocks,
a friend leaving and a third player being turned away, plus a full game
against the computer and resuming a saved game.

## Building

You need Go 1.24 or newer and bash (Git Bash works on Windows). No C
compiler is needed.

```bash
./build.sh
```

This writes `dist/Games/Chess/Chess.exe` along with its README, CREDITS,
CHANGELOG and license files. The version comes from the top entry of
`CHANGELOG.md`.

To build the installer, install Inno Setup 6.3 or newer, run `./build.sh`, then:

```
ISCC.exe installer\ChessSetup.iss
```

The setup adds a Windows Firewall rule so hosting online games works without a prompt.

To run the tests (on Windows, or anywhere for the engine):

```bash
(cd engine && go test ./...)
go test .
```

## Releasing

Add a new version section at the top of `CHANGELOG.md` and push to `main`.
The release workflow builds the game and publishes `Chess-<version>-windows.zip`
as a GitHub release, which Ironite Launcher installs and updates from.

## License

Copyright (c) 2026 Ironite. All rights reserved. Lora is under the SIL Open Font License (`assets/fonts/OFL.txt`).
