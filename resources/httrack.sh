#!/bin/bash

url=$1
domain=$2
now=$3

cd /output/httrack
httrack --near --robots=0 --disable-security-limits --max-rate=0 --extra-log --verbose --path /output/httrack "$url" | tee /output/httrack.log

# Clean up stuff we don't need
rm -rf hts-cache
rm *.gif

# Get rid of integrity and crossorigin stuff
find . -name "*.html" -exec sed -i -E -e 's/integrity="[^"]+"//g' -e 's/crossorigin="[^"]+"//g' {} \;

# Zip up for easy download
zip -q ../httrack-$domain-$now.zip -r .
