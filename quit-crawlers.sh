#!/bin/bash

# kill screen session
screen -S site-archiving-toolkit -X quit

docker_check=`docker version | grep "ERROR: Cannot connect to the Docker daemon"`
if [ -n $docker_check ]; then
echo "It looks like the Docker daemon has not been started. Check to see if Docker is installed and running."
exit
fi

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
