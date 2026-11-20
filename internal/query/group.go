package query

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/luddenig/schedule-lookdown/internal/client"
	"github.com/luddenig/schedule-lookdown/internal/models"
)

// GroupGrid fetches the merged schedule grid for an ad-hoc group in a single
// request. IDs are the directory IDs (models.Person.ID), not usernames.
type GroupGrid struct {
	Term string
	IDs  []string
}

func (q *GroupGrid) Name() string { return "Ad-Hoc Group Schedule" }

func (q *GroupGrid) Validate() error {
	if q.Term == "" {
		return errors.New("term is required")
	}
	if len(q.IDs) < 2 {
		return errors.New("at least two people are required; use Schedule Lookup for one")
	}
	for _, id := range q.IDs {
		if id == "" || strings.Trim(id, "0123456789") != "" {
			return fmt.Errorf("invalid person id %q", id)
		}
	}
	return nil
}

func (q *GroupGrid) Execute(ctx context.Context, c *client.Client) (Result, error) {
	if err := q.Validate(); err != nil {
		return Result{}, err
	}

	// The site takes the IDs space-separated in one id6 param.
	params := url.Values{
		"type":     {"gr_Grid"},
		"termcode": {q.Term},
		"view":     {"tgrid"},
		"id6":      {strings.Join(q.IDs, " ")},
	}
	resp, err := c.Get(ctx, params)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("group grid request failed: HTTP %d", resp.StatusCode)
	}

	g, err := client.ParseGroupSchedule(resp.Body)
	if err != nil {
		return Result{}, err
	}
	return groupResult(g), nil
}

// groupSeparator fills the cells of the rule row drawn between days. The table
// truncates each cell to its column width, so the string only needs to be long.
const groupSeparator = "────────────────────────────────"

// IsGroupSeparator reports whether row is the rule row between days.
func IsGroupSeparator(row []string) bool {
	return len(row) > 0 && strings.HasPrefix(row[0], "─")
}

// groupResult flattens a GroupSchedule into the generic Columns/Rows form the
// results table renders (a table cell cannot hold several lines, so occupants
// are stacked as separate rows). Each day gets only as many rows as its busiest
// slot has people; within a slot the usernames are stacked in alphabetical
// order. Day names are abbreviated ("WEDNESDAY" -> "WED"), days with no entries
// are omitted, and a rule row separates the remaining days.
func groupResult(g models.GroupSchedule) Result {
	cols, rows := FlattenGrid(g)
	members := make([]string, len(g.Members))
	for i, m := range g.Members {
		members[i] = m.Username
	}
	return Result{
		Columns:  cols,
		Rows:     rows,
		Metadata: map[string]string{"members": strings.Join(members, ", ")},
		Mode:     ModeGroup,
		Group:    &g,
	}
}

// FlattenGrid turns a term grid into DAY + hour columns and stacked rows (see
// groupResult for the layout rules).
func FlattenGrid(g models.GroupSchedule) ([]string, [][]string) {
	return flattenGrid(g, false)
}

// FlattenGridAllDays is FlattenGrid but keeps days with no entries (as a single
// blank row).
func FlattenGridAllDays(g models.GroupSchedule) ([]string, [][]string) {
	return flattenGrid(g, true)
}

func flattenGrid(g models.GroupSchedule, keepEmpty bool) ([]string, [][]string) {
	cols := append([]string{"DAY"}, g.Hours...)
	var rows [][]string
	for d, day := range g.Days {
		// Sorted usernames per slot, and the day's maximum overlap.
		slots := make([][]string, len(g.Cells[d]))
		depth := 0
		for h, cell := range g.Cells[d] {
			names := make([]string, len(cell.Entries))
			for i, e := range cell.Entries {
				names[i] = e.Username
			}
			sort.Strings(names)
			slots[h] = names
			depth = max(depth, len(names))
		}
		if depth == 0 {
			if !keepEmpty {
				continue
			}
			depth = 1
		}

		if len(rows) > 0 {
			sep := make([]string, len(cols))
			for i := range sep {
				sep[i] = groupSeparator
			}
			rows = append(rows, sep)
		}
		for i := 0; i < depth; i++ {
			label := ""
			if i == 0 {
				label = abbreviateDay(day)
			}
			row := []string{label}
			for _, names := range slots {
				text := ""
				if i < len(names) {
					text = names[i]
				}
				row = append(row, text)
			}
			rows = append(rows, row)
		}
	}

	return cols, rows
}

// abbreviateDay shortens a day name to its first three letters, upper-cased
// ("Wednesday" -> "WED"). Shorter names are returned upper-cased.
func abbreviateDay(day string) string {
	day = strings.ToUpper(strings.TrimSpace(day))
	if r := []rune(day); len(r) > 3 {
		return string(r[:3])
	}
	return day
}
