package app

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// Errors
const (
	invalidUrlError     = "provided URL is not valid"
	invalidKeyValuePair = "provided key-value pair is not valid"
	invalidCookieStr    = "provided cookie string is not valid"
	invalidHeaderStr    = "provided header string is not valid"
)

type Settings struct {
	Headless         bool
	SeedUrl          url.URL
	Depth            int
	LockHost         bool
	ThreadCount      int
	ReqDelay         int
	Output           string
	Verbose          bool
	Cookie           map[string]string
	Header           map[string]string
	BinaryEdgeApiKey string
	SerpApiKey       string
	SkipBinaryEdge   bool
	SkipGoogleDork   bool
	SkipAXFR         bool
	Deep             bool
}

func ParseFlags() (Settings, error) {
	headlessPtr := flag.Bool("headless", false, "A bool - if set, all requests will be made by a headless Chrome browser (requires Google Chrome)")
	urlPtr := flag.String("u", "", "A string representing the URL")
	depthPtr := flag.Int("d", 0, "An integer representing the depth of the crawl")
	lockHostPtr := flag.Bool("lock-host", false, "A boolean - if set, it will only save URLs with the same host as the seed")
	threadCountPtr := flag.Int("t", 1, "An integer representing the amount of threads to use for the scans")
	reqDelayPtr := flag.Int("delay-ms", 0, "An integer representing the delay between requests in miliseconds")
	outputPtr := flag.String("o", "", "A string representing the name of the output file")
	verbosePtr := flag.Bool("v", false, "A boolean - if set, it will display all found URLs")
	cookiePtr := flag.String("c", "", "A string representing request cookies")
	headerPtr := flag.String("h", "", "A string representing request header")

	skipBinaryEdgePtr := flag.Bool("skip-binaryedge", false, "A bool - if set, it will skip BinaryEdge subdomain scan")
	skipGoogleDorkPtr := flag.Bool("skip-google-dork", false, "A bool - if set, it will skip the Google filetype scan")
	skipAXFRPtr := flag.Bool("skip-axfr", false, "A bool - if set, it will skip the DNS zone trasnfer attempt")
	deepPtr := flag.Bool("deep", false, "A boolean - if set, it will perform a shortened URL scan (can take several minutes)")

	flag.Parse()

	parsedUrl, err := url.Parse(*urlPtr)
	if err != nil {
		return Settings{}, err
	}

	if parsedUrl.Host == "" || parsedUrl.Scheme == "" {
		return Settings{}, fmt.Errorf(invalidUrlError)
	}

	// parse cookie string
	cookieMap := make(map[string]string)
	if *cookiePtr != "" {
		cookies, err := parseKeyValueStr(*cookiePtr)
		if err != nil {
			return Settings{}, fmt.Errorf(invalidCookieStr)
		}

		cookieMap = cookies
	}

	// parse header string
	headerMap := make(map[string]string)
	if *headerPtr != "" {
		headers, err := parseKeyValueStr(*headerPtr)
		if err != nil {
			return Settings{}, fmt.Errorf(invalidCookieStr)
		}

		headerMap = headers
	}

	// Defaults to empty string
	binaryEdgeApiKey := os.Getenv("BINARYEDGE_API_KEY")
	serpApiKey := os.Getenv("SERP_API_KEY")

	return Settings{
		Headless:         *headlessPtr,
		SeedUrl:          *parsedUrl,
		Depth:            *depthPtr,
		LockHost:         *lockHostPtr,
		ThreadCount:      *threadCountPtr,
		ReqDelay:         *reqDelayPtr,
		Output:           *outputPtr,
		Verbose:          *verbosePtr,
		Cookie:           cookieMap,
		Header:           headerMap,
		BinaryEdgeApiKey: binaryEdgeApiKey,
		SerpApiKey:       serpApiKey,
		SkipBinaryEdge:   *skipBinaryEdgePtr,
		SkipGoogleDork:   *skipGoogleDorkPtr,
		SkipAXFR:         *skipAXFRPtr,
		Deep:             *deepPtr,
	}, nil
}

// Parses strings consisting of key-value pairs (e.g. header and cookies)
func parseKeyValueStr(str string) (map[string]string, error) {
	output := make(map[string]string)

	pairs := strings.Split(str, ",")
	for _, pair := range pairs {
		split := strings.Split(pair, "=")
		if len(split) != 2 {
			return output, fmt.Errorf(invalidKeyValuePair)
		}

		output[split[0]] = split[1]
	}

	return output, nil
}
