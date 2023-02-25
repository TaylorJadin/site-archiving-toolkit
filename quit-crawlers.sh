#!/bin/bash

# stop httrack if its running
is_running=`docker ps -q -f name="httrack"`
if [ -n "$is_running" ]; then
    docker stop httrack
    else
        echo "HTTrack is not running."
fi

# stop browsertrix if its running
is_running=`docker ps -q -f name="webrecorder"`
if [ -n "$is_running" ]; then
    docker stop webrecorder
    else
        echo "Browsertrix Crawler is not running."
fi