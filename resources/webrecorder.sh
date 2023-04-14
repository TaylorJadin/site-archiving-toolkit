#!/bin/bash

url=$1
domain=$2
now=$3

cd /crawls/webrecorder
crawl --url $url --generateWACZ --workers 4 --text --collection archive | tee ../webrecorder.log

# Clean up webrecorder stuff we don't need
mv /crawls/webrecorder/collections/archive/archive.wacz /crawls/webrecorder/archive.wacz
rm -rf collections proxy-certs static templates

# Set up webrecorder to publish
wget -q https://cdn.jsdelivr.net/npm/replaywebpage/ui.js https://cdn.jsdelivr.net/npm/replaywebpage/sw.js
mkdir -p replay
mv *.js replay/
cp /replay-template.html index.html
sed -i -e "s|CRAWL_URL|$url|" index.html

# Zip up for easy download
zip -q ../webrecorder-$domain-$now.zip -r .


