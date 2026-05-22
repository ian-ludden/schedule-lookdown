package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type field struct {
	label string
	key   string
	input textinput.Model
}

type searchModel struct {
	queryType string
	fields    []field
	focused   int
}

func newSearchModel() searchModel { return searchModel{} }

func newSearchModelForQuery(queryType string) searchModel {
	return searchModel{
		queryType: queryType,
		fields:    fieldsForQuery(queryType),
	}
}

func fieldsForQuery(queryType string) []field {
	newInput := func(placeholder string) textinput.Model {
		t := textinput.New()
		t.Placeholder = placeholder
		t.Width = 40
		return t
	}

	switch queryType {
	case "course_search":
		return []field{
			{label: "Term", key: "term", input: newInput("202510")},
			{label: "Course Code", key: "course_code", input: newInput("CSSE 132")},
			{label: "Instructor", key: "instructor", input: newInput("Last name")},
		}
	case "schedule_lookup":
		return []field{
			{label: "Term", key: "term", input: newInput("202510")},
			{label: "Username", key: "username", input: newInput("RHIT username")},
		}
	case "section_availability":
		return []field{
			{label: "Term", key: "term", input: newInput("202510")},
			{label: "CRN", key: "crn", input: newInput("12345")},
		}
	}
	return nil
}

func (m searchModel) Init() tea.Cmd {
	if len(m.fields) == 0 {
		return nil
	}
	return m.fields[0].input.Focus()
}

func (m searchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if len(m.fields) == 0 {
		return m, nil
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return backMsg{} }
		case "tab", "down":
			m.fields[m.focused].input.Blur()
			m.focused = (m.focused + 1) % len(m.fields)
			cmds = append(cmds, m.fields[m.focused].input.Focus())
			return m, tea.Batch(cmds...)
		case "shift+tab", "up":
			m.fields[m.focused].input.Blur()
			m.focused = (m.focused - 1 + len(m.fields)) % len(m.fields)
			cmds = append(cmds, m.fields[m.focused].input.Focus())
			return m, tea.Batch(cmds...)
		case "enter":
			if m.focused == len(m.fields)-1 {
				return m, m.submitCmd()
			}
			m.fields[m.focused].input.Blur()
			m.focused++
			cmds = append(cmds, m.fields[m.focused].input.Focus())
			return m, tea.Batch(cmds...)
		}
	}

	// Route all other messages to the focused input.
	var cmd tea.Cmd
	m.fields[m.focused].input, cmd = m.fields[m.focused].input.Update(msg)
	return m, cmd
}

func (m searchModel) submitCmd() tea.Cmd {
	params := make(map[string]string, len(m.fields))
	for _, f := range m.fields {
		params[f.key] = f.input.Value()
	}
	return func() tea.Msg {
		return searchSubmittedMsg{queryType: m.queryType, params: params}
	}
}

func (m searchModel) View() string {
	s := titleStyle.Render("Schedule Lookdown") + "\n"
	s += subtitleStyle.Render(queryDisplayName(m.queryType)) + "\n\n"
	for i, f := range m.fields {
		label := normalStyle.Render(f.label + ": ")
		if i == m.focused {
			label = selectedStyle.Render(f.label + ": ")
		}
		s += label + f.input.View() + "\n"
	}
	s += "\n" + helpStyle.Render("tab/↑↓ navigate • enter submit • esc back")
	return s
}

func queryDisplayName(key string) string {
	switch key {
	case "course_search":
		return "Course Search"
	case "schedule_lookup":
		return "Schedule Lookup"
	case "section_availability":
		return "Section Availability"
	}
	return key
}

var _ tea.Model = searchModel{}
