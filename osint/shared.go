package osint

import (
	"bufio"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

const CHROME_USER_AGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"

func removeScheme(targetUrl url.URL) string {
	scheme := targetUrl.Scheme + "://"
	filteredScheme := strings.Replace(targetUrl.String(), scheme, "", 1)

	return strings.Replace(filteredScheme, "/", "", 1)
}

// Performs an operation for each line of a given file.
func performOpOverFile(threadCount int, file *os.File, op func(text string)) {
	semaphore := make(chan struct{}, threadCount)
	var wg sync.WaitGroup

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		wg.Add(1)
		semaphore <- struct{}{}

		line := scanner.Text()

		op(line)
	}
}

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
