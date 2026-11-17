package models

import "strings"

// Person is one entry in reg-sched.pl's ad-hoc group selection list
// (<select name="id6">). ID is the opaque value the group POST expects (it is
// NOT the username or the Banner ID); usernames only appear in group results.
type Person struct {
	ID     string // option value, e.g. "2183942"
	Last   string
	First  string
	Detail string // class-year/major for students, e.g. "JR-BIO"; "" for faculty/staff
}

// DisplayName returns "Last, First", or just the last name when First is empty.
func (p Person) DisplayName() string {
	if p.First == "" {
		return p.Last
	}
	return p.Last + ", " + p.First
}

// FilterText is the string a fuzzy filter should match against.
func (p Person) FilterText() string {
	return strings.TrimSpace(p.Last + " " + p.First + " " + p.Detail)
}
