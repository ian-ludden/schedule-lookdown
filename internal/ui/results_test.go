package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestResultsTermNavKeys(t *testing.T) {
	m := resultsModel{
		queryType: "schedule_lookup",
		params:    map[string]string{"term": "202610", "username": "testuser"},
	}

	for _, tc := range []struct {
		key      string
		wantTerm string
	}{
		{"l", "202620"},
		{"h", "202540"},
	} {
		t.Run(tc.key, func(t *testing.T) {
			_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.key)})
			if cmd == nil {
				t.Fatal("Update returned nil cmd — key was not handled")
			}
			msg := cmd()
			ct, ok := msg.(changeTermMsg)
			if !ok {
				t.Fatalf("expected changeTermMsg, got %T: %v", msg, msg)
			}
			if ct.term != tc.wantTerm {
				t.Errorf("got term %q, want %q", ct.term, tc.wantTerm)
			}
		})
	}
}
