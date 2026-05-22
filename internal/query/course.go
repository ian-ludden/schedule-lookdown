package query

import (
	"context"
	"errors"
	"net/url"

	"github.com/luddenig/schedule-lookdown/internal/client"
)

type CourseSearch struct {
	Term       string // academic term, e.g. "202510"
	CourseCode string // e.g. "CSSE 132"
	Instructor string
}

func (q *CourseSearch) Name() string { return "Course Search" }

func (q *CourseSearch) Validate() error {
	if q.Term == "" {
		return errors.New("term is required")
	}
	if q.CourseCode == "" && q.Instructor == "" {
		return errors.New("course code or instructor is required")
	}
	return nil
}

func (q *CourseSearch) Execute(ctx context.Context, c *client.Client) (Result, error) {
	if err := q.Validate(); err != nil {
		return Result{}, err
	}

	// TODO: verify actual parameter names and whether GET or POST is used.
	// Observed pattern from HTML: type=Roster&termcode=202630&view=tgrid&id=CSSE474-01
	params := url.Values{
		"termcode": {q.Term},
		"id":       {q.CourseCode},
		"view":     {"tgrid"},
	}
	if q.Instructor != "" {
		params.Set("instructor", q.Instructor)
	}

	resp, err := c.Get(ctx, params)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	cols, rows, err := client.ParseTable(resp.Body)
	if err != nil {
		return Result{}, err
	}
	return Result{Columns: cols, Rows: rows}, nil
}
