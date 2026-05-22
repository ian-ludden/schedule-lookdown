package ui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luddenig/schedule-lookdown/internal/auth"
	"github.com/luddenig/schedule-lookdown/internal/client"
	"github.com/luddenig/schedule-lookdown/internal/query"
)

type Screen int

const (
	ScreenLogin Screen = iota
	ScreenMenu
	ScreenSearch
	ScreenResults
)

type App struct {
	screen      Screen
	session     *auth.Session
	fixturePath string
	login       loginModel
	menu        menuModel
	search      searchModel
	results     resultsModel
}

func NewApp(session *auth.Session, initial Screen, fixturePath string) App {
	return App{
		screen:      initial,
		session:     session,
		fixturePath: fixturePath,
		login:       newLoginModel(),
		menu:        newMenuModel(),
		search:      newSearchModel(),
		results:     newResultsModel(),
	}
}

func (a App) Init() tea.Cmd {
	if a.screen == ScreenLogin {
		return a.login.Init()
	}
	return a.menu.Init()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
	case authSuccessMsg:
		a.session = msg.session
		a.screen = ScreenMenu
		return a, a.menu.Init()
	case querySelectedMsg:
		a.search = newSearchModelForQuery(msg.queryType)
		a.screen = ScreenSearch
		return a, a.search.Init()
	case searchSubmittedMsg:
		a.screen = ScreenResults
		return a, executeQueryCmd(a.session, msg.queryType, msg.params, a.fixturePath)
	case queryResultMsg:
		a.results = newResultsModelWithData(msg.result)
		a.screen = ScreenResults
		return a, nil
	case backMsg:
		a.screen = ScreenMenu
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
	case ScreenResults:
		m, cmd := a.results.Update(msg)
		a.results = m.(resultsModel)
		return a, cmd
	}
	return a, nil
}

func (a App) View() string {
	switch a.screen {
	case ScreenLogin:
		return a.login.View()
	case ScreenMenu:
		return a.menu.View()
	case ScreenSearch:
		return a.search.View()
	case ScreenResults:
		return a.results.View()
	}
	return ""
}

func executeQueryCmd(session *auth.Session, queryType string, params map[string]string, fixturePath string) tea.Cmd {
	return func() tea.Msg {
		var c *client.Client
		var err error
		if fixturePath != "" {
			c, err = client.NewFixture(fixturePath)
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
		default:
			return errMsg{fmt.Errorf("unknown query type: %s", queryType)}
		}

		result, err := q.Execute(context.Background(), c)
		if err != nil {
			return errMsg{err}
		}
		return queryResultMsg{result}
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
type queryResultMsg struct{ result query.Result }
type backMsg struct{}
type errMsg struct{ err error }

var _ tea.Model = App{}
