package models

// GroupMember is one person in an ad-hoc group result's roster table.
type GroupMember struct {
	Username string
	Name     string // "First Last", whitespace collapsed
	BannerID string
	Email    string
}

// GroupEntry is one person's occupancy of a grid slot ("adamg1 in O269").
type GroupEntry struct {
	Username string
	Room     string
}

// GroupCell is one day/hour slot of the merged grid. Conflict is true when the
// server shaded the slot (two or more people occupy it).
type GroupCell struct {
	Entries  []GroupEntry
	Conflict bool
}

// GroupSchedule is the parsed reg-sched.pl group grid page. Cells is indexed
// [day][hour] matching Days and Hours.
type GroupSchedule struct {
	Members []GroupMember
	Days    []string // e.g. MONDAY ... SATURDAY
	Hours   []string // e.g. 8:00 ... 5:00, EVENING
	Cells   [][]GroupCell
}
