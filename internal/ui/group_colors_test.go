package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/luddenig/schedule-lookdown/internal/models"
	"github.com/muesli/termenv"
)

// withColor forces ANSI-256 output for the test (lipgloss emits no escape codes
// when stdout is not a terminal).
func withColor(t *testing.T) {
	t.Helper()
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() { lipgloss.SetColorProfile(old) })
}

func groupOf(users ...string) *models.GroupSchedule {
	g := &models.GroupSchedule{}
	for _, u := range users {
		g.Members = append(g.Members, models.GroupMember{Username: u})
	}
	return g
}

func TestGroupColorerAssignsByAlphabeticalOrderAndCycles(t *testing.T) {
	var users []string
	n := len(groupPalette) + 2
	for i := n; i >= 1; i-- {
		users = append(users, fmt.Sprintf("u%02d", i)) // supplied in reverse order
	}
	c := newGroupColorer(groupOf(users...))

	// Alphabetical: u01 -> palette[0], u02 -> palette[1], ...; wraps after the palette.
	for i := 1; i <= len(users); i++ {
		got := c.styles[fmt.Sprintf("u%02d", i)].GetForeground()
		if want := groupPalette[(i-1)%len(groupPalette)]; got != want {
			t.Errorf("u%02d color = %v, want %v", i, got, want)
		}
	}
	if c.styles["u01"].GetForeground() == c.styles["u02"].GetForeground() {
		t.Error("adjacent people must get different colors")
	}
}

func TestGroupColorizeWholeNamesKeepsWidth(t *testing.T) {
	withColor(t)
	c := newGroupColorer(groupOf("ludden", "luddenig", "adamg1"))

	line := "│ MON   ludden    luddenig   adamg1 │"
	out := c.colorize(line)

	if out == line {
		t.Fatal("expected colored output")
	}
	if lipgloss.Width(out) != lipgloss.Width(line) {
		t.Errorf("visible width changed: %d -> %d", lipgloss.Width(line), lipgloss.Width(out))
	}
	// luddenig must be colored as one token, not as "ludden" + "ig".
	if !strings.Contains(out, c.styles["luddenig"].Render("luddenig")) {
		t.Errorf("luddenig not matched whole: %q", out)
	}
	if strings.Contains(out, c.styles["ludden"].Render("ludden")+"ig") {
		t.Errorf("ludden matched inside luddenig: %q", out)
	}
	// Words that merely contain a username are untouched.
	if got := c.colorize("xludden"); got != "xludden" {
		t.Errorf("substring colored: %q", got)
	}
}

func TestGroupColorizeSkipsStyledLines(t *testing.T) {
	withColor(t)
	c := newGroupColorer(groupOf("a1", "b2"))
	styled := "\x1b[41mMON a1 b2\x1b[0m"
	plain := "TUE a1"
	out := c.colorize(styled + "\n" + plain)
	lines := strings.Split(out, "\n")
	if lines[0] != styled {
		t.Errorf("styled line modified: %q", lines[0])
	}
	if lines[1] == plain {
		t.Error("plain line should be colored")
	}
}

func TestGroupColorerNilSafe(t *testing.T) {
	var c *groupColorer
	if c.colorize("x") != "x" || c.name("u") != "u" {
		t.Error("nil colorer must be a no-op")
	}
	if newGroupColorer(nil) != nil || newGroupColorer(groupOf()) != nil {
		t.Error("no people -> nil colorer")
	}
}
