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

# Copy the two canonical variants used by documentation and README pictures.
for theme in light dark; do
    THEME_SOURCE="$SCREENSHOTS_SOURCE/$theme"
    if [ ! -d "$THEME_SOURCE" ]; then
        echo "Missing $theme screenshots. Run pnpm screenshot first."
        exit 1
    fi
    mkdir -p "$DOCS_IMAGES_DEST/$theme"
    cp "$THEME_SOURCE"/*.png "$DOCS_IMAGES_DEST/$theme/"
done
echo "Copied light and dark screenshots to $DOCS_IMAGES_DEST"
