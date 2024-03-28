package osint

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type HunterClient struct {
	baseUrl url.URL
}

func NewHunterClient(key string) (HunterClient, error) {
	url, err := url.Parse("https://api.hunter.how/search")
	if err != nil {
		return HunterClient{}, nil
	}

	query := url.Query()
	query.Set("api-key", key)

	url.RawQuery = query.Encode()

	return HunterClient{
		baseUrl: *url,
	}, nil
}

func (hunter *HunterClient) DorkExtensions(targetUrl url.URL, extensions []string) error {
	url := hunter.GenerateFiletypeUrl(targetUrl, extensions)

	resp, err := http.Get(url.String())
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(body))

	return nil
}

// Returns API URL for querying filetypes for a given URL
func (hunter *HunterClient) GenerateFiletypeUrl(targetUrl url.URL, extensions []string) url.URL {
	queryStr := hunter.generateFiletypeQuery(targetUrl, extensions)
	encoded := base64.StdEncoding.EncodeToString([]byte(queryStr))

	url := hunter.baseUrl

	query := url.Query()
	query.Set("query", encoded)

	url.RawQuery = query.Encode()

	return url
}

// Returns query string for searching for filetypes given a URL
func (hunter *HunterClient) generateFiletypeQuery(targetUrl url.URL, extensions []string) string {
	domain := removeScheme(targetUrl)
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

func (hunter *HunterClient) dorkFileNames(names []string) {}
