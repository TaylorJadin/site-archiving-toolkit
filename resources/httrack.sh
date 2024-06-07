#!/bin/bash

url=$1
domain=$2
now=$3

source /archive.ini

cd /output/httrack
httrack $httrack_parameters --path /output/httrack "$url" | tee /output/httrack.log

# Clean up stuff we don't need
rm -rf hts-cache
rm *.gif

# Get rid of integrity and crossorigin stuff
find . -name "*.html" -exec sed -i -E -e 's/integrity="[^"]+"//g' -e 's/crossorigin="[^"]+"//g' {} \;

# Make sure permissions are correct
find . -type d -exec chmod 755 {} \;
find . -type f -exec chmod 644 {} \;

# Zip up for easy download
zip -q ../httrack-$domain-$now.zip -r .
