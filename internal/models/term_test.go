package models

import "testing"

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
