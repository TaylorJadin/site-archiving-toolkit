package archive

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"

	"github.com/TaylorJadin/site-archiving-toolkit/resources"
)

// Config holds toolkit settings loaded from .env.
type Config struct {
	BrowsertrixParameters       string
	BrowsertrixRedirectTemplate bool
	CreateWebrecorderZip        bool
	SkipExistingCrawls          bool
	BackgroundModeDefault       bool
	RootDir                     string
}

// LoadConfig reads configuration from .env in rootDir, creating it if missing.
func LoadConfig(rootDir string) (*Config, error) {
	envPath := filepath.Join(rootDir, ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		if err := os.WriteFile(envPath, resources.EnvExample, 0o644); err != nil {
			return nil, fmt.Errorf("create .env: %w", err)
		}
	}

	vals, err := godotenv.Read(envPath)
	if err != nil {
		return nil, fmt.Errorf("read .env: %w", err)
	}

	return &Config{
		BrowsertrixParameters:       get(vals, "browsertrix_parameters", "--workers 4 --text"),
		BrowsertrixRedirectTemplate: truthy(get(vals, "browsertrix_redirect_template", "FALSE")),
		CreateWebrecorderZip:        truthy(get(vals, "create_webrecorder_zip", "TRUE")),
		SkipExistingCrawls:          truthy(get(vals, "skip_existing_crawls", "FALSE")),
		BackgroundModeDefault:       truthy(get(vals, "background_mode_default", "FALSE")),
		RootDir:                     rootDir,
	}, nil
}

// crawlEnv returns the crawl settings as environment variables for webrecorder.sh.
func (c *Config) crawlEnv() []string {
	return []string{
		"browsertrix_parameters=" + c.BrowsertrixParameters,
		"browsertrix_redirect_template=" + boolStr(c.BrowsertrixRedirectTemplate),
		"create_webrecorder_zip=" + boolStr(c.CreateWebrecorderZip),
	}
}

func get(m map[string]string, key, fallback string) string {
	if v, ok := m[key]; ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

func truthy(v string) bool {
	switch strings.ToUpper(strings.TrimSpace(v)) {
	case "TRUE", "1", "YES", "ON":
		return true
	default:
		return false
	}
}

func boolStr(v bool) string {
	if v {
		return "TRUE"
	}
	return "FALSE"
}
