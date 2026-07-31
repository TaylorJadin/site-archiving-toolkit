package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"

	"github.com/TaylorJadin/site-archiving-toolkit/internal/archive"
	"github.com/TaylorJadin/site-archiving-toolkit/internal/tui"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// run dispatches a single archive invocation. Archives, .env and session state
// all live in the current working directory.
func run(args []string) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}

	if len(args) > 0 {
		switch args[0] {
		case "quit", "quit-crawlers":
			msg, err := archive.QuitCrawlers(root)
			if err != nil {
				return err
			}
			fmt.Println(msg)
			return nil
		case "session-run":
			return archive.RunSession(root)
		case "server":
			return server(root, args[1:])
		case "resume":
			return resume(root, args[1:])
		case "help", "-h", "--help":
			fmt.Print(usage)
			return nil
		}
	}

	if len(args) > 0 && !strings.HasPrefix(args[0], "-") && !archive.ValidURL(args[0]) {
		return fmt.Errorf("unknown command %q; URLs must start with http:// or https:// (see archive help)", args[0])
	}

	urls, background := archive.CollectURLsFromArgs(args)
	stdinURLs, err := archive.CollectURLsFromStdin()
	if err != nil {
		return fmt.Errorf("read URLs from stdin: %w", err)
	}
	return start(root, append(urls, stdinURLs...), background)
}

// start archives urls, choosing between the TUI, a detached runner, and
// streaming logs to stdout based on the flags and whether we own a terminal.
func start(root string, urls []string, background bool) error {
	cfg, err := archive.LoadConfig(root)
	if err != nil {
		return err
	}

	switch {
	case len(urls) > 0 && (background || cfg.BackgroundModeDefault):
		session := archive.NewSession(urls)
		if err := archive.StartSessionRunner(root, session); err != nil {
			return err
		}
		fmt.Printf("Crawl started in background (%d URLs). Run archive to reattach.\n", len(session.Jobs))
		return nil
	case len(urls) > 0 && !interactive():
		return archive.RunHeadless(root, urls, os.Stdout)
	default:
		return tui.Run(cfg, urls)
	}
}

func resume(root string, args []string) error {
	session, err := archive.LoadSession(root)
	if err != nil {
		return err
	}
	if !session.CanResume() {
		return errors.New("no incomplete session to resume")
	}
	_, background := archive.CollectURLsFromArgs(args)
	return start(root, session.ResumableURLs(), background)
}

func server(root string, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: archive server <start|stop>")
	}
	switch args[0] {
	case "start":
		if err := archive.StartServer(root); err != nil {
			return err
		}
		fmt.Println("Preview server started at http://localhost")
	case "stop":
		if err := archive.StopServer(); err != nil {
			return err
		}
		fmt.Println("Preview server stopped.")
	default:
		return errors.New("usage: archive server <start|stop>")
	}
	return nil
}

// interactive reports whether we can drive a full-screen TUI.
func interactive() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) && isatty.IsTerminal(os.Stdout.Fd())
}

const usage = `Site Archiving Toolkit — Webrecorder archives via Browsertrix Crawler

Archives, .env and session state live in the directory you run this from.

Usage:
  archive                              Launch the interactive TUI
  archive <url> [url...]               Start crawling URL(s) immediately
  archive --background <url> [url...]  Start crawl(s) detached and exit
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
`
