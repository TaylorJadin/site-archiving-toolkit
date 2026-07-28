package archive

import (
	"regexp"
	"strings"
)

var (
	unsafeChars  = regexp.MustCompile(`[<>:"/\\|?* ]`)
	multiDash    = regexp.MustCompile(`-+`)
	controlChars = regexp.MustCompile(`[\x00-\x1f]`)
)

// NormalizeURL converts a URL into a filesystem-safe name.
func NormalizeURL(url string) string {
	u := url
	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "http://")
	u = strings.TrimSuffix(u, "/")
	u = unsafeChars.ReplaceAllString(u, "-")
	u = controlChars.ReplaceAllString(u, "")
	u = multiDash.ReplaceAllString(u, "-")
	u = strings.Trim(u, "-")
	return u
}

// ValidURL reports whether the URL starts with http:// or https://.
func ValidURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}
