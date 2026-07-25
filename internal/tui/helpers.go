package tui

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	btable "github.com/evertras/bubble-table/table"
)

func dynamicPageSize(h int) int {
	if h < 13 {
		return 5
	}
	return h - 8
}

func contentHeight(termHeight int) int {
	// returns the height available for the main content area
	// status bar takes up 2 lines (one blank separator line + the status line).
	return termHeight - 2
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
			Bold(true)).
		HighlightStyle(lipgloss.NewStyle().
			Background(lipgloss.Color("#444")).
			Foreground(lipgloss.Color("#eee")))
}

// makeListTable builds a two-column list table (ID + name) used for the
// database type, database, and table selection screens.
func makeListTable(title string, items []string, width, pageSize, minHeight int) btable.Model {
	columns := []btable.Column{
		btable.NewColumn("id", "ID", 4),
		btable.NewFlexColumn("name", title, 1),
	}
	rows := make([]btable.Row, 0, len(items))
	for i, name := range items {
		rows = append(rows, btable.NewRow(btable.RowData{
			"id":   strconv.Itoa(i + 1),
			"name": name,
		}))
	}
	return styledTable(columns, rows).WithTargetWidth(width).WithPageSize(pageSize).WithMinimumHeight(minHeight)
}

// fitTable resizes a table to the current terminal dimensions. Scrollable
// tables cap their total width and scroll horizontally instead of flexing
// columns to fill it.
func (m *Model) fitTable(t btable.Model, scrollable bool) btable.Model {
	if scrollable {
		t = t.WithMaxTotalWidth(m.termWidth)
	} else {
		t = t.WithTargetWidth(m.termWidth)
	}
	return t.WithPageSize(m.pageSize).WithMinimumHeight(contentHeight(m.termHeight))
}

func makeDetailTable(row []string, columns []string, width, pageSize, minHeight int) btable.Model {
	colWidth := width / 4

	colCol := btable.NewColumn("column", "Column", colWidth).
		WithStyle(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245")))
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
		WithTargetWidth(width).
		WithPageSize(pageSize).
		WithMinimumHeight(minHeight).
		HighlightStyle(lipgloss.NewStyle().
			Background(lipgloss.Color("#444")).
			Foreground(lipgloss.Color("#eee")))
}

// Row table with horizontal scrolling
func (m *Model) openRowTable(tableName string) error {
	cols, err := m.dbConn.GetColumns(tableName)
	if err != nil {
		return err
	}
	m.rowTableName = tableName
	m.rowColumns = cols
	m.primaryKeyCol = m.dbConn.GetPrimaryKey(tableName)
	m.autoIncrementCols, _ = m.dbConn.GetAutoIncrementColumns(tableName)
	m.filters = nil
	m.rowOffset = 0
	m.rowLimit = 500
	m.rowHasMore = true
	m.fetchRowWindow(0)

	return nil
}

func (m *Model) fetchRowWindow(offset int) {
	var rows [][]string
	var err error
	if len(m.filters) > 0 {
		rows, err = m.dbConn.GetRowsMultiFiltered(m.rowTableName, m.rowColumns, m.filters, offset, m.rowLimit)
	} else {
		rows, err = m.dbConn.GetRowsPaginated(m.rowTableName, m.rowColumns, offset, m.rowLimit)
	}
	if err != nil {
		m.errMsg = fmt.Sprintf("Failed to fetch rows: %v", err)
		return
	}

	if len(rows) == 0 && offset > 0 {
		m.rowHasMore = false
		return
	}

	m.rowOffset = offset
	m.rowHasMore = len(rows) == m.rowLimit

	finalWidths := computeColumnWidths(m.rowColumns, rows, m.termWidth)

	columns := make([]btable.Column, len(m.rowColumns))
	for i, name := range m.rowColumns {
		columns[i] = btable.NewColumn(name, name, finalWidths[i])
	}

	bRows := rowsToTableData(rows, m.rowColumns)

	m.rowTable = styledTable(columns, bRows).
		WithPageSize(m.pageSize).
		WithMaxTotalWidth(m.termWidth).
		WithHorizontalFreezeColumnCount(1).
		WithMinimumHeight(contentHeight(m.termHeight))
}

func computeColumnWidths(columns []string, rows [][]string, termWidth int) []int {
	widths := make([]int, len(columns))
	if len(columns) == 0 {
		return widths
	}

	for i, name := range columns {
		widths[i] = ansi.StringWidth(name)
	}
	for _, row := range rows {
		for j, val := range row {
			if j < len(widths) {
				if w := ansi.StringWidth(val); w > widths[j] {
					widths[j] = w
				}
			}
		}
	}

	total := 0
	for i := range widths {
		widths[i]++
		total += widths[i]
	}

	// Borders and column dividers take up len(columns)+1 cells.
	target := termWidth - (len(columns) + 1)

	if total > target {
		total = 0
		for i := range widths {
			widths[i] = min(widths[i], maxColWidth)
			total += widths[i]
		}
	}

	if total < target {
		widths[len(widths)-1] += target - total
	}

	return widths
}

func rowsToTableData(rows [][]string, columns []string) []btable.Row {
	bRows := make([]btable.Row, len(rows))
	for i, row := range rows {
		data := make(btable.RowData, len(row))
		for j, val := range row {
			data[columns[j]] = val
		}
		bRows[i] = btable.NewRow(data)
	}
	return bRows
}

func (m *Model) initFormInputs(columns, skipCols, values []string) {
	m.updateInputs = make([]textinput.Model, len(columns))
	firstFocus := -1
	for i, col := range columns {
		ti := textinput.New()
		ti.Prompt = col + ": "
		ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
		ti.Width = 50
		if i < len(values) {
			ti.SetValue(values[i])
		}
		m.updateInputs[i] = ti
		if firstFocus == -1 && !slices.Contains(skipCols, col) {
			firstFocus = i
		}
	}
	if firstFocus >= 0 {
		m.updateInputs[firstFocus].Focus()
		m.updateFocused = firstFocus
	} else if len(m.updateInputs) > 0 {
		m.updateInputs[0].Focus()
		m.updateFocused = 0
	}
}

func (m *Model) focusInputs(direction int) {
	m.updateInputs[m.updateFocused].Blur()
	m.updateFocused = cycleFocus(len(m.updateInputs), m.updateFocused, direction, m.pkIdx)
	m.updateInputs[m.updateFocused].Focus()
}
