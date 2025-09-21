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

set -e

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

echo "🦀 Building backend..."
cd "$PROJECT_ROOT/core"
go build -o "$SCRIPT_DIR/src-tauri/whodb-core"

echo "📱 Building Tauri app..."
cd "$SCRIPT_DIR"
pnpm run tauri:build

echo "✅ Build complete! The desktop app is ready in src-tauri/target/release/"
