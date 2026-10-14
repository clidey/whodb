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

package platform

import "sync/atomic"

// The platform records which first-party surface made each request in its
// audit ledger and analytics. The label is sent on every request as the
// audit.ClientHeader header. Commands and the MCP server set it at startup.
var clientNameValue atomic.Pointer[string]

// SetClientName sets the label this process sends to identify itself, for
// example "whodb-cli/1.2.3" or "whodb-cli-mcp/1.2.3".
func SetClientName(name string) {
	clientNameValue.Store(&name)
}

func clientName() string {
	if name := clientNameValue.Load(); name != nil && *name != "" {
		return *name
	}
	return "whodb-cli"
}
