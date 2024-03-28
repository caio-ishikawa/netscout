package shared

import (
	"net/url"
)

type Relevance string

const (
	High   Relevance = "HIGH"
	Medium Relevance = "MEDIUM"
	Low    Relevance = "LOW"
)

type Source string

const (
	Crawler Source = "CRAWLER"
)

type ScannedItem struct {
	Url       url.URL
	Relevance Relevance
	Source    Source
}

// TODO: check relevance and source values?
func NewScannedItem(url url.URL, relevance, source string) ScannedItem {
	return ScannedItem{
		Url:       url,
		Relevance: Relevance(relevance),
		Source:    Source(source),
	}
}
