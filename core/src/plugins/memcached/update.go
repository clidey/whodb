// Copyright 2025 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package memcached

import (
	"errors"

	"github.com/clidey/whodb/core/src/plugins"
)

func Update(config *plugins.Config, storageUnit string, values map[string]string, updatedColumns []string) (bool, error) {
	// Memcached doesn't support updating values in place
	// You can only set new values
	return false, errors.New("update operation is not supported for Memcached - use set to replace values")
}