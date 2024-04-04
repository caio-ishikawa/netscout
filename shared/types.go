package shared

import (
	"fmt"
	"net/url"
)

type Source string

const (
	Axfr         Source = "DNS_AXFR"
	Crawler      Source = "CRAWLER"
	BinaryEdge   Source = "BINARY_EDGE"
	Serp         Source = "SERP"
	ShortenedUrl Source = "SHORTENED_URL"
)

type ScannedItem struct {
	Url    url.URL
	Source Source
}

func (si *ScannedItem) Format() string {
	return fmt.Sprintf("[%s] %s\n", si.Source, si.Url.String())
}

type CommsChannels struct {
	DataChan          chan ScannedItem
	WarningChan       chan string
	CrawlDoneChan     chan struct{}
	ShortenedDoneChan chan struct{}
}

func NewCommsChannels() CommsChannels {
	return CommsChannels{
		DataChan:          make(chan ScannedItem),
		WarningChan:       make(chan string),
		CrawlDoneChan:     make(chan struct{}),
		ShortenedDoneChan: make(chan struct{}),
	}
}
