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

// http_ai_stream_unsupported.go provides stub implementations for armv7 platforms
// where BAML is not supported. AI streaming features will return user-friendly errors.

package graph

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// aiChatStreamHandler returns an error on unsupported platforms
func aiChatStreamHandler(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send error via SSE
	errorData := map[string]string{
		"error": "AI features are not supported on this platform (arm/riscv64). BAML requires amd64 or arm64 architecture.",
	}
	data, _ := json.Marshal(errorData)
	fmt.Fprintf(w, "event: error\ndata: %s\n\n", data)
	flusher.Flush()
}
