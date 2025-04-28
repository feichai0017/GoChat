#!/bin/bash

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' 

# Function to check if a container is running
check_container() {
    docker ps -q -f name=$1
}

# Function to start a container if not running
start_container() {
    local name=$1
    local image=$2
    local ports=$3
    local cmd=$4

    echo -e "${GREEN}Checking $name...${NC}"
    if [ -z "$(check_container $name)" ]; then
        echo -e "${GREEN}Starting $name...${NC}"
        docker run -d \
            --name $name \
            $ports \
            $image \
            $cmd
    else
        echo -e "${GREEN}$name is already running${NC}"
    fi
}

# Start etcd
start_container "etcd" \
    "quay.io/coreos/etcd:v3.5.4" \
    "-p 2379:2379 -p 2380:2380" \
    "/usr/local/bin/etcd --advertise-client-urls http://0.0.0.0:2379 --listen-client-urls http://0.0.0.0:2379"

# Start Redis
start_container "redis" \
    "redis:latest" \
    "-p 6379:6379" \
    ""

# Wait for services to be ready
echo -e "${GREEN}Waiting for services to be ready...${NC}"
sleep 5

# Check if services are running
echo -e "${GREEN}Checking service status:${NC}"
echo "etcd: $(check_container etcd && echo 'Running' || echo 'Not running')"
echo "redis: $(check_container redis && echo 'Running' || echo 'Not running')"

echo -e "${GREEN}All services started!${NC}"
echo -e "etcd endpoint: localhost:2379"
echo -e "redis endpoint: localhost:6379"