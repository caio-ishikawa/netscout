package app

import (
	"flag"
	"fmt"
	"net/url"
	"os"
)

// Errors
const (
	invalidUrlError = "provided URL is not valid"
)

type Settings struct {
	SeedUrl          url.URL
	Depth            int
	ThreadCount      int
	ReqDelay         int
	Output           bool
	Verbose          bool
	BinaryEdgeApiKey string
}

func ParseFlags() (Settings, error) {
	urlPtr := flag.String("u", "", "A string representing the URL")
	depthPtr := flag.Int("d", 0, "An integer representing the depth of the crawl.")
	threadCountPtr := flag.Int("t", 5, "An integer representing the amount of threads to use for the scans.")
	reqDelayPtr := flag.Int("delay", 0, "An integer representing the delay between requests in miliseconds.")
	outputPtr := flag.Bool("o", false, "A boolean - it will output URLs to recon.mimir")

	verbosePtr := flag.Bool("v", false, "A boolean - if set, it will display all found URLs")

	flag.Parse()

	parsedUrl, err := url.Parse(*urlPtr)
	if err != nil {
		return Settings{}, err
	}

	if parsedUrl.Host == "" || parsedUrl.Scheme == "" {
		return Settings{}, fmt.Errorf(invalidUrlError)
	}

	// Defaults to empty string
	binaryEdgeApiKey := os.Getenv("BINARYEDGE_API_KEY")

	return Settings{
		SeedUrl:          *parsedUrl,
		Depth:            *depthPtr,
		BinaryEdgeApiKey: binaryEdgeApiKey,
		ThreadCount:      *threadCountPtr,
		ReqDelay:         *reqDelayPtr,
		Output:           *outputPtr,
		Verbose:          *verbosePtr,
	}, nil
}
