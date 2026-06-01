package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/luddenig/schedule-lookdown/internal/config"
	"github.com/luddenig/schedule-lookdown/internal/models"
)

// writeCourseFixture writes a one-row course-search HTML table and returns its path.
func writeCourseFixture(t *testing.T, courseID string) string {
	t.Helper()
	html := `<html><body>
		<table border=1>
			<tr><th>Course</th><th>CRN</th><th>Title</th></tr>
			<tr><td>` + courseID + `</td><td>1234</td><td>Intro</td></tr>
		</table>
	</body></html>`
	path := filepath.Join(t.TempDir(), "course.html")
	if err := os.WriteFile(path, []byte(html), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestExecuteQueryCmdSingleCourseJumpsToRoster(t *testing.T) {
	fixtures := map[string]string{"course_search": writeCourseFixture(t, "CSSE132-01")}
	cmd := executeQueryCmd(nil, "course_search",
		map[string]string{"term": "202630", "course_code": "CSSE 132"}, fixtures, true, nil)

	msg := cmd()
	ss, ok := msg.(searchSubmittedMsg)
	if !ok {
		t.Fatalf("expected searchSubmittedMsg, got %T (%v)", msg, msg)
	}
	if ss.queryType != "roster_view" {
		t.Errorf("queryType = %q, want roster_view", ss.queryType)
	}
	if ss.params["course_id"] != "CSSE132-01" {
		t.Errorf("course_id = %q, want CSSE132-01", ss.params["course_id"])
	}
	if ss.params["term"] != "202630" {
		t.Errorf("term = %q, want 202630", ss.params["term"])
	}
}

func TestExecuteQueryCmdSingleCourseNoJumpWhenDisabled(t *testing.T) {
	fixtures := map[string]string{"course_search": writeCourseFixture(t, "CSSE132-01")}
	cmd := executeQueryCmd(nil, "course_search",
		map[string]string{"term": "202630", "course_code": "CSSE 132"}, fixtures, false, nil)

	if _, ok := cmd().(queryResultMsg); !ok {
		t.Errorf("with jump disabled, expected queryResultMsg, got %T", cmd())
	}
}

func TestResolvedDefaultTerm(t *testing.T) {
	current := models.CurrentTerm(time.Now())

	cases := []struct {
		name       string
		cfg        config.Config
		latestTerm string
		want       string
	}{
		{"current mode", config.Config{DefaultTerm: config.DefaultTermCurrent}, "202710", current},
		{"latest with term", config.Config{DefaultTerm: config.DefaultTermLatest}, "202710", "202710"},
		{"latest not yet fetched", config.Config{DefaultTerm: config.DefaultTermLatest}, "", current},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := App{config: c.cfg, latestTerm: c.latestTerm}
			if got := a.resolvedDefaultTerm(); got != c.want {
				t.Errorf("resolvedDefaultTerm() = %q, want %q", got, c.want)
			}
		})
	}
}
