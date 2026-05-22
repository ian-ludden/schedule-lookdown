package query

import (
	"context"
	"errors"
	"net/url"

	"github.com/luddenig/schedule-lookdown/internal/client"
)

type SectionAvailability struct {
	Term string
	CRN  string
}

func (q *SectionAvailability) Name() string { return "Section Availability" }

func (q *SectionAvailability) Validate() error {
	if q.Term == "" || q.CRN == "" {
		return errors.New("term and CRN are required")
	}
	return nil
}

func (q *SectionAvailability) Execute(ctx context.Context, c *client.Client) (Result, error) {
	if err := q.Validate(); err != nil {
		return Result{}, err
	}

	// TODO: verify actual parameter names for CRN-based lookup.
	params := url.Values{
		"termcode": {q.Term},
		"crn":      {q.CRN},
		"view":     {"tgrid"},
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
