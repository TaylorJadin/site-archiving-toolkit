#!/bin/bash
workdir=`pwd`/crawls
url=$1
domain=`echo $url | cut -d '/' -f 3`

if [[ ${url} == http* ]];
	then
	docker build . -t archive-toolkit
    mkdir -p $workdir/$domain/httrack/
	mkdir -p $workdir/$domain/webrecorder/
	
	# start browsertrix and detach
	docker run -d -p 9037:9037 -v $workdir/$domain/webrecorder/:/crawls/collections/output/ -it archive-toolkit crawl --url $url --generateWACZ --workers 8 --text --collection output
	
	# start httrack crawl
	docker run -v $workdir/$domain/httrack/:/crawls/httrack archive-toolkit httrack --robots=0 --path /crawls/httrack $url
    
	# strip crossorigin and integrity retrictions out of html
	docker run -v $workdir/$domain/httrack/:/crawls/httrack archive-toolkit find . -name "*.html" -exec sed -i -E -e 's/integrity="[^"]+"//g' -e 's/crossorigin="[^"]+"//g' {} \;

    else
    	echo "URL must start with http:// or https://"
        echo "Ex: ./archive.sh https://reclaimed.tech"
        exit
fi