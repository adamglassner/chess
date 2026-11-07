// Package engine holds the rules of chess and the computer opponent.
//
// The board is 0x88: square = rank*16 + file (rank 0 is White's first rank).
// Pieces: 1 pawn, 2 knight, 3 bishop, 4 rook, 5 queen, 6 king; White
// positive, Black negative.
//
// (c) 2026 Ironite
package engine

import (
	"errors"
	"strconv"
	"strings"
)

const (
	Pawn   = 1
	Knight = 2
	Bishop = 3
	Rook   = 4
	Queen  = 5
	King   = 6

	White = 1
	Black = -1
)

// Move flags.
const (
	FlagCapture   = 1
	FlagEnPassant = 2
	FlagCastle    = 4
	FlagDouble    = 8
	FlagPromo     = 16
)

// StartFEN is the normal starting position.
const StartFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

const letters = " PNBRQK"

var (
	knightOff = [8]int{33, 31, 18, 14, -33, -31, -18, -14}
	bishopOff = [4]int{17, 15, -17, -15}
	rookOff   = [4]int{16, 1, -16, -1}
	kingOff   = [8]int{17, 15, -17, -15, 16, 1, -16, -1}
)

// Move is one move. Piece, Captured and Promo carry their colour sign.
type Move struct {
	From, To int
	Piece    int8
	Captured int8
	Promo    int8
	Flags    uint8
	score    int32
}

// SquareName gives "e4" for a 0x88 square.
func SquareName(s int) string {
	return string([]byte{"abcdefgh"[s&7], byte('1' + s>>4)})
}

// ParseSquare reads "e4"; -1 if it isn't a square.
func ParseSquare(n string) int {
	if len(n) != 2 || n[0] < 'a' || n[0] > 'h' || n[1] < '1' || n[1] > '8' {
		return -1
	}
	return int(n[1]-'1')*16 + int(n[0]-'a')
}

// UCI writes a move as e2e4 / e7e8q.
func (m Move) UCI() string {
	s := SquareName(m.From) + SquareName(m.To)
	if m.Promo != 0 {
		s += string("nbrq"[abs8(m.Promo)-2])
	}
	return s
}

// Same reports whether two moves go between the same squares (and promote alike).
func (m Move) Same(o Move) bool {
	return m.From == o.From && m.To == o.To && abs8(m.Promo) == abs8(o.Promo)
}

func abs8(x int8) int {
	if x < 0 {
		return -int(x)
	}
	return int(x)
}

// Zobrist keys.
var (
	zPiece  [13 * 128]uint64
	zCastle [16]uint64
	zEP     [8]uint64
	zSide   uint64
)

func init() {
	seed := uint64(0x9e3779b97f4a7c15)
	rnd := func() uint64 {
		seed ^= seed << 13
		seed ^= seed >> 7
		seed ^= seed << 17
		return seed
	}
	for i := range zPiece {
		zPiece[i] = rnd()
	}
	for i := range zCastle {
		zCastle[i] = rnd()
	}
	for i := range zEP {
		zEP[i] = rnd()
	}
	zSide = rnd()
}

func zi(p int8, s int) int { return (int(p)+6)*128 + s }

var castleMask [128]uint8

func init() {
	for i := range castleMask {
		castleMask[i] = 15
	}
	castleMask[0x04] = 15 &^ 3
	castleMask[0x00] = 15 &^ 2
	castleMask[0x07] = 15 &^ 1
	castleMask[0x74] = 15 &^ 12
	castleMask[0x70] = 15 &^ 8
	castleMask[0x77] = 15 &^ 4
}

type undo struct {
	m      Move
	null   bool
	castle uint8
	ep     int
	half   int
	hash   uint64
}

// Position is a chess position with enough history to undo moves and spot repetitions.
type Position struct {
	Board  [128]int8
	Side   int // White or Black
	Castle uint8
	EP     int // en-passant target square, or -1
	Half   int // half-moves since the last capture or pawn move
	Full   int
	Kings  [2]int // [white, black]
	Hash   uint64
	stack  []undo
	keys   []uint64
}

// NewPosition sets up a position from FEN ("" = the starting position).
func NewPosition(fen string) (*Position, error) {
	p := &Position{}
	if fen == "" {
		fen = StartFEN
	}
	return p, p.Load(fen)
}

// MustPosition is NewPosition for FENs known to be good.
func MustPosition(fen string) *Position {
	p, err := NewPosition(fen)
	if err != nil {
		panic(err)
	}
	return p
}

// Load replaces the position with the one in a FEN string.
func (p *Position) Load(fen string) error {
	parts := strings.Fields(fen)
	if len(parts) < 4 {
		return errors.New("bad FEN")
	}
	*p = Position{EP: -1}
	r, f := 7, 0
	for _, ch := range parts[0] {
		switch {
		case ch == '/':
			r--
			f = 0
		case ch >= '1' && ch <= '8':
			f += int(ch - '0')
		default:
			t := strings.IndexRune(letters, toUpper(ch))
			if t < 1 || r < 0 || f > 7 {
				return errors.New("bad FEN piece")
			}
			v := int8(t)
			if ch >= 'a' {
				v = -v
			}
			p.Board[r*16+f] = v
			f++
		}
	}
	p.Side = White
	if parts[1] == "b" {
		p.Side = Black
	}
	for _, c := range parts[2] {
		switch c {
		case 'K':
			p.Castle |= 1
		case 'Q':
			p.Castle |= 2
		case 'k':
			p.Castle |= 4
		case 'q':
			p.Castle |= 8
		}
	}
	p.EP = -1
	if parts[3] != "-" {
		p.EP = ParseSquare(parts[3])
	}
	if len(parts) > 4 {
		p.Half, _ = strconv.Atoi(parts[4])
	}
	p.Full = 1
	if len(parts) > 5 {
		p.Full, _ = strconv.Atoi(parts[5])
	}
	for s := 0; s < 128; s++ {
		if s&0x88 != 0 {
			s += 7
			continue
		}
		switch p.Board[s] {
		case King:
			p.Kings[0] = s
		case -King:
			p.Kings[1] = s
		}
	}
	p.computeHash()
	p.keys = []uint64{p.Hash}
	return nil
}

func toUpper(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - 32
	}
	return r
}

func (p *Position) computeHash() {
	var h uint64
	for s := 0; s < 128; s++ {
		if s&0x88 != 0 {
			s += 7
			continue
		}
		if v := p.Board[s]; v != 0 {
			h ^= zPiece[zi(v, s)]
		}
	}
	h ^= zCastle[p.Castle]
	if p.EP >= 0 {
		h ^= zEP[p.EP&7]
	}
	if p.Side == Black {
		h ^= zSide
	}
	p.Hash = h
}

// Clone copies the position (including its history).
func (p *Position) Clone() *Position {
	q := *p
	q.stack = append([]undo(nil), p.stack...)
	q.keys = append([]uint64(nil), p.keys...)
	return &q
}

// FEN writes the position as FEN.
func (p *Position) FEN() string {
	var b strings.Builder
	for r := 7; r >= 0; r-- {
		e := 0
		for f := 0; f < 8; f++ {
			v := p.Board[r*16+f]
			if v == 0 {
				e++
				continue
			}
			if e > 0 {
				b.WriteByte(byte('0' + e))
				e = 0
			}
			c := letters[abs8(v)]
			if v < 0 {
				c += 32
			}
			b.WriteByte(c)
		}
		if e > 0 {
			b.WriteByte(byte('0' + e))
		}
		if r > 0 {
			b.WriteByte('/')
		}
	}
	side := "w"
	if p.Side == Black {
		side = "b"
	}
	c := ""
	if p.Castle&1 != 0 {
		c += "K"
	}
	if p.Castle&2 != 0 {
		c += "Q"
	}
	if p.Castle&4 != 0 {
		c += "k"
	}
	if p.Castle&8 != 0 {
		c += "q"
	}
	if c == "" {
		c = "-"
	}
	ep := "-"
	if p.EP >= 0 {
		ep = SquareName(p.EP)
	}
	return b.String() + " " + side + " " + c + " " + ep + " " + strconv.Itoa(p.Half) + " " + strconv.Itoa(p.Full)
}

func kingIdx(side int) int {
	if side == White {
		return 0
	}
	return 1
}

// Attacked reports whether square sq is attacked by side `by`.
func (p *Position) Attacked(sq, by int) bool {
	b := &p.Board
	pd := 16
	if by == White {
		pd = -16
	}
	for _, d := range [2]int{pd - 1, pd + 1} {
		s := sq + d
		if s&0x88 == 0 && int(b[s]) == Pawn*by {
			return true
		}
	}
	for _, d := range knightOff {
		s := sq + d
		if s&0x88 == 0 && int(b[s]) == Knight*by {
			return true
		}
	}
	for _, d := range kingOff {
		s := sq + d
		if s&0x88 == 0 && int(b[s]) == King*by {
			return true
		}
	}
	for _, d := range bishopOff {
		for s := sq + d; s&0x88 == 0; s += d {
			if v := int(b[s]); v != 0 {
				if v == Bishop*by || v == Queen*by {
					return true
				}
				break
			}
		}
	}
	for _, d := range rookOff {
		for s := sq + d; s&0x88 == 0; s += d {
			if v := int(b[s]); v != 0 {
				if v == Rook*by || v == Queen*by {
					return true
				}
				break
			}
		}
	}
	return false
}

// InCheck reports whether the side to move is in check.
func (p *Position) InCheck() bool { return p.SideInCheck(p.Side) }

// SideInCheck reports whether `side`'s king is attacked.
func (p *Position) SideInCheck(side int) bool {
	return p.Attacked(p.Kings[kingIdx(side)], -side)
}

// Pseudo appends pseudo-legal moves (captures and promotions only when capturesOnly).
func (p *Position) Pseudo(capturesOnly bool, moves []Move) []Move {
	b := &p.Board
	us := p.Side
	add := func(from, to int, piece, captured int8, flags uint8, promo int8) {
		moves = append(moves, Move{From: from, To: to, Piece: piece, Captured: captured, Flags: flags, Promo: promo})
	}
	for s := 0; s < 128; s++ {
		if s&0x88 != 0 {
			s += 7
			continue
		}
		v := b[s]
		if v == 0 || (v > 0) != (us > 0) {
			continue
		}
		t := int(v) * us
		switch t {
		case Pawn:
			fwd := 16 * us
			startRank, lastRank := 1, 7
			if us == Black {
				startRank, lastRank = 6, 0
			}
			one := s + fwd
			if one&0x88 == 0 && b[one] == 0 {
				if one>>4 == lastRank {
					for _, pr := range [4]int{Queen, Rook, Bishop, Knight} {
						add(s, one, v, 0, FlagPromo, int8(pr*us))
					}
				} else if !capturesOnly {
					add(s, one, v, 0, 0, 0)
					two := one + fwd
					if s>>4 == startRank && b[two] == 0 {
						add(s, two, v, 0, FlagDouble, 0)
					}
				}
			}
			for _, d := range [2]int{fwd - 1, fwd + 1} {
				to := s + d
				if to&0x88 != 0 {
					continue
				}
				c := b[to]
				if c != 0 && (c > 0) != (us > 0) {
					if to>>4 == lastRank {
						for _, pr := range [4]int{Queen, Rook, Bishop, Knight} {
							add(s, to, v, c, FlagCapture|FlagPromo, int8(pr*us))
						}
					} else {
						add(s, to, v, c, FlagCapture, 0)
					}
				} else if to == p.EP {
					add(s, to, v, int8(-us*Pawn), FlagCapture|FlagEnPassant, 0)
				}
			}
		case Knight, King:
			offs := knightOff[:]
			if t == King {
				offs = kingOff[:]
			}
			for _, d := range offs {
				to := s + d
				if to&0x88 != 0 {
					continue
				}
				c := b[to]
				if c == 0 {
					if !capturesOnly {
						add(s, to, v, 0, 0, 0)
					}
				} else if (c > 0) != (us > 0) {
					add(s, to, v, c, FlagCapture, 0)
				}
			}
			if t == King && !capturesOnly {
				home := 0x04
				kBit, qBit := uint8(1), uint8(2)
				if us == Black {
					home, kBit, qBit = 0x74, 4, 8
				}
				if s == home {
					if p.Castle&kBit != 0 && b[s+1] == 0 && b[s+2] == 0 && int(b[s+3]) == Rook*us &&
						!p.Attacked(s, -us) && !p.Attacked(s+1, -us) && !p.Attacked(s+2, -us) {
						add(s, s+2, v, 0, FlagCastle, 0)
					}
					if p.Castle&qBit != 0 && b[s-1] == 0 && b[s-2] == 0 && b[s-3] == 0 && int(b[s-4]) == Rook*us &&
						!p.Attacked(s, -us) && !p.Attacked(s-1, -us) && !p.Attacked(s-2, -us) {
						add(s, s-2, v, 0, FlagCastle, 0)
					}
				}
			}
		default:
			var dirs []int
			switch t {
			case Bishop:
				dirs = bishopOff[:]
			case Rook:
				dirs = rookOff[:]
			default:
				dirs = kingOff[:]
			}
			for _, d := range dirs {
				for to := s + d; to&0x88 == 0; to += d {
					c := b[to]
					if c == 0 {
						if !capturesOnly {
							add(s, to, v, 0, 0, 0)
						}
						continue
					}
					if (c > 0) != (us > 0) {
						add(s, to, v, c, FlagCapture, 0)
					}
					break
				}
			}
		}
	}
	return moves
}

// Make plays a (pseudo-legal) move.
func (p *Position) Make(m Move) {
	b := &p.Board
	us := p.Side
	p.stack = append(p.stack, undo{m: m, castle: p.Castle, ep: p.EP, half: p.Half, hash: p.Hash})
	h := p.Hash ^ zCastle[p.Castle]
	if p.EP >= 0 {
		h ^= zEP[p.EP&7]
	}
	h ^= zPiece[zi(m.Piece, m.From)]
	b[m.From] = 0
	if m.Flags&FlagEnPassant != 0 {
		cs := m.To - 16*us
		h ^= zPiece[zi(b[cs], cs)]
		b[cs] = 0
	} else if m.Captured != 0 {
		h ^= zPiece[zi(m.Captured, m.To)]
	}
	placed := m.Piece
	if m.Promo != 0 {
		placed = m.Promo
	}
	b[m.To] = placed
	h ^= zPiece[zi(placed, m.To)]
	if m.Flags&FlagCastle != 0 {
		rf, rt := m.From-4, m.From-1
		if m.To > m.From {
			rf, rt = m.From+3, m.From+1
		}
		rook := b[rf]
		b[rf] = 0
		b[rt] = rook
		h ^= zPiece[zi(rook, rf)] ^ zPiece[zi(rook, rt)]
	}
	if int(m.Piece)*us == King {
		p.Kings[kingIdx(us)] = m.To
	}
	p.Castle &= castleMask[m.From] & castleMask[m.To]
	p.EP = -1
	if m.Flags&FlagDouble != 0 {
		p.EP = m.From + 16*us
	}
	if int(m.Piece)*us == Pawn || m.Captured != 0 {
		p.Half = 0
	} else {
		p.Half++
	}
	if us == Black {
		p.Full++
	}
	p.Side = -us
	h ^= zCastle[p.Castle] ^ zSide
	if p.EP >= 0 {
		h ^= zEP[p.EP&7]
	}
	p.Hash = h
	p.keys = append(p.keys, h)
}

// Unmake takes back the last move made with Make.
func (p *Position) Unmake() Move {
	st := p.stack[len(p.stack)-1]
	p.stack = p.stack[:len(p.stack)-1]
	p.keys = p.keys[:len(p.keys)-1]
	m := st.m
	b := &p.Board
	p.Side = -p.Side
	us := p.Side
	if us == Black {
		p.Full--
	}
	b[m.From] = m.Piece
	if m.Flags&FlagEnPassant != 0 {
		b[m.To] = 0
		b[m.To-16*us] = m.Captured
	} else {
		b[m.To] = m.Captured
	}
	if m.Flags&FlagCastle != 0 {
		rf, rt := m.From-4, m.From-1
		if m.To > m.From {
			rf, rt = m.From+3, m.From+1
		}
		b[rf] = b[rt]
		b[rt] = 0
	}
	if int(m.Piece)*us == King {
		p.Kings[kingIdx(us)] = m.From
	}
	p.Castle, p.EP, p.Half, p.Hash = st.castle, st.ep, st.half, st.hash
	return m
}

func (p *Position) makeNull() {
	p.stack = append(p.stack, undo{null: true, castle: p.Castle, ep: p.EP, half: p.Half, hash: p.Hash})
	h := p.Hash ^ zSide
	if p.EP >= 0 {
		h ^= zEP[p.EP&7]
	}
	p.EP = -1
	p.Side = -p.Side
	p.Half++
	p.Hash = h
	p.keys = append(p.keys, h)
}

func (p *Position) unmakeNull() {
	st := p.stack[len(p.stack)-1]
	p.stack = p.stack[:len(p.stack)-1]
	p.keys = p.keys[:len(p.keys)-1]
	p.Side = -p.Side
	p.Castle, p.EP, p.Half, p.Hash = st.castle, st.ep, st.half, st.hash
}

// Legal returns every legal move for the side to move.
func (p *Position) Legal() []Move {
	us := p.Side
	var out []Move
	for _, m := range p.Pseudo(false, make([]Move, 0, 64)) {
		p.Make(m)
		if !p.Attacked(p.Kings[kingIdx(us)], -us) {
			out = append(out, m)
		}
		p.Unmake()
	}
	return out
}

// FindUCI finds the legal move written as UCI, if there is one.
func (p *Position) FindUCI(u string) (Move, bool) {
	for _, m := range p.Legal() {
		if m.UCI() == u {
			return m, true
		}
	}
	return Move{}, false
}

// Repetitions counts how often the current position has occurred.
func (p *Position) Repetitions() int {
	k := p.keys[len(p.keys)-1]
	n := 0
	for i := len(p.keys) - 1; i >= 0; i -= 2 {
		if p.keys[i] == k {
			n++
		}
	}
	return n
}

// Insufficient reports a dead draw by material (K v K, K+minor v K, bishops all on one colour).
func (p *Position) Insufficient() bool {
	type minor struct{ t, color int }
	var all []minor
	for s := 0; s < 128; s++ {
		if s&0x88 != 0 {
			s += 7
			continue
		}
		v := p.Board[s]
		t := abs8(v)
		if v == 0 || t == King {
			continue
		}
		if t == Pawn || t == Rook || t == Queen {
			return false
		}
		all = append(all, minor{t, ((s >> 4) + (s & 7)) & 1})
	}
	if len(all) <= 1 {
		return true
	}
	for _, m := range all {
		if m.t != Bishop || m.color != all[0].color {
			return false
		}
	}
	return true
}

// Status says whether the game is over for the side to move.
type Status struct {
	Over   bool
	Result string // "1-0", "0-1", "1/2-1/2"
	Reason string // "checkmate", "stalemate", ...
	Winner int    // White, Black or 0
	Moves  []Move // legal moves when not over
}

// Status works out checkmate, stalemate and the automatic draws.
func (p *Position) Status() Status {
	moves := p.Legal()
	if len(moves) == 0 {
		if p.InCheck() {
			if p.Side == White {
				return Status{Over: true, Result: "0-1", Reason: "checkmate", Winner: Black}
			}
			return Status{Over: true, Result: "1-0", Reason: "checkmate", Winner: White}
		}
		return Status{Over: true, Result: "1/2-1/2", Reason: "stalemate"}
	}
	if p.Insufficient() {
		return Status{Over: true, Result: "1/2-1/2", Reason: "insufficient material"}
	}
	if p.Half >= 100 {
		return Status{Over: true, Result: "1/2-1/2", Reason: "fifty-move rule"}
	}
	if p.Repetitions() >= 3 {
		return Status{Over: true, Result: "1/2-1/2", Reason: "threefold repetition"}
	}
	return Status{Moves: moves}
}

// SAN writes a legal move in standard algebraic notation (pass the legal
// moves when you already have them).
func (p *Position) SAN(m Move, legal []Move) string {
	t := abs8(m.Piece)
	var s string
	if m.Flags&FlagCastle != 0 {
		s = "O-O-O"
		if m.To > m.From {
			s = "O-O"
		}
	} else {
		if t != Pawn {
			s = string(letters[t])
			if legal == nil {
				legal = p.Legal()
			}
			sameFile, sameRank, any := false, false, false
			for _, o := range legal {
				if o.To == m.To && o.Piece == m.Piece && o.From != m.From {
					any = true
					if o.From&7 == m.From&7 {
						sameFile = true
					}
					if o.From>>4 == m.From>>4 {
						sameRank = true
					}
				}
			}
			if any {
				switch {
				case !sameFile:
					s += string("abcdefgh"[m.From&7])
				case !sameRank:
					s += string(byte('1' + m.From>>4))
				default:
					s += SquareName(m.From)
				}
			}
		} else if m.Captured != 0 {
			s = string("abcdefgh"[m.From&7])
		}
		if m.Captured != 0 {
			s += "x"
		}
		s += SquareName(m.To)
		if m.Promo != 0 {
			s += "=" + string(letters[abs8(m.Promo)])
		}
	}
	p.Make(m)
	if p.InCheck() {
		if len(p.Legal()) > 0 {
			s += "+"
		} else {
			s += "#"
		}
	}
	p.Unmake()
	return s
}

// Perft counts leaf nodes to depth d (for testing move generation).
func (p *Position) Perft(d int) int {
	if d == 0 {
		return 1
	}
	n := 0
	us := p.Side
	for _, m := range p.Pseudo(false, make([]Move, 0, 64)) {
		p.Make(m)
		if !p.Attacked(p.Kings[kingIdx(us)], -us) {
			if d == 1 {
				n++
			} else {
				n += p.Perft(d - 1)
			}
		}
		p.Unmake()
	}
	return n
}
