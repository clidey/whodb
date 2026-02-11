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
# Screenshot Copy Script
# This script copies all generated screenshots from the Playwright screenshots
# directory to the docs/images directory for documentation purposes.

set -e

# Get the script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Define source and destination paths
SCREENSHOTS_SOURCE="$PROJECT_ROOT/frontend/e2e/screenshots/postgres"
DOCS_IMAGES_DEST="$PROJECT_ROOT/docs/images"

echo "📸 WhoDB Screenshot Copy Utility"
echo "=================================="
echo ""

# Check if source directory exists
if [ ! -d "$SCREENSHOTS_SOURCE" ]; then
    echo "❌ Error: Screenshots directory not found at:"
    echo "   $SCREENSHOTS_SOURCE"
    echo ""
    echo "💡 Tip: Run 'pnpm screenshot' first to generate screenshots"
    exit 1
fi

# Count screenshots
SCREENSHOT_COUNT=$(find "$SCREENSHOTS_SOURCE" -name "*.png" 2>/dev/null | wc -l | tr -d ' ')

if [ "$SCREENSHOT_COUNT" -eq 0 ]; then
    echo "❌ Error: No screenshots found in:"
    echo "   $SCREENSHOTS_SOURCE"
    echo ""
    echo "💡 Tip: Run 'pnpm screenshot' first to generate screenshots"
    exit 1
fi

echo "📁 Source: $SCREENSHOTS_SOURCE"
echo "📁 Destination: $DOCS_IMAGES_DEST"
echo "📊 Found $SCREENSHOT_COUNT screenshot(s)"
echo ""

# Create destination directory if it doesn't exist
if [ ! -d "$DOCS_IMAGES_DEST" ]; then
    echo "📂 Creating destination directory..."
    mkdir -p "$DOCS_IMAGES_DEST"
fi

# Copy screenshots
echo "🚀 Copying screenshots..."
cp -v "$SCREENSHOTS_SOURCE"/*.png "$DOCS_IMAGES_DEST/" 2>&1 | while read line; do
    # Extract just the filename for cleaner output
    filename=$(basename "$line" | sed "s/'//g" | awk '{print $NF}')
    echo "   ✓ $filename"
done

echo ""
echo "=================================="
echo "✅ Successfully copied $SCREENSHOT_COUNT screenshot(s) to docs/images/"
echo ""
echo "💡 Screenshots are now available at:"
echo "   $DOCS_IMAGES_DEST"
echo "=================================="
