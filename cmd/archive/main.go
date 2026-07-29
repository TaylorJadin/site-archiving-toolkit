package main

import (
	"fmt"
	"os"
	"path/filepath"

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
		default:
			fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
			printHelp()
			os.Exit(1)
		}
	}

	cfg, err := config.Load(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	if err := tui.Run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Print(`Site Archiving Toolkit — Webrecorder archives via Browsertrix Crawler

Usage:
  archive              Launch the interactive TUI to archive one or more URLs
  archive quit         Stop any running crawl
  archive server start Start a local preview server at http://localhost
  archive server stop  Stop the local preview server
  archive help         Show this help

TUI keys:
  enter         Start archiving
  shift+enter   Add another URL on a new line
  s             Skip the current URL
  c             Cancel the entire archive run
  q / esc       Quit

Override crawl settings in the .env file (created automatically on first run).
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

	// Fall back to executable directory
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
