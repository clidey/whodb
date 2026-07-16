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

package analytics

import "strings"

// ErrorCode maps raw errors into a stable low-cardinality taxonomy.
func ErrorCode(err error) string {
	if err == nil {
		return ""
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(message, "unauthenticated"):
		return "unauthenticated"
	case strings.Contains(message, "unauthorized"), strings.Contains(message, "forbidden"), strings.Contains(message, "denied"):
		return "access_denied"
	case strings.Contains(message, "not found"), strings.Contains(message, "does not exist"):
		return "not_found"
	case strings.Contains(message, "quota"), strings.Contains(message, "limit reached"):
		return "quota_exceeded"
	case strings.Contains(message, "rate limit"):
		return "rate_limited"
	case strings.Contains(message, "validation"), strings.Contains(message, "invalid"):
		return "invalid_input"
	case strings.Contains(message, "timeout"), strings.Contains(message, "deadline"):
		return "timeout"
	case strings.Contains(message, "connection"):
		return "connection_failed"
	case strings.Contains(message, "provider"):
		return "provider_failed"
	case strings.Contains(message, "execution"), strings.Contains(message, "execute"):
		return "execution_failed"
	default:
		return "internal_error"
	}
}
