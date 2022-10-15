#!/bin/bash
workdir=`pwd`/crawls
url=$1
domain=`echo $url | cut -d '/' -f 3`

if [[ ${url} == http* ]];
	then
	docker build . -t archive-toolkit
    mkdir -p $workdir/$domain/httrack/
	mkdir -p $workdir/$domain/webrecorder/
	docker run -d -p 9037:9037 -v $workdir/$domain/webrecorder/:/crawls/collections/ -it archive-toolkit crawl --url $url --generateWACZ --workers 8 --text
	docker run -it --rm -v $workdir/$domain/httrack/:/crawls/httrack archive-toolkit httrack --robots=0 --path /crawls/httrack $url
	docker run -it --rm -v $workdir/$domain/httrack/:/crawls/httrack archive-toolkit grep -Rl 'crossorigin="' . | xargs sed -i.bak -E -e 's/ integrity="[^"]+"//g' -e 's/ crossorigin="[^"]+"//g'
    else
    	echo "URL must start with http:// or https://"
        echo "Ex: ./archive.sh https://reclaimed.tech"
        exit
fi