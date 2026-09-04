package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	btable "github.com/evertras/bubble-table/table"
)

const (
	msgEmptyRows      = "No records found"
	msgEmptyTables    = "No tables found"
	msgEmptyDatabases = "No databases found"
)

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

func renderModalWithDimensions(title, content string, termWidth, termHeight int, bg string, width, height int) string {
	var formContent strings.Builder

	styledTitle := lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Render(title)
	formContent.WriteString(lipgloss.NewStyle().Width(width - 4).Align(lipgloss.Center).PaddingBottom(1).Render(styledTitle))
	formContent.WriteString(content)

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("13")).
		Padding(1, 2).
		Width(width)
	if height > 0 {
		modalStyle = modalStyle.Height(height)
	}
	modal := modalStyle.Render(formContent.String())
	return overlay(bg, modal, termWidth, max(contentHeight(termHeight), 1))
}

func renderDetailModal(detailTable btable.Model, termWidth, termHeight int, bg string) string {
	layout := detailLayoutFor(termWidth, termHeight)
	return renderModalWithDimensions(
		"Record Details",
		detailTable.View(),
		termWidth,
		termHeight,
		bg,
		layout.modalWidth,
		layout.modalHeight,
	)
}

func overlay(bg, fg string, width, height int) string {
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")

	fgWidth := 0
	for _, line := range fgLines {
		w := ansi.StringWidth(line)
		if w > fgWidth {
			fgWidth = w
		}
	}
	fgHeight := len(fgLines)

	startX := (width - fgWidth) / 2
	startY := (height - fgHeight) / 2
	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	for len(bgLines) < startY+fgHeight {
		bgLines = append(bgLines, "")
	}

	for i, fgLine := range fgLines {
		bgIdx := startY + i
		if bgIdx >= len(bgLines) {
			break
		}
		bgLine := bgLines[bgIdx]
		bgWidth := ansi.StringWidth(bgLine)
		if bgWidth < startX {
			bgLine = bgLine + strings.Repeat(" ", startX-bgWidth)
		}
		left := ansi.Cut(bgLine, 0, startX)
		fgLineWidth := ansi.StringWidth(fgLine)
		right := ansi.TruncateLeft(bgLine, startX+fgLineWidth, "")
		bgLines[bgIdx] = left + fgLine + right
	}

	return strings.Join(bgLines, "\n")
}

func emptyStateMsg(msg string, width, height int) string {
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(lipgloss.Color("240")).
		Render(msg)
}

func tableOrEmpty(t btable.Model, msg string, width, height int) string {
	if len(t.GetVisibleRows()) == 0 {
		return emptyStateMsg(msg, width, height)
	}
	return t.View()
}

func buildFormInputs(inputs []textinput.Model, pkIdx int, columns []string) string {
	var sb strings.Builder
	for i, input := range inputs {
		if i == pkIdx {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(columns[i] + ": " + input.Value()))
		} else {
			sb.WriteString(input.View())
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

func (m *Model) statusBar() string {
	if m.errMsg != "" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("Error: " + m.errMsg)
	}

	switch m.currentScreen() {
	case screenDBType:
		return "↑↓: navigate • Enter: select • q: quit"
	case screenSQLitePath:
		return "Enter: connect • Esc: back"
	case screenConnectionForm:
		return "Tab: next field • Shift+Tab: prev field • Enter: connect • Esc: back"
	case screenDatabases:
		return "↑↓: navigate • Enter: select • Esc: back • q: quit"
	case screenTables:
		return "↑↓: navigate • Enter: open table • Esc: back • q: quit"
	case screenRows:
		if m.detailOpen {
			return "↑↓/j/k: scroll details • Esc/b: close • q: quit"
		}
		if len(m.rowTable.GetVisibleRows()) == 0 {
			if len(m.filters) > 0 {
				return "No matching records • S: search • c: clear • Esc: back • q: quit"
			}
			return "I: insert • S: search • Esc: back • q: quit"
		}
		start := m.rowOffset + 1
		end := m.rowOffset + len(m.rowTable.GetVisibleRows())
		if len(m.filters) > 0 {
			var parts []string
			for _, col := range m.rowColumns {
				if val, ok := m.filters[col]; ok {
					parts = append(parts, fmt.Sprintf("%s LIKE '%%%s%%'", col, val))
				}
			}
			return fmt.Sprintf("Filter: %s • Rows %d-%d • S: search • c: clear", strings.Join(parts, " AND "), start, end)
		}
		return fmt.Sprintf("↑↓: navigate • h/l or shift+←→: scroll • Enter: view • S: search • Esc: back • q: quit • Rows %d-%d", start, end)
	case screenUpdate:
		if m.confirmUpdate {
			return "y: confirm • n/esc: cancel"
		}
		return "Tab: next field • Shift+Tab: prev field • Enter: save • Esc: cancel"
	case screenDelete:
		return "y: confirm • n/esc: cancel"
	case screenInsert:
		if m.confirmUpdate {
			return "y: confirm • n/esc: cancel"
		}
		return "Tab: next field • Shift+Tab: prev field • Enter: save • Esc: cancel"
	case screenSearch:
		return "Tab: next field • Shift+Tab: prev field • Enter: search • Esc: cancel"
	}
	return ""
}

func (m *Model) selectedPKValue() (string, bool) {
	if m.primaryKeyCol == "" {
		return "", false
	}
	selected := m.rowTable.HighlightedRow()
	if selected.Data == nil {
		return "", false
	}
	pkVal, ok := selected.Data[m.primaryKeyCol]
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%v", pkVal), true
}

func (m *Model) requirePKValue() (string, bool) {
	pkVal, ok := m.selectedPKValue()
	if !ok && m.primaryKeyCol == "" {
		m.errMsg = "No primary key found for this table"
	}
	return pkVal, ok
}

func selectedName(t btable.Model) (string, bool) {
	selected := t.HighlightedRow()
	if selected.Data == nil {
		return "", false
	}
	val, ok := selected.Data["name"]
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%v", val), true
}

func (m *Model) renderFormModal(title, confirmText, hint string, pkIdx int, columns []string) string {
	bg := tableOrEmpty(m.rowTable, msgEmptyRows, m.termWidth, contentHeight(m.termHeight))
	var modalContent string
	if m.confirmUpdate {
		modalContent = confirmText
	} else {
		modalContent = buildFormInputs(m.updateInputs, pkIdx, columns)
		styledHint := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(hint)
		modalContent += lipgloss.NewStyle().Width(56).Align(lipgloss.Right).PaddingTop(1).Render(styledHint)
	}
	return renderModalWithDimensions(title, modalContent, m.termWidth, m.termHeight, bg, 60, 0)
}

func cycleFocus(count, current, direction, skipIdx int) int {
	next := current
	for {
		next += direction
		if next >= count {
			next = 0
		}
		if next < 0 {
			next = count - 1
		}
		if next != skipIdx {
			break
		}
	}
	return next
}
