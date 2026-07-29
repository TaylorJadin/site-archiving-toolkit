# Site Archiving Toolkit

## What is this thing?

The Site Archiving Toolkit makes [Webrecorder](https://webrecorder.net) / WACZ archives of websites using [Browsertrix Crawler](https://github.com/webrecorder/browsertrix-crawler) in Docker. A Go terminal UI (built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss)) lets you queue multiple URLs, watch crawl progress, and skip or cancel jobs.

Check out this video to see what it does and how to use it:

[Site Archiving Toolkit - reclaim.tv](https://archive.reclaim.tv/w/qYeNBzUdDWDxWi8pFSLNB1)

Example archives:

[archiving.ca.reclaim.cloud](https://archiving.ca.reclaim.cloud/)

[digciz.jadin.me](https://digciz.jadin.me)

### Features

- Crawl an entire site / domain for offline browsing or preservation (WACZ + ReplayWeb.page)
- Interactive TUI for entering multiple URLs
- Live crawl progress (`Site 1/10`) with skip / cancel controls and a scrolling log panel
- Preview archived pages using a local web server
- Automatically creates zip files for easy download/upload
- Override crawl settings using the `.env` file (delete it to return to defaults)

## Requirements

- [Docker](https://www.docker.com/) (Docker Desktop on macOS/Windows)
- A release binary, or [Go 1.24+](https://go.dev/dl/) to build from source

## How do I use it?

### On Reclaim Cloud

Install the Site Archiving Toolkit from the Marketplace, open a terminal, and run:

```bash
./archive
```

Enter one or more URLs in the TUI (`shift+enter` / `ctrl+j` for additional lines, or paste a multiline list), then press `enter` to start. While a crawl is running:

- `s` — skip the current URL
- `c` — cancel the entire archive run

Stop a runaway crawl from another terminal:

```bash
./archive quit
```

### Previewing and downloading archives

Visit the environment URL to browse completed and in-progress crawls, or start a local preview server:

```bash
./archive server start   # http://localhost
./archive server stop
```

Crawl output lives in the `crawls` directory (on Reclaim Cloud: `/root/site-archiving-toolkit/crawls`).

### On your own computer

1. Install and launch [Docker Desktop](https://www.docker.com/products/docker-desktop/)
2. Download the latest release for your OS from the [releases page](https://github.com/TaylorJadin/site-archiving-toolkit/releases)
3. Unzip somewhere convenient and open a terminal in that folder
4. Run the binary:

```bash
# macOS / Linux
./archive

# Windows
.\archive.exe
```

### Build from source

```bash
go build -o archive ./cmd/archive
./archive
```

### Commands

| Command | Description |
| --- | --- |
| `archive` | Launch the interactive TUI |
| `archive quit` | Stop any running Browsertrix crawl |
| `archive server start` | Start the local preview server |
| `archive server stop` | Stop the local preview server |
| `archive help` | Show help |

### Configuration

On first run a `.env` file is created from `resources/env.example`. Useful options:

- `browsertrix_parameters` — extra flags for Browsertrix Crawler
- `create_webrecorder_zip` — zip each archive for download (`TRUE`/`FALSE`)
- `browsertrix_redirect_template` — include Apache redirect helpers
- `skip_existing_crawls` — skip URLs that already have a completed crawl
