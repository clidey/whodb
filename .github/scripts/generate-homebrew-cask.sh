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

set -e

# Script to generate Homebrew cask file from template
# Usage: ./generate-homebrew-cask.sh <version> <dmg-path> [output-file]

VERSION="${1}"
DMG_PATH="${2}"
OUTPUT_FILE="${3:-whodb.rb}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATE_FILE="${SCRIPT_DIR}/../templates/homebrew-cask.rb.template"

if [ -z "$VERSION" ]; then
  echo "❌ Error: Version is required"
  echo "Usage: $0 <version> <dmg-path> [output-file]"
  exit 1
fi

if [ -z "$DMG_PATH" ]; then
  echo "❌ Error: DMG path is required"
  echo "Usage: $0 <version> <dmg-path> [output-file]"
  exit 1
fi

if [ ! -f "$DMG_PATH" ]; then
  echo "❌ Error: DMG file not found: $DMG_PATH"
  exit 1
fi

if [ ! -f "$TEMPLATE_FILE" ]; then
  echo "❌ Error: Template file not found: $TEMPLATE_FILE"
  exit 1
fi

echo "📝 Generating Homebrew cask for WhoDB v${VERSION}"
echo "📦 Using DMG: $DMG_PATH"
echo "📄 Template: $TEMPLATE_FILE"
echo "💾 Output: $OUTPUT_FILE"
echo ""

# Calculate SHA256
echo "🔐 Calculating SHA256..."
SHA256=$(shasum -a 256 "$DMG_PATH" | cut -d' ' -f1)
echo "SHA256: $SHA256"
echo ""

# Generate cask from template
echo "✍️  Generating cask file..."
sed -e "s/{{VERSION}}/${VERSION}/g" \
    -e "s/{{SHA256}}/${SHA256}/g" \
    "$TEMPLATE_FILE" > "$OUTPUT_FILE"

echo "✅ Generated $OUTPUT_FILE"
echo ""
cat "$OUTPUT_FILE"
