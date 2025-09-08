# Copyright 2025 Clidey, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#!/bin/bash

# Simple prebuild script for current platform
# Builds backend and frontend for current platform only

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

# Default values
BUILD_EE=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --ee)
            BUILD_EE=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --ee         Build Enterprise Edition"
            echo "  -h, --help   Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

print_status "Building WhoDB for current platform..."

# Build frontend
print_status "Building frontend..."
cd ../frontend

if [ "$BUILD_EE" = true ]; then
    pnpm run build:ee
else
    pnpm run build
fi

# Copy frontend build to core
print_status "Copying frontend build to core..."
rm -rf ../core/build
cp -r build ../core/

cd ../desktop

# Build backend
print_status "Building backend..."
cd ../core

# Determine build command
if [ "$BUILD_EE" = true ]; then
    GOWORK=$PWD/../go.work.ee go build -tags ee -o dist/whodb .
else
    go build -o dist/whodb .
fi

cd ../desktop

print_success "Backend and frontend built successfully!"
print_status "Current platform executable is now available in ../core/dist/whodb"

