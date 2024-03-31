package osint

import (
	"net/url"
	"testing"
)

func TestParsePath(t *testing.T) {
	u1, _ := url.Parse("https://localhost")
	u2, _ := url.Parse("https://localhost/test_endpoint")

	cases := []struct {
		urlStr   string
		host     string
		scheme   string
		expected url.URL
		err      bool
	}{
		{
			urlStr:   "https://localhost",
			host:     "localhost",
			scheme:   "https",
			expected: *u1,
			err:      false,
		},
		{
			urlStr:   "/test_endpoint",
			host:     "localhost",
			scheme:   "https",
			expected: *u2,
			err:      false,
		},
		{
			urlStr:   ":",
			host:     "localhost",
			scheme:   "https",
			expected: url.URL{},
			err:      true,
		},
	}

	for _, tc := range cases {
		res, err := parsePath(tc.urlStr, tc.host, tc.scheme)
		if (err != nil) != tc.err {
			t.Errorf("parsePath returned unexpected error: %s", err.Error())
			continue
		}

		if res != tc.expected {
			t.Errorf("parsePath expected %v; got %v", tc.expected, res)
		}
	}
}
