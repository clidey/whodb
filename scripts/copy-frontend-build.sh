#!/bin/bash

# Script to copy frontend build to core directory
# This ensures the frontend build is available for the desktop app

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Check if we're in the right directory
if [ ! -d "frontend" ] || [ ! -d "core" ]; then
    echo "Error: This script must be run from the project root directory"
    exit 1
fi

# Check if frontend build exists
if [ ! -d "frontend/build" ]; then
    print_status "Frontend build not found. Building frontend first..."
    cd frontend
    pnpm run build
    cd ..
fi

# Copy frontend build to core directory
print_status "Copying frontend build to core directory..."
rm -rf core/build
cp -r frontend/build core/

print_success "Frontend build copied to core/build"

