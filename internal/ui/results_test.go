package ui

import (
	"reflect"
	"sort"
	"testing"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/luddenig/schedule-lookdown/internal/query"
)

// ---- computeColWidths and helper tests ----

func TestBuildPreferredWidths(t *testing.T) {
	cols := []string{"Course", "CRN", "UNKNOWN"}
	widths, total := buildPreferredWidths(cols)
	if widths["Course"] != preferredWidths["Course"] {
		t.Errorf("Course: got %d, want %d", widths["Course"], preferredWidths["Course"])
	}
	if widths["CRN"] != preferredWidths["CRN"] {
		t.Errorf("CRN: got %d, want %d", widths["CRN"], preferredWidths["CRN"])
	}
	if widths["UNKNOWN"] != defaultColWidth {
		t.Errorf("UNKNOWN: got %d, want %d (defaultColWidth)", widths["UNKNOWN"], defaultColWidth)
	}
	want := preferredWidths["Course"] + preferredWidths["CRN"] + defaultColWidth
	if total != want {
		t.Errorf("total: got %d, want %d", total, want)
	}
}

func TestCalcAvailable(t *testing.T) {
	for _, tc := range []struct {
		termWidth, numCols, want int
	}{
		{100, 5, 100 - outerBorderWidth - cellPadding*5},
		// floor: tiny terminal forces at least numCols*minColWidth
		{10, 5, 5 * minColWidth},
	} {
		got := calcAvailable(tc.termWidth, tc.numCols)
		if got != tc.want {
			t.Errorf("calcAvailable(%d,%d) = %d, want %d", tc.termWidth, tc.numCols, got, tc.want)
		}
	}
}

func TestPartitionByPriority(t *testing.T) {
	cols := []string{"Course", "CrHrs", "Comments", "MYSTERY"}
	widths := map[string]int{"Course": 11, "CrHrs": 7, "Comments": 25, "MYSTERY": 15}
	high, med, low, sumHigh, sumMed := partitionByPriority(cols, widths)

	if !reflect.DeepEqual(high, []string{"Course"}) {
		t.Errorf("high = %v, want [Course]", high)
	}
	// CrHrs is priority 2; MYSTERY falls back to defaultPriority (2)
	wantMed := []string{"CrHrs", "MYSTERY"}
	gotMed := append([]string{}, med...)
	sort.Strings(gotMed)
	sort.Strings(wantMed)
	if !reflect.DeepEqual(gotMed, wantMed) {
		t.Errorf("med = %v, want %v", med, wantMed)
	}
	if !reflect.DeepEqual(low, []string{"Comments"}) {
		t.Errorf("low = %v, want [Comments]", low)
	}
	if sumHigh != 11 {
		t.Errorf("sumHigh = %d, want 11", sumHigh)
	}
	if sumMed != 7+15 {
		t.Errorf("sumMed = %d, want %d", sumMed, 7+15)
	}
}

func TestSetToMin(t *testing.T) {
	widths := map[string]int{"A": 20, "B": 10}
	setToMin([]string{"A", "B"}, widths)
	for _, k := range []string{"A", "B"} {
		if widths[k] != minColWidth {
			t.Errorf("widths[%q] = %d, want %d", k, widths[k], minColWidth)
		}
	}
}

func TestScaleProportionally(t *testing.T) {
	widths := map[string]int{"A": 20, "B": 10}
	scaleProportionally([]string{"A", "B"}, widths, 15, 30)
	// A: 20/30*15 = 10; B: 10/30*15 = 5
	if widths["A"] != 10 {
		t.Errorf("A = %d, want 10", widths["A"])
	}
	if widths["B"] != 5 {
		t.Errorf("B = %d, want 5", widths["B"])
	}
}

func TestScaleProportionally_Floor(t *testing.T) {
	widths := map[string]int{"A": 1}
	scaleProportionally([]string{"A"}, widths, 1, 100)
	if widths["A"] != minColWidth {
		t.Errorf("A = %d, want minColWidth (%d)", widths["A"], minColWidth)
	}
}

func TestDistributeSurplus(t *testing.T) {
	widths := map[string]int{"A": 20, "B": 10}
	distributeSurplus([]string{"A", "B"}, widths, 15, 30)
	// A: minColWidth + 20/30*15 = 3+10 = 13; B: minColWidth + 10/30*15 = 3+5 = 8
	if widths["A"] != minColWidth+10 {
		t.Errorf("A = %d, want %d", widths["A"], minColWidth+10)
	}
	if widths["B"] != minColWidth+5 {
		t.Errorf("B = %d, want %d", widths["B"], minColWidth+5)
	}
}

func TestComputeColWidths(t *testing.T) {
	// columns covering all three priority tiers
	cols := []string{"Course", "CrHrs", "Comments"}
	prefCourse := preferredWidths["Course"]     // 11, HIGH
	prefCrHrs := preferredWidths["CrHrs"]       // 7,  MED
	prefComments := preferredWidths["Comments"] // 25, LOW

	t.Run("termWidth=0 returns preferred", func(t *testing.T) {
		w := computeColWidths(cols, 0)
		if w["Course"] != prefCourse || w["CrHrs"] != prefCrHrs || w["Comments"] != prefComments {
			t.Errorf("unexpected widths: %v", w)
		}
	})

	t.Run("wide terminal returns preferred", func(t *testing.T) {
		w := computeColWidths(cols, 9999)
		if w["Course"] != prefCourse || w["CrHrs"] != prefCrHrs || w["Comments"] != prefComments {
			t.Errorf("unexpected widths: %v", w)
		}
	})

	t.Run("HIGH+MED fit, LOW gets surplus", func(t *testing.T) {
		// available must be >= sumHigh+sumMed but < totalPreferred
		// sumHigh=11, sumMed=7, prefComments=25
		// target available = 11+7+10 = 28 (surplus of 10 for LOW)
		// available = termWidth - outerBorderWidth - cellPadding*3
		// 28 = termWidth - 2 - 6 => termWidth = 36
		w := computeColWidths(cols, 36)
		if w["Course"] != prefCourse {
			t.Errorf("Course: got %d, want %d (preferred)", w["Course"], prefCourse)
		}
		if w["CrHrs"] != prefCrHrs {
			t.Errorf("CrHrs: got %d, want %d (preferred)", w["CrHrs"], prefCrHrs)
		}
		if w["Comments"] <= minColWidth {
			t.Errorf("Comments should be above minColWidth, got %d", w["Comments"])
		}
		if w["Comments"] >= prefComments {
			t.Errorf("Comments should be below preferred, got %d", w["Comments"])
		}
	})

	t.Run("only HIGH fits, MED scaled, LOW at min", func(t *testing.T) {
		// remainAfterLow must be >= sumHigh but < sumHigh+sumMed
		// sumHigh=11, sumMed=7; target remainAfterLow = 14
		// remainAfterLow = available - lowMin = available - 1*3
		// available = 14+3 = 17 = termWidth - 2 - 6 => termWidth = 25
		w := computeColWidths(cols, 25)
		if w["Course"] != prefCourse {
			t.Errorf("Course: got %d, want %d (preferred)", w["Course"], prefCourse)
		}
		if w["CrHrs"] >= prefCrHrs {
			t.Errorf("CrHrs should be scaled below preferred, got %d", w["CrHrs"])
		}
		if w["Comments"] != minColWidth {
			t.Errorf("Comments: got %d, want minColWidth (%d)", w["Comments"], minColWidth)
		}
	})

	t.Run("very narrow, HIGH scaled, MED+LOW at min", func(t *testing.T) {
		// remainAfterLow < sumHigh (11)
		// remainAfterLow = available - 3 < 11 => available < 14
		// available = termWidth - 2 - 6 => termWidth < 22; use termWidth=20 => available=12
		w := computeColWidths(cols, 20)
		if w["Course"] >= prefCourse {
			t.Errorf("Course should be scaled below preferred, got %d", w["Course"])
		}
		if w["CrHrs"] != minColWidth {
			t.Errorf("CrHrs: got %d, want minColWidth (%d)", w["CrHrs"], minColWidth)
		}
		if w["Comments"] != minColWidth {
			t.Errorf("Comments: got %d, want minColWidth (%d)", w["Comments"], minColWidth)
		}
	})

	t.Run("unknown column uses defaults, no panic", func(t *testing.T) {
		w := computeColWidths([]string{"MYSTERY"}, 0)
		if w["MYSTERY"] != defaultColWidth {
			t.Errorf("MYSTERY: got %d, want %d (defaultColWidth)", w["MYSTERY"], defaultColWidth)
		}
	})
}

// ---- key-handler tests ----

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

func TestResultsNextTermBlockedAtLatest(t *testing.T) {
	m := resultsModel{
		queryType:  "schedule_lookup",
		params:     map[string]string{"term": "202630", "username": "testuser"},
		latestTerm: "202630",
	}

	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	if cmd != nil {
		t.Fatalf("expected nil cmd when next term is blocked, got a command")
	}
	rm := model.(resultsModel)
	if rm.termWarning == "" {
		t.Error("expected termWarning to be set when next term is blocked")
	}
	if rm.params["term"] != "202630" {
		t.Errorf("term changed to %q, want it unchanged at 202630", rm.params["term"])
	}

	// h still navigates back from the latest term, and clears the warning.
	model, cmd = rm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	if cmd == nil {
		t.Fatal("h returned nil cmd at the latest term")
	}
	if _, ok := cmd().(changeTermMsg); !ok {
		t.Errorf("expected changeTermMsg from h, got %T", cmd())
	}
	if model.(resultsModel).termWarning != "" {
		t.Error("expected termWarning cleared after pressing h")
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

// enter on the section picker drills into the highlighted section's roster,
// tagged with from_sections so esc can return.
func TestResultsSectionPickerEnter(t *testing.T) {
	tbl := table.New(
		table.WithColumns([]table.Column{{Title: "Course", Width: 12}}),
		table.WithRows([]table.Row{{"CSSE474-02"}}),
		table.WithFocused(true),
	)
	m := resultsModel{
		queryType: "roster_view",
		params:    map[string]string{"term": "202630", "course_id": "CSSE474"},
		result:    query.Result{Mode: query.ModeSections},
		table:     tbl,
	}
	ss := expectSearchSubmitted(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if ss.queryType != "roster_view" {
		t.Errorf("queryType = %q, want roster_view", ss.queryType)
	}
	if ss.params["course_id"] != "CSSE474-02" {
		t.Errorf("course_id = %q, want CSSE474-02", ss.params["course_id"])
	}
	if ss.params["from_sections"] != "1" {
		t.Errorf("from_sections = %q, want 1", ss.params["from_sections"])
	}
}

// 's' on a single-section roster toggles to the combined roster, remembering
// which section to return to.
func TestResultsRosterToggleToCombined(t *testing.T) {
	m := resultsModel{
		queryType: "roster_view",
		params:    map[string]string{"term": "202630", "course_id": "CSSE474-02", "from_sections": "1"},
		result:    query.Result{},
	}
	ss := expectSearchSubmitted(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if ss.params["course_id"] != "CSSE474" {
		t.Errorf("course_id = %q, want CSSE474 (base)", ss.params["course_id"])
	}
	if ss.params["combined"] != "1" {
		t.Errorf("combined = %q, want 1", ss.params["combined"])
	}
	if ss.params["selected_section"] != "CSSE474-02" {
		t.Errorf("selected_section = %q, want CSSE474-02", ss.params["selected_section"])
	}
	if ss.params["from_sections"] != "1" {
		t.Errorf("from_sections = %q, want 1", ss.params["from_sections"])
	}
}

// 's' on the combined roster toggles back to the previously selected section.
func TestResultsRosterToggleToSection(t *testing.T) {
	m := resultsModel{
		queryType: "roster_view",
		params: map[string]string{
			"term": "202630", "course_id": "CSSE474", "combined": "1",
			"selected_section": "CSSE474-02", "from_sections": "1",
		},
		result: query.Result{},
	}
	ss := expectSearchSubmitted(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if ss.params["course_id"] != "CSSE474-02" {
		t.Errorf("course_id = %q, want CSSE474-02", ss.params["course_id"])
	}
	if ss.params["combined"] != "" {
		t.Errorf("combined = %q, want empty", ss.params["combined"])
	}
}

// 'c' (change section) jumps from the combined roster back to the section picker.
func TestResultsRosterChangeSection(t *testing.T) {
	m := resultsModel{
		queryType: "roster_view",
		params: map[string]string{
			"term": "202630", "course_id": "CSSE474", "combined": "1",
			"selected_section": "CSSE474-02", "from_sections": "1",
		},
		result: query.Result{},
	}
	ss := expectSearchSubmitted(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	if ss.queryType != "roster_view" {
		t.Errorf("queryType = %q, want roster_view", ss.queryType)
	}
	if ss.params["course_id"] != "CSSE474" {
		t.Errorf("course_id = %q, want CSSE474 (picker)", ss.params["course_id"])
	}
	if ss.params["combined"] != "" {
		t.Errorf("picker should not carry combined, got %q", ss.params["combined"])
	}
}

// esc on a roster reached via the picker returns to the picker, not the menu.
func TestResultsRosterEscReturnsToPicker(t *testing.T) {
	m := resultsModel{
		queryType: "roster_view",
		params:    map[string]string{"term": "202630", "course_id": "CSSE474-02", "from_sections": "1"},
		result:    query.Result{},
	}
	ss := expectSearchSubmitted(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if ss.queryType != "roster_view" {
		t.Errorf("queryType = %q, want roster_view", ss.queryType)
	}
	if ss.params["course_id"] != "CSSE474" {
		t.Errorf("course_id = %q, want CSSE474 (picker)", ss.params["course_id"])
	}
	if ss.params["from_sections"] != "" {
		t.Errorf("picker should not carry from_sections, got %q", ss.params["from_sections"])
	}
}

// esc on a directly-opened roster (no picker origin) goes back to the menu.
func TestResultsRosterEscWithoutPickerGoesBack(t *testing.T) {
	m := resultsModel{
		queryType: "roster_view",
		params:    map[string]string{"term": "202630", "course_id": "CSSE474-02"},
		result:    query.Result{},
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc returned nil cmd")
	}
	if _, ok := cmd().(backMsg); !ok {
		t.Errorf("expected backMsg, got %T", cmd())
	}
}

func expectSearchSubmitted(t *testing.T, m resultsModel, key tea.KeyMsg) searchSubmittedMsg {
	t.Helper()
	_, cmd := m.Update(key)
	if cmd == nil {
		t.Fatal("key returned nil cmd")
	}
	msg := cmd()
	ss, ok := msg.(searchSubmittedMsg)
	if !ok {
		t.Fatalf("expected searchSubmittedMsg, got %T", msg)
	}
	return ss
}
