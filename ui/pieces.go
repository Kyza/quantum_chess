package ui

import (
	"image/color"

	"quantum_chess/game"
)

// PieceSymbol returns the Unicode chess glyph for a piece.
func PieceSymbol(p *game.Piece) string {
	if p == nil {
		return ""
	}
	switch p.Type {
	case game.King:
		return "♚"
	case game.Queen:
		return "♛"
	case game.Rook:
		return "♜"
	case game.Bishop:
		return "♝"
	case game.Knight:
		return "♞"
	case game.Pawn:
		return "♟"
	}
	return "?"
}

// PieceColor returns the rendering color for a piece.
func PieceColor(p *game.Piece) color.NRGBA {
	if p.Color == game.White {
		return color.NRGBA{R: 255, G: 255, B: 230, A: 255}
	}
	return color.NRGBA{R: 20, G: 20, B: 20, A: 255}
}
