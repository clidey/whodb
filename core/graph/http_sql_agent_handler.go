//go:build !arm && !riscv64

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

package graph

import "net/http"

// SQLAgentHandlerFunc is the type for the SQL agent stream handler.
// EE registers its implementation via RegisterSQLAgentHandler.
type SQLAgentHandlerFunc func(w http.ResponseWriter, r *http.Request)

var registeredSQLAgentHandler SQLAgentHandlerFunc
var registeredPermitHandler SQLAgentHandlerFunc

// RegisterSQLAgentHandler allows EE to register its SQL agent stream handler.
func RegisterSQLAgentHandler(handler SQLAgentHandlerFunc) {
	registeredSQLAgentHandler = handler
}

// RegisterPermitHandler allows EE to register its SQL agent permission handler.
func RegisterPermitHandler(handler SQLAgentHandlerFunc) {
	registeredPermitHandler = handler
}

// sqlAgentStreamHandler delegates to the registered implementation.
func sqlAgentStreamHandler(w http.ResponseWriter, r *http.Request) {
	if registeredSQLAgentHandler != nil {
		registeredSQLAgentHandler(w, r)
		return
	}
	http.Error(w, "SQL Agent not available in this edition", http.StatusNotImplemented)
}

// sqlAgentPermitHandler delegates to the registered implementation.
func sqlAgentPermitHandler(w http.ResponseWriter, r *http.Request) {
	if registeredPermitHandler != nil {
		registeredPermitHandler(w, r)
		return
	}
	http.Error(w, "SQL Agent not available in this edition", http.StatusNotImplemented)
}
