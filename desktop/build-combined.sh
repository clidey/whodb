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

# Install dependencies
echo "📦 Installing dependencies..."
cd "$SCRIPT_DIR"
pnpm install

# Build frontend
echo "🔨 Building frontend..."
pnpm run build

# Build Go backend FIRST (before Tauri needs it)
echo "🦀 Building backend..."
cd "$PROJECT_ROOT/core"

BIN_DIR="$SCRIPT_DIR/src-tauri/bin"
mkdir -p "$BIN_DIR"

# Clear old binaries
rm -f "$BIN_DIR"/*

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

# Verify the binary exists
echo "Binary created:"
ls -la "$BIN_DIR"

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
