package osint

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/theckman/yacspin"
)

const (
	bold  = "\033[1m"
	reset = "\033[0m"
)

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
	// Set up for crawler by default, since it's the first step
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
