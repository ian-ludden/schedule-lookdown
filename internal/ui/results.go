package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
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
const defaultPriority = 2

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
	"Final Exam Schedule": 20,
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

// columnPriority controls which columns absorb width compression first.
// 3 = HIGH (protected; only scaled as a last resort)
// 2 = MEDIUM (scaled after LOW is exhausted)
// 1 = LOW (scaled first)
var columnPriority = map[string]int{
	"Course": 3, "CRN": 3, "USERNAME": 3,
	"Course Title": 3, "Instructor": 3, "Term Schedule": 3, "NAME": 3, "MAJOR": 3,
	"CrHrs": 2, "Enrl": 2, "Cap": 2, "CLASS": 2, "YEAR": 2,
	"Comments": 1, "Final Exam Schedule": 1, "Term Dates": 1,
	"BANNER ID": 1, "ADVISOR": 1, "EMAIL": 1,
}

type resultsModel struct {
	table      table.Model
	result     query.Result
	queryType  string
	params     map[string]string
	err        error
	spinner    spinner.Model
	loading    bool
	showDetail bool
	width      int
	height     int
}

func newResultsModel() resultsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = selectedStyle
	return resultsModel{spinner: s, loading: true}
}

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

	// Partition columns by priority tier.
	var highCols, medCols, lowCols []string
	sumHigh, sumMed := 0, 0
	for _, c := range cols {
		p := columnPriority[c]
		if p == 0 {
			p = defaultPriority
		}
		switch {
		case p >= 3:
			highCols = append(highCols, c)
			sumHigh += widths[c]
		case p == 2:
			medCols = append(medCols, c)
			sumMed += widths[c]
		default:
			lowCols = append(lowCols, c)
		}
	}

	lowMin := len(lowCols) * minColWidth
	remainAfterLow := available - lowMin

	switch {
	case remainAfterLow >= sumHigh+sumMed:
		// HIGH + MEDIUM fit at preferred; distribute surplus to LOW proportionally.
		surplus := remainAfterLow - sumHigh - sumMed
		sumLow := 0
		for _, c := range lowCols {
			sumLow += widths[c]
		}
		for _, c := range lowCols {
			extra := 0
			if sumLow > 0 {
				extra = int(float64(widths[c]) * float64(surplus) / float64(sumLow))
			}
			widths[c] = minColWidth + extra
		}
		// HIGH and MEDIUM stay at preferred (already set).

	case remainAfterLow >= sumHigh:
		// HIGH fits at preferred; scale MEDIUM into remainder; LOW gets minimum.
		for _, c := range lowCols {
			widths[c] = minColWidth
		}
		medAvail := remainAfterLow - sumHigh
		for _, c := range medCols {
			scaled := int(float64(widths[c]) * float64(medAvail) / float64(sumMed))
			if scaled < minColWidth {
				scaled = minColWidth
			}
			widths[c] = scaled
		}
		// HIGH stays at preferred.

	default:
		// Even HIGH doesn't fit; scale HIGH proportionally; LOW and MEDIUM get minimum.
		for _, c := range lowCols {
			widths[c] = minColWidth
		}
		for _, c := range medCols {
			widths[c] = minColWidth
		}
		for _, c := range highCols {
			scaled := int(float64(widths[c]) * float64(remainAfterLow) / float64(sumHigh))
			if scaled < minColWidth {
				scaled = minColWidth
			}
			widths[c] = scaled
		}
	}
	return widths
}

func (m resultsModel) Init() tea.Cmd {
	if m.loading {
		return m.spinner.Tick
	}
	return nil
}

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
		if m.showDetail {
			switch msg.String() {
			case "d", "esc", "q":
				m.showDetail = false
			}
			return m, nil
		}
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
			if m.queryType == "roster_view" || m.queryType == "person_search" {
				row := m.table.SelectedRow()
				if len(row) > 0 && row[0] != "" {
					username, term := row[0], m.params["term"]
					destType := "schedule_lookup"
					if m.queryType == "person_search" {
						destType = "instructor_lookup"
					}
					return m, func() tea.Msg {
						return searchSubmittedMsg{
							queryType: destType,
							params:    map[string]string{"term": term, "username": username},
						}
					}
				}
			}
		case "a":
			switch m.queryType {
			case "roster_view":
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
			case "schedule_lookup":
				if name := m.result.Metadata["advisor_name"]; name != "" {
					term := m.params["term"]
					return m, func() tea.Msg {
						return advisorSearchMsg{advisorName: name, term: term}
					}
				}
			}
		case "r":
			if m.queryType == "instructor_lookup" || m.queryType == "schedule_lookup" || m.queryType == "course_search" {
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
			}
		case "i":
			if m.queryType == "course_search" {
				row := m.table.SelectedRow()
				if len(row) > 3 && row[3] != "" {
					term := m.params["term"]
					if start := strings.LastIndex(row[3], "("); start != -1 {
						if end := strings.LastIndex(row[3], ")"); end > start {
							username := row[3][start+1 : end]
							return m, func() tea.Msg {
								return searchSubmittedMsg{
									queryType: "instructor_lookup",
									params:    map[string]string{"term": term, "username": username},
								}
							}
						}
					}
				}
			}
		case "ctrl+r":
			qt, params := m.queryType, m.params
			return m, func() tea.Msg {
				return refreshCurrentQueryMsg{queryType: qt, params: params}
			}
		case "d":
			if len(m.result.Columns) > 0 && len(m.table.SelectedRow()) > 0 {
				m.showDetail = true
				return m, nil
			}
		}
	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
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
	case "person_search":
		if ln := params["last_name"]; ln != "" {
			return "Person Search — " + ln + "*"
		}
		return "Person Search"
	}
	return "Results"
}

func (m resultsModel) View() string {
	if m.err != nil {
		return errorStyle.Render("Error: "+m.err.Error()) +
			"\n" + helpStyle.Render("Press esc to go back")
	}
	title := queryResultTitle(m.queryType, m.params)

	if advisorID, ok := m.result.Metadata["advisor"]; ok {
		title += " - " + advisorID
	}

	if term := m.params["term"]; term != "" {
		title += " — " + models.TermDisplayName(term)
	}

	if m.loading {
		return titleStyle.Render("Loading...") + "\n\n" +
			m.spinner.View() + " Retrieving results..." + "\n\n" +
			helpStyle.Render("esc/q back")
	}

	if m.showDetail {
		return titleStyle.Render(title) + "\n" +
			resultsBaseStyle.Render(m.renderDetail()) +
			"\n" + helpStyle.Render("d/esc: close detail")
	}

	help := "↑/↓ navigate • h/l prev/next term • d: detail • ctrl+r refresh • esc/q back"
	switch m.queryType {
	case "roster_view":
		help += " • enter: view schedule • a: view advisor schedule"
	case "course_search":
		help += " • r: view roster • i: view instructor schedule"
	case "instructor_lookup":
		help += " • r: view roster"
	case "schedule_lookup":
		help += " • r: view roster"
		if m.result.Metadata["advisor_name"] != "" {
			help += " • a: view advisor schedule"
		}
	case "person_search":
		help = "↑/↓ navigate • d: detail • enter: view advisor schedule • ctrl+r refresh • esc/q back"
	}

	return titleStyle.Render(title) + "\n" +
		resultsBaseStyle.Render(m.table.View()) +
		"\n" + helpStyle.Render(help)
}

// renderDetail returns a string with all columns and their values for the
// selected row, formatted as padded label: value pairs.
func (m resultsModel) renderDetail() string {
	row := m.table.SelectedRow()
	cols := m.result.Columns

	labelWidth := 0
	for _, c := range cols {
		if len(c) > labelWidth {
			labelWidth = len(c)
		}
	}

	var sb strings.Builder
	sb.WriteString("\n")
	for i, c := range cols {
		val := ""
		if i < len(row) {
			val = row[i]
		}
		fmt.Fprintf(&sb, "  %-*s  %s\n", labelWidth, c, val)
	}
	return sb.String()
}

var _ tea.Model = resultsModel{}
