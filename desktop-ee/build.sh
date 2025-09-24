#!/bin/bash
#
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
#

# Build script for WhoDB Desktop - both CE and EE editions

set -e

# Parse command line arguments
EDITION="ce"
if [ "$1" = "ee" ]; then
    EDITION="ee"
fi

echo "Building WhoDB Desktop Application - ${EDITION^^} Edition..."

# Set workspace based on edition
if [ "$EDITION" = "ee" ]; then
    WORKSPACE="../go.work.desktop-ee"
    BUILD_TAGS="-tags ee"
    OUTPUT_PREFIX="whodb-ee"
    BUILD_CMD="build:ee"
else
    WORKSPACE="../go.work.desktop-ce"
    BUILD_TAGS=""
    OUTPUT_PREFIX="whodb-ce"
    BUILD_CMD="build:ce"
fi

# Build frontend first
echo "Building ${EDITION^^} frontend..."
cd ../frontend
pnpm install
pnpm run $BUILD_CMD

cd ../desktop-ee

# Windows builds
echo "Building ${EDITION^^} for Windows AMD64..."
GOWORK=$WORKSPACE wails build -clean -platform windows/amd64 \
    $BUILD_TAGS \
    -windowsconsole=false \
    -ldflags="-s -w" \
    -o ${OUTPUT_PREFIX}-windows-amd64.exe

echo "Building ${EDITION^^} for Windows ARM64..."
GOWORK=$WORKSPACE wails build -clean -platform windows/arm64 \
    $BUILD_TAGS \
    -windowsconsole=false \
    -ldflags="-s -w" \
    -o ${OUTPUT_PREFIX}-windows-arm64.exe

# macOS builds
echo "Building ${EDITION^^} for macOS Universal..."
GOWORK=$WORKSPACE wails build -clean -platform darwin/universal \
    $BUILD_TAGS \
    -ldflags="-s -w" \
    -o ${OUTPUT_PREFIX}-macos

# Linux builds
echo "Building ${EDITION^^} for Linux AMD64..."
GOWORK=$WORKSPACE wails build -clean -platform linux/amd64 \
    $BUILD_TAGS \
    -ldflags="-s -w" \
    -o ${OUTPUT_PREFIX}-linux-amd64

echo "Building ${EDITION^^} for Linux ARM64..."
GOWORK=$WORKSPACE wails build -clean -platform linux/arm64 \
    $BUILD_TAGS \
    -ldflags="-s -w" \
    -o ${OUTPUT_PREFIX}-linux-arm64

echo "Build complete! ${EDITION^^} binaries are in build/bin/"
echo ""
echo "Usage: ./build.sh [ce|ee]"
echo "  ce - Build Community Edition (default)"
echo "  ee - Build Enterprise Edition"