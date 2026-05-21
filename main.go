package main

import (
	"lazydb/internal/tui"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	m := tui.NewModel()

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		println("Error:", err.Error())
		os.Exit(1)
	}
}
