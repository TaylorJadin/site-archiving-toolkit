#!/bin/bash

completed=`ls crawls | wc -l`
total=`wc -l $1 | cut -d ' ' -f 5`
echo "Percentage complete:"
echo "$completed / $total * 100" | bc -l