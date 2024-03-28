package osint

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/caio-ishikawa/mimir/shared"
	"golang.org/x/net/html"
)

// Errors
const (
	requestWarning   = "Could not make request to server."
	htmlParseWarning = "Could not parse response."
	pathParseWarning = "Could not parse found URL."
)

const CRAWLER_NAME = "CRAWLER"

type Crawler struct {
	seedUrl     url.URL
	maxDepth    int
	toCrawl     []url.URL
	urlMap      map[string]url.URL
	DataChan    chan shared.ScannedItem
	UpdateChan  chan string
	WarningChan chan string
	DoneChan    chan struct{}
}

func NewCrawler(
	seedUrl url.URL,
	maxDepth int,
	dataChan chan shared.ScannedItem,
	updateChan chan string,
	warningChan chan string,
	doneChan chan struct{},
) Crawler {
	return Crawler{
		seedUrl:     seedUrl,
		maxDepth:    maxDepth,
		toCrawl:     []url.URL{},
		urlMap:      map[string]url.URL{},
		DataChan:    dataChan,
		UpdateChan:  updateChan,
		WarningChan: warningChan,
		DoneChan:    doneChan,
	}
}

func (crawler *Crawler) Crawl(currDepth int) {
	if currDepth == 0 {
		crawler.toCrawl = []url.URL{crawler.seedUrl}
	}

	if len(crawler.toCrawl) == 0 {
		close(crawler.DoneChan)
		return
	}

	if currDepth == crawler.maxDepth {
		close(crawler.DoneChan)
		return
	}

	toCrawl := crawler.toCrawl
	crawler.toCrawl = []url.URL{}

	for i := range toCrawl {
		// keeps it from crawling out-of-scope websites
		if toCrawl[i].Host != crawler.seedUrl.Host {
			continue
		}

		crawler.UpdateChan <- toCrawl[i].String()

		resp, err := http.Get(toCrawl[i].String())
		if err != nil {
			crawler.UpdateChan <- requestWarning
			continue
		}
		defer resp.Body.Close()

		crawler.getLinks(resp.Body, toCrawl[i])
	}

	crawler.Crawl(currDepth + 1)
}

func (crawler *Crawler) getLinks(doc io.ReadCloser, seedUrl url.URL) {
	htmlDoc, err := html.Parse(doc)
	if err != nil {
		crawler.WarningChan <- htmlParseWarning
		return
	}

	crawler.traverseHtml(htmlDoc, seedUrl)
}

// Gets URLs of a page recursively and returns it.
// does not return the urls from parent
func (crawler *Crawler) traverseHtml(node *html.Node, currUrl url.URL) {
	if node.Type == html.ElementNode && (node.Data == "a" || node.Data == "img") {
		for _, attr := range node.Attr {
			if attr.Key == "href" || attr.Key == "src" {
				crawler.propagateScannedItem(attr.Val, currUrl.Host, currUrl.Scheme)
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		crawler.traverseHtml(child, currUrl)
	}
}

// creates scannedItem based on scanned URL
func (crawler *Crawler) propagateScannedItem(urlStr, host, scheme string) {
	url, err := parsePath(urlStr, host, scheme)
	if err != nil {
		crawler.WarningChan <- pathParseWarning
		return
	}

	_, exists := crawler.urlMap[url.String()]
	if !exists {
		crawler.toCrawl = append(crawler.toCrawl, url)
		crawler.urlMap[url.String()] = url

		scanned := crawler.analyzeUrl(url)

		crawler.DataChan <- scanned
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
