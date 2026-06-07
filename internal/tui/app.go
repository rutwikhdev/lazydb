package tui

import (
	"fmt"
	"lazydb/internal/db"
	"lazydb/internal/utils"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	btable "github.com/evertras/bubble-table/table"
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
	screenUpdate
	screenDelete
)

const maxColWidth = 40

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

	// Update state
	updateInputs  []textinput.Model
	updateFocused int
	pkIdx         int
	confirmUpdate bool
	confirmDelete bool
	updatePKVal   string
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

func (m *Model) currentScreen() screen {
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
				tables, err := m.dbConn.FetchTables()
				if err != nil {
					m.errMsg = err.Error()
					return m, nil
				}

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
				tables, err := m.dbConn.FetchTables()
				if err != nil {
					m.errMsg = fmt.Sprintf("Failed to fetch tables: %v", err.Error())
					return m, nil
				}

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
				err := m.openRowTable(tableName, m.termWidth, m.pageSize)
				if err != nil {
					m.errMsg = err.Error()
					return m, nil
				}

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
					prevOffset = max(prevOffset, 0)
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
			case "U":
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
				m.updatePKVal = fmt.Sprintf("%v", pkVal)
				row, err := m.dbConn.GetRowByPK(m.rowTableName, m.rowColumns, m.primaryKeyCol, m.updatePKVal)
				if err != nil {
					m.errMsg = fmt.Sprintf("Failed to fetch record: %v", err)
					return m, nil
				}
				m.updateInputs = make([]textinput.Model, len(m.rowColumns))
				m.pkIdx = -1
				firstEditable := -1
				for i, col := range m.rowColumns {
					if col == m.primaryKeyCol {
						m.pkIdx = i
					}
					ti := textinput.New()
					ti.Prompt = col + ": "
					ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
					ti.Width = 50
					if i < len(row) {
						ti.SetValue(row[i])
					}
					m.updateInputs[i] = ti
					if firstEditable == -1 && col != m.primaryKeyCol {
						firstEditable = i
					}
				}
				if firstEditable >= 0 {
					m.updateInputs[firstEditable].Focus()
					m.updateFocused = firstEditable
				} else {
					m.updateFocused = 0
				}
				m.confirmUpdate = false
				m.push(screenUpdate)
				m.errMsg = ""
				return m, nil
			case "D":
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
				m.updatePKVal = fmt.Sprintf("%v", pkVal)
				m.confirmDelete = true
				m.push(screenDelete)
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

	case screenUpdate:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if m.confirmUpdate {
				switch msg.String() {
				case "y":
					values := make([]string, len(m.updateInputs))
					for i, ti := range m.updateInputs {
						values[i] = ti.Value()
					}
					err := m.dbConn.UpdateRow(m.rowTableName, m.rowColumns, values, m.primaryKeyCol, m.updatePKVal)
					if err != nil {
						m.errMsg = fmt.Sprintf("Failed to update: %v", err)
					} else {
						m.errMsg = ""
						m.fetchRowWindow(m.rowOffset)
					}
					m.pop()
					return m, nil
				case "n", "esc":
					m.confirmUpdate = false
					return m, nil
				}
				return m, nil
			}
			switch msg.String() {
			case "esc":
				m.pop()
				return m, nil
			case "tab":
				m.updateInputs[m.updateFocused].Blur()
				for {
					m.updateFocused++
					if m.updateFocused >= len(m.updateInputs) {
						m.updateFocused = 0
					}
					if m.updateFocused != m.pkIdx {
						break
					}
				}
				m.updateInputs[m.updateFocused].Focus()
				return m, nil
			case "shift+tab":
				m.updateInputs[m.updateFocused].Blur()
				for {
					m.updateFocused--
					if m.updateFocused < 0 {
						m.updateFocused = len(m.updateInputs) - 1
					}
					if m.updateFocused != m.pkIdx {
						break
					}
				}
				m.updateInputs[m.updateFocused].Focus()
				return m, nil
			case "enter":
				m.confirmUpdate = true
				return m, nil
			}
		}
		m.updateInputs[m.updateFocused], cmd = m.updateInputs[m.updateFocused].Update(msg)
		return m, cmd
	case screenDelete:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if m.confirmDelete {
				switch msg.String() {
				case "y":
					logger := utils.NewLogger()
					logger.Println("Deleting.........")
					err := m.dbConn.DeleteRow(m.rowTableName, m.primaryKeyCol, m.updatePKVal)
					if err != nil {
						m.errMsg = fmt.Sprintf("Failed to delete: %v", err)
					} else {
						m.errMsg = ""
						m.fetchRowWindow(m.rowOffset)
					}
					m.confirmDelete = false
					m.pop()
					return m, nil
				case "n", "esc":
					m.confirmDelete = false
					m.pop()
					return m, nil
				}
				return m, nil
			}
			switch msg.String() {
			case "esc":
				m.pop()
				return m, nil
			case "enter":
				m.confirmDelete = true
				return m, nil
			}
		}
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
	case screenDelete:
		bg := m.rowTable.View()
		var formContent strings.Builder
		if m.confirmDelete {
			formContent.WriteString("Confirm delete? (y/n)\n")
		}
		modalStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("13")).
			Padding(1, 2).
			Width(60)
		modal := modalStyle.Render(formContent.String())
		content = overlay(bg, modal, m.termWidth, m.termHeight)
	case screenUpdate:
		bg := m.rowTable.View()
		var formContent strings.Builder
		if m.confirmUpdate {
			formContent.WriteString("Confirm update? (y/n)\n")
		} else {
			for i, input := range m.updateInputs {
				if i == m.pkIdx {
					formContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(m.rowColumns[i] + ": " + input.Value()))
				} else {
					formContent.WriteString(input.View())
				}
				formContent.WriteByte('\n')
			}
			hint := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(`Hit "Enter" when done`)
			formContent.WriteString(lipgloss.NewStyle().Width(56).Align(lipgloss.Right).Render(hint))
		}
		modalStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("13")).
			Padding(1, 2).
			Width(60)
		modal := modalStyle.Render(formContent.String())
		content = overlay(bg, modal, m.termWidth, m.termHeight)
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
		case screenUpdate:
			if m.confirmUpdate {
				statusBar = "y: confirm • n/esc: cancel"
			} else {
				statusBar = "Tab: next field • Shift+Tab: prev field • Enter: save • Esc: cancel"
			}
		case screenDelete:
			if m.confirmDelete {
				statusBar = "y: confirm • n/esc: cancel"
			} else {
				statusBar = "Enter: confirm delete • Esc: cancel"
			}
		}
	}

	if statusBar != "" {
		return content + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(statusBar)
	}
	return content
}
