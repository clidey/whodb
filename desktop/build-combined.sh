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

# Simple build script for Linux/Mac and Windows (via WSL)

set -euo pipefail

TARGET="${1:-linux-x64}"

echo "🚀 Building WhoDB Desktop for $TARGET..."

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Build main frontend first (desktop app depends on its CSS)
echo "📦 Building main frontend application..."
cd "$PROJECT_ROOT/frontend"

# Clean ALL old frontend build artifacts
echo "🧹 Cleaning frontend build artifacts..."
rm -rf build .cache node_modules/.cache

if [ ! -d "node_modules" ]; then
  echo "Installing frontend dependencies..."
  pnpm install
fi

# Force clean build
NODE_ENV=production pnpm run build

# Verify frontend build succeeded
if [ ! -f "build/index.html" ]; then
  echo "ERROR: Frontend build failed - build/index.html not found!"
  exit 1
fi
CSS_COUNT=$(find build/assets -name "*.css" 2>/dev/null | wc -l)
if [ "$CSS_COUNT" -eq 0 ]; then
  echo "ERROR: Frontend build failed - no CSS files found in build/assets!"
  exit 1
fi
echo "✓ Frontend build verified - found $CSS_COUNT CSS file(s)"

# Install desktop dependencies
echo "📦 Installing desktop dependencies..."
cd "$SCRIPT_DIR"

# Clean ALL old desktop build artifacts
echo "🧹 Cleaning desktop build artifacts..."
rm -rf dist .cache node_modules/.cache
# Clean Tauri build directories
if [ -d "src-tauri/target" ]; then
  echo "🧹 Cleaning Tauri target directory (this may take a moment)..."
  rm -rf src-tauri/target
fi

pnpm install

# Build desktop frontend with clean cache
echo "🔨 Building desktop frontend..."
NODE_ENV=production pnpm run build

# Verify desktop build succeeded and CSS was copied
if [ ! -f "dist/index.html" ]; then
  echo "ERROR: Desktop build failed - dist/index.html not found!"
  exit 1
fi
DESKTOP_CSS_COUNT=$(find dist/assets -name "*.css" 2>/dev/null | wc -l)
if [ "$DESKTOP_CSS_COUNT" -eq 0 ]; then
  echo "ERROR: Desktop build failed - no CSS files found in dist/assets!"
  echo "This usually means the frontend CSS wasn't copied properly."
  exit 1
fi
echo "✓ Desktop build verified - found $DESKTOP_CSS_COUNT CSS file(s)"

# Build Go backend FIRST (before Tauri needs it)
echo "🦀 Building backend..."
cd "$PROJECT_ROOT/core"

# Clean Go build cache to ensure fresh build (but keep module cache for speed)
echo "🧹 Cleaning Go build cache..."
go clean -cache -testcache

# Clean any existing binaries
BIN_DIR="$SCRIPT_DIR/src-tauri/bin"
if [ -d "$BIN_DIR" ]; then
  echo "🧹 Cleaning old backend binaries..."
  rm -rf "$BIN_DIR"
fi
mkdir -p "$BIN_DIR"

# Ensure fresh module downloads
echo "📦 Downloading Go modules..."
go mod download
go mod verify

case "$TARGET" in
  win-x64|windows)
    GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o "$BIN_DIR/whodb-core-x86_64-pc-windows-gnu.exe" .
    # Also copy with the expected name
    cp "$BIN_DIR/whodb-core-x86_64-pc-windows-gnu.exe" "$BIN_DIR/whodb-core.exe"
    ;;
  linux-x64|linux)
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o "$BIN_DIR/whodb-core" .
    touch "$BIN_DIR/.keep"
    ;;
  darwin|mac)
    # Build universal binary for Mac
    GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o "$BIN_DIR/whodb-core-x64" .
    GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o "$BIN_DIR/whodb-core-arm64" .
    lipo -create -output "$BIN_DIR/whodb-core" "$BIN_DIR/whodb-core-x64" "$BIN_DIR/whodb-core-arm64"
    rm "$BIN_DIR/whodb-core-x64" "$BIN_DIR/whodb-core-arm64"
    touch "$BIN_DIR/.keep"
    ;;
  *)
    echo "Unknown target: $TARGET. Use: win-x64, linux-x64, or darwin"
    exit 1
    ;;
esac

# Verify the binary exists and is fresh
echo "Binary created:"
ls -la "$BIN_DIR"

# Count binaries to ensure build succeeded
BIN_COUNT=$(find "$BIN_DIR" -type f -name "whodb-core*" | wc -l)
if [ "$BIN_COUNT" -eq 0 ]; then
  echo "ERROR: Backend build failed - no binaries found in $BIN_DIR"
  exit 1
fi
echo "✓ Backend build verified - found $BIN_COUNT binary file(s)"

# Build Tauri app
echo "📱 Building Tauri app..."
cd "$SCRIPT_DIR"

case "$TARGET" in
  win-x64|windows)
    pnpm run tauri:build -- --target x86_64-pc-windows-gnu
    ;;
  linux-x64|linux)
    pnpm run tauri:build -- --target x86_64-unknown-linux-gnu
    ;;
  darwin|mac)
    pnpm run tauri:build -- --target universal-apple-darwin
    ;;
esac

echo "✅ Build complete!"
