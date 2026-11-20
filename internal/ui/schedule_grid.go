package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/luddenig/schedule-lookdown/internal/models"
	"github.com/luddenig/schedule-lookdown/internal/query"
)

// minTableRows is the number of course-table lines that must remain visible for
// the term grid to be shown beneath it.
const minTableRows = 4

// renderScheduleGrid renders the term grid of an individual schedule as a static
// day x hour table, colored per course. Every hour column (EVENING and SATURDAY only when filled) and day is shown, and cells are truncated to fit width. It returns "" when there is
// nothing to show.
func renderScheduleGrid(result query.Result, width int) string {
	if result.Grid == nil {
		return ""
	}
	cols, rows := query.FlattenGridAllDays(withoutEmptySaturday(*result.Grid))
	if len(rows) == 0 {
		return ""
	}

	// Show every hour column, occupied or not, except EVENING, which is shown
	// only when it holds a class.
	var keep []int
	for c := range cols {
		if strings.EqualFold(cols[c], "EVENING") && !columnFilled(rows, c) {
			continue
		}
		keep = append(keep, c)
	}
	widths := make([]int, len(keep))
	for i, c := range keep {
		widths[i] = lipgloss.Width(cols[c])
		for _, r := range rows {
			if !query.IsGroupSeparator(r) && c < len(r) {
				widths[i] = max(widths[i], lipgloss.Width(r[c]))
			}
		}
	}
	// Shrink the widest column until the grid fits (1 space between columns).
	if width > 0 {
		for {
			total := len(widths) - 1
			for _, w := range widths {
				total += w
			}
			if total <= width {
				break
			}
			widest := 0
			for i, w := range widths {
				if w > widths[widest] {
					widest = i
				}
			}
			if widths[widest] <= 1 {
				break
			}
			widths[widest]--
		}
	}

	line := func(vals func(c int) string) string {
		parts := make([]string, len(keep))
		for i, c := range keep {
			v := vals(c)
			if lipgloss.Width(v) > widths[i] {
				v = string([]rune(v)[:widths[i]])
			}
			parts[i] = v + strings.Repeat(" ", widths[i]-lipgloss.Width(v))
		}
		return strings.TrimRight(strings.Join(parts, " "), " ")
	}
	rule := func() string {
		parts := make([]string, len(widths))
		for i, w := range widths {
			parts[i] = strings.Repeat("─", w)
		}
		return strings.Join(parts, " ")
	}

	var sb strings.Builder
	sb.WriteString(subtitleStyle.Render("Term grid") + "\n")
	sb.WriteString(line(func(c int) string { return cols[c] }) + "\n")
	sb.WriteString(rule())
	for _, r := range rows {
		sb.WriteString("\n")
		if query.IsGroupSeparator(r) {
			sb.WriteString(rule())
			continue
		}
		sb.WriteString(line(func(c int) string {
			if c < len(r) {
				return r[c]
			}
			return ""
		}))
	}
	return newGroupColorer(result.Grid).colorize(sb.String())
}

// scheduleGridFor returns the grid block (with its trailing spacing) and its
// height, or "" and 0 when it is absent or the window is too short to also keep
// minTableRows of the course table visible.
func scheduleGridFor(result query.Result, queryType string, width, height, reserved int) (string, int) {
	if queryType != "schedule_lookup" || height <= 0 {
		return "", 0
	}
	g := renderScheduleGrid(result, width)
	if g == "" {
		return "", 0
	}
	h := lipgloss.Height(g) + 1
	if height-resultsOverheadLines-reserved-h < minTableRows {
		return "", 0
	}
	return g, h
}

// columnFilled reports whether any non-rule row has text in column c.
func columnFilled(rows [][]string, c int) bool {
	for _, r := range rows {
		if !query.IsGroupSeparator(r) && c < len(r) && r[c] != "" {
			return true
		}
	}
	return false
}

// withoutEmptySaturday returns g without its Saturday row when nothing is
// scheduled then. The input is not modified.
func withoutEmptySaturday(g models.GroupSchedule) models.GroupSchedule {
	out := g
	out.Days, out.Cells = nil, nil
	for d, day := range g.Days {
		if strings.EqualFold(strings.TrimSpace(day), "SATURDAY") && d < len(g.Cells) {
			filled := false
			for _, c := range g.Cells[d] {
				if len(c.Entries) > 0 {
					filled = true
					break
				}
			}
			if !filled {
				continue
			}
		}
		out.Days = append(out.Days, day)
		if d < len(g.Cells) {
			out.Cells = append(out.Cells, g.Cells[d])
		}
	}
	return out
}
