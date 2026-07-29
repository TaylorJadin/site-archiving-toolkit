package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
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

const envExample = `# Browsertrix / Webrecorder crawl options
# Make sure to add quotes at the start and end of the parameters
browsertrix_parameters="--workers 4 --text"

# Create redirect.php and .htaccess files for browsertrix archives (TRUE or FALSE)
# These two files handle redirects automatically if the archive files are placed
# at the webroot of an Apache server with .htaccess rules enabled.
browsertrix_redirect_template=FALSE

# Create zip files for easy download (TRUE or FALSE)
create_webrecorder_zip=TRUE

# Skip crawls for sites that already exist in the crawl directory based on the
# normalized URL (TRUE or FALSE)
skip_existing_crawls=FALSE

# Start crawls in the background by default, detaching from the TUI (TRUE or FALSE)
background_mode_default=FALSE
`

// EnsureEnv creates .env from defaults when missing.
func EnsureEnv(rootDir string) error {
	envPath := filepath.Join(rootDir, ".env")
	if _, err := os.Stat(envPath); err == nil {
		return nil
	}
	example := filepath.Join(rootDir, "resources", "env.example")
	data, err := os.ReadFile(example)
	if err != nil {
		data = []byte(envExample)
	}
	return os.WriteFile(envPath, data, 0o644)
}

// Load reads configuration from .env in rootDir.
func Load(rootDir string) (*Config, error) {
	if err := EnsureEnv(rootDir); err != nil {
		return nil, fmt.Errorf("ensure .env: %w", err)
	}

	envPath := filepath.Join(rootDir, ".env")
	vals, err := godotenv.Read(envPath)
	if err != nil {
		return nil, fmt.Errorf("read .env: %w", err)
	}

	cfg := &Config{
		BrowsertrixParameters:       get(vals, "browsertrix_parameters", "--workers 4 --text"),
		BrowsertrixRedirectTemplate: truthy(get(vals, "browsertrix_redirect_template", "FALSE")),
		CreateWebrecorderZip:        truthy(get(vals, "create_webrecorder_zip", "TRUE")),
		SkipExistingCrawls:          truthy(get(vals, "skip_existing_crawls", "FALSE")),
		BackgroundModeDefault:       truthy(get(vals, "background_mode_default", "FALSE")),
		RootDir:                     rootDir,
	}
	return cfg, nil
}

// WriteArchiveINI writes a shell-sourceable archive.ini for the crawler container.
func (c *Config) WriteArchiveINI(path string) error {
	redirect := boolStr(c.BrowsertrixRedirectTemplate)
	zip := boolStr(c.CreateWebrecorderZip)
	content := fmt.Sprintf(
		"browsertrix_parameters=%q\nbrowsertrix_redirect_template=%s\ncreate_webrecorder_zip=%s\n",
		c.BrowsertrixParameters,
		redirect,
		zip,
	)
	return os.WriteFile(path, []byte(content), 0o644)
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
