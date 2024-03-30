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

func (ns *NetScout) Start() {
	if ns.settings.Output != "" {
		ns.createOutputFile(ns.settings.Output)
	}

	subdomains, err := ns.getBinaryEdgeSubdomains()
	if err != nil {
		ns.displayWarning(err.Error())
	}

	for _, subdomain := range subdomains {
		ns.displayMsg(subdomain.String())
	}

	// crawling happens concurrently, and it updates the state as it finds URLs
	toCrawl := append(subdomains, ns.settings.SeedUrl)
	ns.crawl(true, toCrawl)

	filetypeLinks, err := ns.getFiletypeResults()
	if err != nil {
		ns.displayWarning(err.Error())
	}

	for _, found := range filetypeLinks.OrganicResults {
		ns.displayMsg(found.Link)
	}
}

func (ns *NetScout) createOutputFile(name string) {
	file, err := os.Create(name)
	if err != nil {
		ns.displayError("failed to create output file - proceeding with scan")
		return
	}

	ns.outputFile = file
}

func (ns *NetScout) getBinaryEdgeSubdomains() ([]url.URL, error) {
	if ns.settings.SkipBinaryEdge {
		return []url.URL{}, fmt.Errorf("skipping BinaryEdge subdomain search")
	}
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
		toCrawl,
		ns.settings.Depth,
		comms,
	)

	go crawler.Crawl(0)

	ns.handleComms(
		comms.DataChan,
		comms.WarningChan,
		comms.DoneChan,
	)
}

func (ns *NetScout) getFiletypeResults() (osint.GoogleResults, error) {
	if ns.settings.SkipGoogleDork {
		return osint.GoogleResults{}, fmt.Errorf("skipping Goole dork")
	}

	scanMsg := "Scanning for"
	for _, ext := range ns.Extensions {
		scanMsg = scanMsg + " " + ext
	}

	ns.displaySuccess(scanMsg)

	serpClient, err := osint.NewSerpClient(ns.settings.SerpApiKey)
	if err != nil {
		return osint.GoogleResults{}, err
	}

	queryStr := osint.GenerateFiletypeQuery(ns.settings.SeedUrl, ns.Extensions)
	results, err := serpClient.SearchGoogle(queryStr)
	if err != nil {
		return osint.GoogleResults{}, nil
	}

	return results, nil
}

// handles the consumption of incoming messages until process is done.
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
			time.Sleep(10 * time.Millisecond)
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
