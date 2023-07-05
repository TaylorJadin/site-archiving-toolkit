#!/bin/bash

# screen -d -m watch -n 30 ./status.sh umw1.sh

dirs=`ls crawls/ | wc -l`
lines_in_file=$(wc -l < "$1")
total=$(( $lines_in_file - 3 ))

rm -f "**Progress**"*
echo "Progress $dirs of $total" > "**Progress** $dirs of $total"