#!/bin/bash

# WhoDB Desktop Prebuild Script for Current Platform
# This script builds the Go backend only for the current platform

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}WhoDB Desktop Prebuild Script (Current Platform)${NC}"
echo "================================================"

# Detect current platform
CURRENT_OS=$(uname -s | tr '[:upper:]' '[:lower:]')
CURRENT_ARCH=$(uname -m)

# Convert to Go naming
case $CURRENT_OS in
    darwin) GOOS="darwin" ;;
    linux) GOOS="linux" ;;
    mingw*|msys*|cygwin*) GOOS="windows" ;;
    *) echo -e "${RED}Unsupported OS: $CURRENT_OS${NC}"; exit 1 ;;
esac

case $CURRENT_ARCH in
    x86_64) GOARCH="amd64" ;;
    aarch64|arm64) GOARCH="arm64" ;;
    *) echo -e "${RED}Unsupported architecture: $CURRENT_ARCH${NC}"; exit 1 ;;
esac

echo -e "${YELLOW}Building for $GOOS/$GOARCH${NC}"

# Determine if building EE or CE
IS_EE=false
if [ -n "$GOWORK" ] && [[ "$GOWORK" == *"go.work.ee"* ]]; then
    IS_EE=true
    echo -e "${YELLOW}Building Enterprise Edition${NC}"
else
    echo -e "${YELLOW}Building Community Edition${NC}"
fi

# Create binaries directory
mkdir -p src-tauri/binaries

# Step 1: Build frontend
echo -e "\n${GREEN}Step 1: Building frontend...${NC}"
cd ../frontend
if [ "$IS_EE" = true ]; then
    pnpm run build:ee
else
    pnpm run build
fi
echo -e "${GREEN}Frontend build complete${NC}"

# Step 2: Copy frontend build to core
echo -e "\n${GREEN}Step 2: Copying frontend build to core...${NC}"
rm -rf ../core/build
cp -r build ../core/
echo -e "${GREEN}Frontend copied to core/build${NC}"

# Step 3: Build Go backend
cd ../core
echo -e "\n${GREEN}Step 3: Building Go backend...${NC}"

# Set output name
if [ "$GOOS" = "windows" ]; then
    OUTPUT="../desktop/src-tauri/binaries/whodb.exe"
else
    OUTPUT="../desktop/src-tauri/binaries/whodb"
fi

# Set build tags
BUILD_TAGS=""
if [ "$IS_EE" = true ]; then
    BUILD_TAGS="-tags ee"
fi

# Build command
if [ "$IS_EE" = true ]; then
    GOWORK=$GOWORK go build $BUILD_TAGS -o "$OUTPUT" .
else
    go build -o "$OUTPUT" .
fi

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Built successfully${NC}"
    if [ "$GOOS" != "windows" ]; then
        chmod +x "$OUTPUT"
    fi
    ls -lh "$OUTPUT"
else
    echo -e "${RED}✗ Build failed${NC}"
    exit 1
fi

echo -e "\n${GREEN}Prebuild complete!${NC}"