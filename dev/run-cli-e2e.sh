#!/bin/bash
#
# Copyright 2026 Clidey, Inc.
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

# Get the script directory (so it works from any location)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "Running CLI E2E tests..."

# Setup
bash "$SCRIPT_DIR/setup-cli-e2e.sh"

# Run tests
echo "Running CLI E2E tests..."
cd "$PROJECT_ROOT/cli"
go test -tags=e2e_postgres -v ./e2e/...
RESULT=$?

# Cleanup
bash "$SCRIPT_DIR/cleanup-cli-e2e.sh"

exit $RESULT
