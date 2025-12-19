/*
 * // Copyright 2025 Clidey, Inc.
 * //
 * // Licensed under the Apache License, Version 2.0 (the "License");
 * // you may not use this file except in compliance with the License.
 * // You may obtain a copy of the License at
 * //
 * //     http://www.apache.org/licenses/LICENSE-2.0
 * //
 * // Unless required by applicable law or agreed to in writing, software
 * // distributed under the License is distributed on an "AS IS" BASIS,
 * // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * // See the License for the specific language governing permissions and
 * // limitations under the License.
 */

package mongodb

import (
	"errors"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
)

// handleMongoError converts low-level MongoDB errors into user-friendly messages.
func handleMongoError(err error) error {
	if err == nil {
		return nil
	}

	var we mongo.WriteException
	if errors.As(err, &we) {
		for _, werr := range we.WriteErrors {
			switch werr.Code {
			case 11000:
				return fmt.Errorf("duplicate key: a document with the same identifier already exists")
			default:
				return fmt.Errorf("write error (%d): %s", werr.Code, werr.Message)
			}
		}
	}

	var ce mongo.CommandError
	if errors.As(err, &ce) {
		switch ce.Code {
		case 48: // NamespaceExists
			return fmt.Errorf("collection already exists")
		case 121: // Document validation failure
			return fmt.Errorf("document validation failed: %s", ce.Message)
		}
		return fmt.Errorf("command error (%d): %s", ce.Code, ce.Message)
	}

	// Default: sanitized error text
	msg := err.Error()
	msg = strings.ReplaceAll(msg, "\n", " ")
	return fmt.Errorf("mongodb operation failed: %s", msg)
}
