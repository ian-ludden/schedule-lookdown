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
	table       table.Model
	result      query.Result
	queryType   string
	params      map[string]string
	meta        string // styled metadata header rendered above the table
	latestTerm  string // furthest-future available term; "" means no upper bound
	termWarning string // transient message shown when forward nav is blocked
	err         error
	spinner     spinner.Model
	loading     bool
	showDetail  bool
	width       int
	height      int
}

func newResultsModel() resultsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = selectedStyle
	return resultsModel{spinner: s, loading: true}
}

func newResultsModelWithData(result query.Result, queryType string, params map[string]string, width, height int, latestTerm string) resultsModel {
	m := resultsModel{result: result, queryType: queryType, params: params, width: width, height: height, latestTerm: latestTerm}
	m.meta = renderMetadataBlock(queryType, result, params, width)
	m.table = buildResultsTable(result, width, height, metaReservedLines(m.meta))
	return m
}

// metaReservedLines is the number of vertical lines the metadata header (and its
// trailing blank line) occupies, for sizing the table beneath it.
func metaReservedLines(meta string) int {
	if meta == "" {
		return 0
	}
	return lipgloss.Height(meta)
}

// buildResultsTable creates a table.Model sized to fit within width × height,
// reserving `reserved` lines for a metadata header above the table.
func buildResultsTable(result query.Result, width, height, reserved int) table.Model {
	colWidthMap := computeColWidths(result.Columns, width)

	cols := make([]table.Column, len(result.Columns))
	for i, c := range result.Columns {
		cols[i] = table.Column{Title: c, Width: colWidthMap[c]}
	}

	rows := make([]table.Row, len(result.Rows))
	for i, r := range result.Rows {
		rows[i] = table.Row(r)
	}

	tableHeight := max(1, height-resultsOverheadLines-reserved)

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
	widths, totalPreferred := buildPreferredWidths(cols)
	if termWidth <= 0 {
		return widths
	}
	available := calcAvailable(termWidth, len(cols))
	if available >= totalPreferred {
		return widths
	}
	high, med, low, sumHigh, sumMed := partitionByPriority(cols, widths)
	lowMin := len(low) * minColWidth
	remainAfterLow := available - lowMin
	switch {
	case remainAfterLow >= sumHigh+sumMed:
		// HIGH + MEDIUM fit at preferred; distribute surplus to LOW proportionally.
		sumLow := 0
		for _, c := range low {
			sumLow += widths[c]
		}
		distributeSurplus(low, widths, remainAfterLow-sumHigh-sumMed, sumLow)
	case remainAfterLow >= sumHigh:
		// HIGH fits at preferred; scale MEDIUM into remainder; LOW gets minimum.
		setToMin(low, widths)
		scaleProportionally(med, widths, remainAfterLow-sumHigh, sumMed)
	default:
		// Even HIGH doesn't fit; scale HIGH proportionally; LOW and MEDIUM get minimum.
		setToMin(low, widths)
		setToMin(med, widths)
		scaleProportionally(high, widths, remainAfterLow, sumHigh)
	}
	return widths
}

// buildPreferredWidths returns a widths map initialised from preferredWidths (or
// defaultColWidth for unknown columns) and the sum of those preferred widths.
func buildPreferredWidths(cols []string) (map[string]int, int) {
	widths := make(map[string]int, len(cols))
	total := 0
	for _, c := range cols {
		w, ok := preferredWidths[c]
		if !ok {
			w = defaultColWidth
		}
		widths[c] = w
		total += w
	}
	return widths, total
}

// calcAvailable returns the usable pixel width after subtracting borders and
// per-cell padding, floored so every column can fit at minColWidth.
func calcAvailable(termWidth, numCols int) int {
	available := termWidth - outerBorderWidth - cellPadding*numCols
	if available < numCols*minColWidth {
		available = numCols * minColWidth
	}
	return available
}

// partitionByPriority splits cols into high-, medium-, and low-priority slices
// using columnPriority (defaultPriority for unknown columns) and returns the
// sum of preferred widths for the high and medium groups.
func partitionByPriority(cols []string, widths map[string]int) (high, med, low []string, sumHigh, sumMed int) {
	for _, c := range cols {
		p := columnPriority[c]
		if p == 0 {
			p = defaultPriority
		}
		switch {
		case p >= 3:
			high = append(high, c)
			sumHigh += widths[c]
		case p == 2:
			med = append(med, c)
			sumMed += widths[c]
		default:
			low = append(low, c)
		}
	}
	return
}

// setToMin sets every column in cols to minColWidth in widths.
func setToMin(cols []string, widths map[string]int) {
	for _, c := range cols {
		widths[c] = minColWidth
	}
}

// scaleProportionally scales each column in cols to its proportional share of
// budget (relative to sum), with a minColWidth floor. It mutates widths in place.
func scaleProportionally(cols []string, widths map[string]int, budget, sum int) {
	for _, c := range cols {
		scaled := minColWidth
		if sum > 0 {
			scaled = int(float64(widths[c]) * float64(budget) / float64(sum))
			if scaled < minColWidth {
				scaled = minColWidth
			}
		}
		widths[c] = scaled
	}
}

// distributeSurplus sets each column in cols to minColWidth plus a proportional
// share of surplus (relative to sumCols). It mutates widths in place.
func distributeSurplus(cols []string, widths map[string]int, surplus, sumCols int) {
	for _, c := range cols {
		extra := 0
		if sumCols > 0 {
			extra = int(float64(widths[c]) * float64(surplus) / float64(sumCols))
		}
		widths[c] = minColWidth + extra
	}
}

func (m resultsModel) Init() tea.Cmd {
	if m.loading {
		return m.spinner.Tick
	}
	return nil
}

func (m resultsModel) handleResize(msg tea.WindowSizeMsg) tea.Model {
	m.width = msg.Width
	m.height = msg.Height
	if len(m.result.Columns) > 0 {
		m.meta = renderMetadataBlock(m.queryType, m.result, m.params, m.width)
		m.table = buildResultsTable(m.result, m.width, m.height, metaReservedLines(m.meta))
	}
	return m
}

func (m resultsModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showDetail {
		switch msg.String() {
		case "d", "esc", "q":
			m.showDetail = false
		}
		return m, nil
	}
	// Any key clears a transient term-navigation warning so it disappears on the
	// next interaction.
	m.termWarning = ""
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
			if !models.CanAdvanceTerm(term, m.latestTerm) {
				m.termWarning = models.TermDisplayName(m.latestTerm) + " is the latest available term"
				return m, nil
			}
			next := models.NextTerm(term)
			return m, func() tea.Msg { return changeTermMsg{term: next} }
		}
	case "enter":
		if m.queryType != "roster_view" && m.queryType != "person_search" {
			break
		}
		row := m.table.SelectedRow()
		if len(row) == 0 || row[0] == "" {
			break
		}
		destType := "schedule_lookup"
		if m.queryType == "person_search" {
			destType = "instructor_lookup"
		}
		username, term := row[0], m.params["term"]
		return m, func() tea.Msg {
			return searchSubmittedMsg{
				queryType: destType,
				params:    map[string]string{"term": term, "username": username},
			}
		}
	case "a":
		switch m.queryType {
		case "roster_view":
			row := m.table.SelectedRow()
			if len(row) <= 6 || row[6] == "" {
				break
			}
			advisor, term := row[6], m.params["term"]
			return m, func() tea.Msg {
				return searchSubmittedMsg{
					queryType: "instructor_lookup",
					params:    map[string]string{"term": term, "username": advisor},
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
		if m.queryType != "instructor_lookup" && m.queryType != "schedule_lookup" && m.queryType != "course_search" {
			break
		}
		row := m.table.SelectedRow()
		if len(row) == 0 || row[0] == "" {
			break
		}
		courseID, term := row[0], m.params["term"]
		return m, func() tea.Msg {
			return searchSubmittedMsg{
				queryType: "roster_view",
				params:    map[string]string{"term": term, "course_id": courseID},
			}
		}
	case "i":
		if m.queryType != "course_search" {
			break
		}
		row := m.table.SelectedRow()
		if len(row) <= 3 || row[3] == "" {
			break
		}
		term := m.params["term"]

		// Extract instructor's username from "Last, First (username)" column
		start := strings.LastIndex(row[3], "(")
		end := strings.LastIndex(row[3], ")")
		if start == -1 || end <= start {
			break
		}
		username := row[3][start+1 : end]
		return m, func() tea.Msg {
			return searchSubmittedMsg{
				queryType: "instructor_lookup",
				params:    map[string]string{"term": term, "username": username},
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

	// Forward any other keys to the table
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m resultsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleResize(msg), nil
	case tea.KeyMsg:
		return m.handleKey(msg)
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

// renderMetadataBlock returns the styled, width-wrapped metadata header for the
// current result, or "" when there is nothing to show.
func renderMetadataBlock(queryType string, result query.Result, params map[string]string, width int) string {
	content := metadataContent(queryType, result, params)
	if content == "" {
		return ""
	}
	style := subtitleStyle
	if width > 0 {
		style = style.Width(width)
	}
	return style.Render(content)
}

// metadataContent builds the "Label: value" summary line for a result, joined by
// " │ ". Returns "" when no metadata applies to the query type.
func metadataContent(queryType string, result query.Result, params map[string]string) string {
	meta := result.Metadata
	var parts []string
	add := func(label, value string) {
		if value != "" {
			parts = append(parts, label+": "+value)
		}
	}

	switch queryType {
	case "schedule_lookup", "instructor_lookup":
		add("Name", meta["name"])
		add("Banner ID", meta["banner_id"])
		if meta["dept"] != "" {
			add("Dept", meta["dept"])
		} else {
			add("Major", meta["major"])
			add("Year", meta["year"])
			add("Advisor", meta["advisor_name"])
		}
	case "roster_view":
		add("Title", meta["title"])
		add("CRN", meta["crn"])
		add("Instructor", meta["instructor"])
		add("CrHrs", meta["credits"])
		add("Enrl", meta["enrolled"])
		add("Cap", meta["capacity"])
		add("Schedule", meta["schedule"])
		add("Comments", meta["comments"])
		add("Final Exam", meta["final_exam"])
	case "course_search":
		term := params["course_code"]
		if term == "" {
			term = params["instructor"]
		}
		summary := pluralize(len(result.Rows), "result", "results")
		if term != "" {
			summary = term + " · " + summary
		}
		return summary
	case "person_search":
		summary := pluralize(len(result.Rows), "match", "matches")
		if ln := params["last_name"]; ln != "" {
			summary = ln + "* · " + summary
		}
		return summary
	}

	return strings.Join(parts, " │ ")
}

// pluralize renders a count with the singular or plural noun, e.g. "1 result"
// or "3 results".
func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", singular)
	}
	return fmt.Sprintf("%d %s", n, plural)
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

	header := titleStyle.Render(title) + "\n"
	if m.meta != "" {
		header += m.meta + "\n"
	}
	footer := helpStyle.Render(help)
	prefix := ""
	if m.termWarning != "" {
		// BEL rings the terminal bell once on this render; the styled line is the
		// guaranteed visual signal.
		prefix = "\a"
		footer += "\n" + errorStyle.Render(m.termWarning)
	}
	return prefix + header +
		resultsBaseStyle.Render(m.table.View()) +
		"\n" + footer
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
