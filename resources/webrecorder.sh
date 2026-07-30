#!/usr/bin/env bash
set -euo pipefail

url=$1
normalized_url=$2
now=$3

# Crawl options are passed in as environment variables by the archive binary.
: "${browsertrix_parameters:=--workers 4 --text}"
: "${browsertrix_redirect_template:=FALSE}"
: "${create_webrecorder_zip:=TRUE}"

mkdir -p /output/webrecorder

# shellcheck disable=SC2086
crawl --url "$url" --generateWACZ $browsertrix_parameters --collection archive | tee /output/webrecorder.log

# Clean up webrecorder stuff we don't need
mv /crawls/collections/archive/archive.wacz "/output/webrecorder/${normalized_url}-${now}.wacz"
rm -rf /crawls/collections /crawls/proxy-certs /crawls/static /crawls/templates

# Set up webrecorder to publish
cd /output/webrecorder
wget -q https://cdn.jsdelivr.net/npm/replaywebpage/ui.js https://cdn.jsdelivr.net/npm/replaywebpage/sw.js
mkdir -p replay
mv ./*.js replay/
cp /index.html index.html
if [ "$browsertrix_redirect_template" = TRUE ]; then
	cp /redirect.php redirect.php
	cp /.htaccess .htaccess
	sed -i -e "s|CRAWL_URL|$url|" redirect.php
fi
sed -i -e "s|CRAWL_URL|$url|" index.html
sed -i -e "s|FILE_NAME|$normalized_url-$now|" index.html

# Make sure permissions are correct
find . -type d -exec chmod 755 {} \;
find . -type f -exec chmod 644 {} \;

# Zip up for easy download
if [ "$create_webrecorder_zip" = TRUE ]; then
	zip -q "../webrecorder-${normalized_url}-${now}.zip" -r .
fi
