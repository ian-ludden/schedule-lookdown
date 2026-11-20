package query

import (
	"context"

	"github.com/luddenig/schedule-lookdown/internal/client"
	"github.com/luddenig/schedule-lookdown/internal/models"
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
	// Group carries the structured merged grid for ad-hoc group queries (nil
	// otherwise). Columns/Rows hold a flattened rendering of the same data.
	Group *models.GroupSchedule
	// Grid carries the term grid of an individual schedule lookup (nil when the
	// page had none). Entries' Username field holds the course code.
	Grid *models.GroupSchedule
}

// Result modes.
const (
	ModeSections = "sections"
	// ModeGroup marks an ad-hoc group grid: per-person lane rows with rule rows
	// (see IsGroupSeparator) between days that the UI skips when navigating.
	ModeGroup = "group"
)
