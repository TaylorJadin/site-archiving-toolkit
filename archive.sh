#!/usr/bin/env bash

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

# Function to normalize URL for filesystem use
normalize_url() {
	local url="$1"
	# Remove http:// or https://
	url="${url#http://}"
	url="${url#https://}"
	# Remove trailing slash
	url="${url%/}"
	# Replace unsafe characters with dash
	# Unsafe chars: < > : " / \ | ? * and spaces
	url=$(echo "$url" | tr '<>:"/\\|?* ' '-')
	# Remove any control characters
	url=$(echo "$url" | tr -d '\000-\037')
	# Remove any leading/trailing dashes and collapse multiple dashes
	url=$(echo "$url" | sed 's/-\+/-/g' | sed 's/^-\|-$//g')
	echo "$url"
}

# Archive every URL passed as an argument
for url in "$@"
do

workdir=`pwd`/crawls
normalized_url=$(normalize_url "$url")

# Check if crawl already exists for this normalized URL
if [ "$skip_existing_crawls" = TRUE ]; then
	if [ -d "$workdir" ]; then
		# Remove any INCOMPLETE crawls for this URL
		find "$workdir" -maxdepth 1 -type d -name "*-$normalized_url" | while read -r crawl_dir; do
			dirname=$(basename "$crawl_dir")
			if [[ "$dirname" == INCOMPLETE-* ]]; then
				echo "Removing incomplete crawl: $dirname"
				rm -rf "$crawl_dir"
			fi
		done
		# Check for completed crawls (not starting with INCOMPLETE-)
		completed_crawl=$(find "$workdir" -maxdepth 1 -type d -name "*-$normalized_url" ! -name "INCOMPLETE-*" | head -n 1)
		if [ -n "$completed_crawl" ]; then
			echo "Skipping crawl for $url - completed crawl found: $(basename "$completed_crawl")"
			continue
		fi
	fi
fi

now=`date +%Y-%m-%dT%H%M%S`
crawldir="$workdir/INCOMPLETE-$now-$normalized_url"
completedir="$workdir/$now-$normalized_url"

docker build -f resources/Dockerfile.webrecorder . -t site-archiving-toolkit-webrecorder
docker build -f resources/Dockerfile.httrack . -t site-archiving-toolkit-httrack 

# Start crawling!
if [ "$enable_browsertrix" = TRUE ]; then
mkdir -p $crawldir/webrecorder/
docker run --name webrecorder -d --rm -v $crawldir/:/output site-archiving-toolkit-webrecorder /bin/bash /webrecorder.sh $url $normalized_url $now
fi

if [ "$enable_httrack" = TRUE ]; then
mkdir -p $crawldir/httrack/
docker run --name httrack -d --rm -v $crawldir/:/output site-archiving-toolkit-httrack /bin/bash /httrack.sh $url $normalized_url $now
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
