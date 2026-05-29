package tui

import (
	"fmt"
	"lazydb/internal/db"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	btable "github.com/evertras/bubble-table/table"
	"github.com/mattn/go-runewidth"
)

type screen int

const (
	screenDBType screen = iota
	screenSQLitePath
	screenConnectionForm
	screenDatabases
	screenTables
	screenRows
	screenDetail
)

const maxColWidth = 40

func dynamicPageSize(h int) int {
	if h < 13 {
		return 5
	}
	return h - 8
}

type Model struct {
	stack      []screen
	termWidth  int
	termHeight int
	pageSize   int

	dbConn         *db.Database
	dbTypeList     btable.Model
	sqlitePath     textinput.Model
	connInputs     []textinput.Model
	focusedInput   int
	dbList         btable.Model
	tableList      btable.Model
	rowTable       btable.Model
	selectedDBType string
	errMsg         string

	// Pagination state for row viewer
	rowTableName  string
	rowOffset     int
	rowLimit      int
	rowColumns    []string
	rowHasMore    bool
	primaryKeyCol string
	detailTable   btable.Model
}

var normalBorder = btable.Border{
	Top:            "─",
	Left:           "│",
	Right:          "│",
	Bottom:         "─",
	TopRight:       "┐",
	TopLeft:        "┌",
	BottomRight:    "┘",
	BottomLeft:     "└",
	TopJunction:    "┬",
	LeftJunction:   "├",
	RightJunction:  "┤",
	BottomJunction: "┴",
	InnerJunction:  "┼",
	InnerDivider:   "│",
}

func styledTable(columns []btable.Column, rows []btable.Row) btable.Model {
	return btable.New(columns).
		WithRows(rows).
		Focused(true).
		Border(normalBorder).
		WithBaseStyle(lipgloss.NewStyle().BorderForeground(lipgloss.Color("240"))).
		HeaderStyle(lipgloss.NewStyle().
			Padding(2).
			Foreground(lipgloss.Color("13")). // BrightMagenta
			Bold(true))
}

func NewModel() *Model {
	dbTypeTable := makeDBTypeTable(120, dynamicPageSize(60))

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
		pageSize:     dynamicPageSize(60),
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
		m.pageSize = dynamicPageSize(m.termHeight)

		m.dbTypeList = m.dbTypeList.WithTargetWidth(m.termWidth).WithPageSize(m.pageSize)
		m.dbList = m.dbList.WithTargetWidth(m.termWidth).WithPageSize(m.pageSize)
		m.tableList = m.tableList.WithTargetWidth(m.termWidth).WithPageSize(m.pageSize)
		m.rowTable = m.rowTable.WithMaxTotalWidth(m.termWidth).WithPageSize(m.pageSize)
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
				selected := m.dbTypeList.HighlightedRow()
				if selected.Data == nil {
					return m, nil
				}
				val, ok := selected.Data["name"]
				if !ok {
					return m, nil
				}
				m.selectedDBType = fmt.Sprintf("%v", val)
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
					case db.DB_MYSQL:
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
			case "esc":
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
				m.tableList = makeTableListTable(tables, m.termWidth, m.pageSize)
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
					case db.DB_MYSQL:
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
				m.dbList = makeDatabaseTable(databases, m.termWidth, m.pageSize)
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
				selected := m.dbList.HighlightedRow()
				if selected.Data == nil {
					return m, nil
				}
				val, ok := selected.Data["name"]
				if !ok {
					return m, nil
				}
				dbName := fmt.Sprintf("%v", val)
				if err := m.dbConn.SelectDatabase(dbName); err != nil {
					m.errMsg = fmt.Sprintf("Failed to select database: %v", err)
					return m, nil
				}
				tables := m.dbConn.FetchTables()
				m.tableList = makeTableListTable(tables, m.termWidth, m.pageSize)
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
				selected := m.tableList.HighlightedRow()
				if selected.Data == nil {
					return m, nil
				}
				val, ok := selected.Data["name"]
				if !ok {
					return m, nil
				}
				tableName := fmt.Sprintf("%v", val)
				m.openRowTable(tableName, m.termWidth, m.pageSize)
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
			case "h":
				m.rowTable = m.rowTable.ScrollLeft()
				return m, nil
			case "l":
				m.rowTable = m.rowTable.ScrollRight()
				return m, nil
			case "up", "k":
				idx := m.rowTable.GetHighlightedRowIndex()
				if idx == 0 {
					if m.rowOffset == 0 {
						// At the absolute first record — do nothing
						return m, nil
					}
					// Fetch previous window
					prevOffset := m.rowOffset - m.rowLimit
					if prevOffset < 0 {
						prevOffset = 0
					}
					m.fetchRowWindow(prevOffset)
					visible := m.rowTable.GetVisibleRows()
					if len(visible) > 0 {
						m.rowTable = m.rowTable.WithHighlightedRow(len(visible) - 1)
					}
					return m, nil
				}
			case "down", "j":
				visible := m.rowTable.GetVisibleRows()
				idx := m.rowTable.GetHighlightedRowIndex()
				if idx >= len(visible)-1 {
					if !m.rowHasMore {
						// At the absolute last record — do nothing
						return m, nil
					}
					// Fetch next window
					m.fetchRowWindow(m.rowOffset + m.rowLimit)
					return m, nil
				}
			case "enter":
				if m.primaryKeyCol == "" {
					m.errMsg = "No primary key found for this table"
					return m, nil
				}
				selected := m.rowTable.HighlightedRow()
				if selected.Data == nil {
					return m, nil
				}
				pkVal, ok := selected.Data[m.primaryKeyCol]
				if !ok {
					return m, nil
				}
				row, err := m.dbConn.GetRowByPK(m.rowTableName, m.rowColumns, m.primaryKeyCol, fmt.Sprintf("%v", pkVal))
				if err != nil {
					m.errMsg = fmt.Sprintf("Failed to fetch record: %v", err)
					return m, nil
				}
				m.detailTable = makeDetailTable(row, m.rowColumns, m.termWidth, m.pageSize)
				m.push(screenDetail)
				m.errMsg = ""
				return m, nil
			}
		}
		m.rowTable, cmd = m.rowTable.Update(msg)
		return m, cmd

	case screenDetail:
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
		m.detailTable, cmd = m.detailTable.Update(msg)
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
		var form strings.Builder
		for _, input := range m.connInputs {
			form.WriteString(input.View())
			form.WriteByte('\n')
		}
		content = form.String()
	case screenDatabases:
		content = m.dbList.View()
	case screenTables:
		content = m.tableList.View()
	case screenRows:
		content = m.rowTable.View()
	case screenDetail:
		content = m.detailTable.View()
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
			start := m.rowOffset + 1
			end := m.rowOffset + len(m.rowTable.GetVisibleRows())
			statusBar = fmt.Sprintf("↑↓: navigate • h/l or shift+←→: scroll • Enter: view • Esc: back • q: quit • Rows %d-%d", start, end)
		case screenDetail:
			statusBar = "Esc: back • q: quit"
		}
	}

	if statusBar != "" {
		return content + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(statusBar)
	}
	return content
}

// Helper constructors

func makeDBTypeTable(width, pageSize int) btable.Model {
	columns := []btable.Column{
		btable.NewColumn("id", "ID", 4),
		btable.NewFlexColumn("name", "Database Type", 1),
	}
	rows := []btable.Row{}
	for i, dbType := range db.SupportedDBs {
		rows = append(rows, btable.NewRow(btable.RowData{
			"id":   strconv.Itoa(i + 1),
			"name": dbType,
		}))
	}
	return styledTable(columns, rows).WithTargetWidth(width).WithPageSize(pageSize)
}

func makeTableListTable(tables []string, width, pageSize int) btable.Model {
	columns := []btable.Column{
		btable.NewColumn("id", "ID", 4),
		btable.NewFlexColumn("name", "Table Name", 1),
	}
	rows := []btable.Row{}
	for i, name := range tables {
		rows = append(rows, btable.NewRow(btable.RowData{
			"id":   strconv.Itoa(i + 1),
			"name": name,
		}))
	}
	return styledTable(columns, rows).WithTargetWidth(width).WithPageSize(pageSize)
}

func makeDatabaseTable(databases []string, width, pageSize int) btable.Model {
	columns := []btable.Column{
		btable.NewColumn("id", "ID", 4),
		btable.NewFlexColumn("name", "Database Name", 1),
	}
	rows := []btable.Row{}
	for i, name := range databases {
		rows = append(rows, btable.NewRow(btable.RowData{
			"id":   strconv.Itoa(i + 1),
			"name": name,
		}))
	}
	return styledTable(columns, rows).WithTargetWidth(width).WithPageSize(pageSize)
}

func (m *Model) openRowTable(tableName string, width, pageSize int) {
	cols := m.dbConn.GetColumns(tableName)
	m.rowTableName = tableName
	m.rowColumns = cols
	m.primaryKeyCol = m.dbConn.GetPrimaryKey(tableName)
	m.rowOffset = 0
	m.rowLimit = 500
	m.rowHasMore = true
	m.fetchRowWindow(0)
}

func (m *Model) fetchRowWindow(offset int) {
	rows, err := m.dbConn.GetRowsPaginated(m.rowTableName, m.rowColumns, offset, m.rowLimit)
	if err != nil {
		m.errMsg = fmt.Sprintf("Failed to fetch rows: %v", err)
		return
	}

	if len(rows) == 0 {
		m.rowHasMore = false
		return
	}

	m.rowOffset = offset
	m.rowHasMore = len(rows) == m.rowLimit

	colWidths := make([]int, len(m.rowColumns))
	for i, name := range m.rowColumns {
		colWidths[i] = runewidth.StringWidth(name)
	}
	for _, row := range rows {
		for j, val := range row {
			if j < len(colWidths) {
				w := runewidth.StringWidth(val)
				if w > colWidths[j] {
					colWidths[j] = w
				}
			}
		}
	}

	finalWidths := make([]int, len(m.rowColumns))
	for i := range colWidths {
		finalWidths[i] = colWidths[i] + 1
	}

	totalColWidth := 0
	for _, w := range finalWidths {
		totalColWidth += w
	}

	if totalColWidth < m.termWidth {
		for i := range finalWidths {
			proportion := float64(finalWidths[i]) / float64(totalColWidth)
			finalWidths[i] = int(proportion * float64(m.termWidth-3))
		}
	} else {
		for i := range finalWidths {
			finalWidths[i] = min(finalWidths[i], maxColWidth)
		}
	}

	columns := make([]btable.Column, len(m.rowColumns))
	for i, name := range m.rowColumns {
		columns[i] = btable.NewColumn(name, name, finalWidths[i])
	}

	bRows := make([]btable.Row, len(rows))
	for i, row := range rows {
		data := make(btable.RowData, len(row))
		for j, val := range row {
			data[m.rowColumns[j]] = val
		}
		bRows[i] = btable.NewRow(data)
	}

	m.rowTable = styledTable(columns, bRows).
		WithPageSize(m.pageSize).
		WithMaxTotalWidth(m.termWidth).
		WithHorizontalFreezeColumnCount(1)
}

func makeDetailTable(row []string, columns []string, width, pageSize int) btable.Model {
	colWidth := width / 4

	colCol := btable.NewColumn("column", "Column", colWidth).
		WithStyle(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("13")))
	valCol := btable.NewFlexColumn("value", "Value", 1)

	rows := make([]btable.Row, len(columns))
	for i, col := range columns {
		val := ""
		if i < len(row) {
			val = row[i]
		}
		rows[i] = btable.NewRow(btable.RowData{
			"column": col,
			"value":  val,
		})
	}

	return btable.New([]btable.Column{colCol, valCol}).
		WithRows(rows).
		Focused(true).
		Border(normalBorder).
		WithBaseStyle(lipgloss.NewStyle().BorderForeground(lipgloss.Color("240"))).
		HeaderStyle(lipgloss.NewStyle().
			Padding(2).
			Foreground(lipgloss.Color("13")).
			Bold(true)).
		WithMultiline(true).
		WithTargetWidth(width)
}
