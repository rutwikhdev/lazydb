package tui

import (
	"fmt"
	"lazydb/internal/db"
	"strconv"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen int

const (
	screenDBType screen = iota
	screenSQLitePath
	screenConnectionForm
	screenDatabases
	screenTables
	screenRows
)

type Model struct {
	stack          []screen
	termWidth      int
	termHeight     int

	dbConn         *db.Database
	dbTypeList     table.Model
	sqlitePath     textinput.Model
	connInputs     []textinput.Model
	focusedInput   int
	dbList         table.Model
	tableList      table.Model
	rowTable       table.Model
	selectedDBType string
	errMsg         string
}

func NewModel() *Model {
	dbTypeTable := makeDBTypeTable()

	ti := textinput.New()
	ti.Placeholder = "Enter path to SQLite database file"
	ti.Focus()
	ti.Width = 50

	inputs := make([]textinput.Model, 4)
	labels := []string{"Host", "Port", "Username", "Password"}
	placeholders := []string{db.DEFAULT_HOST, "3306", "", ""}
	for i := range inputs {
		inputs[i] = textinput.New()
		inputs[i].Placeholder = placeholders[i]
		inputs[i].Prompt = labels[i] + ": "
		inputs[i].Width = 50
		if i == 3 {
			inputs[i].EchoMode = textinput.EchoPassword
			inputs[i].EchoCharacter = '•'
		}
	}
	inputs[0].Focus()

	m := &Model{
		stack:        []screen{screenDBType},
		dbTypeList:   dbTypeTable,
		sqlitePath:   ti,
		connInputs:   inputs,
		focusedInput: 0,
		termWidth:    120,
		termHeight:   60,
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
		m.errMsg = ""
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	switch m.currentScreen() {
	case screenDBType:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "enter":
				selected := m.dbTypeList.SelectedRow()
				if len(selected) < 2 {
					return m, nil
				}
				m.selectedDBType = selected[1]
				if m.selectedDBType == db.DB_SQLITE {
					m.push(screenSQLitePath)
					m.sqlitePath.Focus()
				} else {
					m.push(screenConnectionForm)
					m.focusedInput = 0
					for i := range m.connInputs {
						m.connInputs[i].Blur()
					}
					m.connInputs[0].Focus()
					// Prefill port based on DB type
					switch m.selectedDBType {
					case db.DB_MYSQL, db.DB_MARIADB:
						m.connInputs[1].SetValue(strconv.Itoa(db.MYSQL_DEFAULT_PORT))
					case db.DB_POSTGRES:
						m.connInputs[1].SetValue(strconv.Itoa(db.POSTGRES_DEFAULT_PORT))
					}
				}
				return m, nil
			}
		}
		m.dbTypeList, cmd = m.dbTypeList.Update(msg)
		return m, cmd

	case screenSQLitePath:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc", "b":
				m.pop()
				return m, nil
			case "enter":
				path := m.sqlitePath.Value()
				if path == "" {
					m.errMsg = "Path cannot be empty"
					return m, nil
				}
				info := db.ConnectionInfo{
					Type: db.DB_SQLITE,
					Path: path,
				}
				database, err := db.NewDBFromInfo(info)
				if err != nil {
					m.errMsg = fmt.Sprintf("Connection failed: %v", err)
					return m, nil
				}
				m.dbConn = database
				tables := m.dbConn.FetchTables()
				m.tableList = makeTableListTable(tables)
				m.push(screenTables)
				m.errMsg = ""
				return m, nil
			}
		}
		m.sqlitePath, cmd = m.sqlitePath.Update(msg)
		return m, cmd

	case screenConnectionForm:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc", "b":
				m.pop()
				return m, nil
			case "tab":
				m.connInputs[m.focusedInput].Blur()
				m.focusedInput++
				if m.focusedInput >= len(m.connInputs) {
					m.focusedInput = 0
				}
				m.connInputs[m.focusedInput].Focus()
				return m, nil
			case "shift+tab":
				m.connInputs[m.focusedInput].Blur()
				m.focusedInput--
				if m.focusedInput < 0 {
					m.focusedInput = len(m.connInputs) - 1
				}
				m.connInputs[m.focusedInput].Focus()
				return m, nil
			case "enter":
				host := m.connInputs[0].Value()
				portStr := m.connInputs[1].Value()
				username := m.connInputs[2].Value()
				password := m.connInputs[3].Value()

				if host == "" {
					host = db.DEFAULT_HOST
				}
				port, err := strconv.Atoi(portStr)
				if err != nil || portStr == "" {
					switch m.selectedDBType {
					case db.DB_MYSQL, db.DB_MARIADB:
						port = db.MYSQL_DEFAULT_PORT
					case db.DB_POSTGRES:
						port = db.POSTGRES_DEFAULT_PORT
					}
				}

				info := db.ConnectionInfo{
					Type:     m.selectedDBType,
					Host:     host,
					Port:     port,
					Username: username,
					Password: password,
				}
				database, err := db.NewDBFromInfo(info)
				if err != nil {
					m.errMsg = fmt.Sprintf("Connection failed: %v", err)
					return m, nil
				}
				m.dbConn = database
				databases, err := m.dbConn.FetchDatabases()
				if err != nil {
					m.errMsg = fmt.Sprintf("Failed to fetch databases: %v", err)
					return m, nil
				}
				m.dbList = makeDatabaseTable(databases)
				m.push(screenDatabases)
				m.errMsg = ""
				return m, nil
			}
		}
		m.connInputs[m.focusedInput], cmd = m.connInputs[m.focusedInput].Update(msg)
		return m, cmd

	case screenDatabases:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "esc", "b":
				m.pop()
				return m, nil
			case "enter":
				selected := m.dbList.SelectedRow()
				if len(selected) < 2 {
					return m, nil
				}
				dbName := selected[1]
				if err := m.dbConn.SelectDatabase(dbName); err != nil {
					m.errMsg = fmt.Sprintf("Failed to select database: %v", err)
					return m, nil
				}
				tables := m.dbConn.FetchTables()
				m.tableList = makeTableListTable(tables)
				m.push(screenTables)
				m.errMsg = ""
				return m, nil
			}
		}
		m.dbList, cmd = m.dbList.Update(msg)
		return m, cmd

	case screenTables:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "esc", "b":
				m.pop()
				return m, nil
			case "enter":
				selected := m.tableList.SelectedRow()
				if len(selected) < 2 {
					return m, nil
				}
				tableName := selected[1]
				m.rowTable = makeRowTable(m.dbConn, tableName)
				m.push(screenRows)
				return m, nil
			}
		}
		m.tableList, cmd = m.tableList.Update(msg)
		return m, cmd

	case screenRows:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "esc", "b":
				m.pop()
				return m, nil
			}
		}
		m.rowTable, cmd = m.rowTable.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	var content string
	switch m.currentScreen() {
	case screenDBType:
		content = m.dbTypeList.View()
	case screenSQLitePath:
		content = m.sqlitePath.View()
	case screenConnectionForm:
		form := ""
		for _, input := range m.connInputs {
			form += input.View() + "\n"
		}
		content = form
	case screenDatabases:
		content = m.dbList.View()
	case screenTables:
		content = m.tableList.View()
	case screenRows:
		content = m.rowTable.View()
	}

	var statusBar string
	if m.errMsg != "" {
		statusBar = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("Error: " + m.errMsg)
	} else {
		switch m.currentScreen() {
		case screenDBType:
			statusBar = "↑↓: navigate • Enter: select • q: quit"
		case screenSQLitePath:
			statusBar = "Enter: connect • Esc: back"
		case screenConnectionForm:
			statusBar = "Tab: next field • Shift+Tab: prev field • Enter: connect • Esc: back"
		case screenDatabases:
			statusBar = "↑↓: navigate • Enter: select • Esc: back • q: quit"
		case screenTables:
			statusBar = "↑↓: navigate • Enter: open table • Esc: back • q: quit"
		case screenRows:
			statusBar = "↑↓: navigate • Esc: back • q: quit"
		}
	}

	if statusBar != "" {
		return content + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(statusBar)
	}
	return content
}

// Helper constructors

func makeDBTypeTable() table.Model {
	columns := []table.Column{
		{Title: "ID", Width: 4},
		{Title: "Database Type", Width: 20},
	}
	rows := []table.Row{}
	for i, dbType := range db.SupportedDBs {
		rows = append(rows, table.Row{strconv.Itoa(i + 1), dbType})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
	)
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

func makeTableListTable(tables []string) table.Model {
	columns := []table.Column{
		{Title: "ID", Width: 4},
		{Title: "Table Name", Width: 30},
	}
	rows := []table.Row{}
	for i, name := range tables {
		rows = append(rows, table.Row{strconv.Itoa(i + 1), name})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(25),
	)
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

func makeDatabaseTable(databases []string) table.Model {
	columns := []table.Column{
		{Title: "ID", Width: 4},
		{Title: "Database Name", Width: 30},
	}
	rows := []table.Row{}
	for i, name := range databases {
		rows = append(rows, table.Row{strconv.Itoa(i + 1), name})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(25),
	)
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

func makeRowTable(database *db.Database, tableName string) table.Model {
	rawColumns := database.GetColumns(tableName)
	columns := makeColumns(rawColumns)
	rawRows, err := database.GetRows(tableName, rawColumns)
	if err != nil {
		rawRows = [][]string{}
	}
	rows := makeRows(rawRows)

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(25),
	)
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

func makeColumns(columnNames []string) []table.Column {
	columns := []table.Column{}
	for _, name := range columnNames {
		columns = append(columns, table.Column{Title: name, Width: 12})
	}
	return columns
}

func makeRows(rowData [][]string) []table.Row {
	rows := []table.Row{}
	for _, row := range rowData {
		rows = append(rows, row)
	}
	return rows
}
