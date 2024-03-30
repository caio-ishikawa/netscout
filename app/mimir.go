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
	"github.com/theckman/yacspin"
)

const (
	bold   = "\033[1m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	reset  = "\033[0m"
)

type MimirApp struct {
	spinner    yacspin.Spinner
	outputFile *os.File
	settings   Settings
	Extensions []string
}

func NewApp(settings Settings) (MimirApp, error) {
	cfg := yacspin.Config{
		Frequency:       100 * time.Millisecond,
		CharSet:         yacspin.CharSets[69],
		Suffix:          " crawling", // Starting process
		SuffixAutoColon: true,
		Message:         settings.SeedUrl.String(),
		StopCharacter:   "[✓]",
		StopColors:      []string{"fgGreen"},
	}

	spinner, err := yacspin.New(cfg)
	if err != nil {
		panic(err)
	}

	return MimirApp{
		spinner:    *spinner,
		outputFile: nil,
		settings:   settings,
		Extensions: []string{},
	}, nil
}

func (mimir *MimirApp) Start() {
	if mimir.settings.Output {
		mimir.createOutputFile()
	}

	// 1- binaryedge subdomain scan && add domains to list
	// 1- (MAYBE) wayback machine scan for subdomains?
	// 2- crawl seed URL + any found subdomains && save file extensions
	// 3- use serp api to try and find files on google

	mimir.crawl(
		false,                  // headless mode
		true,                   // lockHost (only crawls url with host that matches target host)
		mimir.settings.SeedUrl, // target url
		mimir.settings.Depth,
	)

	mimir.displayCrawlerResults()

	serpClient, err := osint.NewSerpClient(mimir.settings.SerpApiKey)
	if err != nil {
		return
	}

	queryStr := osint.GenerateFiletypeQuery(mimir.settings.SeedUrl, mimir.Extensions)
	results, err := serpClient.SearchGoogle(queryStr)
	for _, i := range results.OrganicResults {
		mimir.displayMsg(i.Link)
	}
}

func (mimir *MimirApp) createOutputFile() {
	file, err := os.Create("recon.mimir")
	if err != nil {
		mimir.displayError("failed to create output file - proceeding with scan")
		return
	}

	mimir.outputFile = file
}

func (mimir *MimirApp) crawl(headless bool, lockHost bool, seedUrl url.URL, depth int) {
	comms := shared.NewCommsChannels()

	crawler := osint.NewCrawler(
		headless,
		lockHost,
		seedUrl,
		mimir.settings.SeedUrl.Host,
		depth,
		comms,
	)

	mimir.displayWarning("crawling")

	go mimir.spinner.Start()

	go crawler.Crawl(0)

	mimir.handleComms(
		comms.DataChan,
		comms.UpdateChan,
		comms.WarningChan,
		comms.DoneChan,
	)

	mimir.spinner.Suffix(" Crawled")

	mimir.spinner.Stop()
}

func (mimir *MimirApp) displayCrawlerResults() {
	crawlOutput := fmt.Sprintf("Found %v file extensions", len(mimir.Extensions))
	mimir.displayMsg(crawlOutput)

	scanMsg := "Scanning for"
	for _, ext := range mimir.Extensions {
		scanMsg = scanMsg + " " + ext
	}

	mimir.displayMsg(scanMsg)
}

// Method responsible for consuming incoming messages until process is done.
func (mimir *MimirApp) handleComms(
	dataChan chan shared.ScannedItem,
	updateChan chan string,
	warningChan chan string,
	doneChan chan struct{},
) {
	var wg sync.WaitGroup
	for {
		select {
		case msg := <-dataChan:
			if mimir.settings.Output {
				mimir.outputFile.Write([]byte(msg.Url.String() + "\n"))
			}
			if !mimir.settings.Verbose && msg.Relevance == shared.Low {
				continue
			}

			wg.Add(1)
			go mimir.CollectFiletypes(msg.Url, &wg)
			mimir.displayMsg(msg.Url.String())
		case msg := <-updateChan:
			mimir.spinner.Message(msg)
		case msg := <-warningChan:
			mimir.displayWarning(msg)
		case <-doneChan:
			wg.Wait()
			return
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

// TODO: add to shared
// WARN: ugly but works
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
	fmt.Print("\033[1A\n\033[2K")
	fmt.Printf("%s[x]%s %s\n", green, reset, item)
}

func (mimir *MimirApp) displaySucess(text string) {
	fmt.Print("\033[1A\n\033[2K")
	fmt.Printf("%s[x] %s%s\n", green, text, reset)
}

func (mimir *MimirApp) displayWarning(text string) {
	fmt.Print("\033[1A\n\033[2K")
	fmt.Printf("%s[x] WARN: %s%s\n", yellow, text, reset)
}

func (mimir *MimirApp) displayError(text string) {
	fmt.Print("\033[1A\n\033[2K")
	fmt.Printf("%s[x] ERR: %s%s\n", red, text, reset)
}
