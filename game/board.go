package game

// Color represents a player's color.
type Color int

const (
	White Color = iota
	Black
)

// PieceType represents the type of a chess piece.
type PieceType int

const (
	None PieceType = iota
	Pawn
	Knight
	Bishop
	Rook
	Queen
	King
)

// Square is a board coordinate.
type Square struct{ Row, Col int }

// inBounds returns true if s is within the 8×8 board.
func (s Square) inBounds() bool {
	return s.Row >= 0 && s.Row < 8 && s.Col >= 0 && s.Col < 8
}

// Piece represents a chess piece, possibly in superposition.
type Piece struct {
	Type            PieceType
	Color           Color
	SuperpositionID int   // 0 = classical; shared across superposition group members
	EntanglementIDs []int // IDs of entanglement groups this piece belongs to
}

// Board holds the complete game state.
type Board struct {
	Cells               [8][8]*Piece
	Turn                Color
	SuperpositionGroups map[int]*SuperpositionGroup
	EntanglementGroups  map[int]*EntanglementGroup
	nextSuperID         int
	nextEntangleID      int

	// Standard chess state
	CastlingRights  [2][2]bool // [color][0=queenside, 1=kingside]
	EnPassantTarget *Square
	HalfMoveClock   int
	FullMoveNumber  int

	// Link-phase tracking
	LinkedThisTurn bool
}

// NewBoard creates a board in the standard chess starting position.
func NewBoard() *Board {
	b := &Board{
		SuperpositionGroups: make(map[int]*SuperpositionGroup),
		EntanglementGroups:  make(map[int]*EntanglementGroup),
		nextSuperID:         1,
		nextEntangleID:      1,
		FullMoveNumber:      1,
	}
	// Castling rights: both sides for both colors
	b.CastlingRights[White][0] = true
	b.CastlingRights[White][1] = true
	b.CastlingRights[Black][0] = true
	b.CastlingRights[Black][1] = true

	placePieces := func(row int, color Color) {
		order := []PieceType{Rook, Knight, Bishop, Queen, King, Bishop, Knight, Rook}
		for col, pt := range order {
			b.Cells[row][col] = &Piece{Type: pt, Color: color}
		}
	}
	placePawns := func(row int, color Color) {
		for col := 0; col < 8; col++ {
			b.Cells[row][col] = &Piece{Type: Pawn, Color: color}
		}
	}

	placePieces(0, Black)
	placePawns(1, Black)
	placePawns(6, White)
	placePieces(7, White)

	return b
}

// Clone performs a deep copy of the board (for move validation).
func (b *Board) Clone() *Board {
	nb := &Board{
		Turn:           b.Turn,
		nextSuperID:    b.nextSuperID,
		nextEntangleID: b.nextEntangleID,
		HalfMoveClock:  b.HalfMoveClock,
		FullMoveNumber: b.FullMoveNumber,
		LinkedThisTurn: b.LinkedThisTurn,
		CastlingRights: b.CastlingRights,
	}
	if b.EnPassantTarget != nil {
		sq := *b.EnPassantTarget
		nb.EnPassantTarget = &sq
	}
	for r := 0; r < 8; r++ {
		for c := 0; c < 8; c++ {
			if b.Cells[r][c] != nil {
				p := *b.Cells[r][c]
				eids := make([]int, len(p.EntanglementIDs))
				copy(eids, p.EntanglementIDs)
				p.EntanglementIDs = eids
				nb.Cells[r][c] = &p
			}
		}
	}
	nb.SuperpositionGroups = make(map[int]*SuperpositionGroup, len(b.SuperpositionGroups))
	for id, sg := range b.SuperpositionGroups {
		sqs := make([]Square, len(sg.Squares))
		copy(sqs, sg.Squares)
		nb.SuperpositionGroups[id] = &SuperpositionGroup{
			ID:      sg.ID,
			Squares: sqs,
			Piece:   sg.Piece,
		}
	}
	nb.EntanglementGroups = make(map[int]*EntanglementGroup, len(b.EntanglementGroups))
	for id, eg := range b.EntanglementGroups {
		sqs := make([]Square, len(eg.Squares))
		copy(sqs, eg.Squares)
		edges := make([][2]Square, len(eg.Edges))
		copy(edges, eg.Edges)
		nb.EntanglementGroups[id] = &EntanglementGroup{
			ID:      eg.ID,
			Squares: sqs,
			Edges:   edges,
		}
	}
	return nb
}

// piece returns the piece at sq, or nil.
func (b *Board) piece(sq Square) *Piece {
	if !sq.inBounds() {
		return nil
	}
	return b.Cells[sq.Row][sq.Col]
}

// opponent returns the other color.
func opponent(c Color) Color {
	if c == White {
		return Black
	}
	return White
}
