package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModelQuitsOnQuitKeys(t *testing.T) {
	testCases := []struct {
		name string
		key  tea.KeyMsg
	}{
		{name: "q", key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}},
		{name: "escape", key: tea.KeyMsg{Type: tea.KeyEsc}},
		{name: "ctrl+c", key: tea.KeyMsg{Type: tea.KeyCtrlC}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, cmd := (model{}).Update(testCase.key)
			if cmd == nil {
				t.Fatal("expected key to return a quit command")
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatal("expected key command to return a quit message")
			}
		})
	}
}

func TestModelViewContainsTitle(t *testing.T) {
	if view := (model{}).View(); !strings.Contains(view, "ts4ccmanager") {
		t.Fatalf("expected view to contain title, got %q", view)
	}
}
