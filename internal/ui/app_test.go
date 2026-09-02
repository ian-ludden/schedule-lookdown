package ui

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/luddenig/schedule-lookdown/internal/client"
	"github.com/luddenig/schedule-lookdown/internal/config"
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

// collectMsgs runs cmd and recursively unwraps any tea.BatchMsg,
// returning all leaf messages.
// Useful for assertions when handlers use tea.Batch.
func collectMsgs(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range batch {
			out = append(out, collectMsgs(c)...)
		}
		return out
	}
	return []tea.Msg{msg}
}

// TestLoadSamplesReachesMenuAndQueries locks the --load-samples path: in
// fixture mode the app must land on the menu (never the username/login prompt,
// since session is nil by design) and run a query straight from fixtures.
func TestLoadSamplesReachesMenuAndQueries(t *testing.T) {
	fixtures := map[string]string{
		"schedule_lookup": "../../sample-responses/sample-student.html",
	}
	app := NewApp(nil, config.Default(), ScreenMenu, fixtures, nil)
	if app.screen != ScreenMenu {
		t.Fatalf("fixture mode landed on screen %v, want ScreenMenu", app.screen)
	}

	// querySelectedMsg → searchSubmittedMsg → queryResultMsg, no auth involved.
	app2, _ := driveUpdate(app, querySelectedMsg{queryType: "schedule_lookup"})
	_, qmsg := driveUpdate(app2, searchSubmittedMsg{
		queryType: "schedule_lookup",
		params:    map[string]string{"term": "202630", "username": "quinna"},
	})

	var qr queryResultMsg
	for _, m := range collectMsgs(func() tea.Msg { return qmsg }) {
		if q, ok := m.(queryResultMsg); ok {
			qr = q
		}
		if em, ok := m.(errMsg); ok {
			t.Fatalf("fixture query produced errMsg: %v", em.err)
		}
	}
	if len(qr.result.Rows) == 0 {
		t.Error("expected fixture query to return rows")
	}
}

func TestAdvisorSearchCmdSingleMatch(t *testing.T) {
	fixtures := map[string]string{
		"person_search": "../../sample-responses/sample-lastname-search.html",
	}
	cmd := advisorSearchCmd(nil, "Robin Vale", "202630", fixtures, nil)
	msg := cmd()

	ss, ok := msg.(searchSubmittedMsg)
	if !ok {
		t.Fatalf("expected searchSubmittedMsg, got %T (%v)", msg, msg)
	}
	if ss.queryType != "instructor_lookup" {
		t.Errorf("queryType = %q, want instructor_lookup", ss.queryType)
	}
	if ss.params["username"] != "valer" {
		t.Errorf("username = %q, want valer", ss.params["username"])
	}
	if ss.params["term"] != "202630" {
		t.Errorf("term = %q, want 202630", ss.params["term"])
	}
}

func TestAdvisorSearchIntegration(t *testing.T) {
	fixtures := map[string]string{
		"person_search":     "../../sample-responses/sample-lastname-search.html",
		"instructor_lookup": "../../sample-responses/sample-instructor.html",
	}
	app := NewApp(nil, config.Default(), ScreenResults, fixtures, nil)

	// Step 1: advisorSearchMsg → batch contains advisorSearchCmd result
	model, cmd1 := app.Update(advisorSearchMsg{advisorName: "Robin Vale",
		term: "202630"})
	app2 := model.(App)

	var ss searchSubmittedMsg
	for _, m := range collectMsgs(cmd1) {
		if s, ok := m.(searchSubmittedMsg); ok {
			ss = s
		}
		if em, ok := m.(errMsg); ok {
			t.Fatalf("advisorSearchMsg produced errMsg: %v", em.err)
		}
	}
	if ss.queryType != "instructor_lookup" {
		t.Errorf("queryType = %q, want instructor_lookup", ss.queryType)
	}
	if ss.params["username"] != "valer" {
		t.Errorf("username = %q, want valer", ss.params["username"])
	}

	// Step 2: searchSubmittedMsg → batch contains executeQueryCmd result
	_, cmd2 := app2.Update(ss)

	var qr queryResultMsg
	for _, m := range collectMsgs(cmd2) {
		if q, ok := m.(queryResultMsg); ok {
			qr = q
		}
		if em, ok := m.(errMsg); ok {
			t.Fatalf("searchSubmittedMsg produced errMsg: %v", em.err)
		}
	}
	if qr.queryType != "instructor_lookup" {
		t.Errorf("queryResultMsg.queryType = %q, want instructor_lookup",
			qr.queryType)
	}
	if len(qr.result.Rows) == 0 {
		t.Error("expected at least one course row in instructor results")
	}
}

func TestAppTermNavIntegration(t *testing.T) {
	fixtures := map[string]string{
		"schedule_lookup": "../../sample-responses/sample-student.html",
	}
	app := NewApp(nil, config.Default(), ScreenResults, fixtures, nil)
	app.results = newResultsModelWithData(
		query.Result{
			Columns: []string{"Course", "CRN"},
			Rows:    [][]string{{"CSSE474-02", "3096"}},
		},
		"schedule_lookup",
		map[string]string{"term": "202610", "username": "quinna"},
		120, 40, "",
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

	// Step 2: changeTermMsg → loading state with spinner; batch also contains executeQueryCmd.
	model3, cmd2 := app2.Update(ct)
	app3 := model3.(App)
	if !app3.results.loading {
		t.Error("expected results.loading to be true after changeTermMsg")
	}

	var qr queryResultMsg
	for _, m := range collectMsgs(cmd2) {
		if q, ok := m.(queryResultMsg); ok {
			qr = q
		}
		if em, isErr := m.(errMsg); isErr {
			t.Fatalf("changeTermMsg produced errMsg: %v", em.err)
		}
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

// TestRosterViewKeepsHistoryPanelOnScreen is a regression test for issue #53:
// the roster view's help line ("↑/↓ navigate • ... • ctrl+s: download roster")
// is the longest of any query type, and rendering it without a width cap made
// the whole main-content block wider than mainWidth. lipgloss.JoinHorizontal
// pads every line in a block to that block's own widest line, so the
// oversized help line pushed the history panel off the right edge of the
// terminal even though the panel itself renders fine. Assert that no
// rendered line of the full app view exceeds the terminal width, and that
// the history panel border is actually present in the output, across a
// range of terminal widths (including ones narrow enough to force the
// results table to compress its columns).
func TestRosterViewKeepsHistoryPanelOnScreen(t *testing.T) {
	f, err := os.Open("../../sample-responses/sample-roster.html")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cols, rows, err := client.ParseRoster(f)
	if err != nil {
		t.Fatal(err)
	}
	result := query.Result{Columns: cols, Rows: rows, Metadata: map[string]string{"title": "Object-Oriented Software Development"}}

	// Widths below ~80 force the roster table's 8 columns below minColWidth,
	// which calcAvailable deliberately refuses to do (a pre-existing, separate
	// tradeoff unrelated to issue #53) — so this sweep starts at a terminal
	// width the table can actually honor.
	for _, width := range []int{80, 90, 120, 160, 220} {
		app := App{screen: ScreenResults, width: width, height: 40}
		app.historyPanel = newHistoryPanelModel()
		app.historyPanel.height = app.height
		app.historyPanel.entries = []HistoryEntry{
			{QueryType: "roster_view", Params: map[string]string{"course_id": "CSSE230-02"}},
		}
		app.results = newResultsModelWithData(result, "roster_view",
			map[string]string{"term": "202710", "course_id": "CSSE230-02"},
			app.mainWidth(), app.height, "")

		out := app.View()
		for i, line := range strings.Split(out, "\n") {
			if w := lipgloss.Width(line); w > width {
				t.Errorf("width=%d: line %d is %d wide (%q)", width, i, w, line)
			}
		}
		if !strings.Contains(out, "Recent Queries") {
			t.Errorf("width=%d: history panel title not found in output", width)
		}
	}
}
