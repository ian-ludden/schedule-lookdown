package query

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"strconv"

	"github.com/luddenig/schedule-lookdown/internal/client"
)

type RosterView struct {
	Term     string
	CourseID string // e.g. "CSSE474-02" or "CSSE474" for all sections
}

func (q *RosterView) Name() string { return "Roster View" }

func (q *RosterView) Validate() error {
	if q.Term == "" || q.CourseID == "" {
		return errors.New("term and course ID are required")
	}
	return nil
}

func (q *RosterView) Execute(ctx context.Context, c *client.Client) (Result, error) {
	if err := q.Validate(); err != nil {
		return Result{}, err
	}

	params := url.Values{
		"type":     {"Roster"},
		"termcode": {q.Term},
		"view":     {"tgrid"},
		"id":       {q.CourseID},
	}

	resp, err := c.Get(ctx, params)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, err
	}

	cols, rows, err := client.ParseRoster(bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}

	// Build a metadata header from the section-detail table (first BORDER=1
	// table) and the user-info table's "Course ID".
	meta, _ := client.ParseUserInfo(bytes.NewReader(body))
	if meta == nil {
		meta = map[string]string{}
	}
	if sections, err := client.ParseSections(bytes.NewReader(body)); err == nil && len(sections) > 0 {
		s := sections[0]
		meta["crn"] = s.CRN
		meta["title"] = s.Title
		meta["instructor"] = s.Instructor
		meta["credits"] = strconv.Itoa(s.Credits)
		meta["enrolled"] = strconv.Itoa(s.Enrolled)
		meta["capacity"] = strconv.Itoa(s.Capacity)
		meta["schedule"] = s.Schedule
		meta["comments"] = s.Comments
		meta["final_exam"] = s.FinalExam
	}

	return Result{Columns: cols, Rows: rows, Metadata: meta}, nil
}
