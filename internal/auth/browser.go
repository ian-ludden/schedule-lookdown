package auth

import (
	"context"
	"net/http"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const registrationURL = "https://prodwebxe-hv.rose-hulman.edu/regweb-cgi/reg-sched.pl"

// Authenticate opens a visible browser for the user to complete Microsoft SAML auth
// and returns session cookies once the redirect back to registrationURL succeeds.
func Authenticate(ctx context.Context) ([]*http.Cookie, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", false),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	chromeCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// TODO: replace WaitVisible with a poll on document.location.href so we
	// reliably wait for the SAML redirect to complete and land back on
	// registrationURL rather than just the Microsoft login page body.
	var rawCookies []*network.Cookie
	err := chromedp.Run(chromeCtx,
		chromedp.Navigate(registrationURL),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			rawCookies, err = network.GetCookies().Do(ctx)
			return err
		}),
	)
	if err != nil {
		return nil, err
	}

	cookies := make([]*http.Cookie, 0, len(rawCookies))
	for _, c := range rawCookies {
		cookies = append(cookies, &http.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HttpOnly: c.HTTPOnly,
		})
	}
	return cookies, nil
}
