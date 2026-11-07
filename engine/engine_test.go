package engine

import (
	"testing"
	"time"
)

// Standard perft positions (chessprogramming.org): move generation must match exactly.
func TestPerft(t *testing.T) {
	cases := []struct {
		fen    string
		counts []int
	}{
		{StartFEN, []int{20, 400, 8902, 197281, 4865609}},
		{"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1", []int{48, 2039, 97862, 4085603}},
		{"8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1", []int{14, 191, 2812, 43238, 674624}},
		{"r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1", []int{6, 264, 9467, 422333}},
		{"rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8", []int{44, 1486, 62379, 2103487}},
		{"r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10", []int{46, 2079, 89890, 3894594}},
	}
	for _, c := range cases {
		p := MustPosition(c.fen)
		for d, want := range c.counts {
			if got := p.Perft(d + 1); got != want {
				t.Errorf("%s depth %d: %d, want %d", c.fen, d+1, got, want)
			}
		}
		if p.FEN() != c.fen {
			t.Errorf("FEN round trip: %s", p.FEN())
		}
	}
}

func TestHashMatchesRecompute(t *testing.T) {
	p := MustPosition("r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")
	var walk func(d int)
	walk = func(d int) {
		if d == 0 {
			return
		}
		for _, m := range p.Legal() {
			p.Make(m)
			h := p.Hash
			p.computeHash()
			if h != p.Hash {
				t.Fatalf("hash mismatch after %s", m.UCI())
			}
			walk(d - 1)
			p.Unmake()
		}
	}
	walk(3)
}

func TestSANAndStatus(t *testing.T) {
	p := MustPosition(StartFEN)
	for _, u := range []string{"e2e4", "e7e5", "d1h5", "b8c6", "f1c4", "g8f6"} {
		m, ok := p.FindUCI(u)
		if !ok {
			t.Fatal("illegal", u)
		}
		p.Make(m)
	}
	m, _ := p.FindUCI("h5f7")
	if s := p.SAN(m, p.Legal()); s != "Qxf7#" {
		t.Errorf("SAN %q, want Qxf7#", s)
	}
	p.Make(m)
	if st := p.Status(); !st.Over || st.Reason != "checkmate" || st.Winner != White {
		t.Errorf("status %+v", st)
	}
	// disambiguation and promotion
	p = MustPosition("6k1/1P6/8/8/8/8/8/R3K2R w KQ - 0 1")
	for _, c := range []struct{ u, san string }{{"b7b8q", "b8=Q+"}, {"e1g1", "O-O"}, {"a1d1", "Rd1"}} {
		m, _ := p.FindUCI(c.u)
		if s := p.SAN(m, p.Legal()); s != c.san {
			t.Errorf("%s: SAN %q want %q", c.u, s, c.san)
		}
	}
	if !MustPosition("8/8/8/4k3/8/8/8/4KB2 w - - 0 1").Insufficient() {
		t.Error("K+B v K should be insufficient")
	}
	if st := MustPosition("7k/5Q2/6K1/8/8/8/8/8 b - - 0 1").Status(); st.Reason != "stalemate" {
		t.Errorf("want stalemate, got %+v", st)
	}
}

func TestBookLinesAreLegal(t *testing.T) {
	for _, l := range bookLines {
		p := MustPosition(StartFEN)
		for _, u := range l.moves {
			m, ok := p.FindUCI(u)
			if !ok {
				t.Fatalf("%s: illegal %s", l.name, u)
			}
			p.Make(m)
		}
	}
	if n := OpeningName([]string{"e2e4", "c7c5", "g1f3", "d7d6", "d2d4", "c5d4", "f3d4", "g8f6"}); n != "Sicilian Defence: Najdorf" {
		t.Errorf("opening name %q", n)
	}
	if n := OpeningName([]string{"e2e4", "c7c5"}); n != "Sicilian Defence" {
		t.Errorf("opening name %q", n)
	}
}

// The computer finds short tactics (a subset of Win At Chess).
func TestTactics(t *testing.T) {
	cases := [][2]string{
		{"6k1/5ppp/8/8/8/8/5PPP/3R2K1 w - - 0 1", "d1d8"},
		{"r1bqkb1r/pppp1ppp/2n2n2/4p2Q/2B1P3/8/PPPP1PPP/RNB1K1NR w KQkq - 4 4", "h5f7"},
		{"2rr3k/pp3pp1/1nnqbN1p/3pN3/2pP4/2P3Q1/PPB4P/R4RK1 w - - 0 1", "g3g6"},
		{"5rk1/1ppb3p/p1pb4/6q1/3P1p1r/2P1R2P/PP1BQ1P1/5RKN w - - 0 1", "e3g3"},
		{"r1bq2rk/pp3pbp/2p1p1pQ/7P/3P4/2PB1N2/PP3PPR/2KR4 w - - 0 1", "h6h7"},
		{"5k2/6pp/p1qN4/1p1p4/3P4/2PKP2Q/PP3r2/3R4 b - - 0 1", "c6c4"},
		{"7k/p7/1R5K/6r1/6p1/6P1/8/8 w - - 0 1", "b6b7"},
		{"rnbqkb1r/pppp1ppp/8/4P3/6n1/7P/PPPNPPP1/R1BQKBNR b KQkq - 0 1", "g4e3"},
		{"r4q1k/p2bR1rp/2p2Q1N/5p2/5p2/2P5/PP3PPP/R5K1 w - - 0 1", "e7f7"},
		{"3q1rk1/p4pp1/2pb3p/3p4/6Pr/1PNQ4/P1PB1PP1/4RRK1 b - - 0 1", "d6h2"},
		{"2br2k1/2q3rn/p2NppQ1/2p1P3/Pp5R/4P3/1P3PPP/3R2K1 w - - 0 1", "h4h7"},
	}
	s := NewSearcher()
	ok := 0
	for _, c := range cases {
		r := s.Think(MustPosition(c[0]), Options{Time: 1500 * time.Millisecond})
		if r.Move.UCI() == c[1] {
			ok++
		} else {
			t.Logf("%s: played %s (depth %d), book move %s", c[0], r.Move.UCI(), r.Depth, c[1])
		}
	}
	if ok < len(cases)-1 {
		t.Errorf("solved %d of %d", ok, len(cases))
	}
}

// Full-strength play converts basic won endgames to checkmate.
func TestEndgamesConvert(t *testing.T) {
	for _, fen := range []string{"8/8/8/4k3/8/8/8/4K2Q w - - 0 1", "8/8/8/4k3/8/8/8/4K2R w - - 0 1", "7k/8/8/8/8/8/8/KQ6 b - - 0 1"} {
		p := MustPosition(fen)
		s := NewSearcher()
		var st Status
		for ply := 0; ply < 160; ply++ {
			if st = p.Status(); st.Over {
				break
			}
			r := s.Think(p, Options{Time: 250 * time.Millisecond})
			p.Make(r.Move)
		}
		if st.Reason != "checkmate" || st.Winner != White {
			t.Errorf("%s: ended %+v", fen, st)
		}
	}
}
