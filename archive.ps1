# Make archive.ini if it doesn't exist
if (!(Test-Path -Path archive.ini)) {
    Copy-Item -Path resources/defaults.ini -Destination archive.ini
}

# Read in archive.ini and set each line as a variable
Get-Content archive.ini | ForEach-Object {
    $key, $value = $_ -split '=', 2
    Set-Variable -Name $key -Value $value
}

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
docker.exe build -f resources/Dockerfile.webrecorder . -t site-archiving-toolkit-webrecorder
docker.exe build -f resources/Dockerfile.httrack . -t site-archiving-toolkit-httrack

# do browsertrix crawl
if ($enable_browsertrix -eq "TRUE") {
    mkdir $crawldir\webrecorder
    docker run --name webrecorder -d --rm -p 9037:9037 -v $crawldir/:/output -it site-archiving-toolkit-webrecorder  /bin/bash /webrecorder.sh $url $domain $now
}

# do httrack crawl
if ($enable_httrack -eq "TRUE") {
    mkdir $crawldir\httrack
    docker run --name httrack -d --rm -v $crawldir/:/output site-archiving-toolkit-httrack /bin/bash /httrack.sh $url $domain $now
}

# attach to crawls as they run
$is_running=docker ps -q -f name="httrack"
if ($is_running) {
    docker attach --sig-proxy=false httrack
    }

$is_running=docker ps -q -f name="webrecorder"
if ($is_running) {
    docker attach --sig-proxy=false webrecorder
    }

Write-Output "Crawl of $url complete!"