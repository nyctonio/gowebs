#!/bin/bash

# Define the Docker image name and port range
IMAGE_NAME="bunweb"
PORT_START=8000
PORT_END=8005

# Loop through the port range and start a Docker container on each port
for ((port=$PORT_START; port<=$PORT_END; port++)); do
    docker run -d -p $port:8080 $IMAGE_NAME
    echo "Started container on port $port"
done