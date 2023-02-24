#!/bin/bash

# Clean up httrack stuff we don't need
cd /crawls/httrack
rm -rf hts-cache
rm *.gif

# Get rid of integrity and crossorigin stuff
find . -name "*.html" -exec sed -i -E -e 's/integrity="[^"]+"//g' -e 's/crossorigin="[^"]+"//g' {} \;

# Clean up webrecorder stuff we don't need
cd /crawls/webrecorder
rm -rf archive/ logs/ pages/

# Set up webrecorder to publish
wget https://cdn.jsdelivr.net/npm/replaywebpage/ui.js https://cdn.jsdelivr.net/npm/replaywebpage/sw.js
mkdir -p replay
mv *.js replay/
cp /replay-template.html index.html