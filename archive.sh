#!/bin/bash
workdir=/root/webroot
url=$1
domain=`echo $url | cut -d '/' -f 3`

if [[ ${url} == http* ]];
	then
    	mkdir -p "$workdir/$domain/httrack/"
		mkdir -p "$workdir/$domain/browsertrix-crawler/"
		mkdir -p "$workdir/$domain/httrack/"
		docker run -d -p 9037:9037 -v "$workdir/$domain/browsertrix-crawler/:/crawls/collections/" -it webrecorder/browsertrix-crawler crawl --url $url --generateWACZ --workers 8 --text
		docker run -it --rm -v $workdir/$domain/httrack/:/data taylorjadin/httrack bash -c "httrack --robots=0 $url"
    else
    	echo "URL must start with http:// or https://"
        echo "Ex: ./archive.sh https://reclaimed.tech"
        exit
fi

