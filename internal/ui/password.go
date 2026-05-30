package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type passwordModel struct {
	input textinput.Model
}

func newPasswordModel() passwordModel {
	t := textinput.New()
	t.Placeholder = "Microsoft password"
	t.EchoMode = textinput.EchoPassword
	t.Width = 40
	t.CharLimit = 128
	t.Focus()
	return passwordModel{input: t}
}

func (m passwordModel) Init() tea.Cmd { return m.input.Focus() }

func (m passwordModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.input.Value() != "" {
				return m, func() tea.Msg { return passwordSubmittedMsg{password: m.input.Value()} }
			}
		case "esc":
			return m, func() tea.Msg { return passwordCancelledMsg{} }
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m passwordModel) View() string {
	s := titleStyle.Render("Schedule Lookdown") + "\n"
	s += subtitleStyle.Render("Microsoft authentication") + "\n\n"
	s += normalStyle.Render("Password: ") + m.input.View() + "\n"
	s += "\n" + helpStyle.Render("enter sign in • esc cancel")
	return s
}

type passwordSubmittedMsg struct{ password string }
type passwordCancelledMsg struct{}

var _ tea.Model = passwordModel{}
