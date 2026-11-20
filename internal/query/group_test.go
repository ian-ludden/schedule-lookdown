package query

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luddenig/schedule-lookdown/internal/client"
	"github.com/luddenig/schedule-lookdown/internal/models"
)

const groupPage = `<TABLE BORDER=1><TR><TH>USERNAME</TH><TH>NAME</TH><TH>BANNER ID</TH><TH>EMAIL</TH></TR>
<TR><TD>doea</TD><TD>Ann Doe</TD><TD>1</TD><TD>a@x</TD></TR><TR><TD>roeb</TD><TD>Bob Roe</TD><TD>2</TD><TD>b@x</TD></TR></TABLE>
<TABLE BORDER=1><TR><TH>TERM GRID</TH><TH>8:00</TH><TH>9:00</TH></TR>
<TR><TH>MONDAY</TH><TD><TABLE BGCOLOR=#FFCCCC><TR><TD>doea in O1</TD></TR><TR><TD>roeb in O2</TD></TR></TABLE></TD><TD><TABLE>&nbsp;</TABLE></TD></TR></TABLE>`

func TestGroupGridValidate(t *testing.T) {
	cases := []struct {
		name string
		q    GroupGrid
		ok   bool
	}{
		{"ok", GroupGrid{Term: "202710", IDs: []string{"1", "2"}}, true},
		{"no term", GroupGrid{IDs: []string{"1", "2"}}, false},
		{"one person", GroupGrid{Term: "202710", IDs: []string{"1"}}, false},
		{"non-numeric id", GroupGrid{Term: "202710", IDs: []string{"1", "x&y"}}, false},
	}
	for _, tc := range cases {
		if err := tc.q.Validate(); (err == nil) != tc.ok {
			t.Errorf("%s: err = %v, ok want %v", tc.name, err, tc.ok)
		}
	}
}

func TestGroupGridExecuteFixture(t *testing.T) {
	path := filepath.Join(t.TempDir(), "g.html")
	if err := os.WriteFile(path, []byte(groupPage), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := client.NewFixture(path)
	if err != nil {
		t.Fatal(err)
	}

	res, err := (&GroupGrid{Term: "202710", IDs: []string{"11", "22"}}).Execute(context.Background(), c)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Join(res.Columns, "|") != "DAY|8:00|9:00" {
		t.Errorf("columns = %v", res.Columns)
	}
	// One day; the busiest slot (8:00) has two people, stacked alphabetically.
	want := [][]string{
		{"MON", "doea", ""},
		{"", "roeb", ""},
	}
	if len(res.Rows) != len(want) {
		t.Fatalf("rows = %v, want %v", res.Rows, want)
	}
	for i := range want {
		if strings.Join(res.Rows[i], "|") != strings.Join(want[i], "|") {
			t.Errorf("row %d = %v, want %v", i, res.Rows[i], want[i])
		}
	}
	if res.Mode != ModeGroup {
		t.Errorf("mode = %q", res.Mode)
	}
	if res.Metadata["members"] != "doea, roeb" {
		t.Errorf("members = %q", res.Metadata["members"])
	}
	if res.Group == nil || len(res.Group.Members) != 2 || !res.Group.Cells[0][0].Conflict {
		t.Errorf("Group = %+v", res.Group)
	}
}

func TestGroupGridRequestShape(t *testing.T) {
	var got http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = *r.Clone(r.Context())
		_, _ = w.Write([]byte(groupPage))
	}))
	defer srv.Close()

	c, err := client.NewWithBaseURL(srv.URL, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (&GroupGrid{Term: "202710", IDs: []string{"11", "22", "33"}}).Execute(context.Background(), c); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	q := got.URL.Query()
	if got.Method != http.MethodGet || q.Get("type") != "gr_Grid" || q.Get("termcode") != "202710" ||
		q.Get("view") != "tgrid" || q.Get("id6") != "11 22 33" {
		t.Errorf("request = %s %s", got.Method, got.URL.String())
	}
}

func TestGroupResultRowsFollowMaxOverlapAndSortAlphabetically(t *testing.T) {
	g := models.GroupSchedule{
		Members: []models.GroupMember{{Username: "zed"}, {Username: "amy"}},
		Hours:   []string{"8:00", "9:00"},
		Days:    []string{"MONDAY", "TUESDAY", "WEDNESDAY"},
		Cells: [][]models.GroupCell{
			// Monday: one person at a time -> a single row.
			{{Entries: []models.GroupEntry{{Username: "zed"}}}, {Entries: []models.GroupEntry{{Username: "amy"}}}},
			{{}, {}}, // Tuesday: nobody, omitted
			// Wednesday: three people at 9:00 (listed out of order) -> three rows.
			{{Entries: []models.GroupEntry{{Username: "zed"}}},
				{Entries: []models.GroupEntry{{Username: "zed"}, {Username: "bob"}, {Username: "amy"}}}},
		},
	}
	res := groupResult(g)
	got := make([]string, len(res.Rows))
	for i, r := range res.Rows {
		if IsGroupSeparator(r) {
			got[i] = "--"
		} else {
			got[i] = strings.Join(r, "|")
		}
	}
	want := []string{
		"MON|zed|amy",
		"--",
		"WED|zed|amy",
		"||bob",
		"||zed",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("rows =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestAbbreviateDay(t *testing.T) {
	for in, want := range map[string]string{"WEDNESDAY": "WED", "Monday": "MON", "TU": "TU", "": ""} {
		if got := abbreviateDay(in); got != want {
			t.Errorf("abbreviateDay(%q) = %q, want %q", in, got, want)
		}
	}
}
