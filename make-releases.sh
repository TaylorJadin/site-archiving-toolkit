#!/usr/bin/env bash
set -euo pipefail

# Builds a self-contained archive binary for each supported platform. All the
# runtime assets are embedded in the binary, so a release is just the binary
# plus the README.

release_root=releases
rm -rf "$release_root"
mkdir -p "$release_root"

targets=(
	"linux amd64 Linux"
	"darwin amd64 macOS-Intel"
	"darwin arm64 macOS-AppleSilicon"
	"windows amd64 Windows .exe"
)

for target in "${targets[@]}"; do
	read -r goos goarch label ext <<<"$target"
	dir="$release_root/site-archiving-toolkit"

	echo "Building ${label} (${goos}/${goarch})..."
	mkdir -p "$dir"
	GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -o "$dir/archive${ext:-}" ./cmd/archive
	cp README.md "$dir/"

	(cd "$release_root" && zip -qr "${label}-site-archiving-toolkit.zip" site-archiving-toolkit)
	rm -rf "$dir"
	echo "Wrote $release_root/${label}-site-archiving-toolkit.zip"
done

echo "Done."
