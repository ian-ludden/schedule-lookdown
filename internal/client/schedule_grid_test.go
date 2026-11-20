package client

import (
	"os"
	"strings"
	"testing"
)

func TestParseScheduleGrid(t *testing.T) {
	f, err := os.Open("../../sample-responses/sample-student.html")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g, err := ParseScheduleGrid(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Days) == 0 || g.Days[0] != "MONDAY" {
		t.Fatalf("days = %v", g.Days)
	}
	if len(g.Hours) < 2 || g.Hours[1] != "9:00" {
		t.Fatalf("hours = %v", g.Hours)
	}
	e := g.Cells[0][1].Entries
	if len(e) != 1 || e[0].Username != "CSSE474-02" || e[0].Room != "O159" {
		t.Fatalf("monday 9:00 = %+v", e)
	}
}

func TestParseScheduleGridMissing(t *testing.T) {
	g, err := ParseScheduleGrid(strings.NewReader("<html><table border=1><tr><th>x</th></tr></table></html>"))
	if err != nil || len(g.Days) != 0 {
		t.Fatalf("g=%+v err=%v", g, err)
	}
}
