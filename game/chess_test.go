package game

import (
	"testing"
)

// sq is a helper to build a Square from algebraic notation (e.g., "e2" → {6,4}).
func sq(file byte, rank int) Square {
	col := int(file - 'a')
	row := 8 - rank
	return Square{row, col}
}

// countMoves counts all legal moves for the given color.
func countAllMoves(b *Board, color Color) int {
	total := 0
	for r := 0; r < 8; r++ {
		for c := 0; c < 8; c++ {
			p := b.Cells[r][c]
			if p == nil || p.Color != color {
				continue
			}
			total += len(b.LegalMoves(Square{r, c}))
		}
	}
	return total
}

// TestStartingPosition verifies white has exactly 20 legal moves from start.
func TestStartingPosition(t *testing.T) {
	b := NewBoard()
	moves := countAllMoves(b, White)
	if moves != 20 {
		t.Errorf("expected 20 legal moves at start, got %d", moves)
	}
}

// TestScholarsMate plays a scholar's mate sequence and checks checkmate.
// 1.e4 e5 2.Bc4 Nc6 3.Qh5 Nf6?? 4.Qxf7#
func TestScholarsMate(t *testing.T) {
	b := NewBoard()

	moves := [][2]Square{
		{sq('e', 2), sq('e', 4)}, // 1. e4
		{sq('e', 7), sq('e', 5)}, // 1...e5
		{sq('f', 1), sq('c', 4)}, // 2. Bc4
		{sq('b', 8), sq('c', 6)}, // 2...Nc6
		{sq('d', 1), sq('h', 5)}, // 3. Qh5
		{sq('g', 8), sq('f', 6)}, // 3...Nf6??
		{sq('h', 5), sq('f', 7)}, // 4. Qxf7#
	}

	for i, mv := range moves {
		if err := b.ApplyMove(mv[0], mv[1]); err != nil {
			t.Fatalf("move %d (%v->%v) failed: %v", i+1, mv[0], mv[1], err)
		}
	}

	if !b.IsCheckmate(Black) {
		t.Error("expected Black to be in checkmate after scholar's mate")
	}
}

// TestEnPassant verifies en passant capture works.
func TestEnPassant(t *testing.T) {
	b := NewBoard()

	// 1.e4 a5 2.e5 d5 (white pawn on e5, black pawn on d5 → en passant exd6)
	moves := [][2]Square{
		{sq('e', 2), sq('e', 4)},
		{sq('a', 7), sq('a', 5)},
		{sq('e', 4), sq('e', 5)},
		{sq('d', 7), sq('d', 5)},
	}
	for i, mv := range moves {
		if err := b.ApplyMove(mv[0], mv[1]); err != nil {
			t.Fatalf("setup move %d failed: %v", i+1, err)
		}
	}

	// White pawn on e5 captures en passant to d6.
	from := sq('e', 5)
	to := sq('d', 6)
	if err := b.ApplyMove(from, to); err != nil {
		t.Fatalf("en passant move failed: %v", err)
	}

	// Black pawn on d5 should be gone.
	if b.Cells[sq('d', 5).Row][sq('d', 5).Col] != nil {
		t.Error("captured pawn should be removed after en passant")
	}
	// White pawn should be on d6.
	p := b.Cells[sq('d', 6).Row][sq('d', 6).Col]
	if p == nil || p.Type != Pawn || p.Color != White {
		t.Error("white pawn should be on d6 after en passant")
	}
}

// TestCastling verifies kingside and queenside castling.
func TestCastlingKingside(t *testing.T) {
	b := NewBoard()

	// Clear the f1 and g1 squares manually for white kingside castling.
	b.Cells[7][5] = nil // f1
	b.Cells[7][6] = nil // g1

	from := sq('e', 1)
	to := sq('g', 1)
	if err := b.ApplyMove(from, to); err != nil {
		t.Fatalf("kingside castling failed: %v", err)
	}

	king := b.Cells[7][6]
	rook := b.Cells[7][5]
	if king == nil || king.Type != King {
		t.Error("king should be on g1 after kingside castling")
	}
	if rook == nil || rook.Type != Rook {
		t.Error("rook should be on f1 after kingside castling")
	}
}

func TestCastlingQueenside(t *testing.T) {
	b := NewBoard()

	// Clear b1, c1, d1 for white queenside castling.
	b.Cells[7][1] = nil // b1
	b.Cells[7][2] = nil // c1
	b.Cells[7][3] = nil // d1

	from := sq('e', 1)
	to := sq('c', 1)
	if err := b.ApplyMove(from, to); err != nil {
		t.Fatalf("queenside castling failed: %v", err)
	}

	king := b.Cells[7][2]
	rook := b.Cells[7][3]
	if king == nil || king.Type != King {
		t.Error("king should be on c1 after queenside castling")
	}
	if rook == nil || rook.Type != Rook {
		t.Error("rook should be on d1 after queenside castling")
	}
}

// TestQuantumSplit verifies a piece can split into superposition.
func TestQuantumSplit(t *testing.T) {
	b := NewBoard()

	// Split white knight from b1 to c3 (standard knight move).
	from := sq('b', 1)
	to := sq('c', 3)
	if err := b.QuantumSplit(from, to); err != nil {
		t.Fatalf("quantum split failed: %v", err)
	}

	pFrom := b.Cells[from.Row][from.Col]
	pTo := b.Cells[to.Row][to.Col]
	if pFrom == nil || pFrom.SuperpositionID == 0 {
		t.Error("original square should have a superposed piece")
	}
	if pTo == nil || pTo.SuperpositionID == 0 {
		t.Error("target square should have a superposed piece")
	}
	if pFrom.SuperpositionID != pTo.SuperpositionID {
		t.Error("both ghosts should share the same superposition group ID")
	}
}

// TestCollapse verifies superposition collapses correctly.
func TestCollapse(t *testing.T) {
	b := NewBoard()

	from := sq('b', 1)
	to := sq('c', 3)
	if err := b.QuantumSplit(from, to); err != nil {
		t.Fatalf("quantum split failed: %v", err)
	}

	p := b.Cells[from.Row][from.Col]
	gid := p.SuperpositionID

	// Collapse to `from`.
	b.CollapseToSquare(gid, from)

	pFrom := b.Cells[from.Row][from.Col]
	pTo := b.Cells[to.Row][to.Col]
	if pFrom == nil || pFrom.SuperpositionID != 0 {
		t.Error("chosen square should have a classical piece after collapse")
	}
	if pTo != nil {
		t.Error("ghost square should be empty after collapse")
	}
}

// TestEntanglement verifies Link and TriggerEntanglement work.
func TestEntanglement(t *testing.T) {
	b := NewBoard()
	b.Turn = White

	a := sq('a', 2) // white pawn
	c := sq('c', 2) // white pawn
	if err := b.Link(a, c); err != nil {
		t.Fatalf("link failed: %v", err)
	}

	// Trigger entanglement by capturing the pawn at a2.
	b.TriggerEntanglement(a)

	if b.Cells[c.Row][c.Col] != nil {
		t.Error("entangled piece at c2 should have been removed")
	}
}

// TestUnlinkSplit verifies that unlinking a bridge splits the group.
func TestUnlinkSplit(t *testing.T) {
	b := NewBoard()
	b.Turn = White

	a := sq('a', 2)
	c := sq('c', 2)
	e := sq('e', 2)

	// Link a-c and c-e forming a chain.
	if err := b.Link(a, c); err != nil {
		t.Fatal(err)
	}
	b.LinkedThisTurn = false // reset for next link
	if err := b.Link(c, e); err != nil {
		t.Fatal(err)
	}
	b.LinkedThisTurn = false

	// Now unlink c-e; a-c stays together, e becomes lone.
	if err := b.Unlink(c, e); err != nil {
		t.Fatalf("unlink failed: %v", err)
	}

	// e should not be in any entanglement group.
	pe := b.Cells[e.Row][e.Col]
	if pe != nil && len(pe.EntanglementIDs) > 0 {
		t.Error("e2 should not be in any entanglement group after unlink")
	}
	// a and c should still be linked.
	pa := b.Cells[a.Row][a.Col]
	if pa == nil || len(pa.EntanglementIDs) == 0 {
		t.Error("a2 should still be in an entanglement group")
	}
}
