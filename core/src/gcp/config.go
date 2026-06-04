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

package gcp

import (
	"context"

	"google.golang.org/api/option"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

// GCPConfig holds the resolved GCP configuration for creating service clients.
// Unlike AWS which returns a single aws.Config, GCP SDK uses option.ClientOption
// slices passed to each service client constructor.
type GCPConfig struct {
	ProjectID string
	Region    string
	Options   []option.ClientOption
}

// LoadGCPConfig creates a GCP SDK configuration from WhoDB credentials.
// This is the primary entry point for GCP plugins.
func LoadGCPConfig(ctx context.Context, creds *engine.Credentials) (*GCPConfig, error) {
	gcpCreds, err := ParseFromWhoDB(creds)
	if err != nil {
		return nil, err
	}

	return loadConfigFromGCPCredentials(ctx, gcpCreds)
}

func loadConfigFromGCPCredentials(_ context.Context, gcpCreds *GCPCredentialConfig) (*GCPConfig, error) {
	var opts []option.ClientOption

	if gcpCreds.IsServiceAccountKeyAuth() && gcpCreds.ServiceAccountKeyPath != "" {
		opts = append(opts, option.WithAuthCredentialsFile(option.ServiceAccount, gcpCreds.ServiceAccountKeyPath))
	}
	// For AuthMethodDefault, no explicit option needed — the SDK uses ADC automatically.

	log.WithFields(map[string]any{
		"projectID":  gcpCreds.ProjectID,
		"region":     gcpCreds.Region,
		"authMethod": gcpCreds.AuthMethod,
	}).Debug("GCP configuration loaded successfully")

	return &GCPConfig{
		ProjectID: gcpCreds.ProjectID,
		Region:    gcpCreds.Region,
		Options:   opts,
	}, nil
}
