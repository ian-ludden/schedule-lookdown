package client

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
)

const baseURL = "https://prodwebxe-hv.rose-hulman.edu/regweb-cgi/reg-sched.pl"

type Client struct {
	http        *http.Client
	base        *url.URL
	fixturePath string
}

func New(cookies []*http.Cookie) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	jar.SetCookies(base, cookies)

	return &Client{
		http: &http.Client{Jar: jar},
		base: base,
	}, nil
}

// NewFixture returns a client that serves all GET requests from a local file
// instead of making real HTTP requests. Useful for testing without auth.
func NewFixture(path string) (*Client, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	return &Client{base: base, fixturePath: path}, nil
}

// Get issues a GET request with the given query params.
// If a fixture path is set, the file is returned instead of hitting the network.
func (c *Client) Get(ctx context.Context, params url.Values) (*http.Response, error) {
	if c.fixturePath != "" {
		f, err := os.Open(c.fixturePath)
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: http.StatusOK, Body: f}, nil
	}

	u := *c.base
	u.RawQuery = params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	return c.http.Do(req)
}

// Post issues a form-encoded POST request.
func (c *Client) Post(ctx context.Context, params url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.http.Do(req)
}
