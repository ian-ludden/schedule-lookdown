package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/luddenig/schedule-lookdown/internal/auth"
	"github.com/luddenig/schedule-lookdown/internal/client"
	"github.com/luddenig/schedule-lookdown/internal/config"
	"github.com/luddenig/schedule-lookdown/internal/models"
	"github.com/luddenig/schedule-lookdown/internal/query"
)

// sampleUsername seeds the stored username in fixture mode so the Schedule
// Lookup form prefills without a keyring entry.
const sampleUsername = "quinna"

type Screen int

const (
	ScreenLogin Screen = iota
	ScreenUsername
	ScreenPassword
	ScreenMFACode
	ScreenMenu
	ScreenSearch
	ScreenResults
)

type App struct {
	screen         Screen
	session        *auth.Session
	config         config.Config
	latestTerm     string // furthest-future term from reg-sched.pl; "" until fetched
	storedUsername string
	fixtures       map[string]string // query type → sample file path; nil means use real HTTP
	width          int
	height         int
	login          loginModel
	usernamePrompt usernameModel
	passwordPrompt passwordModel
	mfaCode        mfaCodeModel
	mfaCodeCh      chan<- string // non-nil while waiting for user to enter SMS code
	menu           menuModel
	search         searchModel
	results        resultsModel
	history        queryHistory
	historyPanel   historyPanelModel
}

// mainWidth returns the width available for the main content area.
// On the login screen (no panel) this is the full terminal width.
func (a App) mainWidth() int {
	if a.width <= 0 {
		return 0
	}
	if a.screen == ScreenLogin || a.screen == ScreenPassword || a.screen == ScreenUsername || a.screen == ScreenMFACode {
		return a.width
	}
	w := a.width - panelTotalWidth
	if w < 20 {
		return 20
	}
	return w
}

func NewApp(session *auth.Session, cfg config.Config, initial Screen, fixtures map[string]string) App {
	app := App{
		screen:         initial,
		session:        session,
		config:         cfg,
		fixtures:       fixtures,
		login:          newLoginModel(),
		passwordPrompt: newPasswordModel(),
		menu:           newMenuModel(),
		search:         newSearchModel(),
		results:        newResultsModel(),
		historyPanel:   newHistoryPanelModel(),
	}

	// Load persisted history.
	if data, err := auth.RetrieveHistory(); err == nil {
		_ = app.history.unmarshal(data)
	}
	app.historyPanel.entries = app.history.sorted()

	// In fixture mode there is no auth at all, so never prompt for a username;
	// seed a fake one so the Schedule Lookup form prefills.
	if app.fixtureMode() {
		app.storedUsername = sampleUsername
	} else if initial == ScreenMenu {
		// When bypassing auth with a valid cached session, authSuccessMsg never
		// fires, so check for a stored username here instead.
		stored, err := auth.RetrieveUsername()
		if err != nil {
			app.usernamePrompt = newUsernameModel()
			app.screen = ScreenUsername
		} else {
			app.storedUsername = stored
		}
	}
	return app
}

// fixtureMode reports whether the app serves queries from local sample files
// instead of authenticating and hitting the network.
func (a App) fixtureMode() bool { return a.fixtures != nil }

// resolvedDefaultTerm returns the term code the search form should pre-select,
// honouring the user's default_term setting. When set to "latest" it uses the
// fetched latest term, falling back to the computed current term if the fetch
// hasn't completed (or failed).
func (a App) resolvedDefaultTerm() string {
	if a.config.DefaultTerm == config.DefaultTermLatest && a.latestTerm != "" {
		return a.latestTerm
	}
	return models.CurrentTerm(time.Now())
}

// maybeFetchTermsCmd returns a command that fetches the latest available term
// when the user defaults to "latest" and a live session is available, else nil.
func (a App) maybeFetchTermsCmd() tea.Cmd {
	if a.config.DefaultTerm != config.DefaultTermLatest || a.fixtureMode() || a.session == nil {
		return nil
	}
	return fetchDefaultTermCmd(a.session)
}

func (a App) Init() tea.Cmd {
	switch a.screen {
	case ScreenLogin:
		return a.login.Init()
	case ScreenUsername:
		return a.usernamePrompt.Init()
	case ScreenPassword:
		return a.passwordPrompt.Init()
	case ScreenMFACode:
		return a.mfaCode.Init()
	default:
		return tea.Batch(a.menu.Init(), a.maybeFetchTermsCmd())
	}
}

func (a *App) applyWindowSizeToMenu() {
	mainW := a.mainWidth()
	if mainW > 0 && a.height > 0 {
		a.menu.list.SetSize(mainW, a.height-4)
	}
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle global key events before screen delegation.
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "ctrl+c" {
			return a, tea.Quit
		}
		if keyMsg.String() == "ctrl+h" && a.screen != ScreenLogin {
			a.historyPanel.focused = !a.historyPanel.focused
			return a, nil
		}
		// Route key events to the history panel when it is focused.
		if a.historyPanel.focused {
			m, cmd := a.historyPanel.Update(keyMsg)
			a.historyPanel = m.(historyPanelModel)
			return a, cmd
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.historyPanel.height = msg.Height
		// Resize sub-models to the reduced main width; search and username use
		// fixed-width text inputs and don't need explicit resizing.
		mainW := a.mainWidth()
		if mainW > 0 {
			a.menu.list.SetSize(mainW, a.height-4)
		}
		if len(a.results.result.Columns) > 0 {
			a.results = newResultsModelWithData(a.results.result, a.results.queryType, a.results.params, mainW, a.height)
		} else {
			a.results.width = mainW
			a.results.height = a.height
		}
		return a, nil

	case authSuccessMsg:
		a.session = msg.session
		if a.storedUsername == "" {
			stored, err := auth.RetrieveUsername()
			if err != nil {
				a.usernamePrompt = newUsernameModel()
				a.screen = ScreenUsername
				return a, a.usernamePrompt.Init()
			}
			a.storedUsername = stored
		}
		a.screen = ScreenMenu
		a.applyWindowSizeToMenu()
		return a, tea.Batch(a.menu.Init(), a.maybeFetchTermsCmd())

	case termsLoadedMsg:
		a.latestTerm = msg.latest
		return a, nil

	case authFailedMsg:
		// A wrong password stored in the keyring would otherwise leave headless
		// auth failing on every launch with no way to re-enter it (the password
		// screen only shows when nothing is stored). On a credential rejection,
		// clear the stored password and re-prompt so the user can recover.
		if auth.IsWSL2() && isCredentialError(msg.err) {
			_ = auth.DeletePassword()
			a.passwordPrompt = newPasswordModel()
			a.passwordPrompt.err = msg.err
			a.screen = ScreenPassword
			return a, a.passwordPrompt.Init()
		}
		// Otherwise surface the error on the login screen (existing behaviour).
		m, cmd := a.login.Update(msg)
		a.login = m.(loginModel)
		return a, cmd

	case usernameNeededMsg:
		a.usernamePrompt = newUsernameModel()
		a.screen = ScreenUsername
		return a, a.usernamePrompt.Init()

	case passwordNeededMsg:
		a.passwordPrompt = newPasswordModel()
		a.screen = ScreenPassword
		return a, a.passwordPrompt.Init()

	case passwordSubmittedMsg:
		_ = auth.StorePassword(msg.password) // best-effort persistence; may fail without a keyring
		a.login = newLoginModel()
		a.screen = ScreenLogin
		// Drive auth with the password we just collected rather than re-reading
		// it from the keyring, which may be unavailable (e.g. WSL2 with no Secret
		// Service) and would otherwise submit an empty password.
		username := a.storedUsername
		if username == "" {
			if u, err := auth.RetrieveUsername(); err == nil {
				username = u
			}
		}
		if username != "" {
			return a, tea.Batch(a.login.spinner.Tick, doHeadlessAuthCmd(username, msg.password))
		}
		return a, a.login.Init()

	case passwordCancelledMsg:
		return a, tea.Quit

	case mfaCodeNeededMsg:
		a.mfaCodeCh = msg.codeCh
		a.mfaCode = newMFACodeModel()
		a.screen = ScreenMFACode
		return a, a.mfaCode.Init()

	case mfaCodeSubmittedMsg:
		if a.mfaCodeCh != nil {
			a.mfaCodeCh <- msg.code
			a.mfaCodeCh = nil
		}
		a.screen = ScreenLogin
		return a, nil

	case usernameSubmittedMsg:
		a.storedUsername = msg.username
		_ = auth.StoreUsername(msg.username)
		if a.session == nil && !a.fixtureMode() {
			// Bypass keyring round-trip; drive auth with the username we just got.
			a.login = newLoginModel()
			a.screen = ScreenLogin
			return a, tea.Batch(a.login.spinner.Tick, doAuthCmdForUsername(msg.username))
		}
		a.screen = ScreenMenu
		a.applyWindowSizeToMenu()
		return a, tea.Batch(a.menu.Init(), a.maybeFetchTermsCmd())

	case usernameSkippedMsg:
		if a.session == nil && !a.fixtureMode() {
			// Can't proceed without a username on WSL2 headless auth.
			return a, tea.Quit
		}
		a.screen = ScreenMenu
		a.applyWindowSizeToMenu()
		return a, a.menu.Init()

	case querySelectedMsg:
		a.search = newSearchModelForQuery(msg.queryType, a.storedUsername, a.resolvedDefaultTerm())
		a.screen = ScreenSearch
		return a, a.search.Init()

	case searchSubmittedMsg:
		a.results = newResultsModel()
		a.screen = ScreenResults
		return a, tea.Batch(a.results.Init(), executeQueryCmd(a.session, msg.queryType, msg.params, a.fixtures, a.config.JumpToRosterOnSingleResult))

	case advisorSearchMsg:
		a.results = newResultsModel()
		a.screen = ScreenResults
		return a, tea.Batch(a.results.Init(), advisorSearchCmd(a.session, msg.advisorName, msg.term, a.fixtures))

	case queryResultMsg:
		a.results = newResultsModelWithData(msg.result, msg.queryType, msg.params, a.mainWidth(), a.height)
		a.screen = ScreenResults
		a.history.add(HistoryEntry{
			QueryType: msg.queryType,
			Params:    msg.params,
			Result:    msg.result,
			FetchedAt: time.Now(),
		})
		a.historyPanel.entries = a.history.sorted()
		if data, err := a.history.marshal(); err == nil {
			_ = auth.StoreHistory(data)
		}
		return a, nil

	case historyEntrySelectedMsg:
		a.results = newResultsModelWithData(msg.result, msg.queryType, msg.params, a.mainWidth(), a.height)
		a.screen = ScreenResults
		a.historyPanel.focused = false
		return a, nil

	case historyClearedMsg:
		a.history.clear()
		a.historyPanel.entries = nil
		a.historyPanel.cursor = 0
		if data, err := a.history.marshal(); err == nil {
			_ = auth.StoreHistory(data)
		}
		return a, nil

	case refreshCurrentQueryMsg:
		a.results = newResultsModel()
		a.results.width = a.mainWidth()
		a.results.height = a.height
		return a, tea.Batch(a.results.Init(), executeQueryCmd(a.session, msg.queryType, msg.params, a.fixtures, false))

	case changeTermMsg:
		queryType := a.results.queryType
		newParams := make(map[string]string, len(a.results.params))
		for k, v := range a.results.params {
			newParams[k] = v
		}
		newParams["term"] = msg.term
		a.results = newResultsModel()
		a.results.width = a.mainWidth()
		a.results.height = a.height
		a.screen = ScreenResults
		return a, tea.Batch(a.results.Init(), executeQueryCmd(a.session, queryType, newParams, a.fixtures, false))

	case backMsg:
		a.screen = ScreenMenu
		a.applyWindowSizeToMenu()
		return a, a.menu.Init()
	}

	return a.delegateToScreen(msg)
}

func (a App) delegateToScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch a.screen {
	case ScreenLogin:
		m, cmd := a.login.Update(msg)
		a.login = m.(loginModel)
		return a, cmd
	case ScreenMenu:
		m, cmd := a.menu.Update(msg)
		a.menu = m.(menuModel)
		return a, cmd
	case ScreenSearch:
		m, cmd := a.search.Update(msg)
		a.search = m.(searchModel)
		return a, cmd
	case ScreenUsername:
		m, cmd := a.usernamePrompt.Update(msg)
		a.usernamePrompt = m.(usernameModel)
		return a, cmd
	case ScreenPassword:
		m, cmd := a.passwordPrompt.Update(msg)
		a.passwordPrompt = m.(passwordModel)
		return a, cmd
	case ScreenMFACode:
		m, cmd := a.mfaCode.Update(msg)
		a.mfaCode = m.(mfaCodeModel)
		return a, cmd
	case ScreenResults:
		m, cmd := a.results.Update(msg)
		a.results = m.(resultsModel)
		return a, cmd
	}
	return a, nil
}

func (a App) View() string {
	var mainContent string
	switch a.screen {
	case ScreenLogin:
		mainContent = a.login.View()
	case ScreenMenu:
		mainContent = a.menu.View()
	case ScreenSearch:
		mainContent = a.search.View()
	case ScreenUsername:
		mainContent = a.usernamePrompt.View()
	case ScreenPassword:
		mainContent = a.passwordPrompt.View()
	case ScreenMFACode:
		mainContent = a.mfaCode.View()
	case ScreenResults:
		mainContent = a.results.View()
	}

	if a.width <= 0 || a.height <= 0 {
		return mainContent
	}

	// Auth screens have no history panel.
	if a.screen == ScreenLogin || a.screen == ScreenPassword || a.screen == ScreenUsername || a.screen == ScreenMFACode {
		return lipgloss.Place(a.width, a.height, lipgloss.Left, lipgloss.Top, mainContent)
	}

	placed := lipgloss.Place(a.mainWidth(), a.height, lipgloss.Left, lipgloss.Top, mainContent)
	return lipgloss.JoinHorizontal(lipgloss.Top, placed, a.historyPanel.View())
}

func executeQueryCmd(session *auth.Session, queryType string, params map[string]string, fixtures map[string]string, jumpToRoster bool) tea.Cmd {
	return func() tea.Msg {
		var c *client.Client
		var err error
		if fixtures != nil {
			key := queryType + ":" + params["term"]
			path, ok := fixtures[key]
			if !ok {
				path, ok = fixtures[queryType]
			}
			if !ok {
				return errMsg{fmt.Errorf("no sample fixture available for query type: %s", queryType)}
			}
			c, err = client.NewFixture(path)
		} else {
			if session == nil {
				return errMsg{fmt.Errorf("no active session")}
			}
			c, err = client.New(session.Cookies)
		}
		if err != nil {
			return errMsg{err}
		}

		var q query.Query
		switch queryType {
		case "course_search":
			q = &query.CourseSearch{
				Term:       params["term"],
				CourseCode: params["course_code"],
				Instructor: params["instructor"],
			}
		case "schedule_lookup":
			q = &query.ScheduleLookup{
				Term:     params["term"],
				Username: params["username"],
			}
		case "roster_view":
			q = &query.RosterView{
				Term:     params["term"],
				CourseID: params["course_id"],
			}
		case "instructor_lookup":
			q = &query.InstructorLookup{
				Term:     params["term"],
				Username: params["username"],
			}
		case "person_search":
			q = &query.PersonSearch{
				Term:     params["term"],
				LastName: params["last_name"],
			}
		default:
			return errMsg{fmt.Errorf("unknown query type: %s", queryType)}
		}

		result, err := q.Execute(context.Background(), c)
		if err != nil {
			return errMsg{err}
		}
		// When enabled, a single course-search hit jumps straight to that
		// course's roster instead of showing a one-row table. row[0] is the
		// course id, matching the 'r' key course→roster navigation in results.go.
		if jumpToRoster && queryType == "course_search" && len(result.Rows) == 1 {
			row := result.Rows[0]
			if len(row) > 0 && row[0] != "" {
				return searchSubmittedMsg{
					queryType: "roster_view",
					params:    map[string]string{"term": params["term"], "course_id": row[0]},
				}
			}
		}
		return queryResultMsg{result: result, queryType: queryType, params: params}
	}
}

// fetchDefaultTermCmd fetches reg-sched.pl's available terms and reports the
// furthest-future one via termsLoadedMsg. Any failure yields an empty latest so
// the app silently falls back to the computed current term.
func fetchDefaultTermCmd(session *auth.Session) tea.Cmd {
	return func() tea.Msg {
		if session == nil {
			return termsLoadedMsg{}
		}
		c, err := client.New(session.Cookies)
		if err != nil {
			return termsLoadedMsg{}
		}
		codes, err := query.FetchAvailableTerms(context.Background(), c)
		if err != nil {
			return termsLoadedMsg{}
		}
		return termsLoadedMsg{latest: models.LatestTerm(codes)}
	}
}

// Message types for inter-screen transitions.
type authSuccessMsg struct{ session *auth.Session }
type termsLoadedMsg struct{ latest string }
type authFailedMsg struct{ err error }
type querySelectedMsg struct{ queryType string }
type searchSubmittedMsg struct {
	queryType string
	params    map[string]string
}
type queryResultMsg struct {
	result    query.Result
	queryType string
	params    map[string]string
}
type changeTermMsg struct{ term string }
type backMsg struct{}
type errMsg struct{ err error }
type historyEntrySelectedMsg struct {
	queryType string
	params    map[string]string
	result    query.Result
}
type historyClearedMsg struct{}
type refreshCurrentQueryMsg struct {
	queryType string
	params    map[string]string
}
type advisorSearchMsg struct{ advisorName, term string }

// advisorSearchCmd looks up the advisor's username via a person search by last
// name, filters by first name, and either auto-navigates (1 match) or returns
// a disambiguation table (2+ matches).
func advisorSearchCmd(session *auth.Session, advisorName, term string, fixtures map[string]string) tea.Cmd {
	return func() tea.Msg {
		parts := strings.Fields(advisorName)
		if len(parts) == 0 {
			return errMsg{fmt.Errorf("advisor name is empty")}
		}
		lastName := parts[len(parts)-1]
		firstName := strings.Join(parts[:len(parts)-1], " ")

		var c *client.Client
		var err error
		if fixtures != nil {
			key := "person_search"
			path, ok := fixtures[key]
			if !ok {
				return errMsg{fmt.Errorf("no fixture available for person_search")}
			}
			c, err = client.NewFixture(path)
		} else {
			if session == nil {
				return errMsg{fmt.Errorf("no active session")}
			}
			c, err = client.New(session.Cookies)
		}
		if err != nil {
			return errMsg{err}
		}

		q := &query.PersonSearch{Term: term, LastName: strings.ToLower(lastName)}
		result, err := q.Execute(context.Background(), c)
		if err != nil {
			return errMsg{err}
		}

		// Filter by first name match (NAME is column 2).
		var matches [][]string
		for _, row := range result.Rows {
			if len(row) < 3 {
				continue
			}
			if firstName == "" || strings.Contains(strings.ToLower(row[2]), strings.ToLower(firstName)) {
				matches = append(matches, row)
			}
		}

		switch len(matches) {
		case 0:
			return errMsg{fmt.Errorf("no person named %q found", advisorName)}
		case 1:
			return searchSubmittedMsg{
				queryType: "instructor_lookup",
				params:    map[string]string{"term": term, "username": matches[0][0]},
			}
		default:
			return queryResultMsg{
				result:    query.Result{Columns: result.Columns, Rows: matches},
				queryType: "person_search",
				params:    map[string]string{"term": term, "last_name": lastName},
			}
		}
	}
}

// isCredentialError reports whether err is Microsoft rejecting the account or
// password, as opposed to a transient/flow error, so we know to re-prompt for
// the password rather than just displaying the failure.
func isCredentialError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "incorrect") ||
		strings.Contains(msg, "account or password") ||
		strings.Contains(msg, "isn't correct") ||
		strings.Contains(msg, "is not correct")
}

var _ tea.Model = App{}
