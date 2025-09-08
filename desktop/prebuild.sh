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

# WhoDB Prebuild Script
# Builds the Go backend and frontend for all platforms before desktop packaging

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
PLATFORM=""
ARCH=""
BUILD_EE=false
CLEAN=false
BUILD_ALL=true  # Default to building for all platforms

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --platform PLATFORM    Target platform (darwin, linux, windows)"
    echo "  --arch ARCH            Target architecture (amd64, arm64)"
    echo "  --ee                   Build Enterprise Edition"
    echo "  --clean                Clean build directories before building"
    echo "  --all                  Build for all platforms and architectures (default)"
    echo "  -h, --help             Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 --platform darwin --arch arm64"
    echo "  $0 --platform linux --arch amd64"
    echo "  $0 --all"
    echo "  $0 --ee --all"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --platform)
            PLATFORM="$2"
            shift 2
            ;;
        --arch)
            ARCH="$2"
            shift 2
            ;;
        --ee)
            BUILD_EE=true
            shift
            ;;
        --clean)
            CLEAN=true
            shift
            ;;
        --all)
            BUILD_ALL=true
            shift
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Function to build Go backend
build_backend() {
    local platform=$1
    local arch=$2
    local is_ee=$3
    
    print_status "Building Go backend for $platform/$arch (EE: $is_ee)"
    
    cd ../core
    
    # Set environment variables for cross-compilation
    export GOOS=$platform
    export GOARCH=$arch
    
    # Handle CGO for cross-compilation
    if [ "$platform" = "linux" ] && [ "$(uname -s)" = "Darwin" ]; then
        # Cross-compiling to Linux from macOS - disable CGO
        export CGO_ENABLED=0
        print_status "Cross-compiling to Linux from macOS - CGO disabled"
    elif [ "$platform" = "windows" ] && [ "$(uname -s)" = "Darwin" ]; then
        # Cross-compiling to Windows from macOS - disable CGO
        export CGO_ENABLED=0
        print_status "Cross-compiling to Windows from macOS - CGO disabled"
    else
        # Native compilation or supported cross-compilation
        export CGO_ENABLED=1
    fi
    
    # Determine build tags and workspace
    local build_tags=""
    local gowork=""
    
    if [ "$is_ee" = true ]; then
        build_tags="-tags ee"
        gowork="GOWORK=$PWD/../go.work.ee"
    fi
    
    # Create dist directory if it doesn't exist
    mkdir -p dist
    
    # Determine executable name
    local executable_name="whodb"
    if [ "$platform" = "windows" ]; then
        executable_name="whodb.exe"
    else
        executable_name="whodb-$platform-$arch"
    fi
    
    # Build the executable
    if [ "$is_ee" = true ]; then
        eval "$gowork go build $build_tags -o dist/$executable_name ."
    else
        go build -o dist/$executable_name .
    fi
    
    if [ $? -eq 0 ]; then
        print_success "Backend built successfully: dist/$executable_name"
    else
        print_error "Backend build failed"
        exit 1
    fi
    
    cd ../desktop
}

# Function to build frontend
build_frontend() {
    local is_ee=$1
    
    print_status "Building frontend (EE: $is_ee)"
    
    cd ../frontend
    
    # Install dependencies if needed
    if [ ! -d "node_modules" ]; then
        print_status "Installing frontend dependencies..."
        pnpm install
    fi
    
    # Build frontend
    if [ "$is_ee" = true ]; then
        pnpm run build:ee
    else
        pnpm run build
    fi
    
    if [ $? -eq 0 ]; then
        print_success "Frontend built successfully"
    else
        print_error "Frontend build failed"
        exit 1
    fi
    
    # Copy frontend build to core directory
    print_status "Copying frontend build to core directory..."
    rm -rf ../core/build
    cp -r build ../core/
    
    print_success "Frontend build copied to core/build"
    
    cd ../desktop
}

# Function to clean build directories
clean_builds() {
    print_status "Cleaning build directories..."
    
    rm -rf ../core/dist/*
    rm -rf ../core/build/*
    rm -rf ../frontend/build/*
    
    print_success "Build directories cleaned"
}

# Main build logic
main() {
    print_status "Starting WhoDB prebuild process..."
    
    # Clean if requested
    if [ "$CLEAN" = true ]; then
        clean_builds
    fi
    
    # Build frontend first (needed for all builds)
    build_frontend $BUILD_EE
    
    # Determine what to build
    if [ "$BUILD_ALL" = true ]; then
        # Build for all platforms and architectures
        platforms=("darwin" "linux" "windows")
        arches=("amd64" "arm64")
        
        for platform in "${platforms[@]}"; do
            for arch in "${arches[@]}"; do
                # Skip unsupported combinations
                if [ "$platform" = "windows" ] && [ "$arch" = "arm64" ]; then
                    continue
                fi
                
                build_backend $platform $arch $BUILD_EE
            done
        done
    else
        # Use provided or default platform/arch
        if [ -z "$PLATFORM" ]; then
            PLATFORM=$(uname -s | tr '[:upper:]' '[:lower:]')
            if [ "$PLATFORM" = "darwin" ]; then
                PLATFORM="darwin"
            fi
        fi
        
        if [ -z "$ARCH" ]; then
            ARCH=$(uname -m)
            if [ "$ARCH" = "x86_64" ]; then
                ARCH="amd64"
            elif [ "$ARCH" = "arm64" ]; then
                ARCH="arm64"
            fi
        fi
        
        build_backend $PLATFORM $ARCH $BUILD_EE
    fi
    
    print_success "Prebuild process completed successfully!"
    print_status "All platform executables are now available in ../core/dist/"
    print_status "Note: Linux builds from macOS have CGO disabled due to cross-compilation limitations"
    print_status "For CGO-enabled Linux builds, build directly on a Linux system"
}

# Run main function
main

