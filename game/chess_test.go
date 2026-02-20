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

// TestScholarsMate plays a scholar's mate sequence and verifies that Black's
// king is in check after Qxf7. In quantum chess, checkmate is not a win
// condition — the game continues until the king is actually captured.
// 1.e4 e5 2.Bc4 Nc6 3.Qh5 Nf6?? 4.Qxf7+
func TestScholarsMate(t *testing.T) {
	b := NewBoard()

	moves := [][2]Square{
		{sq('e', 2), sq('e', 4)}, // 1. e4
		{sq('e', 7), sq('e', 5)}, // 1...e5
		{sq('f', 1), sq('c', 4)}, // 2. Bc4
		{sq('b', 8), sq('c', 6)}, // 2...Nc6
		{sq('d', 1), sq('h', 5)}, // 3. Qh5
		{sq('g', 8), sq('f', 6)}, // 3...Nf6??
		{sq('h', 5), sq('f', 7)}, // 4. Qxf7+
	}

	for i, mv := range moves {
		if err := b.ApplyMove(mv[0], mv[1]); err != nil {
			t.Fatalf("move %d (%v->%v) failed: %v", i+1, mv[0], mv[1], err)
		}
	}

	// Game is NOT over — the king has not been captured yet.
	if b.GameOver {
		t.Error("game should not be over after scholar's mate (king not captured)")
	}
	// Black king should be in check (warning display).
	if !b.IsInCheck(Black) {
		t.Error("expected Black king to be in check after Qxf7+")
	}
}

// TestKingCapture verifies that capturing the king ends the game.
func TestKingCapture(t *testing.T) {
	b := NewBoard()

	// Clear the board and set up a simple scenario:
	// White queen on d1, Black king on e8 (its start square), nothing in between.
	for r := 0; r < 8; r++ {
		for c := 0; c < 8; c++ {
			b.Cells[r][c] = nil
		}
	}
	// Place white queen on d4 to reach e5.
	b.Cells[4][3] = &Piece{Type: Queen, Color: White} // d4
	// Place black king on e5 where white queen can capture it.
	b.Cells[4][4] = &Piece{Type: King, Color: Black} // e5
	b.Turn = White

	from := Square{4, 3} // d4
	to := Square{4, 4}   // e5

	if err := b.ApplyMove(from, to); err != nil {
		t.Fatalf("king capture move failed: %v", err)
	}

	if !b.GameOver {
		t.Error("expected GameOver == true after king capture")
	}
	if b.Winner != White {
		t.Errorf("expected Winner == White, got %v", b.Winner)
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

// ─── Superposition / Quantum Split tests ─────────────────────────────────────

// clearBoard removes all pieces from the board and resets quantum state.
func clearBoard(b *Board) {
	for r := 0; r < 8; r++ {
		for c := 0; c < 8; c++ {
			b.Cells[r][c] = nil
		}
	}
	b.SuperpositionGroups = make(map[int]*SuperpositionGroup)
	b.EntanglementGroups = make(map[int]*EntanglementGroup)
	b.EnPassantTarget = nil
}

// TestSplitCreatesGroup verifies a quantum split creates a SuperpositionGroup
// with both squares registered.
func TestSplitCreatesGroup(t *testing.T) {
	b := NewBoard()
	from := sq('b', 1) // knight
	to := sq('c', 3)
	if err := b.QuantumSplit(from, to); err != nil {
		t.Fatal(err)
	}
	pFrom := b.Cells[from.Row][from.Col]
	pTo := b.Cells[to.Row][to.Col]
	if pFrom == nil || pFrom.SuperpositionID == 0 {
		t.Error("from square should have a superposed piece")
	}
	if pTo == nil || pTo.SuperpositionID == 0 {
		t.Error("to square should have a superposed piece")
	}
	if pFrom.SuperpositionID != pTo.SuperpositionID {
		t.Error("both squares should share a group ID")
	}
	gid := pFrom.SuperpositionID
	g := b.SuperpositionGroups[gid]
	if g == nil {
		t.Fatal("SuperpositionGroup not found")
	}
	if len(g.Squares) != 2 {
		t.Errorf("expected 2 squares in group, got %d", len(g.Squares))
	}
}

// TestSplitCannotCaptureDestination verifies you cannot split onto an occupied square.
func TestSplitCannotCaptureDestination(t *testing.T) {
	b := NewBoard()
	// b1 knight → a3 is a legal knight move, but a3 has nothing; put a piece there.
	b.Cells[5][0] = &Piece{Type: Pawn, Color: White} // a3
	from := sq('b', 1)
	to := sq('a', 3)
	if err := b.QuantumSplit(from, to); err == nil {
		t.Error("expected error: cannot split onto occupied square")
	}
}

// TestSplitKingForbidden verifies kings cannot be split.
func TestSplitKingForbidden(t *testing.T) {
	b := NewBoard()
	clearBoard(b)
	b.Cells[7][4] = &Piece{Type: King, Color: White} // e1
	b.Turn = White
	from := sq('e', 1)
	to := sq('e', 2)
	if err := b.QuantumSplit(from, to); err == nil {
		t.Error("expected error: king cannot be split")
	}
}

// TestTripleSplit verifies splitting twice creates a 3-square superposition group.
func TestTripleSplit(t *testing.T) {
	b := NewBoard()
	clearBoard(b)
	// Place a rook on a1 so it can slide.
	b.Cells[7][0] = &Piece{Type: Rook, Color: White} // a1
	b.Turn = White

	a1 := sq('a', 1)
	a4 := sq('a', 4)
	a6 := sq('a', 6)

	// First split: a1 → a4
	if err := b.QuantumSplit(a1, a4); err != nil {
		t.Fatalf("first split failed: %v", err)
	}
	b.Turn = White // reset turn for test

	// Second split from the ghost at a4 (the a1→a6 path is blocked by that ghost).
	if err := b.QuantumSplit(a4, a6); err != nil {
		t.Fatalf("second split failed: %v", err)
	}

	gid := b.Cells[a1.Row][a1.Col].SuperpositionID
	g := b.SuperpositionGroups[gid]
	if g == nil {
		t.Fatal("superposition group missing")
	}
	if len(g.Squares) != 3 {
		t.Errorf("expected 3 squares after two splits, got %d", len(g.Squares))
	}
}

// TestSplitTargetsAreEmpty verifies LegalSplitTargets only returns empty squares.
func TestSplitTargetsAreEmpty(t *testing.T) {
	b := NewBoard()
	// Knight on b1; some squares are blocked. All returned targets must be empty.
	targets := b.LegalSplitTargets(sq('b', 1))
	for _, t2 := range targets {
		if b.Cells[t2.Row][t2.Col] != nil {
			t.Errorf("split target %v is not empty", t2)
		}
	}
}

// TestNormalMoveCollapsesAttacker verifies that moving a superposed piece normally
// collapses it: only one square is occupied afterward, no group remains.
func TestNormalMoveCollapsesAttacker(t *testing.T) {
	b := NewBoard()
	clearBoard(b)
	b.Cells[7][0] = &Piece{Type: Rook, Color: White} // a1
	b.Cells[0][7] = &Piece{Type: King, Color: Black}  // h8 (need kings for IsInCheck)
	b.Cells[7][4] = &Piece{Type: King, Color: White}  // e1
	b.Turn = White

	a1 := sq('a', 1)
	a4 := sq('a', 4)

	if err := b.QuantumSplit(a1, a4); err != nil {
		t.Fatal(err)
	}
	b.Turn = White

	gid := b.Cells[a1.Row][a1.Col].SuperpositionID

	// Force collapse to a1 then make a normal move from a1.
	b.CollapseToSquare(gid, a1)

	// Now the rook is classical on a1; move it to a2.
	if err := b.ApplyMove(a1, sq('a', 2)); err != nil {
		t.Fatalf("normal move failed: %v", err)
	}

	// No superposition group should remain.
	if len(b.SuperpositionGroups) != 0 {
		t.Errorf("expected no superposition groups, got %d", len(b.SuperpositionGroups))
	}
	// a1 should be empty, a2 occupied.
	if b.Cells[a1.Row][a1.Col] != nil {
		t.Error("a1 should be empty after move")
	}
	if b.Cells[sq('a', 2).Row][sq('a', 2).Col] == nil {
		t.Error("a2 should be occupied after move")
	}
}

// TestAttackerCollapseSucceeds verifies that a superposed attacker can capture
// when forced to collapse to the attacking square.
// Setup: white rook on a5, split DOWN to a2. Capture target is a8 (UP).
// Ghost at a2 does not block the a5→a8 path.
func TestAttackerCollapseSucceeds(t *testing.T) {
	b := NewBoard()
	clearBoard(b)
	b.Cells[4][4] = &Piece{Type: King, Color: White} // e5 — away from files used
	b.Cells[0][4] = &Piece{Type: King, Color: Black} // e8
	b.Cells[3][0] = &Piece{Type: Rook, Color: White} // a5
	b.Turn = White

	a5 := sq('a', 5)
	a2 := sq('a', 2)
	if err := b.QuantumSplit(a5, a2); err != nil {
		t.Fatalf("split failed: %v", err)
	}
	b.Turn = White

	// Black pawn on a8.
	b.Cells[0][0] = &Piece{Type: Pawn, Color: Black}
	a8 := sq('a', 8)

	// Force collapse to a5 (attacker square) → capture must succeed.
	completed, err := b.ApplyMoveCollapseTo(a5, a8, a5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !completed {
		t.Error("expected move to complete when attacker collapses to attacking square")
	}
	if b.Cells[a8.Row][a8.Col] == nil {
		t.Error("rook should be on a8 after successful capture")
	}
	if b.Cells[a5.Row][a5.Col] != nil {
		t.Error("a5 should be empty after rook moved away")
	}
}

// TestAttackerCollapseFails verifies that a superposed attacker's move fails
// when it collapses to its other ghost square instead.
// Same setup as above; collapse to a2 (not a5) → move must fail.
func TestAttackerCollapseFails(t *testing.T) {
	b := NewBoard()
	clearBoard(b)
	b.Cells[4][4] = &Piece{Type: King, Color: White} // e5
	b.Cells[0][4] = &Piece{Type: King, Color: Black} // e8
	b.Cells[3][0] = &Piece{Type: Rook, Color: White} // a5
	b.Turn = White

	a5 := sq('a', 5)
	a2 := sq('a', 2)
	if err := b.QuantumSplit(a5, a2); err != nil {
		t.Fatalf("split failed: %v", err)
	}
	b.Turn = White

	b.Cells[0][0] = &Piece{Type: Pawn, Color: Black}
	a8 := sq('a', 8)

	// Force collapse to a2 (NOT the attacking square a5) → move must fail.
	completed, err := b.ApplyMoveCollapseTo(a5, a8, a2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if completed {
		t.Error("expected move to fail when attacker collapses away from attacking square")
	}
	// Black pawn should still be alive.
	if b.Cells[a8.Row][a8.Col] == nil {
		t.Error("black pawn should survive when attacker missed")
	}
	// Rook should be on a2 (where it collapsed), a5 empty.
	if b.Cells[a2.Row][a2.Col] == nil {
		t.Error("rook should be on a2 after collapsing there")
	}
	if b.Cells[a5.Row][a5.Col] != nil {
		t.Error("a5 should be empty after collapse to a2")
	}
}

// TestCaptureCollapsesSuperposedTarget verifies that capturing a superposed
// piece collapses it: if it collapsed to the attacked square, capture succeeds;
// if not, the target survives at its collapsed square.
func TestCaptureCollapsesSuperposedTarget(t *testing.T) {
	b := NewBoard()
	clearBoard(b)
	b.Cells[7][4] = &Piece{Type: King, Color: White}
	b.Cells[0][4] = &Piece{Type: King, Color: Black}

	// Black rook in superposition on a8 and c8.
	b.Cells[0][0] = &Piece{Type: Rook, Color: Black, SuperpositionID: 1}
	b.Cells[0][2] = &Piece{Type: Rook, Color: Black, SuperpositionID: 1}
	b.SuperpositionGroups[1] = &SuperpositionGroup{
		ID:      1,
		Squares: []Square{sq('a', 8), sq('c', 8)},
		Piece:   Piece{Type: Rook, Color: Black},
	}
	b.nextSuperID = 2

	// White queen on a1, can capture a8.
	b.Cells[7][0] = &Piece{Type: Queen, Color: White}
	b.Turn = White

	a1 := sq('a', 1)
	a8 := sq('a', 8)
	c8 := sq('c', 8)

	// Case A: target collapses to a8 (attacked) → capture succeeds.
	bCopy := b.Clone()
	bCopy.collapseToSquare(1, a8) // pre-collapse for determinism
	// Now a8 is classical, c8 is empty. Queen captures.
	if err := bCopy.ApplyMove(a1, a8); err != nil {
		t.Fatalf("capture should succeed: %v", err)
	}
	if bCopy.Cells[a8.Row][a8.Col] == nil || bCopy.Cells[a8.Row][a8.Col].Color != White {
		t.Error("white queen should be on a8 after successful capture")
	}
	if bCopy.Cells[c8.Row][c8.Col] != nil {
		t.Error("c8 should be empty (ghost cleared by collapse)")
	}

	// Case B: target collapses to c8 (away from attack) → a8 empty, queen moves to empty square.
	bCopy2 := b.Clone()
	bCopy2.collapseToSquare(1, c8) // pre-collapse
	if err := bCopy2.ApplyMove(a1, a8); err != nil {
		t.Fatalf("move to now-empty a8 should succeed: %v", err)
	}
	if bCopy2.Cells[c8.Row][c8.Col] == nil || bCopy2.Cells[c8.Row][c8.Col].Color != Black {
		t.Error("black rook should be on c8 after collapsing there")
	}
}

// TestSplitPieceLegalMoves verifies that legal moves are reported for both
// ghost squares of a superposed piece.
func TestSplitPieceLegalMoves(t *testing.T) {
	b := NewBoard()
	clearBoard(b)
	b.Cells[7][4] = &Piece{Type: King, Color: White}
	b.Cells[0][4] = &Piece{Type: King, Color: Black}
	b.Cells[7][0] = &Piece{Type: Rook, Color: White}
	b.Turn = White

	a1 := sq('a', 1)
	a4 := sq('a', 4)
	b.QuantumSplit(a1, a4)
	b.Turn = White

	movesFromA1 := b.LegalMoves(a1)
	movesFromA4 := b.LegalMoves(a4)

	if len(movesFromA1) == 0 {
		t.Error("superposed piece on a1 should have legal moves")
	}
	if len(movesFromA4) == 0 {
		t.Error("superposed piece on a4 should have legal moves")
	}
}

// TestSplitGroupCleanedOnCapture verifies the SuperpositionGroup is deleted
// after the piece is fully captured/collapsed.
func TestSplitGroupCleanedOnCapture(t *testing.T) {
	b := NewBoard()
	clearBoard(b)
	b.Cells[7][4] = &Piece{Type: King, Color: White}
	b.Cells[0][4] = &Piece{Type: King, Color: Black}
	// Black rook in superposition on a8 and c8 (manually set up).
	b.Cells[0][0] = &Piece{Type: Rook, Color: Black, SuperpositionID: 1}
	b.Cells[0][2] = &Piece{Type: Rook, Color: Black, SuperpositionID: 1}
	b.SuperpositionGroups[1] = &SuperpositionGroup{
		ID:      1,
		Squares: []Square{sq('a', 8), sq('c', 8)},
		Piece:   Piece{Type: Rook, Color: Black},
	}
	b.nextSuperID = 2

	// Collapse to a8, then have white queen capture it.
	b.CollapseToSquare(1, sq('a', 8))

	b.Cells[7][0] = &Piece{Type: Queen, Color: White}
	b.Turn = White
	if err := b.ApplyMove(sq('a', 1), sq('a', 8)); err != nil {
		t.Fatal(err)
	}

	if len(b.SuperpositionGroups) != 0 {
		t.Errorf("SuperpositionGroups should be empty after capture, got %d", len(b.SuperpositionGroups))
	}
}

// TestCollapsePreservesEntanglement verifies that an entanglement link on a
// superposed piece survives collapse to the chosen square.
func TestCollapsePreservesEntanglement(t *testing.T) {
	b := NewBoard()
	clearBoard(b)
	b.Cells[7][0] = &Piece{Type: Rook, Color: White}  // a1
	b.Cells[7][7] = &Piece{Type: Rook, Color: White}  // h1
	b.Cells[7][4] = &Piece{Type: King, Color: White}
	b.Cells[0][4] = &Piece{Type: King, Color: Black}
	b.Turn = White

	a1 := sq('a', 1)
	a4 := sq('a', 4)
	h1 := sq('h', 1)

	// Split a1 rook.
	b.QuantumSplit(a1, a4)
	b.Turn = White // reset for Link
	// Link a1 ghost to h1.
	if err := b.Link(a1, h1); err != nil {
		t.Fatal(err)
	}
	gid := b.Cells[a1.Row][a1.Col].SuperpositionID

	// Collapse to a1 (the linked square).
	b.CollapseToSquare(gid, a1)

	pa1 := b.Cells[a1.Row][a1.Col]
	if pa1 == nil {
		t.Fatal("rook should be on a1 after collapse")
	}
	if len(pa1.EntanglementIDs) == 0 {
		t.Error("rook on a1 should still be entangled to h1 after collapse")
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
