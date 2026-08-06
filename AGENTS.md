# AGENTS.md

## Cursor Cloud specific instructions

This repo (branch `v2`) is a Go [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI that orchestrates [Browsertrix Crawler](https://github.com/webrecorder/browsertrix-crawler) in Docker to produce Webrecorder/WACZ archives. See `README.md` for user-facing commands; the notes below are the non-obvious bits for developing/running it here.

### Layout
- `cmd/archive` — CLI entrypoint (subcommands: `quit`, `server start|stop`, `resume`, `help`; URLs as args or stdin start immediately; no args launches the TUI).
- `internal/tui` — Bubble Tea UI (reattach/detach/resume; no alt-screen).
- `internal/archive` — config loading, Docker orchestration, session state, and the session runner.
- `resources/` — files embedded into the binary via `resources/embed.go`: the crawler `Dockerfile.webrecorder` build context (`webrecorder.sh`, `index.html`, `redirect.php`, `htaccess`) and the default `env.example`.

### Build / test / run
- Makefile targets: `make build`, `make run` (via `go run`; pass args with `ARGS=...`, e.g. `make run ARGS='https://example.com'`), `make test`, `make vet`, `make fmt`, `make releases`, `make clean`.
- Build: `go build -o archive ./cmd/archive` (or `make build`)
- Test: `go test ./...` — Vet: `go vet ./...` — Format: `gofmt -l .`
- Run: `./archive` (interactive TUI), `./archive <url>...`, `./archive --background <url>`, `./archive resume`, `./archive quit`, `./archive server start` / `./archive server stop`.

### Non-obvious caveats
- Docker is required. The daemon is **not** started automatically in this environment — run `sudo dockerd > /tmp/dockerd.log 2>&1 &` (in a tmux session) and wait a few seconds. Storage driver is `fuse-overlayfs`; the `ubuntu` user is already in the `docker` group.
- On the first archive run the toolkit builds the Docker image `site-archiving-toolkit-webrecorder`. The build context is written to a temp dir from the embedded `resources/` files. It pulls `webrecorder/browsertrix-crawler:latest` and `apt install`s packages, so it needs network egress; the first build takes longer, later runs are cached.
- Crawling also needs egress to the target site and to `cdn.jsdelivr.net` (ReplayWeb.page assets are fetched inside the container).
- The TUI needs a real interactive TTY — you can't drive it through a plain piped shell. `archive <url>` falls back to streaming logs to stdout when stdin or stdout is not a terminal, so `echo https://example.com | ./archive` works non-interactively. Subcommands (`quit`, `server ...`, `help`, `resume`) work from a normal shell.
- Crawls run in a background `archive session-run` subprocess that owns the session file. Press `d` to detach, `q` to quit (also stops the Docker container). `./archive` reattaches to an active session; incomplete sessions can be resumed with `r` or `./archive resume`.
- Everything is relative to the current working directory: `crawls/`, `.env`, and the session files under `crawls/`. Run `./archive` from the directory you want the archives in.
- Container output under `crawls/` is written as root, so cleaning up needs `sudo rm -rf crawls`.
- `.env` (created from the embedded default on first run) and everything under `crawls/` are gitignored runtime artifacts — do not commit them. Delete `.env` to reset config to defaults.
- `./archive server start` binds host port 80 (a plain `httpd` container named `site-archiving-toolkit-preview`).
- Quick end-to-end sanity check: `echo https://example.com | ./archive` — a small single page that crawls in a few seconds and produces a `.wacz` (and a `.zip`) under `crawls/<timestamp>-example.com/`.
