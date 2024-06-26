package osint

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/caio-ishikawa/netscout/shared"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"golang.org/x/net/html"
)

const CRAWLER_NAME = "CRAWLER"

type Crawler struct {
	mutex    sync.Mutex
	headless bool
	lockHost bool
	seedUrl  url.URL
	maxDepth int
	threads  int
	delay    int
	toCrawl  []url.URL
	urlMap   map[string]url.URL
	comms    shared.CommsChannels
	cookies  map[string]string
	headers  map[string]string
}

func NewCrawler(
	headless bool,
	lockHost bool,
	seedUrl url.URL,
	threads int,
	delay int,
	toCrawl []url.URL,
	maxDepth int,
	comms shared.CommsChannels,
	headers map[string]string,
	cookies map[string]string,
) Crawler {
	return Crawler{
		mutex:    sync.Mutex{},
		headless: headless,
		lockHost: lockHost,
		seedUrl:  seedUrl,
		threads:  threads,
		delay:    delay,
		maxDepth: maxDepth,
		toCrawl:  toCrawl,
		urlMap:   map[string]url.URL{},
		comms:    comms,
		cookies:  cookies,
		headers:  headers,
	}
}

func (crawler *Crawler) Crawl(currDepth int) {
	if len(crawler.toCrawl) == 0 || currDepth == crawler.maxDepth {
		close(crawler.comms.CrawlDoneChan)
		return
	}

	toCrawl := crawler.toCrawl
	crawler.toCrawl = []url.URL{}

	semaphore := make(chan struct{}, crawler.threads)
	var wg sync.WaitGroup

	for i := range toCrawl {
		wg.Add(1)

		// crawls page, updates toCrawl, and propagates found URLs through comms
		go crawler.crawlSinglePage(toCrawl[i], &wg, semaphore)
	}

	wg.Wait()

	crawler.Crawl(currDepth + 1)
}

// Orchestrates the crawling of a single page. Gets HTML, finds URLs, propagates it and updates toCrawl
func (crawler *Crawler) crawlSinglePage(url url.URL, wg *sync.WaitGroup, semaphore chan struct{}) {
	defer wg.Done()

	semaphore <- struct{}{}

	var htmlNode *html.Node
	if crawler.headless {
		node, err := crawler.getHtmlContentHeadless(url)
		if err != nil {
			crawler.propagateWarning(err.Error())
			return
		}

		htmlNode = node
	} else {
		node, err := crawler.getHtmlContent(url)
		if err != nil {
			crawler.propagateWarning(err.Error())
			return
		}

		htmlNode = node
	}

	// time request was made
	reqTime := time.Now()

	// TODO: make this asynchronous
	crawler.findLinks(htmlNode, url)

	// verifies how long to timeout before making next request
	elapsed := time.Since(reqTime)
	if int(elapsed.Milliseconds()) < crawler.delay {
		dur := crawler.delay - int(elapsed.Milliseconds())
		time.Sleep(time.Duration(dur) * time.Millisecond)
	}

	<-semaphore
}

// Gets HTML content from page with simple HTTP client
func (crawler *Crawler) getHtmlContent(url url.URL) (*html.Node, error) {
	req, err := generateRequest(url)
	if err != nil {
		return nil, err
	}

	if len(crawler.headers) > 0 {
		for key, value := range crawler.headers {
			req.Header.Add(key, value)
		}
	}

	if len(crawler.cookies) > 0 {
		for key, value := range crawler.cookies {
			expiration := time.Now().Add(365 * 24 * time.Hour)
			cookie := http.Cookie{Name: key, Value: value, Expires: expiration}
			req.AddCookie(&cookie)
		}
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

// Gets HTML content from page with headless Chrome browser
func (crawler *Crawler) getHtmlContentHeadless(url url.URL) (*html.Node, error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	network.Enable()

	// the ActionFunc is nil if the cookie hashmap is empty
	setCookiesFunc, err := crawler.setHeadlessCookie(ctx, url)
	if err != nil {
		return nil, err
	}

	var content string
	if err := chromedp.Run(ctx,
		setCookiesFunc,
		crawler.setHeadlessHeader(),
		chromedp.Navigate(url.String()),
		chromedp.WaitVisible("html", chromedp.ByQuery),
		chromedp.OuterHTML("body", &content),
	); err != nil {
		return nil, err
	}

	c, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return nil, err
	}

	return c, nil
}

// Returns chromedp ActionFunc that sets the cookies per each chrome request
func (crawler *Crawler) setHeadlessCookie(ctx context.Context, url url.URL) (chromedp.Action, error) {
	if len(crawler.cookies) == 0 {
		return chromedp.ActionFunc(func(ctx context.Context) error { return nil }), nil
	}

	return chromedp.ActionFunc(func(ctx context.Context) error {
		expr := cdp.TimeSinceEpoch(time.Now().Add(180 * 24 * time.Hour))

		for key, val := range crawler.cookies {
			err := network.SetCookie(key, val).
				WithDomain(url.Hostname()).
				WithExpires(&expr).
				Do(ctx)
			if err != nil {
				return err
			}
		}
		return nil
	}), nil
}

// Returns a chromedp ActionFunc that sets the headers per each chrome request
func (crawler *Crawler) setHeadlessHeader() chromedp.ActionFunc {
	if len(crawler.headers) == 0 {
		return chromedp.ActionFunc(func(ctx context.Context) error { return nil })
	}

	headers := make(map[string]interface{})
	for key, val := range crawler.headers {
		headers[key] = val
	}

	return chromedp.ActionFunc(func(ctx context.Context) error {
		return network.SetExtraHTTPHeaders(headers).Do(ctx)
	})
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

	crawler.mutex.Lock()
	defer crawler.mutex.Unlock()

	_, exists := crawler.urlMap[url.String()]
	if !exists {

		crawler.toCrawl = append(crawler.toCrawl, url)
		crawler.urlMap[url.String()] = url

		scanned := shared.ScannedItem{
			Url:    url,
			Source: CRAWLER_NAME,
		}

		crawler.propagateData(scanned)
	}
}

func (crawler *Crawler) propagateWarning(str string) {
	crawler.comms.WarningChan <- str
}

func (crawler *Crawler) propagateData(scanned shared.ScannedItem) {
	crawler.comms.DataChan <- scanned
}
