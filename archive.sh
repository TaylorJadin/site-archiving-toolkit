#!/bin/bash

if [ -z "$STY" ]; then exec screen -m -S site-archiving-toolkit /bin/bash "$0" "$@"; fi

trap '' INT

for url in "$@"
do

workdir=`pwd`/crawls
domain=`echo $url | cut -d '/' -f 3`
now=`date +%Y-%m-%dT%H%M%S`
crawldir="$workdir/INCOMPLETE-$domain-$now"
completedir="$workdir/$domain-$now"

if [[ ${url} == http* ]];
	then
	echo "Getting ready..."
	docker build -f resources/Dockerfile . -t archive-toolkit

	# make crawl directories
    mkdir -p $crawldir/httrack/
	mkdir -p $crawldir/webrecorder/
	
	clear
	echo "Starting Browsertrix Crawler and HTTrack..."

	# start browsertrix
	docker run --name webrecorder -d --rm -p 9037:9037 -v $crawldir/:/crawls archive-toolkit  /bin/bash /webrecorder.sh $url $domain $now
	
	# start httrack crawl
	docker run --name httrack -d --rm -v $crawldir/:/crawls archive-toolkit /bin/bash /httrack.sh $url $domain $now

	# attach to httrack if its running
	is_running=`docker ps -q -f name="httrack"`
	if [ -n "$is_running" ]; then
		docker attach --sig-proxy=false httrack
		clear
		echo "HTTrack completed its crawl." 
	fi

	# attach to browsertrix if its still running
	is_running=`docker ps -q -f name="webrecorder"`
	if [ -n "$is_running" ]; then
		echo "Browsertrix Crawler is still working, attaching to container."
		docker attach --sig-proxy=false webrecorder
	fi

	mv $crawldir $completedir
    
	echo "Crawl of $url complete!"

    else
    	echo "URL must start with http:// or https://"
        echo "Ex: ./archive.sh https://reclaimed.tech"
        exit
fi

done



