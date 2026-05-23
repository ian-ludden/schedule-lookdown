package ui

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/luddenig/schedule-lookdown/internal/query"
)

var resultsBaseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("1"))

type resultsModel struct {
	table table.Model
	err   error
}

func newResultsModel() resultsModel { return resultsModel{} }

func newResultsModelWithData(result query.Result) resultsModel {
	colWidths := map[string]int{
		"Course":               11,
		"CRN":                   5,
		"Course Title":         20,
		"Instructor":           25,
		"CrHrs":                 7,
		"Enrl":                  5,
		"Cap":                   5,
		"Term Schedule":        20,
		"Comments":             25,
		"Final Exam Schedule":  30,
		"Term Dates":           11,
	}
	const defaultColWidth = 15

	cols := make([]table.Column, len(result.Columns))
	for i, c := range result.Columns {
		var colWidth int
		var ok bool
		if colWidth, ok = colWidths[c]; !ok {
			colWidth = defaultColWidth
		}
		cols[i] = table.Column{Title: c, Width: colWidth}
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

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("1")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("249")).
		Background(lipgloss.Color("1")).
		Bold(false)
	t.SetStyles(s)

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
		resultsBaseStyle.Render(m.table.View()) +
		"\n" + helpStyle.Render("↑/↓ navigate • esc/q back")
}

var _ tea.Model = resultsModel{}
