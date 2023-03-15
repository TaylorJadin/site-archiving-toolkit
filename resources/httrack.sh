#!/bin/bash

url=$1
domain=$2
now=$3

cd /crawls/httrack
httrack -n --robots=0 --extra-log --verbose --path /crawls/httrack $url | tee ../httrack.log
rm ../httrack.log

# Clean up stuff we don't need
rm -rf hts-cache
rm *.gif

# Get rid of integrity and crossorigin stuff
find . -name "*.html" -exec sed -i -E -e 's/integrity="[^"]+"//g' -e 's/crossorigin="[^"]+"//g' {} \;

# Zip up for easy downloadls
cd /crawls/httrack
zip -q ../httrack-$domain-$now.zip -r .


