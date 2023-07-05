#!/bin/bash

dirs=`ls crawls/ | wc -l`
lines_in_file=$(wc -l < "$1")
total=$(( $lines_in_file - 3 ))

echo "Progress: $dirs of $total"