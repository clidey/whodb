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
// Example:
//
//	WHODB_AWS_PROVIDER='[{
//	  "name": "Production AWS",
//	  "region": "us-west-2",
//	  "auth": "default"
//	}]'
type AWSProviderEnvConfig struct {
	// Name is a human-readable name for this provider.
	Name string `json:"name"`

	// Region is the AWS region to discover resources in.
	Region string `json:"region"`

	// Auth is the authentication method: "default", "static", "profile".
	Auth string `json:"auth"`

	// AccessKeyID for static auth.
	AccessKeyID string `json:"accessKeyId,omitempty"`

	// SecretAccessKey for static auth.
	SecretAccessKey string `json:"secretAccessKey,omitempty"`

	// SessionToken for temporary credentials.
	SessionToken string `json:"sessionToken,omitempty"`

	// ProfileName for profile auth.
	ProfileName string `json:"profileName,omitempty"`

	// DiscoverRDS enables RDS database discovery (defaults to true if omitted).
	DiscoverRDS *bool `json:"discoverRDS,omitempty"`

	// DiscoverElastiCache enables ElastiCache discovery (defaults to true if omitted).
	DiscoverElastiCache *bool `json:"discoverElastiCache,omitempty"`

	// DiscoverDocumentDB enables DocumentDB discovery (defaults to true if omitted).
	DiscoverDocumentDB *bool `json:"discoverDocumentDB,omitempty"`

	// DBUsername is the database username for IAM auth connections.
	DBUsername string `json:"dbUsername,omitempty"`
}
