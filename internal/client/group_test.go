package client

import (
	"strings"
	"testing"
)

// groupHTML mirrors the structure of a real gr_Grid response (synthetic data):
// header table without BORDER, roster table, then the term grid whose cells are
// nested tables. A BGCOLOR on a nested table marks a shared slot.
const groupHTML = `<TABLE WIDTH=100% COLS=2><TR><TD>Group Schedule Grid - Fall 2026-2027</TD></TR></TABLE>
<p><TABLE BORDER=1 CELLSPACING=0 CELLPADDING=3><TR><TH>USERNAME</TH><TH>NAME</TH><TH>BANNER ID</TH><TH>MAJOR</TH><TH>CLASS</TH><TH>YEAR</TH><TH>ADVISOR</TH><TH>EMAIL</TH></TR>
<TR><TD><A HREF="/x?id=doea">doea</A></TD><TD>Ann  Doe</TD><TD>800000001</TD><TD>&nbsp</TD><TD>&nbsp</TD><TD>&nbsp</TD><TD>&nbsp</TD><TD><A HREF="mailto:DOEA@x.edu">DOEA@x.edu</A></TD></TR>
<TR><TD><A HREF="/x?id=roeb">roeb</A></TD><TD>Bob  Roe</TD><TD>800000002</TD><TD>&nbsp</TD><TD>&nbsp</TD><TD>&nbsp</TD><TD>&nbsp</TD><TD><A HREF="mailto:ROEB@x.edu">ROEB@x.edu</A></TD></TR>
</TABLE>
<p><TABLE BORDER=1 CELLSPACING=0 CELLPADDING=3><TR><TH>TERM GRID</TH><TH>8:00</TH><TH>9:00</TH><TH>EVENING</TH></TR>
<TR><TH>MONDAY</TH><TD><TABLE CELLSPACING=0 CELLPADDING=0 BGCOLOR=#FFCCCC><TR><TD>doea in O269</TD></TR><TR><TD>roeb in O157</TD></TR></TABLE></TD><TD><TABLE CELLSPACING=0 CELLPADDING=0><TR><TD>doea in C111</TD></TR></TABLE></TD><TD><TABLE CELLSPACING=0 CELLPADDING=0>&nbsp;</TABLE></TD></TR>
<TR><TH>TUESDAY</TH><TD><TABLE CELLSPACING=0 CELLPADDING=0>&nbsp;</TABLE></TD><TD><TABLE CELLSPACING=0 CELLPADDING=0><TR><TD>roeb in M107</TD></TR></TABLE></TD><TD><TABLE CELLSPACING=0 CELLPADDING=0>&nbsp;</TABLE></TD></TR>
</TABLE>`

func TestParseGroupSchedule(t *testing.T) {
	g, err := ParseGroupSchedule(strings.NewReader(groupHTML))
	if err != nil {
		t.Fatalf("ParseGroupSchedule: %v", err)
	}

	if len(g.Members) != 2 || g.Members[0].Username != "doea" || g.Members[0].Name != "Ann Doe" ||
		g.Members[1].BannerID != "800000002" || g.Members[1].Email != "ROEB@x.edu" {
		t.Errorf("members = %+v", g.Members)
	}
	if strings.Join(g.Hours, ",") != "8:00,9:00,EVENING" {
		t.Errorf("hours = %v", g.Hours)
	}
	if strings.Join(g.Days, ",") != "MONDAY,TUESDAY" {
		t.Errorf("days = %v", g.Days)
	}
	if len(g.Cells) != 2 || len(g.Cells[0]) != 3 {
		t.Fatalf("cells shape = %d x %d", len(g.Cells), len(g.Cells[0]))
	}

	shared := g.Cells[0][0]
	if !shared.Conflict || len(shared.Entries) != 2 ||
		shared.Entries[0].Username != "doea" || shared.Entries[0].Room != "O269" ||
		shared.Entries[1].Username != "roeb" || shared.Entries[1].Room != "O157" {
		t.Errorf("shared slot = %+v", shared)
	}
	if solo := g.Cells[0][1]; solo.Conflict || len(solo.Entries) != 1 {
		t.Errorf("solo slot = %+v", solo)
	}
	if empty := g.Cells[0][2]; empty.Conflict || len(empty.Entries) != 0 {
		t.Errorf("empty slot = %+v", empty)
	}
}

func TestParseGroupScheduleBadPages(t *testing.T) {
	for _, in := range []string{"", "<html>login</html>", "<table border=1><tr><th>X</th></tr></table>"} {
		if _, err := ParseGroupSchedule(strings.NewReader(in)); err == nil {
			t.Errorf("input %q: want error", in)
		}
	}
}
