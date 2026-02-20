package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"

	"quantum_chess/game"
)

// ─── colours ────────────────────────────────────────────────────────────────

var (
	colLight     = color.NRGBA{R: 0xF0, G: 0xD9, B: 0xB5, A: 0xFF}
	colDark      = color.NRGBA{R: 0xB5, G: 0x88, B: 0x63, A: 0xFF}
	colSelected  = color.NRGBA{R: 0x50, G: 0x80, B: 0xFF, A: 0x88}
	colLegal     = color.NRGBA{R: 0x00, G: 0xCC, B: 0x44, A: 0x88}
	colSplit     = color.NRGBA{R: 0xAA, G: 0x00, B: 0xFF, A: 0x88}
	colCheck     = color.NRGBA{R: 0xFF, G: 0x00, B: 0x00, A: 0x66}
	colLinkSel   = color.NRGBA{R: 0xFF, G: 0xCC, B: 0x00, A: 0x99}
	colTransp    = color.NRGBA{A: 0}
	colEntangle  = color.NRGBA{R: 0xFF, G: 0x88, B: 0x00, A: 0xCC}
)

// ─── state machine ───────────────────────────────────────────────────────────

type uiState int

const (
	stateIdle uiState = iota
	stateSelected
	stateLinkA // waiting for first piece in link mode
	stateLinkB // waiting for second piece
)

const maxEntLines = 64

// ─── widget ──────────────────────────────────────────────────────────────────

// BoardWidget is a Fyne widget that renders the quantum chess board.
type BoardWidget struct {
	widget.BaseWidget

	Board *game.Board

	state       uiState
	selected    game.Square
	legalMoves  []game.Square
	splitTargets []game.Square
	linkA       game.Square

	OnStatusChange func(string)
	OnGameOver     func(string)
	LinkButton     *widget.Button
}

// NewBoardWidget creates a BoardWidget for the given board.
func NewBoardWidget(b *game.Board) *BoardWidget {
	bw := &BoardWidget{Board: b}
	bw.ExtendBaseWidget(bw)
	return bw
}

// Reset replaces the board with a fresh starting position.
func (bw *BoardWidget) Reset() {
	bw.Board = game.NewBoard()
	bw.state = stateIdle
	bw.legalMoves = nil
	bw.splitTargets = nil
	bw.Refresh()
	bw.notify()
}

// ToggleLinkMode is called by the Link/Unlink button.
func (bw *BoardWidget) ToggleLinkMode(btn *widget.Button) {
	if bw.state == stateLinkA || bw.state == stateLinkB {
		bw.state = stateIdle
		btn.SetText("Link/Unlink")
		bw.Refresh()
		bw.notify()
		return
	}
	bw.state = stateLinkA
	btn.SetText("Cancel Link")
	bw.legalMoves = nil
	bw.splitTargets = nil
	bw.Refresh()
	bw.notify()
}

// Tapped handles square clicks.
func (bw *BoardWidget) Tapped(ev *fyne.PointEvent) {
	if bw.Board.GameOver {
		return
	}
	sz := bw.Size()
	cw := sz.Width / 8
	ch := sz.Height / 8
	col := int(ev.Position.X / cw)
	row := int(ev.Position.Y / ch)
	if row < 0 || row > 7 || col < 0 || col > 7 {
		return
	}
	sq := game.Square{Row: row, Col: col}
	bw.handleClick(sq)
}

func (bw *BoardWidget) handleClick(sq game.Square) {
	b := bw.Board

	switch bw.state {
	case stateLinkA:
		p := b.Cells[sq.Row][sq.Col]
		if p == nil || p.Color != b.Turn {
			return
		}
		bw.linkA = sq
		bw.state = stateLinkB
		bw.Refresh()

	case stateLinkB:
		p := b.Cells[sq.Row][sq.Col]
		if p == nil || p.Color != b.Turn {
			bw.state = stateLinkA
			bw.Refresh()
			return
		}
		if sq == bw.linkA {
			bw.state = stateLinkA
			bw.Refresh()
			return
		}
		// Try link; if already linked, unlink.
		err := b.Link(bw.linkA, sq)
		if err != nil {
			// Maybe already linked — try unlink.
			_ = b.Unlink(bw.linkA, sq)
		}
		bw.state = stateIdle
		if bw.LinkButton != nil {
			bw.LinkButton.SetText("Link/Unlink")
		}
		bw.legalMoves = nil
		bw.splitTargets = nil
		bw.Refresh()
		bw.notify()

	case stateIdle:
		p := b.Cells[sq.Row][sq.Col]
		if p == nil || p.Color != b.Turn {
			return
		}
		bw.selected = sq
		bw.legalMoves = b.LegalMoves(sq)
		bw.splitTargets = b.LegalSplitTargets(sq)
		bw.state = stateSelected
		bw.Refresh()

	case stateSelected:
		// Check if clicking a legal move square.
		if bw.inList(sq, bw.legalMoves) {
			err := b.ApplyMove(bw.selected, sq)
			if err == nil {
				bw.state = stateIdle
				bw.legalMoves = nil
				bw.splitTargets = nil
				bw.Refresh()
				bw.notify()
				bw.checkGameOver()
				return
			}
		}
		// Check quantum split.
		if bw.inList(sq, bw.splitTargets) {
			err := b.ApplyQuantumSplit(bw.selected, sq)
			if err == nil {
				bw.state = stateIdle
				bw.legalMoves = nil
				bw.splitTargets = nil
				bw.Refresh()
				bw.notify()
				return
			}
		}
		// Click on another friendly piece → re-select.
		p := b.Cells[sq.Row][sq.Col]
		if p != nil && p.Color == b.Turn {
			bw.selected = sq
			bw.legalMoves = b.LegalMoves(sq)
			bw.splitTargets = b.LegalSplitTargets(sq)
			bw.Refresh()
			return
		}
		// Otherwise cancel selection.
		bw.state = stateIdle
		bw.legalMoves = nil
		bw.splitTargets = nil
		bw.Refresh()
	}
}

func (bw *BoardWidget) inList(sq game.Square, list []game.Square) bool {
	for _, s := range list {
		if s == sq {
			return true
		}
	}
	return false
}

func (bw *BoardWidget) notify() {
	if bw.OnStatusChange == nil {
		return
	}
	b := bw.Board
	var msg string
	if b.Turn == game.White {
		msg = "White's turn"
	} else {
		msg = "Black's turn"
	}
	if b.IsInCheck(b.Turn) {
		msg += " — Check!"
	}
	if bw.state == stateLinkA || bw.state == stateLinkB {
		msg += " (Link mode: pick pieces)"
	}
	bw.OnStatusChange(msg)
}

func (bw *BoardWidget) checkGameOver() {
	if bw.Board.GameOver && bw.OnGameOver != nil {
		var winner string
		if bw.Board.Winner == game.White {
			winner = "White"
		} else {
			winner = "Black"
		}
		bw.OnGameOver(winner)
	}
}

// MinSize satisfies fyne.Widget.
func (bw *BoardWidget) MinSize() fyne.Size {
	return fyne.NewSize(480, 480)
}

// CreateRenderer satisfies fyne.Widget.
func (bw *BoardWidget) CreateRenderer() fyne.WidgetRenderer {
	r := &boardRenderer{bw: bw}
	// Allocate all canvas objects once.
	for i := 0; i < 64; i++ {
		r.bgRects[i] = canvas.NewRectangle(colLight)
		r.hlRects[i] = canvas.NewRectangle(colTransp)
		t := canvas.NewText("", color.White)
		t.TextSize = 40
		t.Alignment = fyne.TextAlignCenter
		r.pieces[i] = t
		g := canvas.NewText("", color.NRGBA{R: 0xAA, G: 0x00, B: 0xFF, A: 0x80})
		g.TextSize = 12
		r.ghosts[i] = g
	}
	for i := 0; i < maxEntLines; i++ {
		l := canvas.NewLine(colEntangle)
		l.StrokeWidth = 2
		r.lines[i] = l
	}
	// Build objects list (back to front).
	var objs []fyne.CanvasObject
	for i := 0; i < 64; i++ {
		objs = append(objs, r.bgRects[i])
	}
	for i := 0; i < 64; i++ {
		objs = append(objs, r.hlRects[i])
	}
	for i := 0; i < maxEntLines; i++ {
		objs = append(objs, r.lines[i])
	}
	for i := 0; i < 64; i++ {
		objs = append(objs, r.pieces[i])
	}
	for i := 0; i < 64; i++ {
		objs = append(objs, r.ghosts[i])
	}
	r.objects = objs
	return r
}

// ─── renderer ────────────────────────────────────────────────────────────────

type boardRenderer struct {
	bw      *BoardWidget
	bgRects [64]*canvas.Rectangle
	hlRects [64]*canvas.Rectangle
	pieces  [64]*canvas.Text
	ghosts  [64]*canvas.Text
	lines   [maxEntLines]*canvas.Line
	objects []fyne.CanvasObject
}

func (r *boardRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *boardRenderer) Destroy()                     {}

func (r *boardRenderer) Layout(size fyne.Size) {
	cw := size.Width / 8
	ch := size.Height / 8

	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			i := row*8 + col
			pos := fyne.NewPos(float32(col)*cw, float32(row)*ch)
			sz := fyne.NewSize(cw, ch)
			r.bgRects[i].Move(pos)
			r.bgRects[i].Resize(sz)
			r.hlRects[i].Move(pos)
			r.hlRects[i].Resize(sz)
			// Piece text centered.
			r.pieces[i].Move(fyne.NewPos(float32(col)*cw, float32(row)*ch+ch*0.05))
			r.pieces[i].Resize(fyne.NewSize(cw, ch))
			// Ghost label (ψ) bottom-right corner.
			r.ghosts[i].Move(fyne.NewPos(float32(col)*cw+cw*0.6, float32(row)*ch+ch*0.65))
			r.ghosts[i].Resize(fyne.NewSize(cw*0.4, ch*0.35))
		}
	}
	r.refresh(size)
}

func (r *boardRenderer) MinSize() fyne.Size {
	return r.bw.MinSize()
}

func (r *boardRenderer) Refresh() {
	r.refresh(r.bw.Size())
}

func (r *boardRenderer) refresh(size fyne.Size) {
	bw := r.bw
	b := bw.Board
	cw := size.Width / 8
	ch := size.Height / 8

	// Build highlight sets.
	legalSet := make(map[game.Square]bool)
	splitSet := make(map[game.Square]bool)
	for _, sq := range bw.legalMoves {
		legalSet[sq] = true
	}
	for _, sq := range bw.splitTargets {
		splitSet[sq] = true
	}

	// Find check squares.
	checkSet := make(map[game.Square]bool)
	for rr := 0; rr < 8; rr++ {
		for cc := 0; cc < 8; cc++ {
			p := b.Cells[rr][cc]
			if p != nil && p.Type == game.King && b.IsInCheck(p.Color) {
				checkSet[game.Square{Row: rr, Col: cc}] = true
			}
		}
	}

	// Superposition squares.
	superSet := make(map[game.Square]bool)
	for _, sg := range b.SuperpositionGroups {
		for _, sq := range sg.Squares {
			superSet[sq] = true
		}
	}

	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			i := row*8 + col
			sq := game.Square{Row: row, Col: col}

			// Background colour.
			if (row+col)%2 == 0 {
				r.bgRects[i].FillColor = colLight
			} else {
				r.bgRects[i].FillColor = colDark
			}
			r.bgRects[i].Refresh()

			// Highlight overlay.
			var hl color.Color = colTransp
			switch {
			case checkSet[sq]:
				hl = colCheck
			case bw.state == stateSelected && sq == bw.selected:
				hl = colSelected
			case (bw.state == stateLinkB) && sq == bw.linkA:
				hl = colLinkSel
			case legalSet[sq]:
				hl = colLegal
			case splitSet[sq]:
				hl = colSplit
			}
			r.hlRects[i].FillColor = hl
			r.hlRects[i].Refresh()

			// Piece text.
			p := b.Cells[row][col]
			if p != nil {
				sym := PieceSymbol(p)
				pc := PieceColor(p)
				if superSet[sq] {
					pc.A = 0x66 // ghost: ~40% opacity
				}
				r.pieces[i].Text = sym
				r.pieces[i].Color = pc
				// Ghost ψ label.
				if superSet[sq] {
					r.ghosts[i].Text = "ψ"
				} else {
					r.ghosts[i].Text = ""
				}
			} else {
				r.pieces[i].Text = ""
				r.ghosts[i].Text = ""
			}
			r.pieces[i].Refresh()
			r.ghosts[i].Refresh()
		}
	}

	// Entanglement lines.
	lineIdx := 0
	for _, eg := range b.EntanglementGroups {
		for _, edge := range eg.Edges {
			if lineIdx >= maxEntLines {
				break
			}
			a, bb := edge[0], edge[1]
			ax := (float32(a.Col) + 0.5) * cw
			ay := (float32(a.Row) + 0.5) * ch
			bx := (float32(bb.Col) + 0.5) * cw
			by := (float32(bb.Row) + 0.5) * ch
			r.lines[lineIdx].Position1 = fyne.NewPos(ax, ay)
			r.lines[lineIdx].Position2 = fyne.NewPos(bx, by)
			r.lines[lineIdx].StrokeColor = colEntangle
			r.lines[lineIdx].Refresh()
			lineIdx++
		}
	}
	// Hide unused lines.
	for i := lineIdx; i < maxEntLines; i++ {
		r.lines[i].Position1 = fyne.NewPos(0, 0)
		r.lines[i].Position2 = fyne.NewPos(0, 0)
		r.lines[i].Refresh()
	}
	_ = cw
	_ = ch
}
