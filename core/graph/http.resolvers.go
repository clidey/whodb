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

import "github.com/go-chi/chi/v5"

func init() {
	RegisterLongLivedHTTPRoute("/api/ai-chat/stream")
}

// SetupHTTPServer registers HTTP endpoints that are not served through GraphQL.
func SetupHTTPServer(router chi.Router) {
	router.Post("/api/ai-chat/stream", aiChatStreamHandler)
	router.Post("/api/export", HandleExport)
	registerExtensionHTTPRoutes(router)

	// AI chat streaming endpoint is registered via build tags in http_ai_stream.go (!arm) / http_ai_stream_arm.go (arm)
}
