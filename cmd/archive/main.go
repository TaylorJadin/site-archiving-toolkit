package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mattn/go-isatty"

	"github.com/TaylorJadin/site-archiving-toolkit/internal/archive"
	"github.com/TaylorJadin/site-archiving-toolkit/internal/config"
	"github.com/TaylorJadin/site-archiving-toolkit/internal/tui"
)

func main() {
	root, err := findRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "quit", "quit-crawlers":
			msg, err := archive.QuitCrawlers()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(msg)
			return
		case "session-run":
			if err := archive.RunSession(root); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			return
		case "resume":
			runTUI(root, tui.Options{Resume: true})
			return
		case "server":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: archive server <start|stop>")
				os.Exit(1)
			}
			switch os.Args[2] {
			case "start":
				if err := archive.StartServer(root); err != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
					os.Exit(1)
				}
				fmt.Println("Preview server started at http://localhost")
			case "stop":
				if err := archive.StopServer(root); err != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
					os.Exit(1)
				}
				fmt.Println("Preview server stopped.")
			default:
				fmt.Fprintln(os.Stderr, "usage: archive server <start|stop>")
				os.Exit(1)
			}
			return
		case "help", "-h", "--help":
			printHelp()
			return
		}

		urls, background := collectStartupURLs(os.Args[1:])
		if len(urls) > 0 {
			runCrawl(root, tui.Options{URLs: urls, Background: background})
			return
		}

		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printHelp()
		os.Exit(1)
	}

	if urls, err := archive.CollectURLsFromStdin(); err != nil {
		fmt.Fprintf(os.Stderr, "error reading URLs from stdin: %v\n", err)
		os.Exit(1)
	} else if len(urls) > 0 {
		runCrawl(root, tui.Options{URLs: urls})
		return
	}

	runTUI(root, tui.Options{})
}

func runCrawl(root string, opts tui.Options) {
	cfg, err := config.Load(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}
	headless := !isatty.IsTerminal(os.Stdout.Fd()) || opts.Background || cfg.BackgroundModeDefault
	if headless {
		if err := archive.RunHeadless(root, opts.URLs); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}
	runTUI(root, opts)
}

func collectStartupURLs(args []string) ([]string, bool) {
	argURLs, background := archive.CollectURLsFromArgs(args)
	stdinURLs, err := archive.CollectURLsFromStdin()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: stdin URLs ignored: %v\n", err)
	}
	return append(argURLs, stdinURLs...), background
}

func runTUI(root string, opts tui.Options) {
	cfg, err := config.Load(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}
	if err := tui.Run(cfg, opts); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Print(`Site Archiving Toolkit — Webrecorder archives via Browsertrix Crawler

Usage:
  archive                              Launch the interactive TUI
  archive <url> [url...]               Start crawling URL(s) immediately
  archive --background <url> [url...]  Start crawl(s) in the background
  archive resume                       Resume the last incomplete session
  archive quit                         Stop any running crawl
  archive server start                 Start a local preview server at http://localhost
  archive server stop                  Stop the local preview server
  archive help                         Show this help

  echo "https://example.com" | archive Start crawl(s) from stdin (one URL per line)

TUI keys:
  enter              Start archiving
  shift+enter/ctrl+j Add another URL on a new line
  (spaces/newlines)  Separate multiple URLs
  s                  Skip the current URL
  d                  Detach — crawl continues in background
  q                  Quit (stops the crawl and Docker container)
  r                  Resume last incomplete session (input screen)

Settings in .env (created automatically on first run):
  background_mode_default=TRUE   Start new crawls detached by default
`)
}

func findRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := cwd
	for {
		dockerfile := filepath.Join(dir, "resources", "Dockerfile.webrecorder")
		if _, err := os.Stat(dockerfile); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	exe, err := os.Executable()
	if err == nil {
		dir = filepath.Dir(exe)
		dockerfile := filepath.Join(dir, "resources", "Dockerfile.webrecorder")
		if _, err := os.Stat(dockerfile); err == nil {
			return dir, nil
		}
	}

	return cwd, nil
}
