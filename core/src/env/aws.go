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

package env

// AWSProviderEnvConfig represents AWS provider configuration from environment variables.
// Authentication is handled by the AWS SDK default credential chain or named profiles.
// Example:
//
//	WHODB_AWS_PROVIDER='[{
//	  "name": "Production AWS",
//	  "region": "us-west-2",
//	  "profileName": "production"
//	}]'
type AWSProviderEnvConfig struct {
	// Name is a human-readable name for this provider.
	Name string `json:"name"`

	// Region is the AWS region to discover resources in.
	Region string `json:"region"`

	// ProfileName for profile auth. If set, uses the named AWS profile.
	// If empty, uses the default credential chain.
	ProfileName string `json:"profileName,omitempty"`

	// DiscoverRDS enables RDS database discovery (defaults to true if omitted).
	DiscoverRDS *bool `json:"discoverRDS,omitempty"`

	// DiscoverElastiCache enables ElastiCache discovery (defaults to true if omitted).
	DiscoverElastiCache *bool `json:"discoverElastiCache,omitempty"`

	// DiscoverDocumentDB enables DocumentDB discovery (defaults to true if omitted).
	DiscoverDocumentDB *bool `json:"discoverDocumentDB,omitempty"`
}
