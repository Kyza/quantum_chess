package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"quantum_chess/game"
)

// Run starts the Quantum Chess GUI application.
func Run() {
	a := app.New()
	w := a.NewWindow("Quantum Chess")

	board := game.NewBoard()
	bw := NewBoardWidget(board)

	statusLabel := widget.NewLabel("White's turn")

	var linkBtn *widget.Button
	linkBtn = widget.NewButton("Link/Unlink", func() {
		bw.ToggleLinkMode(linkBtn)
	})
	bw.LinkButton = linkBtn

	var newGameBtn *widget.Button
	newGameBtn = widget.NewButton("New Game", func() {
		bw.Reset()
		statusLabel.SetText("White's turn")
		linkBtn.SetText("Link/Unlink")
		_ = newGameBtn
	})

	bw.OnStatusChange = func(s string) {
		statusLabel.SetText(s)
	}
	bw.OnGameOver = func(winner string) {
		dialog.ShowInformation("Game Over", winner+" wins!", w)
	}

	toolbar := container.NewHBox(newGameBtn, linkBtn)
	content := container.NewBorder(toolbar, statusLabel, nil, nil, bw)

	w.SetContent(content)
	w.Resize(fyne.NewSize(640, 720))
	w.ShowAndRun()
}
