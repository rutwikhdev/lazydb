package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	btable "github.com/evertras/bubble-table/table"
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

func renderModal(title, content string, termWidth, termHeight int, bg string) string {
	var formContent strings.Builder

	styledTitle := lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Render(title)
	formContent.WriteString(lipgloss.NewStyle().Width(56).Align(lipgloss.Center).PaddingBottom(1).Render(styledTitle))
	formContent.WriteString(content)

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("13")).
		Padding(1, 2).
		Width(60)
	modal := modalStyle.Render(formContent.String())
	return overlay(bg, modal, termWidth, termHeight)
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
		start := m.rowOffset + 1
		end := m.rowOffset + len(m.rowTable.GetVisibleRows())
		return fmt.Sprintf("↑↓: navigate • h/l or shift+←→: scroll • Enter: view • Esc: back • q: quit • Rows %d-%d", start, end)
	case screenDetail:
		return "Esc: back • q: quit"
	case screenUpdate:
		if m.confirmUpdate {
			return "y: confirm • n/esc: cancel"
		}
		return "Tab: next field • Shift+Tab: prev field • Enter: save • Esc: cancel"
	case screenDelete:
		if m.confirmDelete {
			return "y: confirm • n/esc: cancel"
		}
		return "Enter: confirm delete • Esc: cancel"
	case screenInsert:
		if m.confirmUpdate {
			return "y: confirm • n/esc: cancel"
		}
		return "Tab: next field • Shift+Tab: prev field • Enter: save • Esc: cancel"
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
