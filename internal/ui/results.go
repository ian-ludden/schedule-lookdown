package ui

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/luddenig/schedule-lookdown/internal/models"
	"github.com/luddenig/schedule-lookdown/internal/query"
)

var resultsBaseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color(ROSE_RED))

// resultsOverheadLines is the number of lines the results view uses outside
// the table body: title (1) + blank (1) + outer-border top/bottom (2) +
// table header (1) + header-separator (1) + blank (1) + help (1).
const resultsOverheadLines = 8

// cellPadding is the left+right padding bubbles/table adds per cell (1 each side).
const cellPadding = 2

// outerBorderWidth is the left+right width added by resultsBaseStyle's border.
const outerBorderWidth = 2

const minColWidth = 3
const defaultColWidth = 15

var preferredWidths = map[string]int{
	"Course":              11,
	"CRN":                 5,
	"Course Title":        20,
	"Instructor":          25,
	"CrHrs":               7,
	"Enrl":                5,
	"Cap":                 5,
	"Term Schedule":       20,
	"Comments":            25,
	"Final Exam Schedule": 30,
	"Term Dates":          11,
	"USERNAME":            12,
	"NAME":                25,
	"BANNER ID":           12,
	"MAJOR":               10,
	"CLASS":               6,
	"YEAR":                6,
	"ADVISOR":             10,
	"EMAIL":               30,
}

type resultsModel struct {
	table     table.Model
	result    query.Result
	queryType string
	params    map[string]string
	err       error
	width     int
	height    int
}

func newResultsModel() resultsModel { return resultsModel{} }

func newResultsModelWithData(result query.Result, queryType string, params map[string]string, width, height int) resultsModel {
	m := resultsModel{result: result, queryType: queryType, params: params, width: width, height: height}
	m.table = buildResultsTable(result, width, height)
	return m
}

// buildResultsTable creates a table.Model sized to fit within width × height.
func buildResultsTable(result query.Result, width, height int) table.Model {
	colWidthMap := computeColWidths(result.Columns, width)

	cols := make([]table.Column, len(result.Columns))
	for i, c := range result.Columns {
		cols[i] = table.Column{Title: c, Width: colWidthMap[c]}
	}

	rows := make([]table.Row, len(result.Rows))
	for i, r := range result.Rows {
		rows[i] = table.Row(r)
	}

	tableHeight := max(1, height-resultsOverheadLines)

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(tableHeight),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(ROSE_RED)).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color(ROSE_SILVER)).
		Background(lipgloss.Color(ROSE_RED)).
		Bold(false)
	t.SetStyles(s)

	return t
}

// computeColWidths returns column widths scaled to fit termWidth.
// When termWidth is 0 (not yet known), preferred widths are returned as-is.
func computeColWidths(cols []string, termWidth int) map[string]int {
	widths := make(map[string]int, len(cols))
	totalPreferred := 0
	for _, c := range cols {
		w, ok := preferredWidths[c]
		if !ok {
			w = defaultColWidth
		}
		widths[c] = w
		totalPreferred += w
	}

	if termWidth <= 0 {
		return widths
	}

	available := termWidth - outerBorderWidth - cellPadding*len(cols)
	if available < len(cols)*minColWidth {
		available = len(cols) * minColWidth
	}

	if available >= totalPreferred {
		return widths
	}

	for k, w := range widths {
		scaled := int(float64(w) * float64(available) / float64(totalPreferred))
		if scaled < minColWidth {
			scaled = minColWidth
		}
		widths[k] = scaled
	}
	return widths
}

func (m resultsModel) Init() tea.Cmd { return nil }

func (m resultsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if len(m.result.Columns) > 0 {
			m.table = buildResultsTable(m.result, m.width, m.height)
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return m, func() tea.Msg { return backMsg{} }
		case "h":
			if term := m.params["term"]; term != "" {
				prev := models.PrevTerm(term)
				return m, func() tea.Msg { return changeTermMsg{term: prev} }
			}
		case "l":
			if term := m.params["term"]; term != "" {
				next := models.NextTerm(term)
				return m, func() tea.Msg { return changeTermMsg{term: next} }
			}
		case "enter":
			if m.queryType == "roster_view" {
				row := m.table.SelectedRow()
				if len(row) > 0 && row[0] != "" {
					username, term := row[0], m.params["term"]
					return m, func() tea.Msg {
						return searchSubmittedMsg{
							queryType: "schedule_lookup",
							params:    map[string]string{"term": term, "username": username},
						}
					}
				}
			}
		case "a":
			if m.queryType == "roster_view" {
				row := m.table.SelectedRow()
				if len(row) > 6 && row[6] != "" {
					advisor, term := row[6], m.params["term"]
					return m, func() tea.Msg {
						return searchSubmittedMsg{
							queryType: "instructor_lookup",
							params:    map[string]string{"term": term, "username": advisor},
						}
					}
				}
			}
		case "r":
			if m.queryType == "instructor_lookup" {
				row := m.table.SelectedRow()
				if len(row) > 0 && row[0] != "" {
					courseID, term := row[0], m.params["term"]
					return m, func() tea.Msg {
						return searchSubmittedMsg{
							queryType: "roster_view",
							params:    map[string]string{"term": term, "course_id": courseID},
						}
					}
				}
			} else {
				qt, params := m.queryType, m.params
				return m, func() tea.Msg {
					return refreshCurrentQueryMsg{queryType: qt, params: params}
				}
			}
		}
	case errMsg:
		m.err = msg.err
		return m, nil
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func queryResultTitle(queryType string, params map[string]string) string {
	switch queryType {
	case "schedule_lookup":
		if u := params["username"]; u != "" {
			return "Schedule — " + u
		}
		return "Schedule Lookup"
	case "course_search":
		code, instr := params["course_code"], params["instructor"]
		switch {
		case code != "" && instr != "":
			return "Course Search — " + code + " / " + instr
		case code != "":
			return "Course Search — " + code
		case instr != "":
			return "Course Search — " + instr
		}
		return "Course Search"
	case "roster_view":
		if id := params["course_id"]; id != "" {
			return "Roster — " + id
		}
		return "Roster View"
	case "instructor_lookup":
		if u := params["username"]; u != "" {
			return "Instructor — " + u
		}
		return "Instructor Lookup"
	}
	return "Results"
}

func (m resultsModel) View() string {
	if m.err != nil {
		return errorStyle.Render("Error: "+m.err.Error()) +
			"\n" + helpStyle.Render("Press esc to go back")
	}
	title := queryResultTitle(m.queryType, m.params)
	if term := m.params["term"]; term != "" {
		title += " — " + models.TermDisplayName(term)
	}
	help := "↑/↓ navigate • h/l prev/next term • r refresh • esc/q back"
	switch m.queryType {
	case "roster_view":
		help += " • enter: view schedule • a: view advisor schedule"
	case "instructor_lookup":
		help += " • r: view roster"
	}
	return titleStyle.Render(title) + "\n" +
		resultsBaseStyle.Render(m.table.View()) +
		"\n" + helpStyle.Render(help)
}

var _ tea.Model = resultsModel{}
