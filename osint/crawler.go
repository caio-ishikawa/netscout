package osint

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/caio-ishikawa/netscout/shared"
	"golang.org/x/net/html"
)

const CRAWLER_NAME = "CRAWLER"

type Crawler struct {
	lockHost bool
	seedUrl  url.URL
	maxDepth int
	toCrawl  []url.URL
	urlMap   map[string]url.URL
	comms    shared.CommsChannels
}

func NewCrawler(
	lockHost bool,
	seedUrl url.URL,
	toCrawl []url.URL,
	maxDepth int,
	comms shared.CommsChannels,
) Crawler {
	return Crawler{
		lockHost: lockHost,
		seedUrl:  seedUrl,
		maxDepth: maxDepth,
		toCrawl:  toCrawl,
		urlMap:   map[string]url.URL{},
		comms:    comms,
	}
}

func (crawler *Crawler) Crawl(currDepth int) {
	if len(crawler.toCrawl) == 0 || currDepth == crawler.maxDepth {
		close(crawler.comms.DoneChan)
		return
	}

	toCrawl := crawler.toCrawl
	crawler.toCrawl = []url.URL{}

	for i := range toCrawl {
		htmlNode, err := crawler.getHtmlContent(toCrawl[i])
		if err != nil {
			crawler.propagateWarning(err.Error())
			continue
		}

		crawler.findLinks(htmlNode, toCrawl[i])
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
		crawler.propagateWarning(err.Error())
		return nil, err
	}

	return htmlDoc, nil
}

// Recursively gets URLs from a page and propagates it
func (crawler *Crawler) findLinks(node *html.Node, currUrl url.URL) {
	if node.Type == html.ElementNode && node.Data == "a" {
		for _, attr := range node.Attr {
			if attr.Key == "href" || attr.Key == "src" {
				crawler.handleFoundUrl(attr.Val, currUrl.Host, currUrl.Scheme)
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		crawler.findLinks(child, currUrl)
	}
}

// Creates scannedItem based on scanned URL and sends it via comms.DataChan
func (crawler *Crawler) handleFoundUrl(urlStr, host, scheme string) {
	url, err := parsePath(urlStr, host, scheme)
	if err != nil {
		crawler.propagateWarning(err.Error())
		return
	}

	if crawler.lockHost && url.Host != crawler.seedUrl.Host {
		return
	}

	_, exists := crawler.urlMap[url.String()]
	if !exists {
		crawler.toCrawl = append(crawler.toCrawl, url)
		crawler.urlMap[url.String()] = url

		relevanceScore := crawler.calcRelevanceScore(url)

		scanned := shared.ScannedItem{
			Url:       url,
			Relevance: relevanceScore,
			Source:    CRAWLER_NAME,
		}

		crawler.propagateData(scanned)
	}
}

// TODO: make it not terrible :)
func (crawler *Crawler) calcRelevanceScore(url url.URL) shared.Relevance {
	score := 0

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
		return shared.Low
	} else if score == 1 {
		return shared.Medium
	} else {
		return shared.High
	}
}

func (crawler *Crawler) propagateWarning(str string) {
	crawler.comms.WarningChan <- str
}

func (crawler *Crawler) propagateData(scanned shared.ScannedItem) {
	crawler.comms.DataChan <- scanned
}
