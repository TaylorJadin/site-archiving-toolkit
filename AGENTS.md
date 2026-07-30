# AGENTS.md

## Cursor Cloud specific instructions

This repo (branch `v2`) is a Go [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI that orchestrates [Browsertrix Crawler](https://github.com/webrecorder/browsertrix-crawler) in Docker to produce Webrecorder/WACZ archives. See `README.md` for user-facing commands; the notes below are the non-obvious bits for developing/running it here.

### Layout
- `cmd/archive` — CLI entrypoint (subcommands: `quit`, `server start|stop`, `resume`, `help`; URLs as args or stdin start immediately; no args launches the TUI).
- `internal/tui` — Bubble Tea UI (reattach/detach/resume; no alt-screen).
- `internal/archive` — Docker orchestration, session runner, Docker/session tests.
- `internal/config` — `.env` loading.
- `resources/` — `Dockerfile.webrecorder`, `webrecorder.sh` (runs inside the container), `docker-compose.yml` (Apache preview server).

### Build / test / run
- Build: `go build -o archive ./cmd/archive`
- Test: `go test ./...` — Vet: `go vet ./...`
- Run: `./archive` (interactive TUI), `./archive <url>...`, `./archive resume`, `./archive quit`, `./archive server start` / `./archive server stop`.

### Non-obvious caveats
- Docker is required and the daemon is already running in this environment (storage driver `fuse-overlayfs`). No `sudo`/daemon start needed.
- On the first archive run the TUI builds the Docker image `site-archiving-toolkit-webrecorder` from `resources/Dockerfile.webrecorder`. This pulls `webrecorder/browsertrix-crawler:latest` and `apt install`s packages, so it needs network egress; the first build takes longer, later runs are cached.
- Crawling also needs egress to the target site and to `cdn.jsdelivr.net` (ReplayWeb.page assets are fetched inside the container).
- The TUI needs a real interactive TTY for the full UI — you can't drive it through a plain piped shell. Non-interactive use: pass URLs as arguments or on stdin (`echo https://example.com | ./archive`). Subcommands (`quit`, `server ...`, `help`, `resume`) work from a normal shell.
- Crawls run in a background `archive session-run` subprocess. Press `d` to detach, `q` to quit (also stops the Docker container). `./archive` reattaches to an active session; incomplete sessions can be resumed with `r` or `./archive resume`.
- `main.findRoot()` locates the repo root by walking up looking for `resources/Dockerfile.webrecorder` (falling back to the executable's dir). Run `./archive` from inside the repo (or keep `resources/` next to the binary).
- `.env` (created from `resources/env.example` on first run), `archive.ini` (regenerated each run), and everything under `crawls/` are gitignored build/runtime artifacts — do not commit them. Delete `.env` to reset config to defaults.
- `./archive server start` binds host port 80.
- Quick end-to-end sanity check: archive `https://example.com` — a small single page that crawls in a few seconds and produces a `.wacz` (and a `.zip`) under `crawls/<timestamp>-example.com/webrecorder/`.
