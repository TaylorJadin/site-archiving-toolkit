#!/bin/bash

# run this with:
# screen -d -m watch -n 10 bash status.sh umw1.sh

ls=`ls crawls/ | wc -l`
dirs=$(( $ls - 1))
lines_in_file=$(wc -l < "$1")
total=$(( $lines_in_file - 3 ))

rm -f crawls/"**Progress** "*
echo "Progress: $dirs of $total" > crawls/"**Progress** $dirs of $total"