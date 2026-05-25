package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type usernameModel struct {
	input textinput.Model
}

func newUsernameModel() usernameModel {
	t := textinput.New()
	t.Placeholder = "RHIT username (e.g. luddenig)"
	t.Width = 40
	t.CharLimit = 64
	t.Focus()
	return usernameModel{input: t}
}

func (m usernameModel) Init() tea.Cmd {
	return m.input.Focus()
}

func (m usernameModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.input.Value() != "" {
				return m, func() tea.Msg { return usernameSubmittedMsg{username: m.input.Value()} }
			}
			return m, func() tea.Msg { return usernameSkippedMsg{} }
		case "esc":
			return m, func() tea.Msg { return usernameSkippedMsg{} }
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m usernameModel) View() string {
	s := titleStyle.Render("Schedule Lookdown") + "\n"
	s += subtitleStyle.Render("Save your RHIT username") + "\n\n"
	s += normalStyle.Render("Username: ") + m.input.View() + "\n"
	s += "\n" + helpStyle.Render("enter save • esc skip")
	return s
}

type usernameSubmittedMsg struct{ username string }
type usernameSkippedMsg struct{}

var _ tea.Model = usernameModel{}
