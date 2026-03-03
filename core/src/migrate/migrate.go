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

// Package migrate tracks deprecated configuration patterns and collects
// migration warnings at startup. When a deprecation period ends, remove
// the corresponding flag and warning from this file.
//
// This package intentionally has no dependencies to avoid import cycles.
// Flags are set by other packages during parsing; warnings are collected
// and logged by the caller (typically src.InitializeEngine).
package migrate

// DeprecatedConfigKey is set when a database credential env var uses
// the legacy "config" JSON key instead of "advanced".
var DeprecatedConfigKey bool

// DeprecatedOpenAICompatibleEnv is set when legacy WHODB_OPENAI_COMPATIBLE_*
// or WHODB_CUSTOM_MODELS env vars are detected.
var DeprecatedOpenAICompatibleEnv bool

// Warnings returns migration messages for any active deprecations.
func Warnings() []string {
	var warnings []string
	if DeprecatedConfigKey {
		warnings = append(warnings, `Deprecated: database connection profile uses "config" key — please rename to "advanced". The "config" key will be removed in a future release.`)
	}
	if DeprecatedOpenAICompatibleEnv {
		warnings = append(warnings, `Deprecated: WHODB_OPENAI_COMPATIBLE_ENDPOINT, WHODB_OPENAI_COMPATIBLE_API_KEY, and WHODB_CUSTOM_MODELS are deprecated and no longer have any effect. Use WHODB_AI_GENERIC_<ID>_* environment variables instead.`)
	}
	return warnings
}
