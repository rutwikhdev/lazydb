package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModelInitialState(t *testing.T) {
	m := NewModel()
	if m.currentScreen() != screenDBType {
		t.Fatalf("initial screen = %v, want %v", m.currentScreen(), screenDBType)
	}
	if m.termWidth != 120 || m.termHeight != 60 {
		t.Fatalf("initial terminal size = %dx%d, want 120x60", m.termWidth, m.termHeight)
	}
	if m.pageSize != dynamicPageSize(60) {
		t.Fatalf("initial page size = %d, want %d", m.pageSize, dynamicPageSize(60))
	}
	if m.logger == nil {
		t.Fatalf("expected logger to be initialized")
	}
}

func TestPushPop(t *testing.T) {
	m := NewModel()
	m.push(screenSQLitePath)
	if m.currentScreen() != screenSQLitePath {
		t.Fatalf("current screen after push = %v, want %v", m.currentScreen(), screenSQLitePath)
	}
	m.errMsg = "boom"
	m.pop()
	if m.currentScreen() != screenDBType {
		t.Fatalf("current screen after pop = %v, want %v", m.currentScreen(), screenDBType)
	}
	if m.errMsg != "" {
		t.Fatalf("expected pop to clear error, got %q", m.errMsg)
	}
	m.pop()
	if len(m.stack) != 1 || m.currentScreen() != screenDBType {
		t.Fatalf("root pop changed stack: %#v", m.stack)
	}
}

func TestUpdateCtrlCQuits(t *testing.T) {
	m := NewModel()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatalf("expected ctrl+c to return quit command")
	}
}

func TestUpdateDBTypeQuit(t *testing.T) {
	m := NewModel()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatalf("expected q on db type screen to return quit command")
	}
}

func TestUpdateSQLiteEmptyPathShowsError(t *testing.T) {
	m := NewModel()
	m.stack = []screen{screenSQLitePath}

	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("expected no command for validation error")
	}
	updated := model.(Model)
	if updated.errMsg != "Path cannot be empty" {
		t.Fatalf("errMsg = %q, want path validation error", updated.errMsg)
	}
}

func TestUpdateConnectionFormBack(t *testing.T) {
	m := NewModel()
	m.stack = []screen{screenDBType, screenConnectionForm}

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := model.(Model)
	if updated.currentScreen() != screenDBType {
		t.Fatalf("screen after esc = %v, want %v", updated.currentScreen(), screenDBType)
	}
}

func TestUpdateConnectionFormFocusCycle(t *testing.T) {
	m := NewModel()
	m.stack = []screen{screenConnectionForm}
	m.focusedInput = 0
	m.connInputs[0].Focus()

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated := model.(Model)
	if updated.focusedInput != 1 {
		t.Fatalf("focused input after tab = %d, want 1", updated.focusedInput)
	}

	model, _ = updated.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	updated = model.(Model)
	if updated.focusedInput != 0 {
		t.Fatalf("focused input after shift+tab = %d, want 0", updated.focusedInput)
	}
}

func TestWindowSizeUpdatesDimensions(t *testing.T) {
	m := NewModel()
	model, cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	if cmd != nil {
		t.Fatalf("expected no command for window resize")
	}
	updated := model.(Model)
	if updated.termWidth != 100 || updated.termHeight != 40 {
		t.Fatalf("terminal size = %dx%d, want 100x40", updated.termWidth, updated.termHeight)
	}
	if updated.pageSize != dynamicPageSize(40) {
		t.Fatalf("page size = %d, want %d", updated.pageSize, dynamicPageSize(40))
	}
}

func TestViewIncludesStatusBar(t *testing.T) {
	m := NewModel()
	view := m.View()
	if !strings.Contains(view, "Enter: select") {
		t.Fatalf("expected view to include status bar, got %q", view)
	}
}
