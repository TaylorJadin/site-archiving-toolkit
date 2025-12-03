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

# Function to normalize URL for filesystem use
function Normalize-Url {
    param([string]$url)
    # Remove http:// or https://
    $url = $url -replace '^https?://', ''
    # Remove trailing slash
    $url = $url -replace '/$', ''
    # Replace unsafe characters with dash
    # Unsafe chars: < > : " / \ | ? * and spaces
    $url = $url -replace '[<>:"/\\|?* ]', '-'
    # Remove any control characters
    $url = $url -replace '[\x00-\x1F]', ''
    # Remove any leading/trailing dashes and collapse multiple dashes
    $url = $url -replace '-+', '-'
    $url = $url -replace '^-|-$', ''
    return $url
}

# set variables
$url=$args[0]
$workdir=Join-Path -Path $pwd -ChildPath "crawls"
$normalized_url=Normalize-Url $url

# Check if crawl already exists for this normalized URL
if ($skip_existing_crawls -eq "TRUE") {
    if (Test-Path -Path $workdir) {
        # Remove any INCOMPLETE crawls for this URL
        $incomplete_crawls = Get-ChildItem -Path $workdir -Directory | Where-Object { $_.Name -like "INCOMPLETE-*-$normalized_url" }
        if ($incomplete_crawls) {
            Write-Output "Removing incomplete crawl(s) for $url:"
            foreach ($incomplete_dir in $incomplete_crawls) {
                Write-Output "  Removing $($incomplete_dir.Name)"
                Remove-Item -Path $incomplete_dir.FullName -Recurse -Force
            }
        }
        # Check for completed crawls (not starting with INCOMPLETE-)
        $completed_crawl = Get-ChildItem -Path $workdir -Directory | Where-Object { $_.Name -like "*-$normalized_url" -and $_.Name -notlike "INCOMPLETE-*" } | Select-Object -First 1
        if ($completed_crawl) {
            Write-Output "Skipping crawl for $url - completed crawl found: $($completed_crawl.Name)"
            exit
        }
    }
}

$now=Get-Date -UFormat '+%Y-%m-%dT%H%M%S'
$crawldir=Join-Path -Path $workdir -ChildPath "$now-$normalized_url"

# build containers
docker.exe build -f resources/Dockerfile.webrecorder . -t site-archiving-toolkit-webrecorder
docker.exe build -f resources/Dockerfile.httrack . -t site-archiving-toolkit-httrack

# do browsertrix crawl
if ($enable_browsertrix -eq "TRUE") {
    mkdir $crawldir\webrecorder
    docker run --name webrecorder -d --rm -p 9037:9037 -v $crawldir/:/output -it site-archiving-toolkit-webrecorder  /bin/bash /webrecorder.sh $url $normalized_url $now
}

# do httrack crawl
if ($enable_httrack -eq "TRUE") {
    mkdir $crawldir\httrack
    docker run --name httrack -d --rm -v $crawldir/:/output site-archiving-toolkit-httrack /bin/bash /httrack.sh $url $normalized_url $now
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