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

# Script to generate a Homebrew Cask formula for WhoDB
# Usage: ./generate-homebrew-cask.sh <version> <dmg-file-path> <output-file>

VERSION="${1:?Version is required}"
DMG_FILE="${2:?DMG file path is required}"
OUTPUT_FILE="${3:?Output file is required}"

if [ ! -f "$DMG_FILE" ]; then
    echo "Error: DMG file not found: $DMG_FILE"
    exit 1
fi

# Calculate SHA256
echo "Calculating SHA256 for DMG file..."
SHA256=$(shasum -a 256 "$DMG_FILE" | cut -d' ' -f1)

echo "Version: $VERSION"
echo "DMG File: $DMG_FILE"
echo "SHA256: $SHA256"

# Generate cask formula
cat > "$OUTPUT_FILE" << EOF
cask "whodb" do
  version "$VERSION"
  sha256 "$SHA256"

  url "https://github.com/clidey/whodb/releases/download/v#{version}/whodb.dmg"
  name "WhoDB"
  desc "Modern database management and visualization tool with AI integration"
  homepage "https://whodb.com/"

  auto_updates true

  app "WhoDB.app"

  livecheck do
    url "https://github.com/clidey/whodb/releases.atom"
    regex(/href=.*?\/v?(\d+(?:\.\d+)*)\//i)
    strategy :github_latest
  end

  uninstall quit:   "com.clidey.whodb.ce",
            signal: ["TERM", "com.clidey.whodb.ce"]

  zap trash: [
    "~/Library/Application Support/com.clidey.whodb.ce",
    "~/Library/Caches/com.clidey.whodb.ce",
    "~/Library/Preferences/com.clidey.whodb.ce.plist",
    "~/Library/Saved Application State/com.clidey.whodb.ce.savedState",
  ]

  caveats do
    "WhoDB is a database management tool. Visit https://whodb.com for documentation."
  end
end
EOF

echo "✅ Homebrew Cask formula generated: $OUTPUT_FILE"
echo ""
echo "Next steps:"
echo "1. Review the generated formula"
echo "2. Test it locally: brew install --cask ./$(basename $OUTPUT_FILE)"
echo "3. Submit to https://github.com/Homebrew/homebrew-core/pulls"
