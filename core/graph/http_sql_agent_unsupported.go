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

// http_sql_agent_unsupported.go provides a stub for arm/riscv64 platforms
// where the SQL Agent is not supported.

package graph

import "net/http"

// sqlAgentStreamHandler returns not-implemented on unsupported platforms.
func sqlAgentStreamHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "SQL Agent not available on this platform", http.StatusNotImplemented)
}

// sqlAgentPermitHandler returns not-implemented on unsupported platforms.
func sqlAgentPermitHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "SQL Agent not available on this platform", http.StatusNotImplemented)
}
