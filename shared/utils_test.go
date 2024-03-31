package shared

import (
	"net/url"
	"testing"
)

func TestRemoveScheme(t *testing.T) {
	u, _ := url.Parse("http://localhost:80")
	u1, _ := url.Parse("https://test-crawler.com")

	cases := []struct {
		url      url.URL
		expected string
	}{
		{*u, "localhost:80"},
		{*u1, "test-crawler.com"},
	}

	for _, testCase := range cases {
		res := RemoveScheme(testCase.url)
		if res != testCase.expected {
			t.Errorf("removeScheme expected %s; got %s", testCase.expected, res)
		}
	}
}
