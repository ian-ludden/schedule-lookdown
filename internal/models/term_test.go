package models

import (
	"testing"
	"time"
)

func TestCurrentTerm(t *testing.T) {
	cases := []struct {
		date time.Time
		want string
	}{
		{time.Date(2026, time.May, 30, 0, 0, 0, 0, time.UTC), "202630"},      // Spring 2025-26
		{time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC), "202710"}, // Fall 2026-27
		{time.Date(2025, time.December, 15, 0, 0, 0, 0, time.UTC), "202620"}, // Winter 2025-26 (Dec)
		{time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC), "202620"},  // Winter 2025-26 (Jan)
		{time.Date(2026, time.July, 4, 0, 0, 0, 0, time.UTC), "202640"},      // Summer 2026
	}
	for _, c := range cases {
		if got := CurrentTerm(c.date); got != c.want {
			t.Errorf("CurrentTerm(%s) = %q, want %q", c.date.Format("2006-01-02"), got, c.want)
		}
	}
}

func TestNextTerm(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"202610", "202620"},
		{"202620", "202630"},
		{"202630", "202640"},
		{"202640", "202710"}, // Summer → Fall next year
		{"bad", "bad"},
		{"", ""},
	}
	for _, c := range cases {
		if got := NextTerm(c.in); got != c.want {
			t.Errorf("NextTerm(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPrevTerm(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"202620", "202610"},
		{"202630", "202620"},
		{"202640", "202630"},
		{"202610", "202540"}, // Fall → Summer previous year
		{"bad", "bad"},
		{"", ""},
	}
	for _, c := range cases {
		if got := PrevTerm(c.in); got != c.want {
			t.Errorf("PrevTerm(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCanAdvanceTerm(t *testing.T) {
	cases := []struct {
		name            string
		current, latest string
		want            bool
	}{
		{"at latest is blocked", "202630", "202630", false},
		{"before latest is allowed", "202620", "202630", true},
		{"empty latest is unbounded", "202630", "", true},
		{"already past latest is blocked", "202640", "202630", false},
		{"steps across year boundary up to latest", "202540", "202610", true},
	}
	for _, c := range cases {
		if got := CanAdvanceTerm(c.current, c.latest); got != c.want {
			t.Errorf("%s: CanAdvanceTerm(%q, %q) = %v, want %v", c.name, c.current, c.latest, got, c.want)
		}
	}
}

func TestLatestTerm(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want string
	}{
		{"picks max", []string{"202610", "202710", "202630"}, "202710"},
		{"order independent", []string{"202710", "202610"}, "202710"},
		{"skips malformed", []string{"bad", "202630", ""}, "202630"},
		{"all invalid", []string{"bad", "12345"}, ""},
		{"empty", nil, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := LatestTerm(c.in); got != c.want {
				t.Errorf("LatestTerm(%v) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestTermDisplayName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"202610", "Fall 2025-26"},
		{"202620", "Winter 2025-26"},
		{"202630", "Spring 2025-26"},
		{"202640", "Summer 2026"},
		{"200010", "Fall 1999-00"},
		{"bad", "bad"},
		{"", ""},
	}
	for _, c := range cases {
		if got := TermDisplayName(c.in); got != c.want {
			t.Errorf("TermDisplayName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
