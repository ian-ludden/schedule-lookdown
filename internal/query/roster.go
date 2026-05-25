package query

import (
	"context"
	"errors"
	"net/url"

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

	cols, rows, err := client.ParseRoster(resp.Body)
	if err != nil {
		return Result{}, err
	}
	return Result{Columns: cols, Rows: rows}, nil
}
