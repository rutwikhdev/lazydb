package tui

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	Table table.Model
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.Table, cmd = m.Table.Update(msg)

	return m, cmd
}

func (m Model) View() string {
	return "\n" + m.Table.View()
}

func SetupNewTable(columns []table.Column, rows []table.Row) table.Model {
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(5),
	)

	// Styling
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Bold(true)

	t.SetStyles(s)

	return t
}

func Columns(columnNames []string) []table.Column {
	columns := []table.Column{
		{Title: "ID", Width: 4},
	}
	for _, name := range columnNames {
		columns = append(columns, table.Column{Title: name, Width: 10})
	}

	return columns
}

func Rows(rowData [][]string) []table.Row {
	rows := []table.Row{}

	for _, row := range rowData {
		rows = append(rows, row)
	}

	return rows
}
