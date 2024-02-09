#!/bin/bash

# Make archive.ini if it doesn't exist
if [ ! -f archive.ini ]; then
	cp resources/defaults.ini archive.ini
fi

source archive.ini

# Check if URL is formatted correctly
if [[ $1 != http* ]]; then
	echo "URL must start with http:// or https://"
	echo "Ex: ./archive.sh https://reclaimed.tech"
	exit
fi

# Is docker running?
docker_check=`docker ps 2>&1`
if [[ $docker_check == *'Cannot connect to the Docker daemon'* ]]; then
	echo $docker_check
	echo "It looks like Docker is not available. Check to see if Docker is installed and running."
	exit
fi

# Are crawls running already?
is_running=`docker ps -q -f name="httrack"`
if [ -n "$is_running" ]; then
	echo "It looks like there is already a crawl running."
	echo "Either check on the status by running ./attach.sh or quit by running ./quit-crawlers.sh"
	exit
fi

is_running=`docker ps -q -f name="webrecorder"`
if [ -n "$is_running" ]; then
	echo "It looks like there is already a crawl running."
	echo "Either check on the status by running ./attach.sh or quit by running ./quit-crawlers.sh"
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
now=`date +%Y-%m-%dT%H%M%S`
crawldir="$workdir/INCOMPLETE-$now-$domain"
completedir="$workdir/$now-$domain"

docker build -f resources/Dockerfile.webrecorder . -t site-archiving-toolkit-webrecorder
docker build -f resources/Dockerfile.httrack . -t site-archiving-toolkit-httrack 

# Start crawling!
if [ "$enable_browsertrix" = TRUE ]; then
mkdir -p $crawldir/webrecorder/
docker run --name webrecorder -d --rm -v $crawldir/:/output site-archiving-toolkit-webrecorder /bin/bash /webrecorder.sh $url $domain $now
fi

if [ "$enable_httrack" = TRUE ]; then
mkdir -p $crawldir/httrack/
docker run --name httrack -d --rm -v $crawldir/:/output site-archiving-toolkit-httrack /bin/bash /httrack.sh $url $domain $now
fi

#  attach to httrack if its running
is_running=`docker ps -q -f name="httrack"`
if [ -n "$is_running" ]; then
	docker attach --sig-proxy=false httrack
fi

# attach to browsertrix if its still running
is_running=`docker ps -q -f name="webrecorder"`
if [ -n "$is_running" ]; then
	docker attach --sig-proxy=false webrecorder
fi

mv $crawldir $completedir

echo "Crawl of $url complete!"
done
