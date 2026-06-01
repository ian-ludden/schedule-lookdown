package client

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
)

const baseURL = "https://prodwebxe-hv.rose-hulman.edu/regweb-cgi/reg-sched.pl"

// downloadURL is the endpoint behind the roster page's "Download Roster" button.
// It returns the roster as a CSV body rather than HTML. It is a var (not const)
// so tests can point it at a local server.
var downloadURL = "https://prodwebxe-hv.rose-hulman.edu/regweb-cgi/reg-download.pl"

type Client struct {
	http        *http.Client
	base        *url.URL
	fixturePath string
	logger      *log.Logger
}

func New(cookies []*http.Cookie, logger *log.Logger) (*Client, error) {
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
		http:   &http.Client{Jar: jar},
		base:   base,
		logger: logger,
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
		if c.logger != nil {
			c.logger.Printf("GET (fixture) params=%s", params.Encode())
		}
		f, err := os.Open(c.fixturePath)
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: http.StatusOK, Body: f}, nil
	}

	u := *c.base
	u.RawQuery = params.Encode()
	if c.logger != nil {
		c.logger.Printf("GET %s", u.String())
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Printf("GET error: %v", err)
		}
		return nil, err
	}
	if c.logger != nil {
		c.logger.Printf("GET response: status=%d", resp.StatusCode)
	}
	return resp, nil
}

// DownloadRoster POSTs the roster-download form for courseID and returns the
// raw CSV body. courseID is e.g. "CSSE220-01" (a section) or "CSSE220"
// (combined). It mirrors the form on the roster page: fields id and download.
func (c *Client) DownloadRoster(ctx context.Context, courseID string) ([]byte, error) {
	form := url.Values{
		"id":       {courseID},
		"download": {"Download Roster"},
	}
	if c.logger != nil {
		c.logger.Printf("POST %s body=%s", downloadURL, form.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, downloadURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Printf("POST error: %v", err)
		}
		return nil, err
	}
	defer resp.Body.Close()
	if c.logger != nil {
		c.logger.Printf("POST response: status=%d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("roster download failed: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
