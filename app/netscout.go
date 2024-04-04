package app

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/caio-ishikawa/netscout/osint"
	"github.com/caio-ishikawa/netscout/shared"
)

const (
	bold   = "\033[1m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	reset  = "\033[0m"
)

type NetScout struct {
	mutex      sync.Mutex
	outputFile *os.File
	settings   Settings
	Extensions []string
}

func NewApp(settings Settings) (NetScout, error) {
	return NetScout{
		outputFile: nil,
		settings:   settings,
		Extensions: []string{},
	}, nil
}

func (ns *NetScout) Scan() {
	if ns.settings.Output != "" {
		ns.createOutputFile(ns.settings.Output)
	}

	// initiate communication channels
	comms := shared.NewCommsChannels()
	go ns.handleComms(comms)

	var wg sync.WaitGroup
	if ns.settings.Deep {
		// download, unzip, and scan shortened URL list
		go ns.getShortenedUrls(comms, &wg)
	}

	// zone transfer
	subdomains, err := ns.attemptAXFR()
	if err != nil {
		ns.displayWarning("failed to perform zone transfer - continuing scan")
	}

	ns.outputUrls(subdomains, shared.Axfr)

	// binary edge subdomain query
	binaryEdgeRes, err := ns.getBinaryEdgeSubdomains()
	if err != nil {
		ns.displayWarning("failed to query BinaryEdge - continuing scan")
	}

	ns.outputUrls(binaryEdgeRes, shared.BinaryEdge)

	// crawling happens concurrently, and it updates the state as it finds URLs
	toCrawl := []url.URL{ns.settings.SeedUrl}
	ns.crawl(toCrawl, comms)

	// google dork
	filetypeLinks, err := ns.getFiletypeResults()
	if err != nil {
		ns.displayWarning("failed to query for filetypes")
	}

	ns.outputUrls(filetypeLinks, shared.Serp)

	// wait for goroutines to finish
	wg.Wait()
}

func (ns *NetScout) createOutputFile(name string) {
	file, err := os.Create(name)
	if err != nil {
		ns.displayError("failed to create output file - proceeding with scan")
		return
	}

	ns.outputFile = file
}

func (ns *NetScout) getShortenedUrls(comms shared.CommsChannels, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	finder := osint.NewShortenedUrlFinder(
		ns.settings.SeedUrl.Host,
		comms,
	)

	err := finder.DownloadShortenedURLs()
	if err != nil {
		comms.WarningChan <- "failed to download shortened URL list"
	}

	ns.displaySuccess("Shortened URL download complete")

	err = finder.UnzipAllDownloads()
	if err != nil {
		comms.WarningChan <- "failed to unzip shortened URL list"
	}

	finder.DecompressXZ()

}

func (ns *NetScout) attemptAXFR() ([]url.URL, error) {
	if ns.settings.SkipAXFR {
		return []url.URL{}, nil
	}

	ns.displaySuccess("Attempting AXFR")

	domain := shared.RemoveScheme(ns.settings.SeedUrl)
	subdomains, errs := osint.ZoneTransfer(domain)
	if len(errs) > 0 {
		ns.outputWarnings(errs)
	}

	if len(subdomains) == 0 {
		return []url.URL{}, fmt.Errorf("AFXR yielded no results")
	}

	return subdomains, nil
}

func (ns *NetScout) getBinaryEdgeSubdomains() ([]url.URL, error) {
	if ns.settings.SkipBinaryEdge {
		return []url.URL{}, nil
	}

	ns.displaySuccess("Querying BinaryEdge")

	client := osint.NewBinaryEdgeClient(ns.settings.BinaryEdgeApiKey)
	res, err := client.QuerySubdomains(ns.settings.SeedUrl)
	if err != nil {
		return []url.URL{}, err
	}

	var output []url.URL
	for _, subdomain := range res.Subdomains {
		u, err := url.Parse(subdomain)
		if err != nil {
			ns.displayWarning(err.Error())
			continue
		}

		// normalize
		if u.Scheme == "" {
			u.Scheme = ns.settings.SeedUrl.Scheme
		}

		if u.String() == ns.settings.SeedUrl.String() {
			continue
		}

		output = append(output, *u)
	}

	return output, nil
}

func (ns *NetScout) crawl(toCrawl []url.URL, comms shared.CommsChannels) {
	ns.displaySuccess("Starting crawl")

	crawler := osint.NewCrawler(
		ns.settings.Headless,
		ns.settings.LockHost,
		ns.settings.SeedUrl,
		ns.settings.ThreadCount,
		ns.settings.ReqDelay,
		toCrawl,
		ns.settings.Depth,
		comms,
	)

	crawler.Crawl(0)
}

func (ns *NetScout) getFiletypeResults() ([]url.URL, error) {
	if ns.settings.SkipGoogleDork {
		return []url.URL{}, nil
	}

	scanMsg := "Scanning for"
	for _, ext := range ns.Extensions {
		scanMsg = scanMsg + " " + ext
	}

	ns.displaySuccess(scanMsg)

	serpClient, err := osint.NewSerpClient(ns.settings.SerpApiKey)
	if err != nil {
		return []url.URL{}, err
	}

	queryStr := osint.GenerateFiletypeQuery(ns.settings.SeedUrl, ns.Extensions)
	results, errs := serpClient.SearchGoogle(queryStr)
	if len(errs) > 0 {
		ns.outputWarnings(errs)
		return []url.URL{}, nil
	}

	return results, nil
}

// Handles the consumption of incoming messages until process is done.
// TODO: refactor this
func (ns *NetScout) handleComms(comms shared.CommsChannels) {
	crawlFinish := false
	shortenedFinish := false

	var wg sync.WaitGroup
	defer wg.Done()

	msgDisplayed := false

	for {
		select {
		case msg := <-comms.DataChan:
			ns.manageDataChan(msg, &wg)
		case msg := <-comms.WarningChan:
			ns.displayWarning(msg)
		case <-comms.CrawlDoneChan:
			ns.manageDoneChan(
				shortenedFinish,
				crawlFinish,
				msgDisplayed,
				&wg,
				"In progress: Shortened URL scan (this can take several minutes)",
			)

			msgDisplayed = true
			crawlFinish = true
		case <-comms.ShortenedDoneChan:
			ns.manageDoneChan(
				shortenedFinish,
				crawlFinish,
				msgDisplayed,
				&wg,
				"In progress: Crawler",
			)

			msgDisplayed = true
			shortenedFinish = true
		default:
			time.Sleep(1 * time.Millisecond)
		}
	}
}

// Manages the incoming messages from the data channel. Starts a goroutine to collect file types from URLs.
func (ns *NetScout) manageDataChan(msg shared.ScannedItem, wg *sync.WaitGroup) {
	if msg.Source == shared.ShortenedUrl {
		if msg.Url.Host != ns.settings.SeedUrl.Host {
			return
		}
	}

	if ns.settings.Output != "" {
		ns.mutex.Lock()
		ns.outputFile.Write([]byte(msg.Format()))
		ns.mutex.Unlock()
	}

	wg.Add(1)

	go ns.CollectFiletypes(msg.Url, wg)

	ns.displayMsg(msg.Url.String())
}

// Manages the incoming messages from the done channel
func (ns *NetScout) manageDoneChan(
	shortenedFinish bool,
	crawlFinish bool,
	msgDisplayed bool,
	wg *sync.WaitGroup,
	msg string,
) {
	wg.Wait()

	if crawlFinish && shortenedFinish {
		return
	}

	if msgDisplayed {
		return
	}

	ns.displaySuccess(msg)
}

// Finds file extensions from list of URLs for google dork
func (ns *NetScout) CollectFiletypes(url url.URL, wg *sync.WaitGroup) {
	defer wg.Done()

	path := url.Path
	pathComponents := strings.Split(path, "/")
	for _, component := range pathComponents {
		ns.updateExtensions(component, url)
	}

	params := url.Query()
	for _, values := range params {
		for _, value := range values {
			ns.updateExtensions(value, url)
		}
	}
}

func (ns *NetScout) updateExtensions(file string, url url.URL) {
	ext := filepath.Ext(file)
	if ext == "" || ext == "." {
		return
	}

	extension := strings.TrimLeft(ext, ".")
	exists := shared.SliceContains(ns.Extensions, ext)

	if !exists {
		ns.Extensions = append(ns.Extensions, extension)
	}
}

// Displays found URLs and writes to output file depedning on settings.output
func (ns *NetScout) outputUrls(urls []url.URL, source shared.Source) {
	for _, u := range urls {
		scannedItem := shared.ScannedItem{Url: u, Source: source}

		if ns.settings.Output != "" {
			ns.outputFile.Write([]byte(scannedItem.Format()))
		}

		ns.displayMsg(u.String())
	}
}

// Displays warnings from error list
func (ns *NetScout) outputWarnings(errs []error) {
	for _, err := range errs {
		ns.displayWarning(err.Error())
	}
}

func (ns *NetScout) displayMsg(item string) {
	fmt.Printf("\n\033[A%s[x]%s %s\n", green, reset, item)
}

func (ns *NetScout) displaySuccess(text string) {
	fmt.Printf("\n\033[A%s[x] %s%s\n", green, text, reset)
}

func (ns *NetScout) displayWarning(text string) {
	fmt.Printf("\n\033[A%s[x] WARN: %s%s\n", yellow, text, reset)
}

func (ns *NetScout) displayError(text string) {
	fmt.Printf("\n\033[A%s[x] ERR: %s%s\n", red, text, reset)
}
