$url=$args[0]
$workdir=Join-Path -Path $pwd -ChildPath "crawls"
$domain=([System.Uri]$url).Host -replace '^www\.'
$now=Get-Date -UFormat '+%Y-%m-%dT%H%M%S'
$crawldir=Join-Path -Path $workdir -ChildPath $domain-$now

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