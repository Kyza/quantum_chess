package game

import (
	"fmt"
	"math/rand"
)

// pseudoMoves returns all squares a piece at `sq` can move to (ignoring check).
// If captureOnly is true, only attacking squares are returned (used for check detection).
func (b *Board) pseudoMoves(sq Square, captureOnly bool) []Square {
	p := b.piece(sq)
	if p == nil {
		return nil
	}
	var moves []Square
	switch p.Type {
	case Pawn:
		moves = b.pawnMoves(sq, p.Color, captureOnly)
	case Knight:
		moves = b.knightMoves(sq, p.Color, captureOnly)
	case Bishop:
		moves = b.slidingMoves(sq, p.Color, [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}}, captureOnly)
	case Rook:
		moves = b.slidingMoves(sq, p.Color, [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}, captureOnly)
	case Queen:
		moves = b.slidingMoves(sq, p.Color, [][2]int{
			{1, 0}, {-1, 0}, {0, 1}, {0, -1},
			{1, 1}, {1, -1}, {-1, 1}, {-1, -1},
		}, captureOnly)
	case King:
		moves = b.kingMoves(sq, p.Color, captureOnly)
	}
	return moves
}

func (b *Board) pawnMoves(sq Square, color Color, captureOnly bool) []Square {
	var moves []Square
	dir := -1
	startRow := 6
	if color == Black {
		dir = 1
		startRow = 1
	}

	if !captureOnly {
		// Forward one.
		fwd := Square{sq.Row + dir, sq.Col}
		if fwd.inBounds() && b.piece(fwd) == nil {
			moves = append(moves, fwd)
			// Forward two from start.
			if sq.Row == startRow {
				fwd2 := Square{sq.Row + 2*dir, sq.Col}
				if fwd2.inBounds() && b.piece(fwd2) == nil {
					moves = append(moves, fwd2)
				}
			}
		}
	}

	// Captures (diagonal).
	for _, dc := range []int{-1, 1} {
		cap := Square{sq.Row + dir, sq.Col + dc}
		if !cap.inBounds() {
			continue
		}
		target := b.piece(cap)
		if captureOnly {
			// Return as attack square regardless of occupancy.
			moves = append(moves, cap)
		} else if target != nil && target.Color != color {
			moves = append(moves, cap)
		} else if b.EnPassantTarget != nil && *b.EnPassantTarget == cap {
			moves = append(moves, cap)
		}
	}
	return moves
}

func (b *Board) knightMoves(sq Square, color Color, captureOnly bool) []Square {
	offsets := [][2]int{{2, 1}, {2, -1}, {-2, 1}, {-2, -1}, {1, 2}, {1, -2}, {-1, 2}, {-1, -2}}
	var moves []Square
	for _, off := range offsets {
		dst := Square{sq.Row + off[0], sq.Col + off[1]}
		if !dst.inBounds() {
			continue
		}
		target := b.piece(dst)
		if captureOnly || target == nil || target.Color != color {
			moves = append(moves, dst)
		}
	}
	return moves
}

func (b *Board) slidingMoves(sq Square, color Color, dirs [][2]int, captureOnly bool) []Square {
	var moves []Square
	for _, d := range dirs {
		for i := 1; i < 8; i++ {
			dst := Square{sq.Row + d[0]*i, sq.Col + d[1]*i}
			if !dst.inBounds() {
				break
			}
			target := b.piece(dst)
			if target == nil {
				if !captureOnly {
					moves = append(moves, dst)
				}
			} else {
				if captureOnly || target.Color != color {
					moves = append(moves, dst)
				}
				break
			}
		}
	}
	return moves
}

func (b *Board) kingMoves(sq Square, color Color, captureOnly bool) []Square {
	offsets := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
	var moves []Square
	for _, off := range offsets {
		dst := Square{sq.Row + off[0], sq.Col + off[1]}
		if !dst.inBounds() {
			continue
		}
		target := b.piece(dst)
		if captureOnly || target == nil || target.Color != color {
			moves = append(moves, dst)
		}
	}
	if !captureOnly {
		moves = append(moves, b.castlingMoves(sq, color)...)
	}
	return moves
}

func (b *Board) castlingMoves(sq Square, color Color) []Square {
	var moves []Square
	row := 7
	if color == Black {
		row = 0
	}
	if sq.Row != row || sq.Col != 4 {
		return nil // King not on starting square.
	}
	if b.IsAttacked(sq, opponent(color)) {
		return nil // King in check.
	}
	// Kingside.
	if b.CastlingRights[color][1] {
		// Squares between king and rook must be empty.
		if b.piece(Square{row, 5}) == nil && b.piece(Square{row, 6}) == nil {
			// King must not pass through or land on attacked square.
			if !b.IsAttacked(Square{row, 5}, opponent(color)) &&
				!b.IsAttacked(Square{row, 6}, opponent(color)) {
				moves = append(moves, Square{row, 6})
			}
		}
	}
	// Queenside.
	if b.CastlingRights[color][0] {
		if b.piece(Square{row, 3}) == nil && b.piece(Square{row, 2}) == nil && b.piece(Square{row, 1}) == nil {
			if !b.IsAttacked(Square{row, 3}, opponent(color)) &&
				!b.IsAttacked(Square{row, 2}, opponent(color)) {
				moves = append(moves, Square{row, 2})
			}
		}
	}
	return moves
}

// IsAttacked returns true if `sq` is attacked by any piece of `byColor`.
func (b *Board) IsAttacked(sq Square, byColor Color) bool {
	for r := 0; r < 8; r++ {
		for c := 0; c < 8; c++ {
			p := b.Cells[r][c]
			if p == nil || p.Color != byColor {
				continue
			}
			attacks := b.pseudoMoves(Square{r, c}, true)
			for _, a := range attacks {
				if a == sq {
					return true
				}
			}
		}
	}
	return false
}

// findKing returns the square of `color`'s king (or first superposed king square).
func (b *Board) findKing(color Color) (Square, bool) {
	for r := 0; r < 8; r++ {
		for c := 0; c < 8; c++ {
			p := b.Cells[r][c]
			if p != nil && p.Type == King && p.Color == color {
				return Square{r, c}, true
			}
		}
	}
	return Square{}, false
}

// IsInCheck returns whether the given color's king is in check.
// For a superposed king: in check if ANY possible square is attacked
// (used as a warning; players are not forced to escape check).
func (b *Board) IsInCheck(color Color) bool {
	opp := opponent(color)
	for r := 0; r < 8; r++ {
		for c := 0; c < 8; c++ {
			p := b.Cells[r][c]
			if p != nil && p.Type == King && p.Color == color {
				if b.IsAttacked(Square{r, c}, opp) {
					return true
				}
			}
		}
	}
	return false
}

// LegalMoves returns all legal destination squares for the piece at `sq`.
// In quantum chess, players are not forced to escape check; all pseudo-legal
// moves are legal (king capture ends the game instead of checkmate).
func (b *Board) LegalMoves(sq Square) []Square {
	p := b.piece(sq)
	if p == nil || p.Color != b.Turn {
		return nil
	}
	return b.pseudoMoves(sq, false)
}

// LegalSplitTargets returns empty squares the piece at `sq` can quantum-split to.
func (b *Board) LegalSplitTargets(sq Square) []Square {
	p := b.piece(sq)
	if p == nil || p.Color != b.Turn || p.Type == King {
		return nil
	}
	// Non-capturing reachable squares: pseudo moves that land on empty squares.
	candidates := b.pseudoMoves(sq, false)
	var targets []Square
	for _, dst := range candidates {
		if b.piece(dst) == nil {
			targets = append(targets, dst)
		}
	}
	return targets
}

// LegalLinks returns squares of friendly pieces the piece at `sq` can link to.
func (b *Board) LegalLinks(sq Square) []Square {
	p := b.piece(sq)
	if p == nil || p.Color != b.Turn {
		return nil
	}
	var links []Square
	for r := 0; r < 8; r++ {
		for c := 0; c < 8; c++ {
			other := Square{r, c}
			if other == sq {
				continue
			}
			op := b.piece(other)
			if op != nil && op.Color == b.Turn {
				links = append(links, other)
			}
		}
	}
	return links
}

// ApplyMove executes a move from `from` to `to`, enforcing legality.
func (b *Board) ApplyMove(from, to Square) error {
	p := b.piece(from)
	if p == nil {
		return fmt.Errorf("no piece at %v", from)
	}
	if p.Color != b.Turn {
		return fmt.Errorf("not your turn")
	}
	legal := b.LegalMoves(from)
	found := false
	for _, sq := range legal {
		if sq == to {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("illegal move from %v to %v", from, to)
	}
	b.applyMoveUnchecked(from, to)
	return nil
}

// ApplyQuantumSplit executes a quantum split move.
func (b *Board) ApplyQuantumSplit(from, to Square) error {
	return b.QuantumSplit(from, to)
}

// applyMoveUnchecked performs a move without legality checking (used internally).
// Returns true if the move was completed, false if the attacker collapsed away
// (turn is still consumed in that case).
func (b *Board) applyMoveUnchecked(from, to Square) bool {
	p := b.piece(from)
	if p == nil {
		return false
	}

	// If the attacker is in superposition, collapse it to a random square.
	// If it collapses somewhere other than `from` the move fails — the piece
	// wasn't actually there — but the turn is still consumed.
	if p.SuperpositionID != 0 {
		sid := p.SuperpositionID
		sg := b.SuperpositionGroups[sid]
		chosen := sg.Squares[rand.Intn(len(sg.Squares))]
		b.collapseToSquare(sid, chosen)
		if chosen != from {
			// Piece collapsed elsewhere — move doesn't happen.
			b.advanceTurn()
			return false
		}
		p = b.piece(from) // refresh after collapse
	}

	// Reset en passant target.
	prevEP := b.EnPassantTarget
	b.EnPassantTarget = nil

	// If the destination holds a superposed piece, collapse it now.
	// The piece collapses to a random square; if it collapses away from `to`
	// the square is now empty and the move lands without a capture.
	if target := b.piece(to); target != nil && target.SuperpositionID != 0 {
		sid := target.SuperpositionID
		sg := b.SuperpositionGroups[sid]
		chosen := sg.Squares[rand.Intn(len(sg.Squares))]
		b.collapseToSquare(sid, chosen)
		// If it collapsed elsewhere, `to` is now empty — move proceeds normally.
	}

	// Determine if capture.
	captured := b.piece(to)

	// Handle special moves.
	dir := -1
	if p.Color == Black {
		dir = 1
	}

	// En passant capture.
	if p.Type == Pawn && prevEP != nil && *prevEP == to {
		// Remove the captured pawn.
		epCaptureSq := Square{to.Row - dir, to.Col}
		capturedEP := b.piece(epCaptureSq)
		if capturedEP != nil {
			b.TriggerEntanglement(epCaptureSq)
			b.Cells[epCaptureSq.Row][epCaptureSq.Col] = nil
		}
		captured = nil
	}

	// Trigger entanglement for capture.
	if captured != nil {
		b.TriggerEntanglement(to)
		// King capture: game over.
		if captured.Type == King {
			b.GameOver = true
			b.Winner = p.Color
		}
	}

	// Move the piece.
	b.Cells[to.Row][to.Col] = p
	b.Cells[from.Row][from.Col] = nil
	// Update entanglement group records so lines follow the piece.
	b.updateEntanglementSquare(from, to)

	// Castling: also move the rook.
	if p.Type == King {
		row := 7
		if p.Color == Black {
			row = 0
		}
		if from == (Square{row, 4}) {
			if to == (Square{row, 6}) {
				// Kingside.
				rook := b.Cells[row][7]
				b.Cells[row][5] = rook
				b.Cells[row][7] = nil
				b.CastlingRights[p.Color][1] = false
			} else if to == (Square{row, 2}) {
				// Queenside.
				rook := b.Cells[row][0]
				b.Cells[row][3] = rook
				b.Cells[row][0] = nil
				b.CastlingRights[p.Color][0] = false
			}
		}
		b.CastlingRights[p.Color][0] = false
		b.CastlingRights[p.Color][1] = false
	}

	// Rook move revokes castling rights.
	if p.Type == Rook {
		row := 7
		if p.Color == Black {
			row = 0
		}
		if from == (Square{row, 0}) {
			b.CastlingRights[p.Color][0] = false
		} else if from == (Square{row, 7}) {
			b.CastlingRights[p.Color][1] = false
		}
	}

	// Set en passant target if pawn double push.
	if p.Type == Pawn {
		if from.Row-to.Row == 2 || to.Row-from.Row == 2 {
			ep := Square{(from.Row + to.Row) / 2, from.Col}
			b.EnPassantTarget = &ep
		}
	}

	// Pawn promotion (auto-queen).
	if p.Type == Pawn && (to.Row == 0 || to.Row == 7) {
		b.Cells[to.Row][to.Col] = &Piece{
			Type:  Queen,
			Color: p.Color,
		}
	}

	// Half-move clock.
	if p.Type == Pawn || captured != nil {
		b.HalfMoveClock = 0
	} else {
		b.HalfMoveClock++
	}

	b.advanceTurn()
	return true
}

// IsCheckmate returns whether the given color is in checkmate.
func (b *Board) IsCheckmate(color Color) bool {
	if !b.IsInCheck(color) {
		return false
	}
	return !b.hasAnyLegalMove(color)
}

// IsStalemate returns whether the given color is in stalemate.
func (b *Board) IsStalemate(color Color) bool {
	if b.IsInCheck(color) {
		return false
	}
	return !b.hasAnyLegalMove(color)
}

func (b *Board) hasAnyLegalMove(color Color) bool {
	savedTurn := b.Turn
	b.Turn = color
	defer func() { b.Turn = savedTurn }()
	for r := 0; r < 8; r++ {
		for c := 0; c < 8; c++ {
			p := b.Cells[r][c]
			if p == nil || p.Color != color {
				continue
			}
			if len(b.LegalMoves(Square{r, c})) > 0 {
				return true
			}
		}
	}
	return false
}
