# stop httrack if its running
$is_running=docker ps -q -f name="httrack"
if ($is_running) {
    docker stop httrack
    }
    else {
        Write-Output "HTTrack is not running."
    }

# stop browsertrix crawler if its running
$is_running=docker ps -q -f name="webrecorder"
if ($is_running) {
    docker stop webrecorder
    }
    else {
        Write-Output "Browsertrix Crawler is not running."
    }