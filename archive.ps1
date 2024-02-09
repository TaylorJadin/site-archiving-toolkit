# check if crawls are already running and exit if they are
$is_running=docker ps -q -f name="httrack"
if ($is_running) {
    Write-Output "It looks like there is already a crawl running."
	Write-Output "Either check on the status by running .\attach.ps1 or quit by running .\quit-crawlers.ps1"
    exit
    }

$is_running=docker ps -q -f name="webrecorder"
if ($is_running) {
    Write-Output "It looks like there is already a crawl running."
	Write-Output "Either check on the status by running .\attach.ps1 or quit by running .\quit-crawlers.ps1"
    exit
    }

# set variables
$url=$args[0]
$workdir=Join-Path -Path $pwd -ChildPath "crawls"
$domain=([System.Uri]$url).Host -replace '^www\.'
$now=Get-Date -UFormat '+%Y-%m-%dT%H%M%S'
$crawldir=Join-Path -Path $workdir -ChildPath $now-$domain

# build containers
Write-Output "Getting ready..."
docker.exe build -f resources/Dockerfile.webrecorder . -t site-archiving-toolkit-webrecorder
docker.exe build -f resources/Dockerfile.httrack . -t site-archiving-toolkit-httrack

# make directories
mkdir $crawldir\httrack
mkdir $crawldir\webrecorder

Clear-Host

# start browsertrix
docker run --name webrecorder -d --rm -p 9037:9037 -v $crawldir/:/output -it site-archiving-toolkit-webrecorder  /bin/bash /webrecorder.sh $url $domain $now

# start httrack crawl
docker run --name httrack -d --rm -v $crawldir/:/output site-archiving-toolkit-httrack /bin/bash /httrack.sh $url $domain $now

# attach to crawls as they run
$is_running=docker ps -q -f name="httrack"
if ($is_running) {
    docker attach --sig-proxy=false httrack
    Clear-Host
    Write-Output "HTTrack completed its crawl." 
    }

$is_running=docker ps -q -f name="webrecorder"
if ($is_running) {
    Write-Output "Browsertrix Crawler is still working, attaching to container."
    docker attach --sig-proxy=false webrecorder
    }

Write-Output "Crawl of $url complete!"