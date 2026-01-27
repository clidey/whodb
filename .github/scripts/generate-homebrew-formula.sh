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

# Script to generate Homebrew formula for WhoDB CLI (source build)
# Usage: ./generate-homebrew-formula.sh <version> [output-file]

VERSION="${1}"
OUTPUT_FILE="${2:-whodb-cli.rb}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATE_FILE="${SCRIPT_DIR}/../templates/homebrew-formula.rb.template"

if [ -z "$VERSION" ]; then
  echo "Error: Version is required"
  echo "Usage: $0 <version> [output-file]"
  exit 1
fi

if [ ! -f "$TEMPLATE_FILE" ]; then
  echo "Error: Template file not found: $TEMPLATE_FILE"
  exit 1
fi

echo "Generating Homebrew formula for WhoDB CLI v${VERSION}"
echo "Template: $TEMPLATE_FILE"
echo "Output: $OUTPUT_FILE"
echo ""

# Download source tarball and calculate SHA256
TARBALL_URL="https://github.com/clidey/whodb/archive/refs/tags/${VERSION}.tar.gz"
echo "Downloading source tarball: $TARBALL_URL"

TMP_FILE=$(mktemp)
trap "rm -f $TMP_FILE" EXIT

if curl -fsSL -o "$TMP_FILE" "$TARBALL_URL"; then
  SHA256=$(shasum -a 256 "$TMP_FILE" | cut -d' ' -f1)
  echo "SHA256: $SHA256"
else
  echo "Error: Failed to download tarball. Release may not exist yet."
  echo "Using placeholder SHA256 - you'll need to update this after release."
  SHA256="PLACEHOLDER_UPDATE_AFTER_RELEASE"
fi

echo ""

# Generate formula from template
echo "Generating formula..."
sed -e "s/{{VERSION}}/${VERSION}/g" \
    -e "s/{{SHA256}}/${SHA256}/g" \
    "$TEMPLATE_FILE" > "$OUTPUT_FILE"

echo "Generated $OUTPUT_FILE"
echo ""
echo "Formula contents:"
echo "----------------------------------------"
cat "$OUTPUT_FILE"
echo "----------------------------------------"
echo ""
echo "Next steps for homebrew-core submission:"
echo "1. Fork https://github.com/Homebrew/homebrew-core"
echo "2. Add this file to Formula/w/whodb-cli.rb"
echo "3. Test locally: brew install --build-from-source ./whodb-cli.rb"
echo "4. Run: brew audit --new whodb-cli"
echo "5. Submit PR to homebrew-core"
