#!/bin/bash

screen -dr -S site-archiving-toolkit

# attach to httrack if its running
is_running=`docker ps -q -f name="httrack"`
if [ -n "$is_running" ]; then
    clear
    echo "Attaching to HTTrack container..."
    docker attach --sig-proxy=false httrack
    else
        echo "HTTrack is not running."
fi

# attach to browsertrix if its still running
is_running=`docker ps -q -f name="webrecorder"`
if [ -n "$is_running" ]; then
    clear
    echo "Attaching to Browsertrix Crawler container..."
    docker attach --sig-proxy=false webrecorder
    else
        echo "Browsertrix Crawler is not running."
fi
