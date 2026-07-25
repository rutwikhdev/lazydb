package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	btable "github.com/evertras/bubble-table/table"
)

func TestCycleFocusForward(t *testing.T) {
	if got := cycleFocus(3, 2, 1, -1); got != 0 {
		t.Fatalf("cycleFocus forward wrap = %d, want 0", got)
	}
}

func TestCycleFocusBackward(t *testing.T) {
	if got := cycleFocus(3, 0, -1, -1); got != 2 {
		t.Fatalf("cycleFocus backward wrap = %d, want 2", got)
	}
}

func TestCycleFocusSkipsIndex(t *testing.T) {
	if got := cycleFocus(4, 0, 1, 1); got != 2 {
		t.Fatalf("cycleFocus skip index = %d, want 2", got)
	}
}

func TestOverlayPlacesForeground(t *testing.T) {
	got := overlay("background\nsecond", "modal", 40, 10)
	if !strings.Contains(got, "modal") {
		t.Fatalf("expected overlay to contain foreground, got %q", got)
	}
}

func TestRenderModalContainsTitleAndContent(t *testing.T) {
	got := renderModal("Title", "Body", 80, 20, "background")
	for _, want := range []string{"Title", "Body"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected modal to contain %q, got %q", want, got)
		}
	}
}

func TestBuildFormInputs(t *testing.T) {
	inputs := make([]textinput.Model, 2)
	for i := range inputs {
		inputs[i] = textinput.New()
	}
	inputs[0].SetValue("1")
	inputs[1].SetValue("alice")

	got := buildFormInputs(inputs, 0, []string{"id", "name"})
	for _, want := range []string{"id: 1", "alice"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected form inputs to contain %q, got %q", want, got)
		}
	}
}

func TestTableOrEmpty(t *testing.T) {
	cols := []btable.Column{btable.NewColumn("name", "Name", 10)}

	empty := btable.New(cols).WithRows(nil)
	got := tableOrEmpty(empty, "nothing here", 40, 10)
	if !strings.Contains(got, "nothing here") {
		t.Fatalf("expected empty state message, got %q", got)
	}
	if h := lipgloss.Height(got); h != 10 {
		t.Fatalf("empty state height = %d, want 10", h)
	}

	withRows := btable.New(cols).WithRows([]btable.Row{
		btable.NewRow(btable.RowData{"name": "alice"}),
	})
	got = tableOrEmpty(withRows, "nothing here", 40, 10)
	if strings.Contains(got, "nothing here") || !strings.Contains(got, "alice") {
		t.Fatalf("expected table view with rows, got %q", got)
	}
}

func TestStatusBarErrorTakesPrecedence(t *testing.T) {
	m := NewModel()
	m.errMsg = "boom"
	got := m.statusBar()
	if !strings.Contains(got, "Error: boom") {
		t.Fatalf("statusBar error = %q", got)
	}
}

func TestStatusBarScreens(t *testing.T) {
	tests := []struct {
		name string
		s    screen
		want string
	}{
		{name: "db type", s: screenDBType, want: "Enter: select"},
		{name: "sqlite path", s: screenSQLitePath, want: "Enter: connect"},
		{name: "connection form", s: screenConnectionForm, want: "Tab: next field"},
		{name: "tables", s: screenTables, want: "Enter: open table"},
		{name: "detail", s: screenDetail, want: "Esc: back"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel()
			m.stack = []screen{tt.s}
			if got := m.statusBar(); !strings.Contains(got, tt.want) {
				t.Fatalf("statusBar() = %q, want substring %q", got, tt.want)
			}
		})
	}
}

func TestSelectedPKValueMissingPK(t *testing.T) {
	m := NewModel()
	if got, ok := m.selectedPKValue(); ok || got != "" {
		t.Fatalf("selectedPKValue without pk = %q, %v; want empty false", got, ok)
	}
}
