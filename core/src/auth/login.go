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

package auth

import (
	"context"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

func Login(ctx context.Context, input *model.LoginCredentials) (*model.StatusResponse, error) {
	log.Debugf("[Login] type=%s, hostname=%s, username=%s, database=%s, advanced=%d",
		input.Type, input.Hostname, input.Username, input.Database, len(input.Advanced))

	// Note: We no longer set cookies for authentication.
	// Credentials are sent via Authorization header on each request. This avoids the ~4KB cookie size
	// limit which can be exceeded when SSL certificates are included.
	log.Debugf("[Login] Login successful for %s at %s", input.Type, input.Hostname)

	// Persist credentials in OS keychain when an Id is provided (for future use)
	if input.ID != nil && *input.ID != "" {
		advanced := make([]engine.Record, 0, len(input.Advanced))
		for _, rec := range input.Advanced {
			advanced = append(advanced, engine.Record{Key: rec.Key, Value: rec.Value})
		}
		if err := SaveCredentials(*input.ID, &engine.Credentials{
			Id:        input.ID,
			Type:      input.Type,
			Hostname:  input.Hostname,
			Username:  input.Username,
			Password:  input.Password,
			Database:  input.Database,
			Advanced:  advanced,
			IsProfile: false,
		}); err != nil {
			warnKeyringUnavailableOnce(err)
		}
	}

	return &model.StatusResponse{
		Status: true,
	}, nil
}
