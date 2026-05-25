package query

import (
	"context"
	"errors"
	"net/url"

	"github.com/luddenig/schedule-lookdown/internal/client"
)

type InstructorLookup struct {
	Term     string
	Username string // instructor's RHIT username, e.g. "luddenig"
}

func (q *InstructorLookup) Name() string { return "Instructor Lookup" }

func (q *InstructorLookup) Validate() error {
	if q.Term == "" || q.Username == "" {
		return errors.New("term and username are required")
	}
	return nil
}

func (q *InstructorLookup) Execute(ctx context.Context, c *client.Client) (Result, error) {
	if err := q.Validate(); err != nil {
		return Result{}, err
	}

	params := url.Values{
		"type":     {"Instructor"},
		"termcode": {q.Term},
		"view":     {"tgrid"},
		"id":       {q.Username},
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
