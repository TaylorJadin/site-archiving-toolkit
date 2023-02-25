tell application "Finder"
	set myPath to container of (path to me) as alias
end tell

set toolkitPath to (POSIX path of myPath)

set theURL to text returned of (display dialog �
	"Enter URL" default answer �
	"" buttons {"Archive"} �
	default button 1 �
	)

tell application "Terminal"
	activate
	do script "cd " & toolkitPath & " && ./archive.sh " & theURL & " && open crawls/"
end tell