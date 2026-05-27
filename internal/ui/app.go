package ui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/luddenig/schedule-lookdown/internal/auth"
	"github.com/luddenig/schedule-lookdown/internal/client"
	"github.com/luddenig/schedule-lookdown/internal/query"
)

type Screen int

const (
	ScreenLogin Screen = iota
	ScreenUsername
	ScreenMenu
	ScreenSearch
	ScreenResults
)

type App struct {
	screen         Screen
	session        *auth.Session
	storedUsername string
	fixtures       map[string]string // query type → sample file path; nil means use real HTTP
	width          int
	height         int
	login          loginModel
	usernamePrompt usernameModel
	menu           menuModel
	search         searchModel
	results        resultsModel
}

func NewApp(session *auth.Session, initial Screen, fixtures map[string]string) App {
	app := App{
		screen:   initial,
		session:  session,
		fixtures: fixtures,
		login:    newLoginModel(),
		menu:     newMenuModel(),
		search:   newSearchModel(),
		results:  newResultsModel(),
	}
	// When bypassing auth (fixture mode or valid cached session), authSuccessMsg
	// never fires, so check for a stored username here instead.
	if initial == ScreenMenu {
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

func (a App) Init() tea.Cmd {
	switch a.screen {
	case ScreenLogin:
		return a.login.Init()
	case ScreenUsername:
		return a.usernamePrompt.Init()
	default:
		return a.menu.Init()
	}
}

func (a *App) applyWindowSizeToMenu() {
	if a.width > 0 && a.height > 0 {
		a.menu.list.SetSize(a.width, a.height-4)
	}
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
	case authSuccessMsg:
		a.session = msg.session
		stored, err := auth.RetrieveUsername()
		if err != nil {
			a.usernamePrompt = newUsernameModel()
			a.screen = ScreenUsername
			return a, a.usernamePrompt.Init()
		}
		a.storedUsername = stored
		a.screen = ScreenMenu
		a.applyWindowSizeToMenu()
		return a, a.menu.Init()
	case usernameSubmittedMsg:
		a.storedUsername = msg.username
		_ = auth.StoreUsername(msg.username)
		a.screen = ScreenMenu
		a.applyWindowSizeToMenu()
		return a, a.menu.Init()
	case usernameSkippedMsg:
		a.screen = ScreenMenu
		a.applyWindowSizeToMenu()
		return a, a.menu.Init()
	case querySelectedMsg:
		a.search = newSearchModelForQuery(msg.queryType, a.storedUsername)
		a.screen = ScreenSearch
		return a, a.search.Init()
	case searchSubmittedMsg:
		a.results = newResultsModel()
		a.screen = ScreenResults
		return a, executeQueryCmd(a.session, msg.queryType, msg.params, a.fixtures)
	case queryResultMsg:
		a.results = newResultsModelWithData(msg.result, msg.queryType, msg.params, a.width, a.height)
		a.screen = ScreenResults
		return a, nil
	case changeTermMsg:
		queryType := a.results.queryType
		newParams := make(map[string]string, len(a.results.params))
		for k, v := range a.results.params {
			newParams[k] = v
		}
		newParams["term"] = msg.term
		// Show the new term in the title immediately while the query loads.
		a.results = resultsModel{queryType: queryType, params: newParams, width: a.width, height: a.height}
		a.screen = ScreenResults
		return a, executeQueryCmd(a.session, queryType, newParams, a.fixtures)
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
	case ScreenResults:
		m, cmd := a.results.Update(msg)
		a.results = m.(resultsModel)
		return a, cmd
	}
	return a, nil
}

func (a App) View() string {
	var content string
	switch a.screen {
	case ScreenLogin:
		content = a.login.View()
	case ScreenMenu:
		content = a.menu.View()
	case ScreenSearch:
		content = a.search.View()
	case ScreenUsername:
		content = a.usernamePrompt.View()
	case ScreenResults:
		content = a.results.View()
	}
	// lipgloss.Place fills the entire terminal rectangle with content, padding
	// every line to a.width and adding blank lines to reach a.height. This
	// ensures old characters from the previous screen are always overwritten.
	if a.width > 0 && a.height > 0 {
		return lipgloss.Place(a.width, a.height, lipgloss.Left, lipgloss.Top, content)
	}
	return content
}

func executeQueryCmd(session *auth.Session, queryType string, params map[string]string, fixtures map[string]string) tea.Cmd {
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
		case "section_availability":
			q = &query.SectionAvailability{
				Term: params["term"],
				CRN:  params["crn"],
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
		default:
			return errMsg{fmt.Errorf("unknown query type: %s", queryType)}
		}

		result, err := q.Execute(context.Background(), c)
		if err != nil {
			return errMsg{err}
		}
		return queryResultMsg{result: result, queryType: queryType, params: params}
	}
}

// Message types for inter-screen transitions.
type authSuccessMsg struct{ session *auth.Session }
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

var _ tea.Model = App{}
