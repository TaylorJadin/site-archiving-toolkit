#!/bin/bash

url=$1
domain=$2
now=$3

source /archive.ini

crawl --url "$url" --generateWACZ $browsertrix_parameters --collection archive | tee /output/webrecorder.log

# Clean up webrecorder stuff we don't need
mv /crawls/collections/archive/archive.wacz /output/webrecorder/$domain-$now.wacz
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
sed -i -e "s|FILE_NAME|$domain-$now|" index.html

# Zip up for easy download
zip -q ../webrecorder-$domain-$now.zip -r .
