package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestDynamicPageSize(t *testing.T) {
	tests := []struct {
		name   string
		height int
		want   int
	}{
		{name: "small terminal", height: 12, want: 5},
		{name: "normal terminal", height: 30, want: 22},
		{name: "boundary", height: 13, want: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dynamicPageSize(tt.height); got != tt.want {
				t.Fatalf("dynamicPageSize(%d) = %d, want %d", tt.height, got, tt.want)
			}
		})
	}
}

func TestComputeColumnWidthsCapsWideColumns(t *testing.T) {
	columns := []string{"id", "description"}
	rows := [][]string{{"1", strings.Repeat("x", maxColWidth+20)}}

	widths := computeColumnWidths(columns, rows, 20)
	if len(widths) != len(columns) {
		t.Fatalf("got %d widths, want %d", len(widths), len(columns))
	}
	if widths[1] != maxColWidth {
		t.Fatalf("wide column width = %d, want cap %d", widths[1], maxColWidth)
	}
}

func TestComputeColumnWidthsExpandsToTerminalWidth(t *testing.T) {
	columns := []string{"id", "name"}
	rows := [][]string{{"1", "alice"}}

	widths := computeColumnWidths(columns, rows, 80)
	if len(widths) != len(columns) {
		t.Fatalf("got %d widths, want %d", len(widths), len(columns))
	}
	// Non-last columns keep their natural width (widest content + 1).
	if widths[0] != len("id")+1 {
		t.Fatalf("first column width = %d, want natural width %d", widths[0], len("id")+1)
	}
	// The last column absorbs the leftover so the table fills the terminal.
	total := widths[0] + widths[1]
	want := 80 - (len(columns) + 1)
	if total != want {
		t.Fatalf("total column width = %d, want %d", total, want)
	}
}

func TestComputeColumnWidthsFillsAfterCapping(t *testing.T) {
	columns := []string{"id", "data"}
	rows := [][]string{{"1", strings.Repeat("x", 100)}}

	widths := computeColumnWidths(columns, rows, 60)
	total := widths[0] + widths[1]
	want := 60 - (len(columns) + 1)
	if total != want {
		t.Fatalf("total column width = %d, want %d", total, want)
	}
}

func TestRowsToTableData(t *testing.T) {
	rows := rowsToTableData([][]string{{"1", "alice"}, {"2", "bob"}}, []string{"id", "name"})
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if got := rows[0].Data["id"]; got != "1" {
		t.Fatalf("first row id = %v, want 1", got)
	}
	if got := rows[1].Data["name"]; got != "bob" {
		t.Fatalf("second row name = %v, want bob", got)
	}
}

func TestMakeListTableRendersItems(t *testing.T) {
	view := makeListTable("Database Type", []string{"sqlite", "mysql"}, 80, 10, 20).View()
	for _, want := range []string{"Database Type", "sqlite", "mysql"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected list table to contain %q, view: %s", want, view)
		}
	}
}

func TestMakeDetailTableRendersColumnsAndValues(t *testing.T) {
	view := makeDetailTable([]string{"1", "alice"}, []string{"id", "name"}, 80, 10, 20).View()
	for _, want := range []string{"id", "name", "1", "alice"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected detail table to contain %q, view: %s", want, view)
		}
	}
}

func TestTableFillsMinHeight(t *testing.T) {
	view := makeListTable("Table Name", []string{"users"}, 80, 10, 20).View()
	if h := lipgloss.Height(view); h != 20 {
		t.Fatalf("table height = %d, want minimum height 20", h)
	}
}
