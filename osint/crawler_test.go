package osint

import (
	"github.com/caio-ishikawa/netscout/shared"
	"net/url"
	"testing"
	"time"
)

// THIS NEEDS THE DVWA DOCKER CONTAINER TO BE RUNNING ON PORT 80
func TestCrawlerAllHosts(t *testing.T) {
	seed, _ := url.Parse("http://localhost")
	comms := shared.NewCommsChannels()

	crawler := NewCrawler(false, *seed, []url.URL{*seed}, 5, comms)

	go crawler.Crawl(0)

	receivedData := 0
	expectedData := 44

	receivedWarning := 0
	expectedWarning := 0

	for {
		select {
		case <-comms.DataChan:
			receivedData++
		case <-comms.WarningChan:
			receivedWarning++
		case <-comms.DoneChan:
			if receivedData != expectedData {
				t.Errorf("crawl expected %v; got %v", expectedData, receivedData)
				t.Fail()
			}
			if receivedWarning != expectedWarning {
				t.Errorf("crawl expected %v; got %v", expectedWarning, receivedWarning)
				t.Fail()
			}
			return
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

}
