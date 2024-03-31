package osint

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/caio-ishikawa/netscout/shared"
)

const SERP_API_URL = "https://serpapi.com/search.json"

type GoogleResults struct {
	OrganicResults []OrganicResult `json:"organic_results"`
}

type OrganicResult struct {
	Title string `json:"title"`
	Link  string `json:"link"`
}

type SerpClient struct {
	url url.URL
}

func NewSerpClient(apiKey string) (SerpClient, error) {
	u, err := url.Parse(SERP_API_URL)
	if err != nil {
		return SerpClient{}, err
	}

	query := u.Query()
	query.Add("api_key", apiKey)
	query.Add("engine", "google")
	query.Add("google_domain", "google.com")
	query.Add("gl", "us")
	query.Add("hl", "en")

	u.RawQuery = query.Encode()

	return SerpClient{*u}, nil
}

func (serp *SerpClient) SearchGoogle(queryStr string) (GoogleResults, error) {
	query := serp.url.Query()
	query.Add("q", queryStr)
	serp.url.RawQuery = query.Encode()

	resp, err := http.Get(serp.url.String())
	if err != nil {
		return GoogleResults{}, err
	}
	defer resp.Body.Close()

	var res GoogleResults
	if err = json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return GoogleResults{}, err
	}

	return res, nil
}

// Returns query string for searching for filetypes given a URL
func GenerateFiletypeQuery(targetUrl url.URL, extensions []string) string {
	domain := shared.RemoveScheme(targetUrl)
	queryStr := fmt.Sprintf("site:%s ", domain)

	for i, ext := range extensions {
		if i == 0 {
			queryStr = queryStr + "filetype:" + ext
		} else {
			queryStr = queryStr + " OR filetype:" + ext
		}
	}

	return queryStr
}
