package archive

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"strings"
)

var (
	unsafeChars  = regexp.MustCompile(`[<>:"/\\|?* ]`)
	controlChars = regexp.MustCompile(`[\x00-\x1f]`)
	multiDash    = regexp.MustCompile(`-+`)
)

// NormalizeURL converts a URL into a filesystem-safe name.
func NormalizeURL(url string) string {
	u := strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "http://")
	u = strings.TrimSuffix(u, "/")
	u = unsafeChars.ReplaceAllString(u, "-")
	u = controlChars.ReplaceAllString(u, "")
	u = multiDash.ReplaceAllString(u, "-")
	return strings.Trim(u, "-")
}

// ValidURL reports whether the URL starts with http:// or https://.
func ValidURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

// ParseURLs splits raw text on whitespace into URL tokens.
func ParseURLs(raw string) []string {
	return strings.Fields(raw)
}

// CollectURLsFromArgs returns the non-flag CLI arguments as URLs.
func CollectURLsFromArgs(args []string) (urls []string, background bool) {
	for _, arg := range args {
		switch {
		case arg == "--background" || arg == "-b":
			background = true
		case strings.HasPrefix(arg, "-"):
			continue
		default:
			urls = append(urls, ParseURLs(arg)...)
		}
	}
	return urls, background
}

// CollectURLsFromStdin reads URLs from stdin when it is piped, ignoring blank
// lines and # comments. It returns nil when stdin is a terminal.
func CollectURLsFromStdin() ([]string, error) {
	stat, err := os.Stdin.Stat()
	if err != nil || stat.Mode()&os.ModeCharDevice != 0 {
		return nil, err
	}
	return collectURLs(os.Stdin)
}

func collectURLs(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)
	var urls []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, ParseURLs(line)...)
	}
	return urls, scanner.Err()
}
