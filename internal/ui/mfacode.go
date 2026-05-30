package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type mfaCodeModel struct {
	input textinput.Model
}

func newMFACodeModel() mfaCodeModel {
	t := textinput.New()
	t.Placeholder = "6-digit code"
	t.Width = 20
	t.CharLimit = 8
	t.Focus()
	return mfaCodeModel{input: t}
}

func (m mfaCodeModel) Init() tea.Cmd { return m.input.Focus() }

func (m mfaCodeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.input.Value() != "" {
				return m, func() tea.Msg { return mfaCodeSubmittedMsg{code: m.input.Value()} }
			}
		case "esc":
			return m, func() tea.Msg { return passwordCancelledMsg{} }
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m mfaCodeModel) View() string {
	s := titleStyle.Render("Schedule Lookdown") + "\n"
	s += subtitleStyle.Render("Two-factor verification") + "\n\n"
	s += normalStyle.Render("SMS code: ") + m.input.View() + "\n"
	s += "\n" + helpStyle.Render("enter submit • esc cancel")
	return s
}

// mfaCodeNeededMsg is sent when headless auth needs an SMS code from the user.
// codeCh receives the code once the user submits it.
type mfaCodeNeededMsg struct{ codeCh chan<- string }
type mfaCodeSubmittedMsg struct{ code string }

var _ tea.Model = mfaCodeModel{}
