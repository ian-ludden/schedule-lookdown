package ui

import (
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/luddenig/schedule-lookdown/internal/models"
)

// groupPalette is a cyclic set of ANSI-256 colors for telling people apart in
// an ad-hoc group grid. Hues are spread out (loosely following the Okabe-Ito
// colorblind-safe set) and mid-brightness so they read on dark and light
// backgrounds; ROSE_RED is avoided because it is the selected-row highlight.
// With more people than colors, the palette repeats.
var groupPalette = []lipgloss.Color{
	"208", // orange
	"39",  // sky blue
	"36",  // teal green
	"220", // gold
	"170", // orchid
	"141", // lavender
	"114", // light green
	"209", // salmon
}

// groupColorer assigns each person in a group a palette color (alphabetical by
// username, so the assignment is stable across term changes) and recolors
// usernames in rendered text.
type groupColorer struct {
	styles map[string]lipgloss.Style
	re     *regexp.Regexp
}

func newGroupColorer(g *models.GroupSchedule) *groupColorer {
	if g == nil {
		return nil
	}
	seen := map[string]bool{}
	var names []string
	add := func(u string) {
		if u != "" && !seen[u] {
			seen[u] = true
			names = append(names, u)
		}
	}
	for _, m := range g.Members {
		add(m.Username)
	}
	for _, day := range g.Cells {
		for _, cell := range day {
			for _, e := range cell.Entries {
				add(e.Username)
			}
		}
	}
	if len(names) == 0 {
		return nil
	}
	sort.Strings(names)

	c := &groupColorer{styles: make(map[string]lipgloss.Style, len(names))}
	for i, n := range names {
		c.styles[n] = lipgloss.NewStyle().Foreground(groupPalette[i%len(groupPalette)]).Bold(true)
	}

	// Longest first so "luddenig" is matched before its prefix "ludden".
	byLen := append([]string(nil), names...)
	sort.Slice(byLen, func(i, j int) bool { return len(byLen[i]) > len(byLen[j]) })
	quoted := make([]string, len(byLen))
	for i, n := range byLen {
		quoted[i] = regexp.QuoteMeta(n)
	}
	c.re = regexp.MustCompile(`\b(?:` + strings.Join(quoted, "|") + `)\b`)
	return c
}

// name returns username rendered in its person's color.
func (c *groupColorer) name(username string) string {
	if c == nil {
		return username
	}
	if st, ok := c.styles[username]; ok {
		return st.Render(username)
	}
	return username
}

// colorize recolors every known username in a multi-line view. Lines that
// already contain escape sequences (the highlighted selected row, header rules)
// are left alone, since a nested style reset would cut their background short.
// Replacing a name with an equal-width styled name keeps the columns aligned.
func (c *groupColorer) colorize(view string) string {
	if c == nil {
		return view
	}
	lines := strings.Split(view, "\n")
	for i, line := range lines {
		if strings.Contains(line, "\x1b") {
			continue
		}
		lines[i] = c.re.ReplaceAllStringFunc(line, c.name)
	}
	return strings.Join(lines, "\n")
}
