package osint

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/theckman/yacspin"
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

func createSpinner(suffix string) yacspin.Spinner {
	bold := "\033[1m"
	reset := "\033[0m"

	cfg := yacspin.Config{
		Frequency:     100 * time.Millisecond,
		CharSet:       yacspin.CharSets[39],
		Suffix:        fmt.Sprintf("%s%s ", bold, suffix),
		StopCharacter: "[x] ",
		StopColors:    []string{"fgGreen"},
		StopMessage:   fmt.Sprintf("Results:%s", reset),
	}

	spinner, _ := yacspin.New(cfg)

	return *spinner
}

func generateRequest(url url.URL) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return &http.Request{}, err
	}

	req.Header.Set("User-Agent", CHROME_USER_AGENT)

	return req, nil
}
