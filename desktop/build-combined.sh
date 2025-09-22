#!/bin/bash

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

set -euo pipefail

TARGET="darwin-x64"
if [[ ${1:-} == "--target" && -n ${2:-} ]]; then
  TARGET="$2"
fi

echo "🎯 Target: $TARGET"

echo "🚀 Building WhoDB Desktop Application..."

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "📦 Installing dependencies..."
cd "$SCRIPT_DIR"
pnpm install

echo "📦 Installing frontend dependencies..."
cd "$PROJECT_ROOT/frontend"
pnpm install

echo "🔨 Building frontend..."
cd "$SCRIPT_DIR"
pnpm run build

echo "🦀 Building backend (Go)..."
cd "$PROJECT_ROOT/core"

BIN_DIR="$SCRIPT_DIR/src-tauri/bin"
mkdir -p "$BIN_DIR"

build_go() {
  local goos="$1"; local goarch="$2"; local out="$3"; local tags=""
  echo "  - $goos/$goarch -> $out"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -o "$out" .
}

case "$TARGET" in
  darwin-x64)
    build_go darwin amd64 "$BIN_DIR/whodb-core"
    ;;
  darwin-arm64)
    build_go darwin arm64 "$BIN_DIR/whodb-core"
    ;;
  win-x64)
    build_go windows amd64 "$BIN_DIR/whodb-core.exe"
    ;;
  linux-x64)
    build_go linux amd64 "$BIN_DIR/whodb-core"
    ;;
  linux-arm64)
    build_go linux arm64 "$BIN_DIR/whodb-core"
    ;;
  linux-all)
    build_go linux amd64 "$BIN_DIR/whodb-core"
    build_go linux arm64 "$BIN_DIR/whodb-core-arm64"
    ;;
  all)
    build_go darwin amd64 "$BIN_DIR/whodb-core"
    build_go darwin arm64 "$BIN_DIR/whodb-core-arm64"
    build_go windows amd64 "$BIN_DIR/whodb-core.exe"
    build_go linux amd64 "$BIN_DIR/whodb-core"
    build_go linux arm64 "$BIN_DIR/whodb-core-arm64"
    ;;
  *)
    echo "Unknown target: $TARGET" >&2
    exit 1
    ;;
esac

echo "📱 Building Tauri app..."
cd "$SCRIPT_DIR"

TAURI_TARGETS=""
case "$TARGET" in
  darwin-*) TAURI_TARGETS="--target aarch64-apple-darwin --target x86_64-apple-darwin" ;;
  win-*) TAURI_TARGETS="--target x86_64-pc-windows-msvc" ;;
  linux-x64) TAURI_TARGETS="--target x86_64-unknown-linux-gnu" ;;
  linux-arm64) TAURI_TARGETS="--target aarch64-unknown-linux-gnu" ;;
  linux-all) TAURI_TARGETS="--target x86_64-unknown-linux-gnu --target aarch64-unknown-linux-gnu" ;;
  all) TAURI_TARGETS="--target aarch64-apple-darwin --target x86_64-apple-darwin --target x86_64-pc-windows-msvc --target x86_64-unknown-linux-gnu --target aarch64-unknown-linux-gnu" ;;
esac

pnpm run tauri:build -- $TAURI_TARGETS

echo "✅ Build complete! The desktop app is ready in src-tauri/target/release/"
