# Site Archiving Toolkit

## What is this thing?

The Site Archiving Toolkit makes [Webrecorder](https://webrecorder.net) / WACZ archives of websites using [Browsertrix Crawler](https://github.com/webrecorder/browsertrix-crawler) in Docker. A Go terminal UI (built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss)) lets you queue multiple URLs, watch crawl progress, skip a URL, detach, or quit.

Check out this video to see what it does and how to use it:

[Site Archiving Toolkit - reclaim.tv](https://archive.reclaim.tv/w/qYeNBzUdDWDxWi8pFSLNB1)

Example archives:

[archiving.ca.reclaim.cloud](https://archiving.ca.reclaim.cloud/)

[digciz.jadin.me](https://digciz.jadin.me)

### Features

- Crawl an entire site / domain for offline browsing or preservation (WACZ + ReplayWeb.page)
- Interactive TUI for entering multiple URLs
- Live crawl progress (`Site 1/10`) with skip-url / detach / quit controls and a scrolling log panel
- Preview archived pages using a local web server
- Automatically creates zip files for easy download/upload
- Override crawl settings using the `.env` file (delete it to return to defaults)

## Requirements

- [Docker](https://www.docker.com/) (Docker Desktop on macOS/Windows)
- A release binary, or [Go 1.25+](https://go.dev/dl/) to build from source

The binary is self-contained: the crawler image and its helper scripts are built
into it. Archives (`crawls/`), settings (`.env`) and session state all live in
the directory you run `archive` from.

## How do I use it?

### On Reclaim Cloud

Install the Site Archiving Toolkit from the Marketplace, open a terminal, and run:

```bash
./archive
```

Enter one or more URLs in the TUI (separated by spaces or newlines; `shift+enter` / `ctrl+j` also insert a new line), then press `enter` to start. You can also pass URLs on the command line or via stdin:

```bash
./archive https://example.com https://example.org
echo "https://example.com" | ./archive
```

While a crawl is running:

- `s` — skip the current URL
- `d` — detach: crawl continues in the background; run `./archive` again to reattach
- `q` — quit (stops the crawl and the Docker container)

If the last session did not finish, `./archive` offers **resume** (`r` on the input screen, or `./archive resume`).

Start a crawl detached and return to the shell right away with `--background`, or
set `background_mode_default=TRUE` in `.env` to make that the default.

```bash
./archive --background https://example.com
```

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
| `archive <url> [url...]` | Start crawling the given URL(s) immediately |
| `archive --background <url>` | Start a crawl detached and exit |
| `archive resume` | Resume the last incomplete session |
| `archive quit` | Stop any running Browsertrix crawl |
| `archive server start` | Start the local preview server |
| `archive server stop` | Stop the local preview server |
| `archive help` | Show help |

When output is piped or redirected, `archive <url>` skips the TUI and streams the
crawl log to stdout instead.

### Configuration

On first run a `.env` file is created with the defaults. Useful options:

- `browsertrix_parameters` — extra flags for Browsertrix Crawler
- `create_webrecorder_zip` — zip each archive for download (`TRUE`/`FALSE`)
- `browsertrix_redirect_template` — include Apache redirect helpers
- `skip_existing_crawls` — skip URLs that already have a completed crawl
- `background_mode_default` — start new crawls detached
