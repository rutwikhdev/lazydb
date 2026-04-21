package main

import (
	"lazydb/internal/tui"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	columns := tui.Columns([]string{"Name", "Language"})

	// TODO: fetch from query engine and send to list below
	rows := tui.Rows([][]string{
		{"1", "Alice", "Go"},
		{"2", "Bob", "Python"},
		{"3", "Charlie", "Rust"},
	})

	// returns a new table
	t := tui.SetupNewTable(columns, rows)

	// m := model{table: t}
	m := tui.Model{Table: t}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		println("Error:", err.Error())
		os.Exit(1)
	}
}
