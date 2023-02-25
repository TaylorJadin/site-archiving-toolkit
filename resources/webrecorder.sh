#!/bin/bash

url=$1
domain=$2
now=$3

cd /crawls/webrecorder
crawl --url $url --generateWACZ --workers 8 --text --collection archive

# Clean up webrecorder stuff we don't need
mv /crawls/webrecorder/collections/archive/archive.wacz /crawls/webrecorder/archive.wacz
rm -rf collections proxy-certs static templates

# Set up webrecorder to publish
wget https://cdn.jsdelivr.net/npm/replaywebpage/ui.js https://cdn.jsdelivr.net/npm/replaywebpage/sw.js
mkdir -p replay
mv *.js replay/
cp /replay-template.html index.html
sed -i -e "s|CRAWL_URL|$url|" index.html

# Zip up for easy download
zip ../webrecorder-$domain-$now.zip -r .