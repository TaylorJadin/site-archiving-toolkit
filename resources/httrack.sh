#!/bin/bash

url=$1
domain=$2
now=$3

source /archive.ini

# Read in defaults.ini if archive.ini doesn't exist
if [ ! -f archive.ini ]; then
	source defaults.ini
else
	source archive.ini
fi

cd /output/httrack
httrack $httrack_parameters --path /output/httrack "$url" | tee /output/httrack.log

# Clean up stuff we don't need
rm -rf hts-cache
rm *.gif

# Get rid of integrity and crossorigin stuff
find . -name "*.html" -exec sed -i -E -e 's/integrity="[^"]+"//g' -e 's/crossorigin="[^"]+"//g' {} \;

# Zip up for easy download
zip -q ../httrack-$domain-$now.zip -r .
