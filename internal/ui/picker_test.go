package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luddenig/schedule-lookdown/internal/models"
)

var testPeople = []models.Person{
	{ID: "1", Last: "Adams", First: "Ann", Detail: "SR-CS"},
	{ID: "2", Last: "Baker", First: "Bob", Detail: "JR-ME"},
	{ID: "3", Last: "Chen", First: "Cy"},
}

func adhocModel(t *testing.T) searchModel {
	t.Helper()
	m := newSearchModelForQuery("adhoc_group", "", "202710", "")
	m.setSize(100, 40)
	m.setDirectory(testPeople, "")
	// term row is focused first; tab into the picker.
	return press(t, m, tea.KeyMsg{Type: tea.KeyTab})
}

func press(t *testing.T, m searchModel, msg tea.Msg) searchModel {
	t.Helper()
	next, _ := m.Update(msg)
	return next.(searchModel)
}

func pickerOf(m searchModel) *pickerModel {
	for i := range m.fields {
		if m.fields[i].kind == fieldPicker {
			return &m.fields[i].picker
		}
	}
	return nil
}

func TestPickerToggleAndSubmitParams(t *testing.T) {
	m := adhocModel(t)
	m = press(t, m, keyMsg(" "))                   // select Adams
	m = press(t, m, tea.KeyMsg{Type: tea.KeyDown}) // move to Baker
	m = press(t, m, tea.KeyMsg{Type: tea.KeyDown}) // move to Chen
	m = press(t, m, keyMsg(" "))                   // select Chen
	if got := pickerOf(m).selectedIDs(); strings.Join(got, ",") != "1,3" {
		t.Fatalf("selected = %v, want [1 3]", got)
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter with a selection should submit")
	}
	sub, ok := cmd().(searchSubmittedMsg)
	if !ok {
		t.Fatalf("cmd returned %T, want searchSubmittedMsg", cmd())
	}
	if sub.queryType != "adhoc_group" || sub.params["ids"] != "1 3" || sub.params["term"] != "202710" {
		t.Errorf("submit = %+v", sub)
	}

	// Toggling again deselects.
	m = press(t, m, keyMsg(" "))
	if got := pickerOf(m).selectedIDs(); strings.Join(got, ",") != "1" {
		t.Errorf("after untoggle selected = %v, want [1]", got)
	}
}

func TestPickerTooFewSubmitBlocked(t *testing.T) {
	m := adhocModel(t)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("enter with nothing selected must not submit")
	}
	if !strings.Contains(next.(searchModel).termWarning, "two people") {
		t.Errorf("warning = %q", next.(searchModel).termWarning)
	}
}

func TestPickerSinglePersonWarnsToUseScheduleLookup(t *testing.T) {
	m := adhocModel(t)
	m = press(t, m, keyMsg(" "))
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("one person must not submit")
	}
	if w := next.(searchModel).termWarning; !strings.Contains(w, "Schedule Lookup") {
		t.Errorf("warning = %q, want mention of Schedule Lookup", w)
	}
}

func TestPickerEnterWhileFilteringTogglesTopMatchAndKeepsFilter(t *testing.T) {
	m := adhocModel(t)
	m = press(t, m, keyMsg("/"))
	for _, r := range "chen" {
		m = pressDrain(t, m, keyMsg(string(r)))
	}
	m = pressDrain(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	p := pickerOf(m)
	if got := strings.Join(p.selectedIDs(), ","); got != "3" {
		t.Fatalf("selected = %q, want 3", got)
	}
	if !p.typing() {
		t.Error("filter prompt should stay open after enter")
	}
	// Enter again deselects; the filter is still open.
	m = pressDrain(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if n := len(pickerOf(m).selectedIDs()); n != 0 {
		t.Errorf("selected after second enter = %d, want 0", n)
	}
}

func TestPickerClear(t *testing.T) {
	m := adhocModel(t)
	m = press(t, m, keyMsg(" "))
	m = press(t, m, keyMsg("c"))
	if n := len(pickerOf(m).selectedIDs()); n != 0 {
		t.Errorf("selected after clear = %d", n)
	}
}

func TestPickerSelectionSurvivesFilter(t *testing.T) {
	m := adhocModel(t)
	m = press(t, m, keyMsg(" ")) // select Adams
	m = press(t, m, keyMsg("/"))
	for _, r := range "chen" {
		m = pressDrain(t, m, keyMsg(string(r)))
	}
	m = pressDrain(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // apply filter
	p := pickerOf(m)
	if n := len(p.list.VisibleItems()); n != 1 {
		t.Fatalf("visible after filter = %d, want 1", n)
	}
	m = press(t, m, keyMsg(" "))                  // select Chen
	m = press(t, m, tea.KeyMsg{Type: tea.KeyEsc}) // clear filter
	p = pickerOf(m)
	if got := strings.Join(p.selectedIDs(), ","); got != "1,3" {
		t.Errorf("selected = %q, want 1,3", got)
	}
}

func TestPickerTypingSpaceGoesToFilter(t *testing.T) {
	m := adhocModel(t)
	m = press(t, m, keyMsg("/")) // start filtering
	if !pickerOf(m).typing() {
		t.Fatal("expected filter prompt active")
	}
	m = press(t, m, keyMsg(" "))
	if n := len(pickerOf(m).selectedIDs()); n != 0 {
		t.Error("space while typing a filter must not toggle")
	}
}

func TestPickerUpAtTopReturnsToTerm(t *testing.T) {
	m := adhocModel(t)
	m = press(t, m, tea.KeyMsg{Type: tea.KeyUp})
	if m.fields[m.focused].kind != fieldTerm {
		t.Errorf("focused kind = %v, want term", m.fields[m.focused].kind)
	}
}

func TestPickerLoadingAndErrorViews(t *testing.T) {
	m := newSearchModelForQuery("adhoc_group", "", "202710", "")
	if v := m.View(); !strings.Contains(v, "Loading") {
		t.Errorf("want loading view, got:\n%s", v)
	}
	m.setDirectory(nil, "HTTP 500")
	if v := m.View(); !strings.Contains(v, "Couldn't load") || !strings.Contains(v, "HTTP 500") {
		t.Errorf("want error banner, got:\n%s", v)
	}
}

// pressDrain sends a key, then feeds back the messages of any returned
// commands (the list filters asynchronously). Slow commands such as cursor
// blink timers are abandoned after a short timeout.
func pressDrain(t *testing.T, m searchModel, msg tea.Msg) searchModel {
	t.Helper()
	next, cmd := m.Update(msg)
	m = next.(searchModel)
	var run func(c tea.Cmd)
	run = func(c tea.Cmd) {
		if c == nil {
			return
		}
		ch := make(chan tea.Msg, 1)
		go func() { ch <- c() }()
		select {
		case out := <-ch:
			if batch, ok := out.(tea.BatchMsg); ok {
				for _, bc := range batch {
					run(bc)
				}
				return
			}
			if out == nil {
				return
			}
			n, c2 := m.Update(out)
			m = n.(searchModel)
			run(c2)
		case <-time.After(100 * time.Millisecond):
		}
	}
	run(cmd)
	return m
}
