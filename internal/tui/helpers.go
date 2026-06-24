package tui

import (
	"fmt"
	"lazydb/internal/db"
	"strconv"

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

func makeDetailTable(row []string, columns []string, width, pageSize int) btable.Model {
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
		WithPageSize(pageSize)
}

// Row table with horizontal scrolling
func (m *Model) openRowTable(tableName string, width, pageSize int) error {
	cols, err := m.dbConn.GetColumns(tableName)
	if err != nil {
		return err
	}
	m.rowTableName = tableName
	m.rowColumns = cols
	m.termWidth = width
	m.pageSize = pageSize
	m.primaryKeyCol = m.dbConn.GetPrimaryKey(tableName)
	m.autoIncrementCols, _ = m.dbConn.GetAutoIncrementColumns(tableName)
	m.filterCol = ""
	m.filterVal = ""
	m.rowOffset = 0
	m.rowLimit = 500
	m.rowHasMore = true
	m.fetchRowWindow(0)

	return nil
}

func (m *Model) fetchRowWindow(offset int) {
	var rows [][]string
	var err error
	if m.filterCol != "" && m.filterVal != "" {
		rows, err = m.dbConn.GetRowsFiltered(m.rowTableName, m.rowColumns, m.filterCol, m.filterVal, offset, m.rowLimit)
	} else {
		rows, err = m.dbConn.GetRowsPaginated(m.rowTableName, m.rowColumns, offset, m.rowLimit)
	}
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

	finalWidths := computeColumnWidths(m.rowColumns, rows, m.termWidth)

	columns := make([]btable.Column, len(m.rowColumns))
	for i, name := range m.rowColumns {
		columns[i] = btable.NewColumn(name, name, finalWidths[i])
	}

	bRows := rowsToTableData(rows, m.rowColumns)

	m.rowTable = styledTable(columns, bRows).
		WithPageSize(m.pageSize).
		WithMaxTotalWidth(m.termWidth).
		WithHorizontalFreezeColumnCount(1)
}

func computeColumnWidths(columns []string, rows [][]string, termWidth int) []int {
	colWidths := make([]int, len(columns))
	for i, name := range columns {
		colWidths[i] = ansi.StringWidth(name)
	}
	for _, row := range rows {
		for j, val := range row {
			if j < len(colWidths) {
				w := ansi.StringWidth(val)
				if w > colWidths[j] {
					colWidths[j] = w
				}
			}
		}
	}

	finalWidths := make([]int, len(columns))
	for i := range colWidths {
		finalWidths[i] = colWidths[i] + 1
	}

	totalColWidth := 0
	for _, w := range finalWidths {
		totalColWidth += w
	}

	if totalColWidth < termWidth {
		for i := range finalWidths {
			proportion := float64(finalWidths[i]) / float64(totalColWidth)
			finalWidths[i] = int(proportion * float64(termWidth-3))
		}
	} else {
		for i := range finalWidths {
			finalWidths[i] = min(finalWidths[i], maxColWidth)
		}
	}

	return finalWidths
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
