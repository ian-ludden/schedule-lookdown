package query

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/luddenig/schedule-lookdown/internal/client"
	"github.com/luddenig/schedule-lookdown/internal/models"
)

// FetchDirectory retrieves the selectable people list from reg-sched.pl's
// "Ad-hoc Group Schedules" page. That page is reached by POSTing the main
// lookup form with the ad_hoc button; the empty id/dept fields mirror what the
// browser sends. The result is ~2400 entries, so callers should cache it.
func FetchDirectory(ctx context.Context, c *client.Client, term string) ([]models.Person, error) {
	if term == "" {
		return nil, errors.New("term is required")
	}
	form := url.Values{
		"termcode": {term},
		"view":     {"tgrid"},
		"lnameid":  {""},
		"id1":      {""},
		"id4":      {""},
		"id5":      {""},
		"deptid":   {""},
		"ad_hoc":   {"Ad-hoc Group Schedules"},
	}
	resp, err := c.Post(ctx, form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("directory fetch failed: HTTP %d", resp.StatusCode)
	}
	people, err := client.ParseUserOptions(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(people) == 0 {
		return nil, errors.New("directory page contained no people (session expired or page changed?)")
	}
	return people, nil
}
