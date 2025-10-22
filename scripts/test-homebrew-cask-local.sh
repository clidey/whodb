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

# Script to test Homebrew Cask locally
# Usage: ./test-homebrew-cask-local.sh <dmg-file-path> [version]

DMG_FILE="${1:?DMG file path is required}"
VERSION="${2:-1.0.0-test}"
TAP_NAME="test/local"
CASK_NAME="whodb"

if [ ! -f "$DMG_FILE" ]; then
    echo "Error: DMG file not found: $DMG_FILE"
    exit 1
fi

# Get absolute path of DMG
DMG_FILE_ABS=$(cd "$(dirname "$DMG_FILE")" && pwd)/$(basename "$DMG_FILE")

# Calculate SHA256
echo "Calculating SHA256 for DMG file..."
SHA256=$(shasum -a 256 "$DMG_FILE_ABS" | cut -d' ' -f1)

echo "DMG File: $DMG_FILE_ABS"
echo "SHA256: $SHA256"
echo "Version: $VERSION"
echo ""

# Check if tap exists, create if not
if ! brew tap | grep -q "^$TAP_NAME$"; then
    echo "Creating local tap: $TAP_NAME"
    brew tap-new $TAP_NAME
fi

# Get tap path
TAP_PATH=$(brew --repository)/Library/Taps/${TAP_NAME/\///homebrew-}

# Create Casks directory if it doesn't exist
mkdir -p "$TAP_PATH/Casks"

# Generate local test cask
CASK_FILE="$TAP_PATH/Casks/$CASK_NAME.rb"
echo "Generating local test cask: $CASK_FILE"

cat > "$CASK_FILE" << EOF
cask "$CASK_NAME" do
  version "$VERSION"
  sha256 "$SHA256"

  url "file://$DMG_FILE_ABS"
  name "WhoDB"
  desc "Modern database management and visualization tool with AI integration"
  homepage "https://whodb.com/"

  auto_updates true

  app "WhoDB.app"

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

echo "✅ Local test cask generated"
echo ""
echo "To test the cask:"
echo "  Install:   brew install --cask $TAP_NAME/$CASK_NAME"
echo "  Uninstall: brew uninstall --cask $CASK_NAME"
echo "  Reinstall: brew reinstall --cask $CASK_NAME"
echo ""
echo "To remove the test tap when done:"
echo "  brew untap $TAP_NAME"