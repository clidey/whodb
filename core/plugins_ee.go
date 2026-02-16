//go:build ee

/*
 * Copyright 2025 Clidey, Inc.
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

package main

import (
	// Import EE package to register EE plugins
	// This import will only work when using the EE workspace (ee/go.work)
	// which includes the ee module
	_ "github.com/clidey/whodb/ee/core/graph"
	_ "github.com/clidey/whodb/ee/core/src/plugins"
)
