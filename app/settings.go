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
	Output           string
	Verbose          bool
	BinaryEdgeApiKey string
	SerpApiKey       string
	SkipBinaryEdge   bool
	SkipGoogleDork   bool
}

func ParseFlags() (Settings, error) {
	urlPtr := flag.String("u", "", "A string representing the URL")
	depthPtr := flag.Int("d", 0, "An integer representing the depth of the crawl")
	threadCountPtr := flag.Int("t", 5, "An integer representing the amount of threads to use for the scans")
	reqDelayPtr := flag.Int("delay", 0, "An integer representing the delay between requests in miliseconds")
	outputPtr := flag.String("o", "", "A string representing the name of the output file")
	verbosePtr := flag.Bool("v", false, "A boolean - if set, it will display all found URLs")

	skipBinaryEdgePtr := flag.Bool("skip-binaryedge", false, "A bool - if set, it will skip BinaryEdge subdomain scan")
	skipGoogleDorkPtr := flag.Bool("skip-google-dork", false, "A bool - if set, it will skip the Google filetype scan")

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
	serpApiKey := os.Getenv("SERP_API_KEY")

	return Settings{
		SeedUrl:          *parsedUrl,
		Depth:            *depthPtr,
		ThreadCount:      *threadCountPtr,
		ReqDelay:         *reqDelayPtr,
		Output:           *outputPtr,
		Verbose:          *verbosePtr,
		BinaryEdgeApiKey: binaryEdgeApiKey,
		SerpApiKey:       serpApiKey,
		SkipBinaryEdge:   *skipBinaryEdgePtr,
		SkipGoogleDork:   *skipGoogleDorkPtr,
	}, nil
}
