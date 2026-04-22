package main

import (
	"lazydb/internal/db"
	"lazydb/internal/tui"

	"log"
	"os"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// get these values from cli
	db, err := db.NewDB("sqlite", "/home/pheonix/chinook.db")
	if err != nil {
		log.Fatal(err)
	}
	tables := db.FetchTables()

	columns := tui.TransformColumns([]string{"ID", "Table Name"})
	tableRows := [][]string{}

	for i, tableName := range tables {
		tableRows = append(tableRows, []string{strconv.Itoa(i + 1), tableName})
	}

	rows := tui.TransformRows(tableRows)

	// returns a new table
	t := tui.SetupNewTable(columns, rows)

	// m := model{table: t}
	var m *tui.Model
	m = tui.NewModel(t)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		println("Error:", err.Error())
		os.Exit(1)
	}
}
