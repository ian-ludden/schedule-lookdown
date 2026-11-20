package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/luddenig/schedule-lookdown/internal/models"
)

type fieldKind int

const (
	fieldText fieldKind = iota
	fieldTerm
	fieldPicker // multi-select people picker (ad-hoc group)
)

type field struct {
	label  string
	key    string
	kind   fieldKind
	input  textinput.Model // used when kind == fieldText
	term   string          // YYYYTT code, used when kind == fieldTerm
	picker pickerModel     // used when kind == fieldPicker
}

type searchModel struct {
	queryType      string
	fields         []field
	focused        int
	storedUsername string
	latestTerm     string // furthest-future available term; "" means no upper bound
	termWarning    string // transient message shown when forward nav is blocked
	width, height  int    // main-area size; 0 until known
}

func newSearchModel() searchModel { return searchModel{} }

func newSearchModelForQuery(queryType string, storedUsername string, defaultTerm string, latestTerm string) searchModel {
	return searchModel{
		queryType:      queryType,
		fields:         fieldsForQuery(queryType, defaultTerm),
		storedUsername: storedUsername,
		latestTerm:     latestTerm,
	}
}

func fieldsForQuery(queryType string, defaultTerm string) []field {
	newInput := func(placeholder string) textinput.Model {
		t := textinput.New()
		t.Placeholder = placeholder
		t.Width = 40
		return t
	}

	termField := field{label: "Term", key: "term", kind: fieldTerm, term: defaultTerm}

	switch queryType {
	case "course_search":
		return []field{
			termField,
			{label: "Course Code", key: "course_code", input: newInput("CSSE 132")},
			{label: "Instructor", key: "instructor", input: newInput("Last name")},
		}
	case "schedule_lookup":
		return []field{
			termField,
			{label: "Username", key: "username", input: newInput("RHIT username")},
		}
	case "roster_view":
		return []field{
			termField,
			{label: "Course ID", key: "course_id", input: newInput("CSSE474-02")},
		}
	case "adhoc_group":
		return []field{
			termField,
			{label: "People", key: "ids", kind: fieldPicker, picker: newPickerModel()},
		}
	}
	return nil
}

func (m searchModel) Init() tea.Cmd {
	if len(m.fields) == 0 {
		return nil
	}
	if m.fields[0].kind == fieldText {
		return m.fields[0].input.Focus()
	}
	return nil
}

// blurFocused blurs the currently focused field's input, if it is a text field.
func (m *searchModel) blurFocused() {
	switch m.fields[m.focused].kind {
	case fieldText:
		m.fields[m.focused].input.Blur()
	case fieldPicker:
		m.fields[m.focused].picker.setFocused(false)
	}
}

// focusField sets focus to field i and returns a focus cmd if it is a text field.
func (m *searchModel) focusField(i int) tea.Cmd {
	m.focused = i
	switch m.fields[i].kind {
	case fieldText:
		return m.fields[i].input.Focus()
	case fieldPicker:
		m.fields[i].picker.setFocused(true)
	}
	return nil
}

// setSize records the main-area size and resizes any picker to fit.
func (m *searchModel) setSize(width, height int) {
	m.width, m.height = width, height
	for i := range m.fields {
		if m.fields[i].kind == fieldPicker {
			// Reserve rows for the title block, term row, labels, summary,
			// status bar, help and a possible warning line.
			m.fields[i].picker.setSize(width, height-14)
		}
	}
}

// setDirectory hands the loaded people (or a load error) to the picker.
func (m *searchModel) setDirectory(people []models.Person, loadErr string) {
	for i := range m.fields {
		if m.fields[i].kind == fieldPicker {
			m.fields[i].picker.setDirectory(people, loadErr)
		}
	}
}

// updatePicker handles keys while the picker field is focused. It reports
// whether the key was consumed.
func (m searchModel) updatePicker(msg tea.KeyMsg) (searchModel, tea.Cmd, bool) {
	p := &m.fields[m.focused].picker
	key := msg.String()

	// While the filter prompt captures typing, keys belong to the list, except
	// enter: it toggles the top match and leaves the filter open so several
	// people can be picked in a row.
	if p.typing() {
		if key == "enter" {
			p.toggleCurrent()
			return m, nil, true
		}
		var cmd tea.Cmd
		*p, cmd = p.Update(msg)
		return m, cmd, true
	}

	switch key {
	case "esc":
		if p.filtering() { // clear the applied filter first
			var cmd tea.Cmd
			*p, cmd = p.Update(msg)
			return m, cmd, true
		}
		return m, func() tea.Msg { return backMsg{} }, true
	case "tab", "shift+tab":
		return m, nil, false // normal field navigation
	case "up":
		if p.atTop() { // leave the list upwards
			return m, nil, false
		}
	case "enter":
		switch n := len(p.selectedIDs()); {
		case !p.ready() || n == 0:
			m.termWarning = "Select at least two people first (space toggles the highlighted row)"
			return m, nil, true
		case n == 1:
			m.termWarning = "Only one person selected. For a single schedule, use Schedule Lookup instead"
			return m, nil, true
		}
		return m, m.submitCmd(), true
	}
	var cmd tea.Cmd
	*p, cmd = p.Update(msg)
	return m, cmd, true
}

func (m searchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if len(m.fields) == 0 {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Any key clears a transient term-navigation warning so it disappears on
		// the next interaction.
		m.termWarning = ""
		if m.fields[m.focused].kind == fieldPicker {
			if mm, cmd, handled := m.updatePicker(msg); handled {
				return mm, cmd
			}
		}
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return backMsg{} }
		case "^":
			if m.storedUsername != "" && m.queryType == "schedule_lookup" {
				for i, f := range m.fields {
					if f.key == "username" {
						m.fields[i].input.SetValue(m.storedUsername)
					}
				}
			}
			return m, nil
		case "tab", "down":
			m.blurFocused()
			return m, m.focusField((m.focused + 1) % len(m.fields))
		case "shift+tab", "up":
			m.blurFocused()
			return m, m.focusField((m.focused - 1 + len(m.fields)) % len(m.fields))
		case "enter":
			if m.focused == len(m.fields)-1 {
				return m, m.submitCmd()
			}
			m.blurFocused()
			return m, m.focusField(m.focused + 1)
		}

		// Term selector: step through terms with h/l/←/→.
		if m.fields[m.focused].kind == fieldTerm {
			switch msg.String() {
			case "h", "left":
				m.fields[m.focused].term = models.PrevTerm(m.fields[m.focused].term)
			case "l", "right":
				if !models.CanAdvanceTerm(m.fields[m.focused].term, m.latestTerm) {
					m.termWarning = models.TermDisplayName(m.latestTerm) + " is the latest available term"
				} else {
					m.fields[m.focused].term = models.NextTerm(m.fields[m.focused].term)
				}
			}
			return m, nil
		}
	}

	// Non-key messages (e.g. the list's async filter results) go to any picker,
	// focused or not.
	if _, isKey := msg.(tea.KeyMsg); !isKey {
		var cmds []tea.Cmd
		for i := range m.fields {
			if m.fields[i].kind == fieldPicker {
				var cmd tea.Cmd
				m.fields[i].picker, cmd = m.fields[i].picker.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}
	}

	// Route all other messages to the focused text input.
	if m.fields[m.focused].kind == fieldText {
		var cmd tea.Cmd
		m.fields[m.focused].input, cmd = m.fields[m.focused].input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m searchModel) submitCmd() tea.Cmd {
	params := make(map[string]string, len(m.fields))
	for _, f := range m.fields {
		switch f.kind {
		case fieldTerm:
			params[f.key] = f.term
		case fieldPicker:
			// reg-sched.pl takes the IDs space-separated in one id6 param.
			params[f.key] = strings.Join(f.picker.selectedIDs(), " ")
		default:
			params[f.key] = f.input.Value()
		}
	}
	return func() tea.Msg {
		return searchSubmittedMsg{queryType: m.queryType, params: params}
	}
}

func (m searchModel) View() string {
	s := titleStyle.Render("Schedule Lookdown") + "\n"
	s += subtitleStyle.Render(queryDisplayName(m.queryType)) + "\n\n"
	for i, f := range m.fields {
		focused := i == m.focused
		label := normalStyle.Render(f.label + ": ")
		if focused {
			label = selectedStyle.Render(f.label + ": ")
		}
		if f.kind == fieldPicker {
			s += label + "\n" + f.picker.View() + "\n"
			continue
		}
		var val string
		if f.kind == fieldTerm {
			name := models.TermDisplayName(f.term)
			if focused {
				val = selectedStyle.Render("‹ " + name + " ›")
			} else {
				val = normalStyle.Render(name)
			}
		} else {
			val = f.input.View()
		}
		s += label + val + "\n"
	}
	help := "tab/↑↓ navigate • h/l/←/→ change term • enter submit • esc back"
	if m.fields[m.focused].kind == fieldPicker {
		help = "↑↓ move • space toggle • / filter • c clear • enter search • tab/shift+tab term • esc back"
		if m.fields[m.focused].picker.typing() {
			help = "type to filter • enter toggle top match • ↑↓ apply filter • esc cancel filter"
		}
	}
	if m.storedUsername != "" && m.queryType == "schedule_lookup" {
		help += " • ^ fill my username"
	}
	s += "\n" + helpStyle.Render(help)
	if m.termWarning != "" {
		// BEL rings the terminal bell once on this render; the styled line is the
		// guaranteed visual signal.
		s = "\a" + s + "\n" + errorStyle.Render(m.termWarning)
	}
	return s
}

func queryDisplayName(key string) string {
	switch key {
	case "course_search":
		return "Course Search"
	case "schedule_lookup":
		return "Schedule Lookup"
	case "roster_view":
		return "Roster View"
	case "adhoc_group":
		return "Ad-Hoc Group Schedule"
	}
	return key
}

var _ tea.Model = searchModel{}
