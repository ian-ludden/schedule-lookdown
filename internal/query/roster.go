package query

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/luddenig/schedule-lookdown/internal/client"
)

type RosterView struct {
	Term     string
	CourseID string // e.g. "CSSE474-02" or "CSSE474" for all sections
	// Combined requests the merged all-sections roster for a bare course code,
	// bypassing the section picker. Ignored when CourseID names a specific
	// section.
	Combined bool
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

	// A bare course code (no "-NN" suffix) may resolve to multiple sections.
	isBareCode := !strings.Contains(q.CourseID, "-")
	sections, _ := client.ParseSections(bytes.NewReader(body))
	multiSection := len(sections) > 1

	// Section picker: a bare code with several sections lands on a course-list
	// table (identical columns to Course Search) so the user can drill into one.
	if isBareCode && !q.Combined && multiSection {
		cols, rows, err := client.ParseTable(bytes.NewReader(body))
		if err != nil {
			return Result{}, err
		}
		meta := map[string]string{}
		if len(sections) > 0 {
			meta["title"] = sections[0].Title
		}
		meta["sections"] = strconv.Itoa(len(sections))
		return Result{Columns: cols, Rows: rows, Metadata: meta, Mode: ModeSections}, nil
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

	// Combined all-sections roster: per-section fields (CRN, instructor,
	// schedule, final exam, comments) describe only one section and would be
	// misleading, so trim them and report aggregate enrolment instead.
	combined := q.Combined || (isBareCode && multiSection)
	if combined && len(sections) > 0 {
		enrolled, capacity := 0, 0
		for _, s := range sections {
			enrolled += s.Enrolled
			capacity += s.Capacity
		}
		meta["title"] = sections[0].Title
		meta["credits"] = strconv.Itoa(sections[0].Credits)
		meta["enrolled"] = strconv.Itoa(enrolled)
		meta["capacity"] = strconv.Itoa(capacity)
		meta["sections"] = strconv.Itoa(len(sections))
	} else if len(sections) > 0 {
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
