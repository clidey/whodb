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

# Fix line endings for shell scripts to prevent \r issues

set -e

echo "🔧 Fixing line endings for shell scripts..."

# Get the script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Fix line endings in all shell scripts
find "$PROJECT_ROOT" -name "*.sh" -type f -exec sed -i 's/\r$//' {} \;

echo "✅ Fixed line endings for all shell scripts"
echo "💡 To prevent this issue:"
echo "   1. Make sure your editor uses LF line endings for .sh files"
echo "   2. Configure git: git config core.autocrlf input"
echo "   3. The .gitattributes file will help prevent future issues"