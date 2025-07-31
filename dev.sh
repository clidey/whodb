#!/bin/bash

# Development script - runs both frontend and backend
# Usage: ./dev.sh [--ee]

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

MODE="Community"
EE_FLAG=""

if [[ "$1" == "--ee" ]]; then
    MODE="Enterprise"
    EE_FLAG="--ee"
fi

echo -e "${GREEN}Starting WhoDB in $MODE Edition (Development Mode)${NC}"
echo


# Function to kill processes on exit
cleanup() {
    echo -e "\n${BLUE}Stopping all services...${NC}"
    pkill -P $$
    exit
}

trap cleanup EXIT INT TERM

# Start backend
echo -e "${BLUE}Starting backend...${NC}"
if [[ "$1" == "--ee" ]]; then
    (cd core && GOWORK="$PWD/go.work.ee" go run -tags ee .) &
else
    (cd core && go run .) &
fi

# Give backend a moment to start
sleep 2

# Start frontend
echo -e "${BLUE}Starting frontend dev server...${NC}"
(cd frontend && ./run.sh $EE_FLAG) &

echo
echo -e "${GREEN}✅ Development servers running:${NC}"
echo "   Backend: http://localhost:8080"
echo "   Frontend: http://localhost:1234 (with hot-reload)"
echo
echo "Press Ctrl+C to stop all services"

# Wait for all background processes
wait