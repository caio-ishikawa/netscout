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
	binaryEdge osint.BinaryEdgeClient
	crawler    osint.Crawler
	spinner    yacspin.Spinner
	outputFile *os.File
	settings   Settings
	Extensions map[string]interface{}
}

func NewApp(settings Settings) (MimirApp, error) {
	var binaryEdge osint.BinaryEdgeClient
	if settings.BinaryEdgeApiKey != "" {
		binaryEdge = osint.NewBinaryEdgeClient(settings.BinaryEdgeApiKey)
	}

	dataChan := make(chan shared.ScannedItem)
	updateChan := make(chan string)
	warningChan := make(chan string)
	doneChan := make(chan struct{})

	crawler := osint.NewCrawler(
		settings.SeedUrl,
		settings.Depth,
		dataChan,
		updateChan,
		warningChan,
		doneChan,
	)

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
		binaryEdge: binaryEdge,
		crawler:    crawler,
		spinner:    *spinner,
		outputFile: nil,
		settings:   settings,
		Extensions: make(map[string]interface{}),
	}, nil
}

func (mimir *MimirApp) Start() {
	if mimir.settings.Output {
		mimir.createOutputFile()
	}

	mimir.crawl()
}

func (mimir *MimirApp) createOutputFile() {
	file, err := os.Create("recon.mimir")
	if err != nil {
		mimir.displayError("failed to create output file - proceeding with scan")
		return
	}

	mimir.outputFile = file
}

// Method responsible for managing all BinaryEdge requests for a scan.
func (mimir *MimirApp) queryBinaryEdge() {
	subdomains, err := mimir.binaryEdge.QuerySubdomains(mimir.settings.SeedUrl)
	if err != nil {
		mimir.displayWarning("error making request to BinaryEdge")
		return
	}

	if len(subdomains.Events) == 0 {
		mimir.displayWarning("no subdomains found on BinaryEdge")
		return
	}

	mimir.displaySucess("BinaryEdge Subdomains:")
	for i := range subdomains.Events {
		mimir.displayWarning(subdomains.Events[i])
	}
}

func (mimir *MimirApp) crawl() {
	go mimir.spinner.Start()

	go mimir.crawler.Crawl(0)

	mimir.handleOperation(
		mimir.crawler.DataChan,
		mimir.crawler.UpdateChan,
		mimir.crawler.WarningChan,
		mimir.crawler.DoneChan,
	)

	mimir.spinner.Suffix(" Crawled")
	mimir.spinner.Stop()

	// Output results
	crawlOutput := fmt.Sprintf("Found %v file extensions", len(mimir.Extensions))
	mimir.displayMsg(crawlOutput)

	scanMsg := "Scanning for"
	for key := range mimir.Extensions {
		scanMsg = scanMsg + " " + key
	}

	mimir.displayMsg(scanMsg)

}

// Method responsible for consuming incoming messages until process is done.
func (mimir *MimirApp) handleOperation(
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

	_, exists := mimir.Extensions[ext]
	if !exists {
		extension := strings.TrimRight(ext, ".")
		mimir.Extensions[extension] = url
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
