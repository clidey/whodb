#!/usr/bin/env bash
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

set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

run_tidy() {
  local label="$1"
  local module_dir="$2"

  echo "Running go mod tidy in $label"
  (
    cd "$module_dir"
    go mod tidy
  )
}

run_tidy "CE core" "$ROOT_DIR/core"
run_tidy "CE cli" "$ROOT_DIR/cli"
run_tidy "CE desktop-common" "$ROOT_DIR/desktop-common"
run_tidy "CE desktop-ce" "$ROOT_DIR/desktop-ce"

if [ -f "$ROOT_DIR/ee/go.mod" ]; then
  run_tidy "EE root" "$ROOT_DIR/ee"
  run_tidy "EE cli" "$ROOT_DIR/ee/cli"
  run_tidy "EE desktop" "$ROOT_DIR/ee/desktop"
else
  echo "EE module not present, skipping EE go mod tidy"
fi
