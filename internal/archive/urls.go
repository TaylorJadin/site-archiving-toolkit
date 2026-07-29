package archive

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// ParseURLs splits raw text on whitespace into URL tokens.
func ParseURLs(raw string) []string {
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return nil
	}
	return append([]string(nil), fields...)
}

// CollectURLsFromArgs returns non-flag CLI arguments as URLs.
func CollectURLsFromArgs(args []string) (urls []string, background bool) {
	for _, arg := range args {
		switch arg {
		case "--background", "-b":
			background = true
		default:
			if strings.HasPrefix(arg, "-") {
				continue
			}
			urls = append(urls, ParseURLs(arg)...)
		}
	}
	return urls, background
}

// CollectURLsFromStdin reads URLs line-by-line from stdin when piped.
func CollectURLsFromStdin() ([]string, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	if stat.Mode()&os.ModeCharDevice != 0 {
		return nil, nil
	}
	return CollectURLsFromReader(os.Stdin)
}

// CollectURLsFromReader reads URLs from a reader, one or more per line.
func CollectURLsFromReader(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)
	var urls []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, ParseURLs(line)...)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return urls, nil
}
