#!/bin/bash
workdir=`pwd`/crawls
url=$1
domain=`echo $url | cut -d '/' -f 3`

if [[ ${url} == http* ]];
	then
	docker build -f resources/Dockerfile -t archive-toolkit

	# make working directories
    mkdir -p $workdir/$domain/httrack/
	mkdir -p $workdir/$domain/webrecorder/
	
	# start browsertrix
	docker run --name webrecorder -d --rm -p 9037:9037 \
	-v $workdir/$domain/webrecorder/:/crawls/collections/archive/ -it archive-toolkit \
	crawl --url $url --generateWACZ --workers 8 --text --collection archive
	
	# start httrack crawl
	docker run --name httrack -d --rm -v $workdir/$domain/httrack/:/crawls/httrack archive-toolkit \
	httrack --robots=0 --extra-log --verbose --path /crawls/httrack $url

	# attach to httrack if its running
	is_running=$(docker ps -q -f name="httrack")
	if [ -n "$is_running" ]; then
		docker attach --sig-proxy=false httrack
		clear
		echo "HTTrack completed its crawl." 
	fi

	# attach to browsertrix if its still running
	is_running=$(docker ps -q -f name="webrecorder")
	if [ -n "$is_running" ]; then
		echo "Browsertrix Crawler is still working, attaching to container."
		sleep 3
		docker attach --sig-proxy=false webrecorder
	fi

	# Clean up httrack mirror and browsertrix crawl mirrors
	docker run --name archive-toolkit --rm -v $workdir/$domain/:/crawls/ archive-toolkit /bin/bash /cleanup.sh

    else
    	echo "URL must start with http:// or https://"
        echo "Ex: ./archive.sh https://reclaimed.tech"
        exit
fi
