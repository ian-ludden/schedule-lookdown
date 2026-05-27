package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luddenig/schedule-lookdown/internal/query"
)

// driveUpdate calls Update with msg and, if a Cmd is returned, runs it.
func driveUpdate(a App, msg tea.Msg) (App, tea.Msg) {
	model, cmd := a.Update(msg)
	a = model.(App)
	if cmd == nil {
		return a, nil
	}
	return a, cmd()
}

func TestAppTermNavIntegration(t *testing.T) {
	fixtures := map[string]string{
		"schedule_lookup": "../../sample-responses/sample-student.html",
	}
	app := NewApp(nil, ScreenResults, fixtures)
	app.results = newResultsModelWithData(
		query.Result{
			Columns: []string{"Course", "CRN"},
			Rows:    [][]string{{"CSSE474-02", "3096"}},
		},
		"schedule_lookup",
		map[string]string{"term": "202610", "username": "bregginr"},
		120, 40,
	)
	app.screen = ScreenResults

	// Step 1: press l — should produce changeTermMsg.
	app2, msg1 := driveUpdate(app, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	ct, ok := msg1.(changeTermMsg)
	if !ok {
		t.Fatalf("pressing l: expected changeTermMsg, got %T (%v)", msg1, msg1)
	}
	if ct.term != "202620" {
		t.Errorf("changeTermMsg.term = %q, want 202620", ct.term)
	}

	// Step 2: changeTermMsg → should produce queryResultMsg (or errMsg).
	app3, msg2 := driveUpdate(app2, ct)
	qr, ok := msg2.(queryResultMsg)
	if !ok {
		if em, isErr := msg2.(errMsg); isErr {
			t.Fatalf("changeTermMsg produced errMsg: %v", em.err)
		}
		t.Fatalf("changeTermMsg produced %T (%v), want queryResultMsg", msg2, msg2)
	}
	if qr.params["term"] != "202620" {
		t.Errorf("queryResultMsg.params[term] = %q, want 202620", qr.params["term"])
	}

	// Step 3: queryResultMsg → results model should reflect new term.
	app4, _ := driveUpdate(app3, qr)
	if got := app4.results.params["term"]; got != "202620" {
		t.Errorf("after queryResultMsg, results.params[term] = %q, want 202620", got)
	}
}
