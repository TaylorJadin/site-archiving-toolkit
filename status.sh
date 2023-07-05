#!/bin/bash

# run like this ./status.sh umw1.sh

# run in screen and watch every 10 seconds
if [ -z "$STY" ]; then exec screen -m -S status watch -n 10 "$0" "$@"; fi

ls=`ls crawls/ | wc -l`
dirs=$(( $ls - 1))
lines_in_file=$(wc -l < "$1")
total=$(( $lines_in_file - 3 ))

rm -f crawls/"**Progress** "*
echo "Progress: $dirs of $total" > crawls/"**Progress** $dirs of $total"