package ui

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/luddenig/schedule-lookdown/internal/query"
)

type resultsModel struct {
	table table.Model
	err   error
}

func newResultsModel() resultsModel { return resultsModel{} }

func newResultsModelWithData(result query.Result) resultsModel {
	cols := make([]table.Column, len(result.Columns))
	for i, c := range result.Columns {
		cols[i] = table.Column{Title: c, Width: 16}
	}

	rows := make([]table.Row, len(result.Rows))
	for i, r := range result.Rows {
		rows[i] = table.Row(r)
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	return resultsModel{table: t}
}

func (m resultsModel) Init() tea.Cmd { return nil }

func (m resultsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return m, func() tea.Msg { return backMsg{} }
		}
	case errMsg:
		m.err = msg.err
		return m, nil
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m resultsModel) View() string {
	if m.err != nil {
		return errorStyle.Render("Error: "+m.err.Error()) +
			"\n" + helpStyle.Render("Press esc to go back")
	}
	return titleStyle.Render("Results") + "\n" +
		m.table.View() +
		"\n" + helpStyle.Render("↑/↓ navigate • esc/q back")
}

var _ tea.Model = resultsModel{}
