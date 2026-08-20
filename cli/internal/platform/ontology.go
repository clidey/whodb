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

import "fmt"

// DefaultOntologyStorageMode is the storage mode used by CLI ontology creates when omitted.
const DefaultOntologyStorageMode = "operational"

// ValidateOntologyStorageMode validates the immutable storage class accepted by the platform.
func ValidateOntologyStorageMode(storageMode string) error {
	switch storageMode {
	case "operational", "analytical":
		return nil
	default:
		return fmt.Errorf("storage mode must be operational or analytical, got %q", storageMode)
	}
}
