package osint

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Errors
const (
	subdomainQueryFailed = "BinaryEdge subdomains query yielded non-successful status code"
)

type BinaryEdgeClient struct {
	apiKey  string
	baseUrl string
}

type BinaryEdgeSubdomains struct {
	Subdomains []string `json:"events"`
}

func NewBinaryEdgeClient(apiKey string) BinaryEdgeClient {
	return BinaryEdgeClient{
		apiKey:  apiKey,
		baseUrl: "https://api.binaryedge.io/v2",
	}
}

// Queries for subdomains using the BinaryEdge API:
// https://docs.binaryedge.io/api-v2/#domains
func (client *BinaryEdgeClient) QuerySubdomains(targetUrl url.URL) (BinaryEdgeSubdomains, error) {
	targetDomain := removeScheme(targetUrl)
	url := fmt.Sprintf("https://api.binaryedge.io/v2/query/domains/subdomain/%s", targetDomain)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return BinaryEdgeSubdomains{}, err
	}

	req.Header.Set("X-Key", client.apiKey)

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return BinaryEdgeSubdomains{}, err
	}

	if resp.StatusCode != 200 {
		return BinaryEdgeSubdomains{}, fmt.Errorf(subdomainQueryFailed)
	}

	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return BinaryEdgeSubdomains{}, err
	}

	var subdomains BinaryEdgeSubdomains
	err = json.Unmarshal(bytes, &subdomains)
	if err != nil {
		return BinaryEdgeSubdomains{}, err
	}

	return subdomains, nil
}
