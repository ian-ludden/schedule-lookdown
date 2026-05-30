package query

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"

	"github.com/luddenig/schedule-lookdown/internal/client"
)

// ScheduleLookup fetches the course schedule for a given RHIT username and term.
// Term codes observed: 202630 (Spring 2025-2026). Format appears to be YYYYTT
// where TT is 10=Fall, 20=Winter, 30=Spring.
type ScheduleLookup struct {
	Term     string // e.g. "202630"
	Username string // RHIT username, e.g. "valer"
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

	params := url.Values{
		"type":     {"Username"},
		"termcode": {q.Term},
		"view":     {"tgrid"},
		"id":       {q.Username},
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

	cols, rows, err := client.ParseTable(bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}

	meta, _ := client.ParseUserInfo(bytes.NewReader(body))
	return Result{Columns: cols, Rows: rows, Metadata: meta}, nil
}
