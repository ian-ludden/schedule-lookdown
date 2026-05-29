package query

import (
	"context"
	"errors"
	"net/url"

	"github.com/luddenig/schedule-lookdown/internal/client"
)

// PersonSearch looks up people by last name using a wildcard search
// (e.g. id=Ludden* returns everyone whose last name starts with "Ludden").
type PersonSearch struct {
	Term     string
	LastName string
}

func (q *PersonSearch) Name() string { return "Person Search" }

func (q *PersonSearch) Validate() error {
	if q.Term == "" || q.LastName == "" {
		return errors.New("term and last name are required")
	}
	return nil
}

func (q *PersonSearch) Execute(ctx context.Context, c *client.Client) (Result, error) {
	if err := q.Validate(); err != nil {
		return Result{}, err
	}

	params := url.Values{
		"type":     {"Person"},
		"termcode": {q.Term},
		"view":     {"tgrid"},
		"id":       {q.LastName + "*"},
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
