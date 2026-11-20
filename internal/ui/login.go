package ui

import (
	"context"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/luddenig/schedule-lookdown/internal/auth"
)

type loginModel struct {
	spinner spinner.Model
	status  string
	err     error
	// persist mirrors config.PersistChromeProfile: reuse the saved Chrome
	// profile and try a silent restore before prompting for credentials.
	persist bool
}

func newLoginModel(persist bool) loginModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = selectedStyle
	return loginModel{spinner: s, persist: persist}
}

func (m loginModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, doAuthCmd(m.persist))
}

func (m loginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case authStatusMsg:
		m.status = msg.status
		return m, msg.next // re-arm the status listener; nil when channel closes
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
	body := m.status
	if body == "" {
		body = "Starting authentication..."
	}
	return titleStyle.Render("Schedule Lookdown") + "\n\n" +
		m.spinner.View() + " " + body + "\n" +
		helpStyle.Render("\nctrl+c to cancel")
}

// authStatusMsg carries a progress update from a headless auth goroutine.
// next is the command to re-arm the listener; nil when the channel is closed.
type authStatusMsg struct {
	status string
	next   tea.Cmd
}

// passwordNeededMsg is sent when headless auth needs a password that isn't
// stored yet. App.Update handles it by switching to ScreenPassword.
type passwordNeededMsg struct{ username string }

// usernameNeededMsg is sent when headless auth needs a username that isn't
// stored yet. App.Update handles it by switching to ScreenUsername.
type usernameNeededMsg struct{}

// silentAuthFailedMsg is sent when the silent restore could not sign in (or
// could not run); App.Update falls through to the credential flow.
type silentAuthFailedMsg struct{}

func doAuthCmd(persist bool) tea.Cmd {
	if auth.IsWSL2() && persist && auth.HasChromeProfile() {
		return doSilentAuthCmd()
	}
	return doCredentialAuthCmd(persist)
}

// doCredentialAuthCmd is the regular auth flow: stored username/password on
// WSL2, a visible browser elsewhere.
func doCredentialAuthCmd(persist bool) tea.Cmd {
	if auth.IsWSL2() {
		username, err := auth.RetrieveUsername()
		if err != nil || username == "" {
			return func() tea.Msg { return usernameNeededMsg{} }
		}
		password, err := auth.RetrievePassword()
		if err != nil {
			// Password not stored (or keyring unavailable) — ask for it.
			return func() tea.Msg { return passwordNeededMsg{username: username} }
		}
		return doHeadlessAuthCmd(username, password, persist)
	}

	// Non-WSL2: open a visible browser window.
	return func() tea.Msg {
		cookies, err := auth.Authenticate(context.Background())
		if err != nil {
			return authFailedMsg{err}
		}
		session := auth.NewSession(cookies)
		_ = auth.SaveSession(session)
		return authSuccessMsg{session}
	}
}

// doAuthCmdForUsername runs the auth flow with a known username, bypassing the
// keyring lookup for the username. Call this when the username was just
// collected in-session and the keyring may not have persisted it yet.
func doAuthCmdForUsername(username string, persist bool) tea.Cmd {
	if !auth.IsWSL2() {
		return func() tea.Msg {
			cookies, err := auth.Authenticate(context.Background())
			if err != nil {
				return authFailedMsg{err}
			}
			session := auth.NewSession(cookies)
			_ = auth.SaveSession(session)
			return authSuccessMsg{session}
		}
	}
	password, err := auth.RetrievePassword()
	if err != nil {
		return func() tea.Msg { return passwordNeededMsg{username: username} }
	}
	return doHeadlessAuthCmd(username, password, persist)
}

// doSilentAuthCmd restores the sign-in remembered by the persistent Chrome
// profile. It never asks for credentials: on any failure it emits
// silentAuthFailedMsg so the app falls back to the regular flow.
func doSilentAuthCmd() tea.Cmd {
	statusCh := make(chan string, 4)
	resultCh := make(chan tea.Msg, 1)

	go func() {
		defer close(statusCh)
		cookies, err := auth.AuthenticateSilent(
			context.Background(),
			func(s string) { select { case statusCh <- s: default: } },
		)
		if err != nil {
			resultCh <- silentAuthFailedMsg{}
			return
		}
		session := auth.NewSession(cookies)
		_ = auth.SaveSession(session)
		resultCh <- authSuccessMsg{session}
	}()

	var readNext tea.Cmd
	readNext = func() tea.Msg {
		status, ok := <-statusCh
		if !ok {
			return nil
		}
		return authStatusMsg{status: status, next: readNext}
	}
	return tea.Batch(readNext, func() tea.Msg { return <-resultCh })
}

// doHeadlessAuthCmd starts headless auth for WSL2 and returns a batch of
// commands: one that forwards status updates to the TUI, one that waits for
// an MFA code request, and one that waits for the final result.
// doHeadlessAuthCmd takes the password directly rather than re-reading it from
// the keyring: on systems without an OS keyring (e.g. WSL2 with no Secret
// Service) the stored copy is unavailable, and re-fetching it would submit an
// empty password. Callers pass the value they already hold.
func doHeadlessAuthCmd(username, password string, persist bool) tea.Cmd {
	statusCh := make(chan string, 4)
	resultCh := make(chan tea.Msg, 1)
	codeNeededCh := make(chan struct{}, 1)
	codeInputCh := make(chan string, 1)

	go func() {
		defer close(statusCh)
		defer close(codeNeededCh)
		cookies, err := auth.AuthenticateHeadless(
			context.Background(), username, password, persist,
			func(s string) { select { case statusCh <- s: default: } },
			func() string {
				codeNeededCh <- struct{}{}
				return <-codeInputCh
			},
		)
		if err != nil {
			resultCh <- authFailedMsg{err}
			return
		}
		session := auth.NewSession(cookies)
		_ = auth.SaveSession(session)
		resultCh <- authSuccessMsg{session}
	}()

	// Self-carrying status reader: each authStatusMsg re-arms itself.
	var readNext tea.Cmd
	readNext = func() tea.Msg {
		status, ok := <-statusCh
		if !ok {
			return nil
		}
		return authStatusMsg{status: status, next: readNext}
	}

	// Fires once when the goroutine needs an MFA code from the user.
	waitForCode := func() tea.Msg {
		_, ok := <-codeNeededCh
		if !ok {
			return nil
		}
		return mfaCodeNeededMsg{codeCh: codeInputCh}
	}

	return tea.Batch(
		readNext,
		waitForCode,
		func() tea.Msg { return <-resultCh },
	)
}

var _ tea.Model = loginModel{}
