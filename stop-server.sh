#!/bin/bash

docker_check=`docker version | grep "ERROR: Cannot connect to the Docker daemon"`
if [ -n $docker_check ]; then
echo "It looks like the Docker daemon has not been started. Check to see if Docker is installed and running."
exit
fi

docker-compose -f resources/docker-compose.yml down