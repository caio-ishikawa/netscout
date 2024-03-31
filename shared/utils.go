package shared

import (
	"net/url"
	"strings"
)

// Checks if string exists in a slice
func SliceContains(slice []string, toCheck string) bool {
	for _, x := range slice {
		if x == toCheck {
			return true
		}
	}

	return false
}

func SliceContainsURL(slice []url.URL, toCheck url.URL) bool {
	for _, x := range slice {
		if x == toCheck {
			return true
		}
	}

	return false
}

func RemoveScheme(targetUrl url.URL) string {
	scheme := targetUrl.Scheme + "://"
	filteredScheme := strings.Replace(targetUrl.String(), scheme, "", 1)

	return strings.Replace(filteredScheme, "/", "", 1)
}
