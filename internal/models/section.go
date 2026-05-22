package models

// Section mirrors the columns from the schedule table in reg-sched.pl.
// Term Schedule format: "MTRF/8:00/O159" (days/start-time/room).
type Section struct {
	Course     string // e.g. "CSSE474-01"
	CRN        string
	Title      string
	Instructor string
	Credits    int
	Enrolled   int
	Capacity   int
	Schedule   string // e.g. "MTRF/8:00/O159"
	Comments   string
	FinalExam  string
	TermDates  string
}
