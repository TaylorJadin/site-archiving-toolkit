#!/bin/bash

url=$1

cd /crawls
crawl --url $url --generateWACZ --workers 8 --text --scopeType host --collection archive | tee webrecorder.log

# Clean up webrecorder stuff we don't need
mv /crawls/collections/archive/archive.wacz /crawls/archive.wacz
rm -rf collections proxy-certs static templates

# Set up webrecorder to publish
wget -q https://cdn.jsdelivr.net/npm/replaywebpage/ui.js https://cdn.jsdelivr.net/npm/replaywebpage/sw.js
mkdir -p replay
mv *.js replay/
cp /replay-template.html index.html
sed -i -e "s|CRAWL_URL|$url|" index.html