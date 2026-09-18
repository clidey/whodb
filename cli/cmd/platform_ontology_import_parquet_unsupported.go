//go:build arm || riscv64

/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"context"
	"errors"
)

// readOntologyImportParquet is unavailable on architectures without DuckDB
// prebuilt libraries (32-bit ARM, riscv64).
func readOntologyImportParquet(_ context.Context, _ string) ([]map[string]any, error) {
	return nil, errors.New("parquet import is not supported on this platform; convert the file to CSV, JSON, or NDJSON")
}
