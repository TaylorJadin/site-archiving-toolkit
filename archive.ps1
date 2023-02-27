$url=$args[0]
$workdir=Join-Path -Path $pwd -ChildPath "crawls"
$domain=([System.Uri]$url).Host -replace '^www\.'
$now=Get-Date -UFormat '+%Y-%m-%dT%H%M%S'
$crawldir=Join-Path -Path $workdir -ChildPath $now-$domain

Write-Output "Getting ready..."
docker.exe build -f resources/Dockerfile . -t archive-toolkit

mkdir $crawldir\httrack
mkdir $crawldir\webrecorder

Clear-Host
Write-Output "Starting Browsertrix Crawler and HTTrack..."

# start browsertrix
docker run --name webrecorder -d --rm -p 9037:9037 -v $crawldir/:/crawls -it archive-toolkit  /bin/bash /webrecorder.sh $url $domain $now

# start httrack crawl
docker run --name httrack -d --rm -v $crawldir/:/crawls archive-toolkit /bin/bash /httrack.sh $url $domain $now

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