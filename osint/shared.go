package osint

import (
	"net/http"
	"net/url"
)

const CHROME_USER_AGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"

// Validates urlStr, and adds provided host and scheme if none are found in the URL object.
func parsePath(urlStr string, host string, scheme string) (url.URL, error) {
	parsedUrl, err := url.Parse(urlStr)
	if err != nil {
		return url.URL{}, err
	}

	if parsedUrl.Host == "" {
		parsedUrl.Host = host
	}

	if parsedUrl.Scheme == "" {
		parsedUrl.Scheme = scheme
	}

	return *parsedUrl, nil
}

func generateRequest(url url.URL) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return &http.Request{}, err
	}

	req.Header.Set("User-Agent", CHROME_USER_AGENT)

	return req, nil
}
