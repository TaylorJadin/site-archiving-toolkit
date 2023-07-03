#!/bin/bash

# kill screen session
screen -S site-archiving-toolkit -X quit

# stop browsertrix if its running
is_running=`docker ps -q -f name="webrecorder"`
if [ -n "$is_running" ]; then
    docker stop webrecorder > /dev/null
    echo "Succesfully quit Browsertrix Crawler."
    else
        echo "Browsertrix Crawler is not running."
fi
