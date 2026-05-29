package ui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luddenig/schedule-lookdown/internal/query"
)

// makeEntry is a test helper that constructs a HistoryEntry.
func makeEntry(queryType string, params map[string]string, fetchedAt time.Time) HistoryEntry {
	return HistoryEntry{
		QueryType: queryType,
		Params:    params,
		Result:    query.Result{Columns: []string{"Col"}, Rows: [][]string{{"val"}}},
		FetchedAt: fetchedAt,
	}
}

var (
	t0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 = t0.Add(time.Hour)
	t2 = t0.Add(2 * time.Hour)
)

// ── entryIntentKey ────────────────────────────────────────────────────────────

func TestEntryIntentKey_DeterministicAcrossMapOrder(t *testing.T) {
	a := entryIntentKey("schedule_lookup", map[string]string{"term": "202630", "username": "ian"})
	b := entryIntentKey("schedule_lookup", map[string]string{"username": "ian", "term": "202630"})
	if a != b {
		t.Errorf("keys differ for same params: %q vs %q", a, b)
	}
}

func TestEntryIntentKey_DiffersByQueryType(t *testing.T) {
	params := map[string]string{"term": "202630", "username": "ian"}
	if entryIntentKey("schedule_lookup", params) == entryIntentKey("instructor_lookup", params) {
		t.Error("different query types must produce different keys")
	}
}

func TestEntryIntentKey_DiffersByParamValue(t *testing.T) {
	a := entryIntentKey("schedule_lookup", map[string]string{"username": "ian"})
	b := entryIntentKey("schedule_lookup", map[string]string{"username": "bob"})
	if a == b {
		t.Error("different param values must produce different keys")
	}
}

// ── queryHistory.add ──────────────────────────────────────────────────────────

func TestHistoryAdd_BasicInsert(t *testing.T) {
	var h queryHistory
	h.add(makeEntry("schedule_lookup", map[string]string{"username": "ian"}, t0))
	if len(h.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(h.entries))
	}
}

func TestHistoryAdd_DeduplicatesExistingIntent(t *testing.T) {
	var h queryHistory
	params := map[string]string{"username": "ian"}
	h.add(makeEntry("schedule_lookup", params, t0))

	updated := makeEntry("schedule_lookup", params, t1)
	updated.Result = query.Result{Columns: []string{"X"}}
	h.add(updated)

	if len(h.entries) != 1 {
		t.Fatalf("expected 1 entry after dedup, got %d", len(h.entries))
	}
	if h.entries[0].FetchedAt != t1 {
		t.Errorf("FetchedAt not updated: got %v, want %v", h.entries[0].FetchedAt, t1)
	}
	if len(h.entries[0].Result.Columns) != 1 || h.entries[0].Result.Columns[0] != "X" {
		t.Error("Result not updated on dedup")
	}
}

func TestHistoryAdd_EvictsOldestAtCapacity(t *testing.T) {
	var h queryHistory
	// Fill to capacity.
	for i := range historyLimit {
		h.add(makeEntry("schedule_lookup",
			map[string]string{"username": string(rune('a' + i))},
			t0.Add(time.Duration(i)*time.Minute),
		))
	}
	if len(h.entries) != historyLimit {
		t.Fatalf("expected %d entries, got %d", historyLimit, len(h.entries))
	}

	// Adding one more should evict the entry with the earliest FetchedAt (username "a").
	h.add(makeEntry("schedule_lookup", map[string]string{"username": "new"}, t2))

	if len(h.entries) != historyLimit {
		t.Fatalf("expected %d entries after eviction, got %d", historyLimit, len(h.entries))
	}
	for _, e := range h.entries {
		if e.Params["username"] == "a" {
			t.Error("oldest entry 'a' should have been evicted")
		}
	}
}

// ── queryHistory.clear ────────────────────────────────────────────────────────

func TestHistoryClear(t *testing.T) {
	var h queryHistory
	h.add(makeEntry("schedule_lookup", map[string]string{"username": "ian"}, t0))
	h.clear()
	if len(h.entries) != 0 {
		t.Errorf("expected 0 entries after clear, got %d", len(h.entries))
	}
}

// ── queryHistory.sorted ───────────────────────────────────────────────────────

func TestHistorySorted_DescendingOrder(t *testing.T) {
	var h queryHistory
	h.add(makeEntry("schedule_lookup", map[string]string{"username": "ian"}, t0))
	h.add(makeEntry("schedule_lookup", map[string]string{"username": "bob"}, t2))
	h.add(makeEntry("schedule_lookup", map[string]string{"username": "sue"}, t1))

	sorted := h.sorted()
	if len(sorted) != 3 {
		t.Fatalf("expected 3 sorted entries, got %d", len(sorted))
	}
	if sorted[0].Params["username"] != "bob" {
		t.Errorf("first entry should be newest (bob), got %s", sorted[0].Params["username"])
	}
	if sorted[2].Params["username"] != "ian" {
		t.Errorf("last entry should be oldest (ian), got %s", sorted[2].Params["username"])
	}
}

func TestHistorySorted_DoesNotMutateOriginal(t *testing.T) {
	var h queryHistory
	h.add(makeEntry("schedule_lookup", map[string]string{"username": "ian"}, t0))
	h.add(makeEntry("schedule_lookup", map[string]string{"username": "bob"}, t2))

	_ = h.sorted()

	// Original slice order should be unaffected by sorting.
	if h.entries[0].Params["username"] != "ian" {
		t.Error("sorted() must not mutate h.entries order")
	}
}

// ── queryHistory.findOldestIndex ──────────────────────────────────────────────

func TestFindOldestIndex_Empty(t *testing.T) {
	var h queryHistory
	if got := h.findOldestIndex(); got != -1 {
		t.Errorf("expected -1 for empty history, got %d", got)
	}
}

func TestFindOldestIndex_CorrectIndex(t *testing.T) {
	var h queryHistory
	h.add(makeEntry("schedule_lookup", map[string]string{"username": "a"}, t2))
	h.add(makeEntry("schedule_lookup", map[string]string{"username": "b"}, t0)) // oldest
	h.add(makeEntry("schedule_lookup", map[string]string{"username": "c"}, t1))

	idx := h.findOldestIndex()
	if h.entries[idx].Params["username"] != "b" {
		t.Errorf("oldest index %d points to %s, want b", idx, h.entries[idx].Params["username"])
	}
}

// ── marshal / unmarshal ───────────────────────────────────────────────────────

func TestHistoryMarshalUnmarshal_RoundTrip(t *testing.T) {
	var h queryHistory
	h.add(makeEntry("schedule_lookup", map[string]string{"username": "ian", "term": "202630"}, t0))
	h.add(makeEntry("roster_view", map[string]string{"course_id": "CSSE474-02", "term": "202630"}, t1))

	data, err := h.marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var h2 queryHistory
	if err := h2.unmarshal(data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(h2.entries) != 2 {
		t.Fatalf("expected 2 entries after round-trip, got %d", len(h2.entries))
	}
	if h2.entries[0].QueryType != "schedule_lookup" {
		t.Errorf("first entry QueryType = %q, want schedule_lookup", h2.entries[0].QueryType)
	}
}

// ── historyEntryLabel ─────────────────────────────────────────────────────────

func TestHistoryEntryLabel_AllQueryTypes(t *testing.T) {
	cases := []struct {
		entry    HistoryEntry
		wantPfx  string
	}{
		{
			makeEntry("course_search", map[string]string{"course_code": "CSSE474", "term": "202630"}, t0),
			"Course:",
		},
		{
			makeEntry("schedule_lookup", map[string]string{"username": "luddenig", "term": "202630"}, t0),
			"Sched:",
		},
		{
			makeEntry("roster_view", map[string]string{"course_id": "CSSE474-02", "term": "202630"}, t0),
			"Roster:",
		},
		{
			makeEntry("instructor_lookup", map[string]string{"username": "smithj", "term": "202630"}, t0),
			"Instr:",
		},
	}
	for _, tc := range cases {
		label := historyEntryLabel(tc.entry)
		if len(label) == 0 {
			t.Errorf("empty label for %s", tc.entry.QueryType)
		}
		// Label must fit within the panel width (minus cursor prefix).
		if len(label) > panelInnerWidth-2 {
			t.Errorf("%s label too long: %d chars (max %d): %q",
				tc.entry.QueryType, len(label), panelInnerWidth-2, label)
		}
		found := false
		for i := range len(label) - len(tc.wantPfx) + 1 {
			if label[i:i+len(tc.wantPfx)] == tc.wantPfx {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("label %q does not contain prefix %q", label, tc.wantPfx)
		}
	}
}

func TestHistoryEntryLabel_CourseSearchFallsBackToInstructor(t *testing.T) {
	e := makeEntry("course_search", map[string]string{"course_code": "", "instructor": "Smith"}, t0)
	label := historyEntryLabel(e)
	if label == "Course: " {
		t.Error("course_search with empty course_code should fall back to instructor name")
	}
}

func TestTruncateStr(t *testing.T) {
	if got := truncateStr("hello", 10); got != "hello" {
		t.Errorf("no truncation needed: got %q", got)
	}
	if got := truncateStr("hello world", 5); len([]rune(got)) > 5 {
		t.Errorf("truncated string too long: %q", got)
	}
	if got := truncateStr("hi", 1); got != "h" {
		t.Errorf("max=1: got %q, want h", got)
	}
}

// ── historyPanelModel ─────────────────────────────────────────────────────────

func newFocusedPanel(entries []HistoryEntry) historyPanelModel {
	return historyPanelModel{entries: entries, focused: true, height: 20}
}

func TestHistoryPanel_NavigateUpDown(t *testing.T) {
	entries := []HistoryEntry{
		makeEntry("schedule_lookup", map[string]string{"username": "a"}, t0),
		makeEntry("schedule_lookup", map[string]string{"username": "b"}, t1),
	}
	m := newFocusedPanel(entries)

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	p := m2.(historyPanelModel)
	if p.cursor != 1 {
		t.Errorf("after down: cursor = %d, want 1", p.cursor)
	}

	m3, _ := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	p = m3.(historyPanelModel)
	if p.cursor != 0 {
		t.Errorf("after up: cursor = %d, want 0", p.cursor)
	}
}

func TestHistoryPanel_UpAtTopDoesNotWrap(t *testing.T) {
	m := newFocusedPanel([]HistoryEntry{
		makeEntry("schedule_lookup", map[string]string{"username": "a"}, t0),
	})
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m2.(historyPanelModel).cursor != 0 {
		t.Error("cursor should stay at 0 when pressing up at the top")
	}
}

func TestHistoryPanel_DownAtBottomDoesNotWrap(t *testing.T) {
	m := newFocusedPanel([]HistoryEntry{
		makeEntry("schedule_lookup", map[string]string{"username": "a"}, t0),
	})
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m2.(historyPanelModel).cursor != 0 {
		t.Error("cursor should stay at 0 when pressing down at the bottom")
	}
}

func TestHistoryPanel_EnterEmitsSelectedMsg(t *testing.T) {
	entries := []HistoryEntry{
		makeEntry("roster_view", map[string]string{"course_id": "CSSE474-02", "term": "202630"}, t0),
	}
	m := newFocusedPanel(entries)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter with entry selected should return a cmd")
	}
	msg := cmd()
	sel, ok := msg.(historyEntrySelectedMsg)
	if !ok {
		t.Fatalf("expected historyEntrySelectedMsg, got %T", msg)
	}
	if sel.queryType != "roster_view" {
		t.Errorf("selected queryType = %q, want roster_view", sel.queryType)
	}
}

func TestHistoryPanel_XEmitsClearedMsg(t *testing.T) {
	m := newFocusedPanel([]HistoryEntry{
		makeEntry("schedule_lookup", map[string]string{"username": "a"}, t0),
	})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if cmd == nil {
		t.Fatal("x should return a cmd")
	}
	if _, ok := cmd().(historyClearedMsg); !ok {
		t.Error("x should emit historyClearedMsg")
	}
}

func TestHistoryPanel_EscUnfocuses(t *testing.T) {
	m := newFocusedPanel([]HistoryEntry{
		makeEntry("schedule_lookup", map[string]string{"username": "a"}, t0),
	})
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m2.(historyPanelModel).focused {
		t.Error("esc should unfocus the panel")
	}
}

func TestHistoryPanel_UnfocusedIgnoresKeys(t *testing.T) {
	m := historyPanelModel{
		entries: []HistoryEntry{makeEntry("schedule_lookup", map[string]string{"username": "a"}, t0)},
		focused: false,
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("unfocused panel should not handle enter")
	}
}

func TestHistoryPanel_ViewRendersWithoutPanic(t *testing.T) {
	m := historyPanelModel{
		entries: []HistoryEntry{
			makeEntry("schedule_lookup", map[string]string{"username": "ian"}, t0),
		},
		focused: true,
		height:  20,
	}
	v := m.View()
	if v == "" {
		t.Error("View() returned empty string")
	}
}
