#!/bin/bash

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Default values
DEFAULT_CONNECTIONS=10000
GATEWAY_TCP_PORT=8900
GATEWAY_ADDR="127.0.0.1"
LOG_FILE="gateway_test.log"
GATEWAY_LOG="gateway.log"

# Help message
show_help() {
    echo -e "${GREEN}Usage:${NC}"
    echo "  ./scripts/connection-test.sh [options]"
    echo
    echo -e "${GREEN}Options:${NC}"
    echo "  -c, --connections    Target number of connections to test (default: $DEFAULT_CONNECTIONS)"
    echo "  -h, --help           Show this help message"
    echo
    echo -e "${GREEN}Example:${NC}"
    echo "  ./scripts/connection-test.sh -c 50000"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -c|--connections)
            CONNECTIONS="$2"
            shift 2
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            show_help
            exit 1
            ;;
    esac
done

# Set default values if not provided
CONNECTIONS=${CONNECTIONS:-$DEFAULT_CONNECTIONS}

# Print test configuration
echo -e "${GREEN}Connection Test Configuration:${NC}"
echo -e "Target Connections: ${YELLOW}$CONNECTIONS${NC}"
echo -e "Gateway TCP Address: ${YELLOW}$GATEWAY_ADDR:$GATEWAY_TCP_PORT${NC}"

# Check if the binary exists
if [ ! -f "./bin/gochat" ]; then
    echo -e "${RED}Error: gochat binary not found. Please build it first using 'make all'${NC}"
    exit 1
fi

# Clean up function
cleanup() {
    echo -e "\n${YELLOW}Cleaning up...${NC}"
    kill $GATEWAY_PID 2>/dev/null
    kill $PERF_PID 2>/dev/null
    rm -f $LOG_FILE $GATEWAY_LOG
    exit 0
}

# Set up trap for cleanup
trap cleanup SIGINT SIGTERM

# Start gateway service
echo -e "\n${GREEN}Starting Gateway service...${NC}"
./bin/gochat gateway > $GATEWAY_LOG 2>&1 &
GATEWAY_PID=$!

# Wait for gateway to start
echo -e "${YELLOW}Waiting for Gateway to start...${NC}"
for i in {1..30}; do
    if nc -z $GATEWAY_ADDR $GATEWAY_TCP_PORT > /dev/null 2>&1; then
        echo -e "${GREEN}Gateway is ready!${NC}"
        break
    fi
    if [ $i -eq 30 ]; then
        echo -e "${RED}Gateway failed to start within timeout${NC}"
        cleanup
        exit 1
    fi
    sleep 1
done

# Run the performance test
echo -e "\n${GREEN}Starting performance test...${NC}"
./bin/gochat perf --tcp_conn_num=$CONNECTIONS > $LOG_FILE 2>&1 &
PERF_PID=$!

# Wait for performance test to complete
wait $PERF_PID
PERF_EXIT_CODE=$?

# Extract results
MAX_CONNECTIONS=$(grep "MAX_CONNECTIONS=" $LOG_FILE | tail -n1 | cut -d'=' -f2)
GATEWAY_CONNECTIONS=$(grep "tcpNum:" $GATEWAY_LOG | tail -n1 | cut -d':' -f2 | tr -d ' ')

# Print results
echo -e "\n${GREEN}Test Results:${NC}"
echo -e "Performance Test Max Connections: ${YELLOW}$MAX_CONNECTIONS${NC}"
echo -e "Gateway Reported Connections: ${YELLOW}$GATEWAY_CONNECTIONS${NC}"

# Check if test was successful
if [ ! -z "$MAX_CONNECTIONS" ] && [ ! -z "$GATEWAY_CONNECTIONS" ]; then
    if [ "$MAX_CONNECTIONS" -ge "$CONNECTIONS" ] && [ "$GATEWAY_CONNECTIONS" -ge "$CONNECTIONS" ]; then
        echo -e "\n${GREEN}Test successful!${NC}"
        echo -e "Both performance test and gateway reported successful connections"
        echo -e "Logs saved to: ${GREEN}$LOG_FILE${NC} and ${GREEN}$GATEWAY_LOG${NC}"
        cleanup
        exit 0
    else
        echo -e "\n${YELLOW}Test partially successful${NC}"
        echo -e "Target connections: ${YELLOW}$CONNECTIONS${NC}"
        echo -e "Performance test connections: ${YELLOW}$MAX_CONNECTIONS${NC}"
        echo -e "Gateway connections: ${YELLOW}$GATEWAY_CONNECTIONS${NC}"
        cleanup
        exit 1
    fi
else
    echo -e "\n${RED}Failed to determine connection results${NC}"
    echo -e "Logs saved to: ${GREEN}$LOG_FILE${NC} and ${GREEN}$GATEWAY_LOG${NC}"
    cleanup
    exit 1
fi