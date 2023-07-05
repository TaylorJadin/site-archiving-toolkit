#!/bin/bash

if [[ $1 != http* ]];
then
echo "URL must start with http:// or https://"
echo "Ex: ./archive.sh https://reclaimed.tech"
exit
fi

# Open up a screen session 
if [ -z "$STY" ]; then exec screen -m -S site-archiving-toolkit /bin/bash "$0" "$@"; fi

# Ignore ctrl+c
trap '' INT

# Archive every URL passed as an argument
for url in "$@"
do

workdir=`pwd`/crawls
domain=`echo $url | cut -d '/' -f 3`
subdomain=`echo $domain | cut -d '.' -f 1`
now=`date +%Y-%m-%dT%H%M%S`
crawldir="$workdir/INCOMPLETE-$subdomain"
completedir="$workdir/$subdomain"

echo "Getting ready..."
docker build -f resources/Dockerfile . -t archive-toolkit

# make crawl directories
mkdir -p $crawldir

clear
echo "Starting Browsertrix Crawler..."

# start browsertrix
docker run --name webrecorder -d --rm -p 9037:9037 -v $crawldir/:/crawls archive-toolkit  /bin/bash /webrecorder.sh $url

# attach to browsertrix if its still running
is_running=`docker ps -q -f name="webrecorder"`
if [ -n "$is_running" ]; then
	echo "Browsertrix Crawler is still working, attaching to container."
	docker attach --sig-proxy=false webrecorder
fi

mv $crawldir $completedir

echo "Crawl of $url complete!"
done