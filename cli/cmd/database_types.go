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

package cmd

import (
	"github.com/clidey/whodb/cli/internal/sourcetypes"
	"github.com/clidey/whodb/core/src/source"
)

func lookupDatabaseType(input string) (source.TypeSpec, bool) {
	return sourcetypes.Find(input)
}

func getDefaultPort(dbType string) int {
	port, ok := sourcetypes.DefaultPort(dbType)
	if !ok {
		return 0
	}
	return port
}

func isFileBasedDatabaseType(dbType string) bool {
	spec, ok := lookupSourceTypeSpec(dbType)
	if !ok {
		return false
	}
	return spec.Traits.Connection.Transport == source.ConnectionTransportFile
}

func lookupSourceTypeSpec(input string) (source.TypeSpec, bool) {
	return sourcetypes.Find(input)
}

func isConnectionFieldRequired(dbType string, key string) bool {
	return sourcetypes.ConnectionFieldRequired(dbType, key)
}
