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

# Script to clean all coverage data (both frontend and backend)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "🧹 Cleaning all coverage data..."

# Clean backend Go coverage
if [ -f "$PROJECT_ROOT/core/coverage.out" ]; then
    rm "$PROJECT_ROOT/core/coverage.out"
    echo "✅ Backend coverage cleaned"
fi

if [ -f "$PROJECT_ROOT/core/coverage.tmp.out" ]; then
    rm "$PROJECT_ROOT/core/coverage.tmp.out"
fi

if [ -f "$PROJECT_ROOT/core/coverage.prev.out" ]; then
    rm "$PROJECT_ROOT/core/coverage.prev.out"
fi

# Clean frontend coverage
cd "$PROJECT_ROOT/frontend"
if [ -d ".nyc_output" ] || [ -d "coverage" ]; then
    npm run coverage:clean 2>/dev/null || rm -rf .nyc_output coverage
    echo "✅ Frontend coverage cleaned"
fi

echo "✨ All coverage data cleaned!"
echo ""
echo "Next steps:"
echo "  - Run 'pnpm cypress:ce:headless' to generate fresh coverage"
echo "  - Run 'npm run view:coverage' to see frontend coverage"
echo "  - Run 'npm run view:coverage:backend' to see backend coverage"