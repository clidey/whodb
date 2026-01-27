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

// AIChatStreamHandlerFunc is the type for the AI chat stream handler
// EE can override this by registering its own implementation
type AIChatStreamHandlerFunc func(w http.ResponseWriter, r *http.Request)

// registeredAIChatStreamHandler holds the handler implementation
// Default is nil, will be set by CE or EE implementations
var registeredAIChatStreamHandler AIChatStreamHandlerFunc

// RegisterAIChatStreamHandler allows CE/EE implementations to register their handler
func RegisterAIChatStreamHandler(handler AIChatStreamHandlerFunc) {
	registeredAIChatStreamHandler = handler
}

// aiChatStreamHandler delegates to registered implementation
func aiChatStreamHandler(w http.ResponseWriter, r *http.Request) {
	if registeredAIChatStreamHandler != nil {
		registeredAIChatStreamHandler(w, r)
		return
	}
	http.Error(w, "AI chat streaming not available", http.StatusNotImplemented)
}
