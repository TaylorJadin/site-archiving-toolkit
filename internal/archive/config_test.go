package archive

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadConfigCreatesEnvWithDefaults(t *testing.T) {
	dir := t.TempDir()

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".env")); err != nil {
		t.Fatalf(".env was not created: %v", err)
	}

	want := &Config{
		BrowsertrixParameters: "--workers 4 --text",
		CreateWebrecorderZip:  true,
		RootDir:               dir,
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("cfg = %+v, want %+v", cfg, want)
	}

	// A second load must reuse the file rather than reset it.
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("skip_existing_crawls=yes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err = LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.SkipExistingCrawls {
		t.Error("expected skip_existing_crawls=yes to be honoured")
	}
	if cfg.BrowsertrixParameters != "--workers 4 --text" {
		t.Errorf("missing key should fall back to the default, got %q", cfg.BrowsertrixParameters)
	}
}

func TestTruthy(t *testing.T) {
	for _, v := range []string{"TRUE", "true", " Yes ", "1", "on"} {
		if !truthy(v) {
			t.Errorf("truthy(%q) = false", v)
		}
	}
	for _, v := range []string{"FALSE", "no", "", "0", "maybe"} {
		if truthy(v) {
			t.Errorf("truthy(%q) = true", v)
		}
	}
}

func TestCrawlEnv(t *testing.T) {
	cfg := &Config{
		BrowsertrixParameters:       "--workers 2",
		BrowsertrixRedirectTemplate: true,
		CreateWebrecorderZip:        false,
	}
	want := []string{
		"browsertrix_parameters=--workers 2",
		"browsertrix_redirect_template=TRUE",
		"create_webrecorder_zip=FALSE",
	}
	if got := cfg.crawlEnv(); !reflect.DeepEqual(got, want) {
		t.Errorf("crawlEnv() = %q, want %q", got, want)
	}
}
