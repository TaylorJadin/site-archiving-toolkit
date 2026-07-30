// Package resources holds the files that are baked into the archive binary:
// the Docker build context for the crawler image and the default .env file.
package resources

import (
	"embed"
	_ "embed"
)

// Dockerfile is the name of the Dockerfile within the build context.
const Dockerfile = "Dockerfile.webrecorder"

// BuildContext contains every file the crawler image is built from.
//
//go:embed Dockerfile.webrecorder webrecorder.sh index.html redirect.php htaccess
var BuildContext embed.FS

// EnvExample is the default .env written on first run.
//
//go:embed env.example
var EnvExample []byte
