package osint

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/caio-ishikawa/mimir/shared"
	"golang.org/x/net/html" //"github.com/chromedp/chromedp"
)

// Errors
const (
	requestWarning   = "Could not make request to server"
	htmlParseWarning = "Could not parse response"
	pathParseWarning = "Could not parse found URL"
	htmlGetWarning   = "Could not get HTML for page"
)

const CRAWLER_NAME = "CRAWLER"

type Crawler struct {
	headless   bool
	lockHost   bool
	seedUrl    url.URL
	targetHost string
	maxDepth   int
	toCrawl    []url.URL
	urlMap     map[string]url.URL
	comms      shared.CommsChannels
}

func NewCrawler(
	headless bool,
	lockHost bool,
	seedUrl url.URL,
	host string,
	maxDepth int,
	comms shared.CommsChannels,
) Crawler {
	return Crawler{
		headless:   headless,
		lockHost:   lockHost,
		seedUrl:    seedUrl,
		targetHost: host,
		maxDepth:   maxDepth,
		toCrawl:    []url.URL{},
		urlMap:     map[string]url.URL{},
		comms:      comms,
	}
}

func (crawler *Crawler) Crawl(currDepth int) {
	if currDepth == 0 {
		crawler.toCrawl = []url.URL{crawler.seedUrl}
	}

	if len(crawler.toCrawl) == 0 || currDepth == crawler.maxDepth {
		close(crawler.comms.DoneChan)
		return
	}

	toCrawl := crawler.toCrawl
	crawler.toCrawl = []url.URL{}

	for i := range toCrawl {
		crawler.comms.UpdateChan <- toCrawl[i].String() // udpate spinner

		htmlNode, err := crawler.getHtmlContent(toCrawl[i])
		if err != nil {
			crawler.comms.WarningChan <- htmlGetWarning
			continue
		}

		crawler.traverseHtml(htmlNode, toCrawl[i])
	}

	crawler.Crawl(currDepth + 1)
}

func (crawler *Crawler) getHtmlContent(url url.URL) (*html.Node, error) {
	req, err := generateRequest(url)
	if err != nil {
		return nil, err
	}

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	htmlDoc, err := html.Parse(resp.Body)
	if err != nil {
		crawler.comms.WarningChan <- htmlParseWarning
		return nil, err
	}

	return htmlDoc, nil
}

// Gets URLs of a page recursively and propagates it via comms.DataChan
func (crawler *Crawler) traverseHtml(node *html.Node, currUrl url.URL) {
	if node.Type == html.ElementNode && node.Data == "a" {
		for _, attr := range node.Attr {
			if attr.Key == "href" || attr.Key == "src" {
				crawler.propagateUrl(attr.Val, currUrl.Host, currUrl.Scheme)
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		crawler.traverseHtml(child, currUrl)
	}
}

// creates scannedItem based on scanned URL and sends it via comms.DataChan
func (crawler *Crawler) propagateUrl(urlStr, host, scheme string) {
	url, err := parsePath(urlStr, host, scheme)
	if err != nil {
		crawler.comms.WarningChan <- pathParseWarning
		return
	}

	if crawler.lockHost && url.Host != crawler.targetHost {
		return
	}

	_, exists := crawler.urlMap[url.String()]
	if !exists {
		crawler.toCrawl = append(crawler.toCrawl, url)
		crawler.urlMap[url.String()] = url

		scanned := crawler.analyzeUrl(url)

		crawler.comms.DataChan <- scanned
	}
}

// TODO: make it not terrible :)
func (crawler *Crawler) analyzeUrl(url url.URL) shared.ScannedItem {
	score := 0
	var relevance shared.Relevance

	if url.Host == crawler.seedUrl.Host {
		score++
	}

	if url.Scheme == "http" {
		score++
	}

	if strings.Contains(url.Path, "=") {
		score++
	}

	if score == 0 {
		relevance = shared.Low
	} else if score == 1 {
		relevance = shared.Medium
	} else {
		relevance = shared.High
	}

	return shared.ScannedItem{
		Url:       url,
		Relevance: relevance,
		Source:    shared.Crawler,
	}
}
