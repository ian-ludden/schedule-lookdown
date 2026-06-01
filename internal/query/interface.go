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
	Columns  []string
	Rows     [][]string
	Metadata map[string]string // optional extra info, e.g. "advisor"
	// Mode discriminates how the UI should render the result. "" is a normal
	// table/roster; "sections" marks a Roster View section picker (a course-list
	// table whose rows are sections to drill into).
	Mode string
}

// Result modes.
const (
	ModeSections = "sections"
)
