# attach to httrack if its running
$is_running=docker ps -q -f name="httrack"
if ($is_running) {
    docker attach --sig-proxy=false httrack
    Clear-Host
    Write-Output "HTTrack completed its crawl." 
    }
    else {
        Write-Output "HTTrack is not running."
    }

# attach to browsertrix crawler if its running
$is_running=docker ps -q -f name="webrecorder"
if ($is_running) {
    Write-Output "Browsertrix Crawler is still working, attaching to container."
    docker attach --sig-proxy=false webrecorder
    }
    else {
        Write-Output "Browsertrix Crawler is not running."
    }