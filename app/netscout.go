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

	toCrawl := []url.URL{ns.settings.SeedUrl}

	subdomains, err := ns.attemptAXFR()
	if err != nil {
		ns.displayWarning("failed to perform zone transfer - continuing scan")
	}

	ns.outputUrls(subdomains)

	binaryEdgeRes, err := ns.getBinaryEdgeSubdomains()
	if err != nil {
		ns.displayWarning("failed to query BinaryEdge - continuing scan")
	}

	ns.outputUrls(binaryEdgeRes)

	// crawling happens concurrently, and it updates the state as it finds URLs
	ns.crawl(ns.settings.LockHost, toCrawl)

	filetypeLinks, err := ns.getFiletypeResults()
	if err != nil {
		ns.displayWarning("failed to query for filetypes")
	}

	ns.outputUrls(filetypeLinks)
}

func (ns *NetScout) createOutputFile(name string) {
	file, err := os.Create(name)
	if err != nil {
		ns.displayError("failed to create output file - proceeding with scan")
		return
	}

	ns.outputFile = file
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

func (ns *NetScout) crawl(lockHost bool, toCrawl []url.URL) {
	ns.displaySuccess("Starting crawl")

	comms := shared.NewCommsChannels()

	crawler := osint.NewCrawler(
		lockHost,
		ns.settings.SeedUrl,
		ns.settings.ThreadCount,
		ns.settings.ReqDelay,
		toCrawl,
		ns.settings.Depth,
		comms,
	)

	go ns.handleComms(
		comms.DataChan,
		comms.WarningChan,
		comms.DoneChan,
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
func (ns *NetScout) handleComms(
	dataChan chan shared.ScannedItem,
	warningChan chan string,
	doneChan chan struct{},
) {
	var wg sync.WaitGroup
	for {
		select {
		case msg := <-dataChan:
			if ns.settings.Output != "" {
				ns.outputFile.Write([]byte(msg.Url.String() + "\n"))
			}
			if !ns.settings.Verbose && msg.Relevance == shared.Low {
				continue
			}

			wg.Add(1)
			go ns.CollectFiletypes(msg.Url, &wg)
			ns.displayMsg(msg.Url.String())
		case msg := <-warningChan:
			ns.displayWarning(msg)
		case <-doneChan:
			wg.Wait()
			return
		default:
			time.Sleep(1 * time.Millisecond)
		}
	}
}

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
func (ns *NetScout) outputUrls(urls []url.URL) {
	for _, subdomain := range urls {
		if ns.settings.Output != "" {
			ns.outputFile.Write([]byte(subdomain.String() + "\n"))
		}
		ns.displayMsg(subdomain.String())
	}
}

// Displays warnings from error list
func (ns *NetScout) outputWarnings(errs []error) {
	for _, err := range errs {
		ns.displayWarning(err.Error())
	}
}

func (ns *NetScout) displayMsg(item string) {
	fmt.Printf("%s[x]%s %s\n", green, reset, item)
}

func (ns *NetScout) displaySuccess(text string) {
	fmt.Printf("%s[x] %s%s\n", green, text, reset)
}

func (ns *NetScout) displayWarning(text string) {
	fmt.Printf("%s[x] WARN: %s%s\n", yellow, text, reset)
}

func (ns *NetScout) displayError(text string) {
	fmt.Printf("%s[x] ERR: %s%s\n", red, text, reset)
}
