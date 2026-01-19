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

// Package bamlconfig sets BAML environment defaults before the BAML native library loads.
// This package MUST be imported first (before any other imports) in main packages
// to ensure the environment variable is set before baml_client is imported.
//
// Usage in main.go:
//
//	import (
//		_ "github.com/clidey/whodb/core/src/bamlconfig" // Must be first!
//		// ... other imports
//	)
package bamlconfig

import (
	"os"
	"strings"
)

func init() {
	// Don't override if user explicitly set BAML_LOG
	if os.Getenv("BAML_LOG") != "" {
		return
	}

	// Map WHODB_LOG_LEVEL to BAML_LOG
	level := strings.ToLower(os.Getenv("WHODB_LOG_LEVEL"))

	var bamlLevel string
	switch level {
	case "debug":
		bamlLevel = "debug"
	case "info":
		bamlLevel = "info"
	case "warning", "warn":
		bamlLevel = "warn"
	case "error":
		bamlLevel = "error"
	case "none", "off", "disabled":
		bamlLevel = "off"
	default:
		// Default: only show errors (quieter output)
		bamlLevel = "error"
	}

	os.Setenv("BAML_LOG", bamlLevel)
}
