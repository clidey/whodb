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

import (
	"sync"

	"github.com/go-chi/chi/v5"
)

// HTTPRouteRegistrar registers HTTP routes that are not served through GraphQL.
type HTTPRouteRegistrar func(chi.Router)

var (
	httpRouteRegistrarsMu sync.RWMutex
	httpRouteRegistrars   []HTTPRouteRegistrar
)

// RegisterHTTPRoutes adds an HTTP route registrar for platform or edition extensions.
func RegisterHTTPRoutes(register HTTPRouteRegistrar) {
	if register == nil {
		return
	}
	httpRouteRegistrarsMu.Lock()
	defer httpRouteRegistrarsMu.Unlock()
	httpRouteRegistrars = append(httpRouteRegistrars, register)
}

func registerExtensionHTTPRoutes(router chi.Router) {
	httpRouteRegistrarsMu.RLock()
	registrars := append([]HTTPRouteRegistrar(nil), httpRouteRegistrars...)
	httpRouteRegistrarsMu.RUnlock()

	for _, register := range registrars {
		register(router)
	}
}
