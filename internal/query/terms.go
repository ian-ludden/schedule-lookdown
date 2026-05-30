package query

import (
	"context"
	"net/url"

	"github.com/luddenig/schedule-lookdown/internal/client"
)

// FetchAvailableTerms retrieves the term codes offered by reg-sched.pl's
// <select name="termcode"> drop-down by fetching the base form page (a bare GET
// with no params) and parsing its options. Returned codes are in document order;
// use models.LatestTerm to pick the furthest-future one.
//
// TODO(verify-live): confirm a bare GET returns the form page containing the
// term drop-down, and whether any param is required to surface it.
func FetchAvailableTerms(ctx context.Context, c *client.Client) ([]string, error) {
	resp, err := c.Get(ctx, url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return client.ParseTermOptions(resp.Body)
}
