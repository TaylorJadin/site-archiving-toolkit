!/bin/bash

domain=$1
now=$2
url=$3

cd /crawls
crawl --url $url --generateWACZ --workers 8 --text --collection archive

# Clean up webrecorder stuff we don't need
rm -rf archive/ logs/ pages/

# Set up webrecorder to publish
wget https://cdn.jsdelivr.net/npm/replaywebpage/ui.js https://cdn.jsdelivr.net/npm/replaywebpage/sw.js
mkdir -p replay
mv *.js replay/
cp /replay-template.html index.html
sed -i -e "s|CRAWL_URL|$url|" index.html

# Zip up for easy download
zip ../$domain-$now-webrecorder.zip -r .