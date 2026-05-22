package query

import (
	"context"

	"github.com/luddenig/schedule-lookdown/internal/client"
)

// Query represents a parameterized request to the registration system.
type Query interface {
	Name() string
	Validate() error
	Execute(ctx context.Context, c *client.Client) (Result, error)
}

// Result holds tabular data returned from a query.
type Result struct {
	Columns []string
	Rows    [][]string
}
