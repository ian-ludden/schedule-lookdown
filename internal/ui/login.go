package ui

import (
	"context"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/luddenig/schedule-lookdown/internal/auth"
)

type loginModel struct {
	spinner spinner.Model
	err     error
}

func newLoginModel() loginModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = selectedStyle
	return loginModel{spinner: s}
}

func (m loginModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, doAuthCmd())
}

func (m loginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case authFailedMsg:
		m.err = msg.err
		return m, nil
	}
	return m, nil
}

func (m loginModel) View() string {
	if m.err != nil {
		return errorStyle.Render("Authentication failed: "+m.err.Error()) +
			"\n" + helpStyle.Render("Press ctrl+c to quit")
	}
	return titleStyle.Render("Schedule Lookdown") + "\n\n" +
		m.spinner.View() + " Waiting for browser authentication...\n" +
		helpStyle.Render("\nComplete login in the browser window that opened")
}

func doAuthCmd() tea.Cmd {
	return func() tea.Msg {
		cookies, err := auth.Authenticate(context.Background())
		if err != nil {
			return authFailedMsg{err}
		}
		session := auth.NewSession(cookies)
		_ = auth.SaveSession(session) // best-effort persist
		return authSuccessMsg{session}
	}
}

var _ tea.Model = loginModel{}
