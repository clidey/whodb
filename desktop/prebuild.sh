#!/bin/bash

# WhoDB Desktop Prebuild Script for Tauri
# This script builds the Go backend for multiple platforms before Tauri packaging

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}WhoDB Desktop Prebuild Script${NC}"
echo "================================"

# Detect current platform
CURRENT_OS=$(uname -s | tr '[:upper:]' '[:lower:]')
CURRENT_ARCH=$(uname -m)

# Convert to Go naming
case $CURRENT_OS in
    darwin) CURRENT_OS="darwin" ;;
    linux) CURRENT_OS="linux" ;;
    mingw*|msys*|cygwin*) CURRENT_OS="windows" ;;
    *) echo -e "${RED}Unsupported OS: $CURRENT_OS${NC}"; exit 1 ;;
esac

case $CURRENT_ARCH in
    x86_64) CURRENT_ARCH="amd64" ;;
    aarch64|arm64) CURRENT_ARCH="arm64" ;;
    *) echo -e "${RED}Unsupported architecture: $CURRENT_ARCH${NC}"; exit 1 ;;
esac

# Determine if building EE or CE
IS_EE=false
if [ -n "$GOWORK" ] && [[ "$GOWORK" == *"go.work.ee"* ]]; then
    IS_EE=true
    echo -e "${YELLOW}Building Enterprise Edition${NC}"
else
    echo -e "${YELLOW}Building Community Edition${NC}"
fi

# Target platforms
PLATFORMS=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
)

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

# Step 3: Build Go backend for each platform
cd ../core

echo -e "\n${GREEN}Step 3: Building Go backend for all platforms...${NC}"

for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "$platform"
    
    echo -e "\n${YELLOW}Building for $GOOS/$GOARCH...${NC}"
    
    # Set output name
    if [ "$GOOS" = "windows" ]; then
        OUTPUT="../desktop/src-tauri/binaries/whodb-$GOOS-$GOARCH.exe"
    else
        OUTPUT="../desktop/src-tauri/binaries/whodb-$GOOS-$GOARCH"
    fi
    
    # Set build tags
    BUILD_TAGS=""
    if [ "$IS_EE" = true ]; then
        BUILD_TAGS="-tags ee"
    fi
    
    # Disable CGO for cross-compilation from macOS
    CGO_ENABLED=1
    if [ "$CURRENT_OS" = "darwin" ] && [ "$GOOS" != "darwin" ]; then
        CGO_ENABLED=0
        echo -e "${YELLOW}  Note: CGO disabled for cross-compilation from macOS${NC}"
    fi
    
    # Build command
    if [ "$IS_EE" = true ]; then
        GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=$CGO_ENABLED \
            GOWORK=$GOWORK go build $BUILD_TAGS -o "$OUTPUT" .
    else
        GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=$CGO_ENABLED \
            go build -o "$OUTPUT" .
    fi
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}  ✓ Built successfully${NC}"
        ls -lh "$OUTPUT"
    else
        echo -e "${RED}  ✗ Build failed${NC}"
        exit 1
    fi
done

# Copy the current platform binary for development
echo -e "\n${GREEN}Step 4: Setting up development binary...${NC}"
cd ../desktop

if [ "$CURRENT_OS" = "windows" ]; then
    cp "src-tauri/binaries/whodb-$CURRENT_OS-$CURRENT_ARCH.exe" "src-tauri/binaries/whodb.exe"
else
    cp "src-tauri/binaries/whodb-$CURRENT_OS-$CURRENT_ARCH" "src-tauri/binaries/whodb"
    chmod +x "src-tauri/binaries/whodb"
fi

echo -e "${GREEN}Development binary ready${NC}"

echo -e "\n${GREEN}Prebuild complete!${NC}"
echo "All binaries built in: desktop/src-tauri/binaries/"
ls -la src-tauri/binaries/