package osint

import (
	"context"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// Accesses google.com/search, accepts cookies, and returns html content to be parsed
func GetHtml(queryUrl url.URL, ctx context.Context) (io.ReadCloser, error) {
	var htmlContent string
	err := chromedp.Run(ctx,
		chromedp.Navigate(queryUrl.String()),
		chromedp.Sleep(5*time.Second),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.InnerHTML("html", &htmlContent),
	)
	if err != nil {
		return nil, err
	}

	body := io.NopCloser(strings.NewReader(htmlContent))

	return body, nil
}

// Accesses website, and returns html content
func GetGoogleResultsHtml(url url.URL, ctx context.Context) (io.ReadCloser, error) {
	var htmlContent string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url.String()),
		chromedp.WaitVisible("body"),
		chromedp.Click(`
			/html/body/div[@aria-label='Before you continue to Google Search']/div[3]/span/div/div/div/div[3]/div/button/div[contains(text(),'Reject all')]`),
	)
	if err != nil {
		return nil, err
	}

	body := io.NopCloser(strings.NewReader(htmlContent))

	return body, nil
}
