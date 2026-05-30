package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/luddenig/schedule-lookdown/internal/models"
)

type fieldKind int

const (
	fieldText fieldKind = iota
	fieldTerm
)

type field struct {
	label string
	key   string
	kind  fieldKind
	input textinput.Model // used when kind == fieldText
	term  string          // YYYYTT code, used when kind == fieldTerm
}

type searchModel struct {
	queryType      string
	fields         []field
	focused        int
	storedUsername string
}

func newSearchModel() searchModel { return searchModel{} }

func newSearchModelForQuery(queryType string, storedUsername string, defaultTerm string) searchModel {
	return searchModel{
		queryType:      queryType,
		fields:         fieldsForQuery(queryType, defaultTerm),
		storedUsername: storedUsername,
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
	if m.fields[m.focused].kind == fieldText {
		m.fields[m.focused].input.Blur()
	}
}

// focusField sets focus to field i and returns a focus cmd if it is a text field.
func (m *searchModel) focusField(i int) tea.Cmd {
	m.focused = i
	if m.fields[i].kind == fieldText {
		return m.fields[i].input.Focus()
	}
	return nil
}

func (m searchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if len(m.fields) == 0 {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
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
				m.fields[m.focused].term = models.NextTerm(m.fields[m.focused].term)
			}
			return m, nil
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
		if f.kind == fieldTerm {
			params[f.key] = f.term
		} else {
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
	if m.storedUsername != "" && m.queryType == "schedule_lookup" {
		help += " • ^ fill my username"
	}
	s += "\n" + helpStyle.Render(help)
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
	}
	return key
}

var _ tea.Model = searchModel{}
