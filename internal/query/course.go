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

	params := url.Values{
		"type":     {"Course"},
		"termcode": {q.Term},
		"view":     {"tgrid"},
		"id":       {q.CourseCode},
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
