package query

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"

	"github.com/luddenig/schedule-lookdown/internal/client"
)

type InstructorLookup struct {
	Term     string
	Username string // instructor's RHIT username, e.g. "valer"
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
