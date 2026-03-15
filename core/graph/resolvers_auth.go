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
	"context"
	"errors"
	"strings"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/analytics"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
)

// Login is the resolver for the Login field.
func (r *mutationResolver) Login(ctx context.Context, credentials model.LoginCredentials) (*model.StatusResponse, error) {
	if env.DisableCredentialForm {
		log.WithFields(log.Fields{
			"type":     credentials.Type,
			"hostname": credentials.Hostname,
			"username": credentials.Username,
			"database": credentials.Database,
		}).Error("Login with credentials is disabled; use preconfigured connections")
		return nil, errors.New("login with credentials is disabled; use preconfigured connections")
	}

	advanced := make([]engine.Record, 0, len(credentials.Advanced))
	for _, recordInput := range credentials.Advanced {
		advanced = append(advanced, engine.Record{
			Key:   recordInput.Key,
			Value: recordInput.Value,
		})
	}

	hasProfileID := credentials.ID != nil && strings.TrimSpace(*credentials.ID) != ""
	identity := strings.TrimSpace(analytics.MetadataFromContext(ctx).DistinctID)
	hasIdentity := identity != "" && identity != "disabled"

	if hasIdentity {
		analytics.CaptureWithDistinctID(ctx, identity, "login.attempt", map[string]any{
			"database_type":      credentials.Type,
			"profile_id_present": hasProfileID,
		})
	}

	if !src.MainEngine.Choose(engine.DatabaseType(credentials.Type)).IsAvailable(ctx, &engine.PluginConfig{
		Credentials: &engine.Credentials{
			Type:     credentials.Type,
			Hostname: credentials.Hostname,
			Username: credentials.Username,
			Password: credentials.Password,
			Database: credentials.Database,
			Advanced: advanced,
		},
	}) {
		log.WithFields(log.Fields{
			"type":     credentials.Type,
			"hostname": credentials.Hostname,
			"username": credentials.Username,
			"database": credentials.Database,
		}).Error("Database connection failed during login - credentials unauthorized")

		if hasIdentity {
			analytics.CaptureWithDistinctID(ctx, identity, "login.denied", map[string]any{
				"database_type":      credentials.Type,
				"profile_id_present": hasProfileID,
			})
		}
		return nil, errors.New("unauthorized")
	}

	resp, err := auth.Login(ctx, &credentials)
	if err != nil {
		if hasIdentity {
			analytics.CaptureError(ctx, "login.execute", err, map[string]any{
				"database_type":      credentials.Type,
				"profile_id_present": hasProfileID,
			})
		}
		return nil, err
	}

	if hasIdentity {
		traits := map[string]any{
			"profile_id_present": hasProfileID,
		}
		if hashedHost := analytics.HashIdentifier(credentials.Hostname); hashedHost != "" {
			traits["hostname_hash"] = hashedHost
		}
		if hashedDatabase := analytics.HashIdentifier(credentials.Database); hashedDatabase != "" {
			traits["database_hash"] = hashedDatabase
		}

		analytics.IdentifyWithDistinctID(ctx, identity, traits)
		analytics.CaptureWithDistinctID(ctx, identity, "login.success", map[string]any{
			"database_type":      credentials.Type,
			"profile_id_present": hasProfileID,
		})
	}

	return resp, nil
}

// LoginWithProfile is the resolver for the LoginWithProfile field.
func (r *mutationResolver) LoginWithProfile(ctx context.Context, profile model.LoginProfileInput) (*model.StatusResponse, error) {
	profiles := src.GetLoginProfiles()
	for i, loginProfile := range profiles {
		profileId := src.GetLoginProfileId(i, loginProfile)
		if profile.ID == profileId {

			resolved := src.GetLoginCredentials(loginProfile)
			credentials := &model.LoginCredentials{
				ID:       &profile.ID,
				Type:     resolved.Type,
				Hostname: resolved.Hostname,
				Username: resolved.Username,
				Password: resolved.Password,
				Database: resolved.Database,
				Advanced: func() []*model.RecordInput {
					out := make([]*model.RecordInput, 0, len(resolved.Advanced))
					for _, rec := range resolved.Advanced {
						out = append(out, &model.RecordInput{Key: rec.Key, Value: rec.Value})
					}
					return out
				}(),
			}
			if profile.Database != nil && *profile.Database != "" {
				credentials.Database = *profile.Database
				resolved.Database = credentials.Database
			}

			identity := strings.TrimSpace(analytics.MetadataFromContext(ctx).DistinctID)
			hasIdentity := identity != "" && identity != "disabled"

			if hasIdentity {
				analytics.CaptureWithDistinctID(ctx, identity, "login_with_profile.attempt", map[string]any{
					"database_type":  loginProfile.Type,
					"profile_source": loginProfile.Source,
				})
			}

			if !src.MainEngine.Choose(engine.DatabaseType(loginProfile.Type)).IsAvailable(ctx, &engine.PluginConfig{
				Credentials: resolved,
			}) {
				log.WithFields(log.Fields{
					"profile_id": profile.ID,
					"type":       loginProfile.Type,
				}).Error("Database connection failed for login profile - credentials unauthorized")

				if hasIdentity {
					analytics.CaptureWithDistinctID(ctx, identity, "login_with_profile.denied", map[string]any{
						"database_type":  loginProfile.Type,
						"profile_source": loginProfile.Source,
					})
				}
				return nil, errors.New("unauthorized")
			}

			resp, err := auth.Login(ctx, credentials)
			if err != nil {
				if hasIdentity {
					analytics.CaptureError(ctx, "login_with_profile.execute", err, map[string]any{
						"database_type":  loginProfile.Type,
						"profile_source": loginProfile.Source,
					})
				}
				return nil, err
			}

			if hasIdentity {
				traits := map[string]any{
					"profile_source": loginProfile.Source,
					"saved_profile":  true,
				}
				if hashedHost := analytics.HashIdentifier(credentials.Hostname); hashedHost != "" {
					traits["hostname_hash"] = hashedHost
				}
				if hashedDatabase := analytics.HashIdentifier(credentials.Database); hashedDatabase != "" {
					traits["database_hash"] = hashedDatabase
				}

				analytics.IdentifyWithDistinctID(ctx, identity, traits)
				analytics.CaptureWithDistinctID(ctx, identity, "login_with_profile.success", map[string]any{
					"database_type":  loginProfile.Type,
					"profile_source": loginProfile.Source,
				})
			}

			return resp, nil
		}
	}
	log.WithFields(log.Fields{
		"profile_id": profile.ID,
	}).Error("Login profile not found or not authorized")
	return nil, errors.New("login profile does not exist or is not authorized")
}

// Logout is the resolver for the Logout field.
func (r *mutationResolver) Logout(ctx context.Context) (*model.StatusResponse, error) {
	creds := auth.GetCredentials(ctx)
	identity := strings.TrimSpace(analytics.MetadataFromContext(ctx).DistinctID)
	hasIdentity := identity != "" && identity != "disabled"
	hasProfile := false
	dbType := ""
	if creds != nil {
		hasProfile = creds.Id != nil && strings.TrimSpace(*creds.Id) != ""
		dbType = creds.Type
	}

	if hasIdentity {
		analytics.CaptureWithDistinctID(ctx, identity, "logout.attempt", map[string]any{
			"database_type":      dbType,
			"profile_id_present": hasProfile,
		})
	}

	resp, err := auth.Logout(ctx)
	if err != nil {
		if hasIdentity {
			analytics.CaptureError(ctx, "logout.execute", err, map[string]any{
				"database_type":      dbType,
				"profile_id_present": hasProfile,
			})
		}
		return nil, err
	}

	if hasIdentity {
		analytics.CaptureWithDistinctID(ctx, identity, "logout.success", map[string]any{
			"database_type":      dbType,
			"profile_id_present": hasProfile,
		})
	}

	return resp, nil
}
