$url=$args[0]
$workdir=Join-Path -Path $pwd -ChildPath "crawls"
$domain=([System.Uri]$url).Host -replace '^www\.'
$now=Get-Date -UFormat '+%Y-%m-%dT%H%M%S'
$crawldir=Join-Path -Path $workdir -ChildPath $domain-$now

docker.exe build -f resources/Dockerfile . -t archive-toolkit

mkdir $crawldir\httrack
mkdir $crawldir\webrecorder