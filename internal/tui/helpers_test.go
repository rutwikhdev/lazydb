package tui

import (
	"strings"
	"testing"
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
	if widths[0] <= len(columns[0]) || widths[1] <= len("alice") {
		t.Fatalf("expected widths to expand for available terminal width, got %v", widths)
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

func TestMakeDBTypeTableRendersSupportedDBs(t *testing.T) {
	view := makeDBTypeTable(80, 10).View()
	for _, want := range []string{"sqlite", "mysql", "postgres"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected db type table to contain %q, view: %s", want, view)
		}
	}
}

func TestMakeTableListTableRendersTables(t *testing.T) {
	view := makeTableListTable([]string{"users", "orders"}, 80, 10).View()
	for _, want := range []string{"users", "orders"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected table list to contain %q, view: %s", want, view)
		}
	}
}

func TestMakeDatabaseTableRendersDatabases(t *testing.T) {
	view := makeDatabaseTable([]string{"app", "analytics"}, 80, 10).View()
	for _, want := range []string{"app", "analytics"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected database table to contain %q, view: %s", want, view)
		}
	}
}

func TestMakeDetailTableRendersColumnsAndValues(t *testing.T) {
	view := makeDetailTable([]string{"1", "alice"}, []string{"id", "name"}, 80, 10).View()
	for _, want := range []string{"id", "name", "1", "alice"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected detail table to contain %q, view: %s", want, view)
		}
	}
}
