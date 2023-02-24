$url=$args[0]
$workdir=$pwd\crawls


[uri]$url_uri = $url
$domain = $url.Authority -replace '^www\.'
$domain