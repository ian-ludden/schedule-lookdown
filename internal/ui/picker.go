package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/luddenig/schedule-lookdown/internal/models"
)

// defaultPickerRows is the list height used before a window size is known.
const defaultPickerRows = 12

// personItem adapts models.Person to list.Item. FilterValue feeds the list's
// built-in fuzzy filter, so typing matches against last name, first name and
// the (YR-MAJOR) detail.
type personItem struct{ p models.Person }

func (i personItem) FilterValue() string { return i.p.FilterText() }

// pickerDelegate renders one row as "> [x] Last, First (YR-MAJOR)". It holds
// the selection map by reference so toggles show without rebuilding items.
type pickerDelegate struct {
	selected map[string]bool
	focused  *bool
}

func (d pickerDelegate) Height() int                             { return 1 }
func (d pickerDelegate) Spacing() int                            { return 0 }
func (d pickerDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d pickerDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	pi, ok := item.(personItem)
	if !ok {
		return
	}
	box := "[ ]"
	if d.selected[pi.p.ID] {
		box = "[x]"
	}
	text := pi.p.DisplayName()
	if pi.p.Detail != "" {
		text += " (" + pi.p.Detail + ")"
	}
	line := box + " " + text

	cursor := "  "
	style := normalStyle
	if index == m.Index() && *d.focused {
		cursor = "> "
		style = selectedStyle
	} else if d.selected[pi.p.ID] {
		style = selectedStyle
	}
	if m.Width() > 0 {
		style = style.MaxWidth(m.Width() - 2)
	}
	fmt.Fprint(w, cursor+style.Render(line))
}

// pickerModel is a filterable multi-select over the ad-hoc group directory.
// Selections live in a map outside the list, so they survive filter changes.
type pickerModel struct {
	list     list.Model
	selected map[string]bool
	order    []string // selected IDs in the order they were picked
	byID     map[string]models.Person
	focused  *bool // shared with the delegate
	loaded   bool
	loadErr  string
	width    int
}

func newPickerModel() pickerModel {
	selected := map[string]bool{}
	focused := new(bool)
	l := list.New(nil, pickerDelegate{selected: selected, focused: focused}, 0, defaultPickerRows)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(true)
	l.SetStatusBarItemName("person", "people")
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()
	return pickerModel{list: l, selected: selected, byID: map[string]models.Person{}, focused: focused}
}

// setDirectory supplies the loaded people, or the reason loading failed.
func (m *pickerModel) setDirectory(people []models.Person, loadErr string) {
	m.loadErr = loadErr
	m.loaded = loadErr != "" || people != nil
	items := make([]list.Item, len(people))
	m.byID = make(map[string]models.Person, len(people))
	for i, p := range people {
		items[i] = personItem{p}
		m.byID[p.ID] = p
	}
	m.list.SetItems(items)
}

func (m *pickerModel) setSize(width, rows int) {
	m.width = width
	if rows < 3 {
		rows = 3
	}
	m.list.SetSize(width, rows)
}

func (m *pickerModel) setFocused(f bool) { *m.focused = f }

// ready reports whether the directory is loaded and usable.
func (m pickerModel) ready() bool { return m.loaded && m.loadErr == "" }

// typing reports whether the filter prompt is capturing keystrokes.
func (m pickerModel) typing() bool { return m.list.SettingFilter() }

// filtering reports whether any filter is active (being typed or applied).
func (m pickerModel) filtering() bool { return m.list.FilterState() != list.Unfiltered }

// atTop reports whether the cursor is on the first visible row.
func (m pickerModel) atTop() bool { return m.list.Index() == 0 }

// selectedIDs returns the chosen IDs in pick order.
func (m pickerModel) selectedIDs() []string { return append([]string(nil), m.order...) }

func (m *pickerModel) toggleCurrent() {
	it, ok := m.list.SelectedItem().(personItem)
	if !ok {
		return
	}
	id := it.p.ID
	if m.selected[id] {
		delete(m.selected, id)
		for i, v := range m.order {
			if v == id {
				m.order = append(m.order[:i], m.order[i+1:]...)
				break
			}
		}
		return
	}
	m.selected[id] = true
	m.order = append(m.order, id)
}

func (m *pickerModel) clearSelection() {
	clear(m.selected)
	m.order = nil
}

// Update handles space (toggle) and c (clear) while the filter prompt is not
// capturing input; everything else goes to the list.
func (m pickerModel) Update(msg tea.Msg) (pickerModel, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && m.ready() && !m.typing() {
		switch km.String() {
		case " ":
			m.toggleCurrent()
			return m, nil
		case "c":
			m.clearSelection()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m pickerModel) summary() string {
	if len(m.order) == 0 {
		return helpStyle.Render("Selected (0)")
	}
	names := make([]string, len(m.order))
	for i, id := range m.order {
		names[i] = m.byID[id].DisplayName()
	}
	s := fmt.Sprintf("Selected (%d): %s", len(names), strings.Join(names, " • "))
	st := normalStyle
	if m.width > 0 {
		st = st.MaxWidth(m.width)
	}
	return st.Render(s)
}

func (m pickerModel) View() string {
	switch {
	case m.loadErr != "":
		return errorStyle.Render("Couldn't load the people directory: "+m.loadErr) + "\n" +
			helpStyle.Render("Press esc to go back; the directory is retried on the next launch.")
	case !m.loaded:
		return helpStyle.Render("Loading people directory…")
	}
	return m.summary() + "\n" + lipgloss.NewStyle().Render(m.list.View())
}
