#!/usr/bin/env bash

# kill screen session
screen -S site-archiving-toolkit -X quit

# stop httrack if its running
is_running=`docker ps -q -f name="httrack"`
if [ -n "$is_running" ]; then
    docker stop httrack > /dev/null
    echo "Succesfully quit HTTTrack."
    else
        echo "HTTrack is not running."
fi

# stop browsertrix if its running
is_running=`docker ps -q -f name="webrecorder"`
if [ -n "$is_running" ]; then
    docker stop webrecorder > /dev/null
    echo "Succesfully quit Browsertrix Crawler."
    else
        echo "Browsertrix Crawler is not running."
fi
