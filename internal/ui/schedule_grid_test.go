package ui

import (
	"strings"
	"testing"

	"github.com/luddenig/schedule-lookdown/internal/models"
	"github.com/luddenig/schedule-lookdown/internal/query"
)

func testGridResult() query.Result {
	g := &models.GroupSchedule{
		Days:  []string{"MONDAY", "TUESDAY"},
		Hours: []string{"8:00", "9:00", "10:00", "EVENING"},
		Cells: [][]models.GroupCell{
			{{}, {Entries: []models.GroupEntry{{Username: "CSSE474-02", Room: "O159"}}}, {}, {}},
			{{}, {}, {}, {}},
		},
	}
	return query.Result{Columns: []string{"Course"}, Rows: [][]string{{"CSSE474-02"}}, Grid: g}
}

func TestRenderScheduleGrid(t *testing.T) {
	out := renderScheduleGrid(testGridResult(), 80)
	for _, want := range []string{"MON", "9:00", "CSSE474-02"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	for _, want := range []string{"8:00", "10:00"} {
		if !strings.Contains(out, want) {
			t.Errorf("empty hour %q should still be shown:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "TUE") {
		t.Errorf("empty day should still be shown:\n%s", out)
	}
}

func TestScheduleGridPlacementAndHeight(t *testing.T) {
	r := testGridResult()
	m := newResultsModelWithData(r, "schedule_lookup", nil, 100, 40, "")
	v := m.View()
	if i, j := strings.Index(v, "CSSE474-02"), strings.Index(v, "Term grid"); j < 0 || i < 0 {
		t.Fatalf("grid missing:\n%s", v)
	}
	if h := strings.Count(v, "\n") + 1; h > 40 {
		t.Errorf("view height %d exceeds 40", h)
	}
	short := newResultsModelWithData(r, "schedule_lookup", nil, 100, 12, "")
	if short.grid != "" {
		t.Error("grid should be hidden on a short window")
	}
}

func TestRenderScheduleGridEvening(t *testing.T) {
	r := testGridResult()
	if out := renderScheduleGrid(r, 80); strings.Contains(out, "EVENING") {
		t.Errorf("empty EVENING should be hidden:\n%s", out)
	}
	r.Grid.Cells[1][3] = models.GroupCell{Entries: []models.GroupEntry{{Username: "MA111-01"}}}
	if out := renderScheduleGrid(r, 80); !strings.Contains(out, "EVENING") || !strings.Contains(out, "MA111-01") {
		t.Errorf("filled EVENING should be shown:\n%s", out)
	}
}

func TestRenderScheduleGridSaturday(t *testing.T) {
	r := testGridResult()
	r.Grid.Days = append(r.Grid.Days, "SATURDAY")
	r.Grid.Cells = append(r.Grid.Cells, []models.GroupCell{{}, {}, {}, {}})
	if out := renderScheduleGrid(r, 80); strings.Contains(out, "SAT") {
		t.Errorf("empty Saturday should be hidden:\n%s", out)
	}
	r.Grid.Cells[2][1] = models.GroupCell{Entries: []models.GroupEntry{{Username: "PE101-01"}}}
	if out := renderScheduleGrid(r, 80); !strings.Contains(out, "SAT") || !strings.Contains(out, "PE101-01") {
		t.Errorf("filled Saturday should be shown:\n%s", out)
	}
}
