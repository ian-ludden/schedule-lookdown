package query

import (
	"context"
	"errors"
	"net/url"

	"github.com/luddenig/schedule-lookdown/internal/client"
)

// ScheduleLookup fetches the course schedule for a given RHIT username and term.
// Term codes observed: 202630 (Spring 2025-2026). Format appears to be YYYYTT
// where TT is 10=Fall, 20=Winter, 30=Spring.
type ScheduleLookup struct {
	Term     string // e.g. "202630"
	Username string // RHIT username, e.g. "luddenig"
}

func (q *ScheduleLookup) Name() string { return "Schedule Lookup" }

func (q *ScheduleLookup) Validate() error {
	if q.Term == "" || q.Username == "" {
		return errors.New("term and username are required")
	}
	return nil
}

func (q *ScheduleLookup) Execute(ctx context.Context, c *client.Client) (Result, error) {
	if err := q.Validate(); err != nil {
		return Result{}, err
	}

	// Parameters inferred from the example response HTML link patterns, e.g.:
	// /regweb-cgi/reg-sched.pl?type=Instructor&termcode=202630&view=tgrid&id=luddenig
	// The exact type value for a person schedule lookup needs verification.
	params := url.Values{
		"termcode": {q.Term},
		"id":       {q.Username},
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
