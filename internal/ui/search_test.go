package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// termFieldIndex returns the index of the term field, or -1.
func termFieldIndex(m searchModel) int {
	for i, f := range m.fields {
		if f.kind == fieldTerm {
			return i
		}
	}
	return -1
}

func TestSearchTermSelectorStepKeys(t *testing.T) {
	m := newSearchModelForQuery("course_search", "", "202610", "")
	ti := termFieldIndex(m)
	if ti != 0 {
		t.Fatalf("term field index = %d, want 0", ti)
	}
	m.fields[ti].term = "202610" // fix a known starting term

	for _, tc := range []struct {
		key      string
		wantTerm string
	}{
		{"l", "202620"}, // forward
		{"h", "202610"}, // back to start
		{"right", "202620"},
		{"left", "202610"},
	} {
		t.Run(tc.key, func(t *testing.T) {
			updated, _ := m.Update(keyMsg(tc.key))
			m = updated.(searchModel)
			if got := m.fields[ti].term; got != tc.wantTerm {
				t.Errorf("after %q: term = %q, want %q", tc.key, got, tc.wantTerm)
			}
		})
	}
}

func TestSearchTermSelectorBlockedAtLatest(t *testing.T) {
	m := newSearchModelForQuery("course_search", "", "202630", "202630")
	ti := termFieldIndex(m)
	m.fields[ti].term = "202630" // at the latest available term

	// Forward steps are blocked and set a warning; the term stays put.
	for _, key := range []string{"l", "right"} {
		updated, _ := m.Update(keyMsg(key))
		m = updated.(searchModel)
		if got := m.fields[ti].term; got != "202630" {
			t.Errorf("after %q: term = %q, want it unchanged at 202630", key, got)
		}
		if m.termWarning == "" {
			t.Errorf("after %q: expected termWarning to be set", key)
		}
	}

	// Backward still works and clears the warning.
	updated, _ := m.Update(keyMsg("h"))
	m = updated.(searchModel)
	if got := m.fields[ti].term; got != "202620" {
		t.Errorf("after h: term = %q, want 202620", got)
	}
	if m.termWarning != "" {
		t.Error("expected termWarning cleared after pressing h")
	}
}

func TestSearchSubmitEmitsTerm(t *testing.T) {
	m := newSearchModelForQuery("schedule_lookup", "", "202630", "")
	ti := termFieldIndex(m)
	m.fields[ti].term = "202630"

	msg := m.submitCmd()()
	ss, ok := msg.(searchSubmittedMsg)
	if !ok {
		t.Fatalf("expected searchSubmittedMsg, got %T", msg)
	}
	if ss.params["term"] != "202630" {
		t.Errorf("params[term] = %q, want 202630", ss.params["term"])
	}
}

// keyMsg builds a tea.KeyMsg for a key string, handling named keys (left/right)
// and rune keys (h/l).
func keyMsg(key string) tea.KeyMsg {
	switch key {
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
}
