#!/usr/bin/env bash
set -euo pipefail

# Prep
release_root=releases
rm -rf "$release_root"
mkdir -p "$release_root"

build_release() {
	local goos=$1
	local goarch=$2
	local label=$3
	local ext=$4

	local dir="$release_root/site-archiving-toolkit"
	rm -rf "$dir"
	mkdir -p "$dir"

	local bin="archive${ext}"
	echo "Building ${label} (${goos}/${goarch})..."
	GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -o "$dir/$bin" ./cmd/archive

	cp -r resources "$dir/"
	cp README.md "$dir/"

	(
		cd "$release_root"
		zip -r "${label}-site-archiving-toolkit.zip" site-archiving-toolkit >/dev/null
	)
	rm -rf "$dir"
	echo "Wrote $release_root/${label}-site-archiving-toolkit.zip"
}

build_release linux amd64 Linux ""
build_release darwin amd64 macOS-Intel ""
build_release darwin arm64 macOS-AppleSilicon ""
build_release windows amd64 Windows ".exe"

echo "Done."
