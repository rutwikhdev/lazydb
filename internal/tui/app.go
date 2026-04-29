package tui

import (
	"lazydb/internal/db"
	"log"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen int

const (
	screenTables screen = iota
	screenRows
	screenDetails
)

type Model struct {
	stack      []screen
	termWidth  int
	termHeight int

	TableList   table.Model
	RowTable    table.Model
	DetailTable table.Model
}

func NewModel(t table.Model) *Model {
	m := &Model{
		stack:      []screen{screenTables},
		TableList:  t,
		termWidth:  120,
		termHeight: 60,
	}
	return m
}

func (m Model) currentScreen() screen {
	return m.stack[len(m.stack)-1]
}

func (m *Model) push(s screen) {
	m.stack = append(m.stack, s)
}

func (m *Model) pop() {
	if len(m.stack) > 1 {
		m.stack = m.stack[:len(m.stack)-1]
	}
}

func (m Model) Init() tea.Cmd {
	m.stack = []screen{screenTables}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.currentScreen() {
	case screenTables:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "enter":
				selected := m.TableList.SelectedRow()
				m.RowTable = makeRowTable(selected[1])

				m.push(screenRows)
				return m, nil
			}
		}
		m.TableList, cmd = m.TableList.Update(msg)
		return m, cmd
	case screenRows:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "b", "esc":
				m.pop()
				return m, nil
			}

		}
		m.RowTable, cmd = m.RowTable.Update(msg)
		return m, cmd

	}

	return m, nil
}

func (m Model) View() string {
	switch m.currentScreen() {
	case screenTables:
		return m.TableList.View()
	case screenRows:
		return m.RowTable.View()
	case screenDetails:
		return m.DetailTable.View()
	}

	return ""
}

func SetupNewTable(columns []table.Column, rows []table.Row) table.Model {
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(28),
	)

	// Styling
	s := table.DefaultStyles()
	s.Header = s.Header.
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("62")).
		MarginBottom(1).
		Underline(true).
		Bold(true)

	t.SetStyles(s)

	return t
}

func TransformColumns(columnNames []string) []table.Column {
	columns := []table.Column{}

	for _, name := range columnNames {
		columns = append(columns, table.Column{Title: name, Width: 12})
	}

	return columns
}

func TransformRows(rowData [][]string) []table.Row {
	rows := []table.Row{}

	for _, row := range rowData {
		rows = append(rows, row)
	}

	return rows
}

func makeRowTable(tableName string) table.Model {
	db, err := db.NewDB("sqlite", "/home/pheonix/chinook.db")
	if err != nil {
		log.Fatal(err)
	}

	rawColumns := db.GetColumns(tableName)
	columns := TransformColumns(rawColumns)
	rawRows, err := db.GetRows(tableName, rawColumns)
	rows := TransformRows(rawRows)

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(25),
	)

	// Styling
	s := table.DefaultStyles()
	s.Header = s.Header.
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("62")).
		MarginBottom(1).
		Underline(true).
		Bold(true)

	t.SetStyles(s)

	return t
}
