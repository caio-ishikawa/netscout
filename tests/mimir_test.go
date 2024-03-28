package tests

// The tests are ran locally and require the the dvwa container to be running locally.
// https://hub.docker.com/r/citizenstig/dvwa/

import (
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/caio-ishikawa/mimir/app"
	"github.com/caio-ishikawa/mimir/osint"
	"github.com/caio-ishikawa/mimir/shared"
)

// Tests crawling scenario with static website
func TestCrawl(t *testing.T) {
	url, _ := url.Parse("http://localhost:80")
	msgChan := make(chan shared.ScannedItem)
	updateChan := make(chan string)
	warningChan := make(chan string)
	doneChan := make(chan struct{})

	crawler := osint.NewCrawler(
		*url,
		5,
		msgChan,
		updateChan,
		warningChan,
		doneChan,
	)

	go crawler.Crawl(0)

	received := 0
	expected := 46
	for {
		select {
		case <-msgChan:
			received++
		case <-doneChan:
			if received != expected {
				t.Errorf("received unexpected amount of messages - %v messages", received)
				t.Fail()
			}
			return
		case <-updateChan:
		case <-warningChan:
			t.Error("received unexpected warning")
			t.Fail()
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

// Tests that CollectFiletypes updates internal map with correct values
func TestCollectFiletypes(t *testing.T) {
	u, _ := url.Parse("http://localhost:80/path/path2/path3.html?param1=test.php&param2=value")

	settings := app.Settings{
		SeedUrl:          *u,
		Depth:            3,
		ThreadCount:      3,
		ReqDelay:         3,
		BinaryEdgeApiKey: "",
	}

	mimir, _ := app.NewApp(settings)

	var wg sync.WaitGroup
	wg.Add(1)
	mimir.CollectFiletypes(*u, &wg)

	if len(mimir.Extensions) != 2 {
		t.Error("unexpected amount of filetypes")
		t.Fail()
	}

	_, phpExists := mimir.Extensions[".php"]
	_, htmlExists := mimir.Extensions[".html"]
	if !phpExists || !htmlExists {
		t.Error("unexpected filetype")
		t.Fail()
	}
}

// Tests that GenerateFiletypeHunterUrl returns url containing correct query
func TestGenerateFiletypeHunterUrl(t *testing.T) {
	hunter, err := osint.NewHunterClient("test-key")
	if err != nil {
		t.Error("could not create hunter client")
		t.Fail()
	}

	targetUrl, _ := url.Parse("http://localhost:80")
	url := hunter.GenerateFiletypeUrl(*targetUrl, []string{".txt", ".pdf", ".php"})

	expectedQuery := "bG9jYWxob3N0OjgwIChmaWxldHlwZToiLnR4dCIgb3IgZmlsZXR5cGU6Ii5wZGYiIG9yIGZpbGV0eXBlOiIucGhwIik%3D"
	expectedUrl := fmt.Sprintf("https://api.hunter.how/search?api-key=test-key&query=%s", expectedQuery)
	if url.String() != expectedUrl {
		t.Error("url did not match expected")
		t.Fail()
	}
}
