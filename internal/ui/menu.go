package ui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type queryItem struct {
	name string
	key  string
}

func (i queryItem) Title() string       { return i.name }
func (i queryItem) Description() string { return "" }
func (i queryItem) FilterValue() string { return i.name }

var queryTypes = []list.Item{
	queryItem{name: "Course Search", key: "course_search"},
	queryItem{name: "Schedule Lookup", key: "schedule_lookup"},
	queryItem{name: "Roster View", key: "roster_view"},
}

type menuModel struct {
	list list.Model
}

func newMenuModel() menuModel {
	delegate := list.NewDefaultDelegate()
	l := list.New(queryTypes, delegate, 0, 0)
	l.Title = "Schedule Lookdown"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	return menuModel{list: l}
}

func (m menuModel) Init() tea.Cmd { return nil }

func (m menuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-4)
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "enter" {
			if item, ok := m.list.SelectedItem().(queryItem); ok {
				return m, func() tea.Msg { return querySelectedMsg{queryType: item.key} }
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m menuModel) View() string {
	return m.list.View() + "\n" + helpStyle.Render("↑/↓ navigate • enter select • ctrl+c quit")
}

var _ tea.Model = menuModel{}
