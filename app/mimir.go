package app

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/caio-ishikawa/mimir/osint"
	"github.com/caio-ishikawa/mimir/shared"
)

const (
	bold   = "\033[1m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	reset  = "\033[0m"
)

type MimirApp struct {
	outputFile *os.File
	settings   Settings
	Extensions []string
}

func NewApp(settings Settings) (MimirApp, error) {
	return MimirApp{
		outputFile: nil,
		settings:   settings,
		Extensions: []string{},
	}, nil
}

func (mimir *MimirApp) Start() {
	if mimir.settings.Output != "" {
		mimir.createOutputFile(mimir.settings.Output)
	}

	subdomains, err := mimir.getBinaryEdgeSubdomains()
	if err != nil {
		mimir.displayWarning(err.Error())
	}

	for _, subdomain := range subdomains {
		mimir.displayMsg(subdomain.String())
	}

	// crawling happens concurrently, and it updates the state as it finds URLs
	toCrawl := append(subdomains, mimir.settings.SeedUrl)
	mimir.crawl(true, toCrawl)

	filetypeLinks, err := mimir.getFiletypeResults()
	if err != nil {
		mimir.displayWarning(err.Error())
	}

	for _, found := range filetypeLinks.OrganicResults {
		mimir.displayMsg(found.Link)
	}
}

func (mimir *MimirApp) createOutputFile(name string) {
	file, err := os.Create(name)
	if err != nil {
		mimir.displayError("failed to create output file - proceeding with scan")
		return
	}

	mimir.outputFile = file
}

func (mimir *MimirApp) getBinaryEdgeSubdomains() ([]url.URL, error) {
	if mimir.settings.SkipBinaryEdge {
		return []url.URL{}, fmt.Errorf("skipping BinaryEdge subdomain search")
	}
	client := osint.NewBinaryEdgeClient(mimir.settings.BinaryEdgeApiKey)
	res, err := client.QuerySubdomains(mimir.settings.SeedUrl)
	if err != nil {
		return []url.URL{}, err
	}

	var output []url.URL
	for _, subdomain := range res.Subdomains {
		u, err := url.Parse(subdomain)
		if err != nil {
			mimir.displayWarning(err.Error())
			continue
		}

		// normalize
		if u.Scheme == "" {
			u.Scheme = mimir.settings.SeedUrl.Scheme
		}

		if u.String() == mimir.settings.SeedUrl.String() {
			continue
		}

		output = append(output, *u)
	}

	return output, nil
}

func (mimir *MimirApp) crawl(lockHost bool, toCrawl []url.URL) {
	mimir.displaySuccess("Starting crawl")

	comms := shared.NewCommsChannels()

	crawler := osint.NewCrawler(
		lockHost,
		mimir.settings.SeedUrl,
		toCrawl,
		mimir.settings.Depth,
		comms,
	)

	go crawler.Crawl(0)

	mimir.handleComms(
		comms.DataChan,
		comms.WarningChan,
		comms.DoneChan,
	)
}

func (mimir *MimirApp) getFiletypeResults() (osint.GoogleResults, error) {
	if mimir.settings.SkipGoogleDork {
		return osint.GoogleResults{}, fmt.Errorf("skipping Goole dork")
	}

	scanMsg := "Scanning for"
	for _, ext := range mimir.Extensions {
		scanMsg = scanMsg + " " + ext
	}

	mimir.displaySuccess(scanMsg)

	serpClient, err := osint.NewSerpClient(mimir.settings.SerpApiKey)
	if err != nil {
		return osint.GoogleResults{}, err
	}

	queryStr := osint.GenerateFiletypeQuery(mimir.settings.SeedUrl, mimir.Extensions)
	results, err := serpClient.SearchGoogle(queryStr)
	if err != nil {
		return osint.GoogleResults{}, nil
	}

	return results, nil
}

// handles the consumption of incoming messages until process is done.
func (mimir *MimirApp) handleComms(
	dataChan chan shared.ScannedItem,
	warningChan chan string,
	doneChan chan struct{},
) {
	var wg sync.WaitGroup
	for {
		select {
		case msg := <-dataChan:
			if mimir.settings.Output != "" {
				mimir.outputFile.Write([]byte(msg.Url.String() + "\n"))
			}
			if !mimir.settings.Verbose && msg.Relevance == shared.Low {
				continue
			}

			wg.Add(1)
			go mimir.CollectFiletypes(msg.Url, &wg)
			mimir.displayMsg(msg.Url.String())
		case msg := <-warningChan:
			mimir.displayWarning(msg)
		case <-doneChan:
			wg.Wait()
			return
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func (mimir *MimirApp) CollectFiletypes(url url.URL, wg *sync.WaitGroup) {
	defer wg.Done()

	path := url.Path
	pathComponents := strings.Split(path, "/")
	for _, component := range pathComponents {
		mimir.updateExtensions(component, url)
	}

	params := url.Query()
	for _, values := range params {
		for _, value := range values {
			mimir.updateExtensions(value, url)
		}
	}
}

func (mimir *MimirApp) updateExtensions(file string, url url.URL) {
	ext := filepath.Ext(file)
	if ext == "" || ext == "." {
		return
	}

	extension := strings.TrimLeft(ext, ".")
	exists := shared.SliceContains(mimir.Extensions, ext)

	if !exists {
		mimir.Extensions = append(mimir.Extensions, extension)
	}
}

func (mimir *MimirApp) displayMsg(item string) {
	fmt.Printf("%s[x]%s %s\n", green, reset, item)
}

func (mimir *MimirApp) displaySuccess(text string) {
	fmt.Printf("%s[x] %s%s\n", green, text, reset)
}

func (mimir *MimirApp) displayWarning(text string) {
	fmt.Printf("%s[x] WARN: %s%s\n", yellow, text, reset)
}

func (mimir *MimirApp) displayError(text string) {
	fmt.Printf("%s[x] ERR: %s%s\n", red, text, reset)
}
