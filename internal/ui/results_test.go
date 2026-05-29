package ui

import (
	"testing"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/luddenig/schedule-lookdown/internal/query"
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

func TestResultsRefreshKey(t *testing.T) {
	for _, queryType := range []string{"schedule_lookup", "roster_view", "instructor_lookup", "course_search"} {
		t.Run(queryType, func(t *testing.T) {
			m := resultsModel{
				queryType: queryType,
				params:    map[string]string{"term": "202630"},
			}
			_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
			if cmd == nil {
				t.Fatal("ctrl+r returned nil cmd")
			}
			msg := cmd()
			r, ok := msg.(refreshCurrentQueryMsg)
			if !ok {
				t.Fatalf("expected refreshCurrentQueryMsg, got %T", msg)
			}
			if r.queryType != queryType {
				t.Errorf("got queryType %q, want %q", r.queryType, queryType)
			}
		})
	}
}

func TestResultsRosterFromScheduleLookup(t *testing.T) {
	tbl := table.New(
		table.WithColumns([]table.Column{{Title: "Course", Width: 10}}),
		table.WithRows([]table.Row{{"CSSE474-01"}}),
		table.WithFocused(true),
	)
	m := resultsModel{
		queryType: "schedule_lookup",
		params:    map[string]string{"term": "202630", "username": "testuser"},
		table:     tbl,
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if cmd == nil {
		t.Fatal("r returned nil cmd")
	}
	msg := cmd()
	ss, ok := msg.(searchSubmittedMsg)
	if !ok {
		t.Fatalf("expected searchSubmittedMsg, got %T", msg)
	}
	if ss.queryType != "roster_view" {
		t.Errorf("got queryType %q, want roster_view", ss.queryType)
	}
	if ss.params["course_id"] != "CSSE474-01" {
		t.Errorf("got course_id %q, want CSSE474-01", ss.params["course_id"])
	}
}

func TestResultsAdvisorFromScheduleLookup(t *testing.T) {
	m := resultsModel{
		queryType: "schedule_lookup",
		params:    map[string]string{"term": "202630", "username": "testuser"},
		result:    query.Result{Metadata: map[string]string{"advisor_name": "John Smith"}},
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if cmd == nil {
		t.Fatal("a returned nil cmd")
	}
	msg := cmd()
	as, ok := msg.(advisorSearchMsg)
	if !ok {
		t.Fatalf("expected advisorSearchMsg, got %T", msg)
	}
	if as.advisorName != "John Smith" {
		t.Errorf("got advisorName %q, want John Smith", as.advisorName)
	}
	if as.term != "202630" {
		t.Errorf("got term %q, want 202630", as.term)
	}
}

func TestResultsPersonSearchEnter(t *testing.T) {
	tbl := table.New(
		table.WithColumns([]table.Column{{Title: "USERNAME", Width: 12}}),
		table.WithRows([]table.Row{{"smithj"}}),
		table.WithFocused(true),
	)
	m := resultsModel{
		queryType: "person_search",
		params:    map[string]string{"term": "202630", "last_name": "Smith"},
		table:     tbl,
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter returned nil cmd")
	}
	msg := cmd()
	ss, ok := msg.(searchSubmittedMsg)
	if !ok {
		t.Fatalf("expected searchSubmittedMsg, got %T", msg)
	}
	if ss.queryType != "instructor_lookup" {
		t.Errorf("got queryType %q, want instructor_lookup", ss.queryType)
	}
	if ss.params["username"] != "smithj" {
		t.Errorf("got username %q, want smithj", ss.params["username"])
	}
}
