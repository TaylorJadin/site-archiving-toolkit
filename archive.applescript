tell application "Finder"
	set myPath to container of (path to me) as alias
end tell

set toolkitPath to (POSIX path of myPath)

set theURL to text returned of (display dialog Â
	"Enter URL" default answer Â
	"" buttons {"Archive"} Â
	default button 1 Â
	)

tell application "Terminal"
	activate
	do script "cd " & toolkitPath & " && ./archive.sh " & theURL & " && open crawls/"
end tell