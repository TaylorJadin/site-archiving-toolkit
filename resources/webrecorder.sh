#!/bin/bash

url=$1
normalized_url=$2
now=$3

source /archive.ini

crawl --url "$url" --generateWACZ $browsertrix_parameters --collection archive | tee /output/webrecorder.log

# Clean up webrecorder stuff we don't need
mv /crawls/collections/archive/archive.wacz /output/webrecorder/$normalized_url-$now.wacz
rm -rf collections proxy-certs static templates

# Set up webrecorder to publish
cd /output/webrecorder
wget -q https://cdn.jsdelivr.net/npm/replaywebpage/ui.js https://cdn.jsdelivr.net/npm/replaywebpage/sw.js
mkdir -p replay
mv *.js replay/
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
zip -q ../webrecorder-$normalized_url-$now.zip -r .
