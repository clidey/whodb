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

package aws

import (
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
)

// AuthMethod represents the AWS authentication method to use.
type AuthMethod string

const (
	// AuthMethodStatic uses explicit access key and secret key credentials.
	AuthMethodStatic AuthMethod = "static"

	// AuthMethodProfile uses a named profile from the AWS shared credentials file.
	AuthMethodProfile AuthMethod = "profile"

	// AuthMethodIAM uses EC2/ECS/Lambda instance role credentials.
	AuthMethodIAM AuthMethod = "iam"

	// AuthMethodEnv uses AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables.
	AuthMethodEnv AuthMethod = "env"

	// AuthMethodDefault uses the AWS SDK's default credential chain.
	// This is the recommended method as it automatically handles:
	// environment variables, shared credentials file, IAM roles, etc.
	AuthMethodDefault AuthMethod = "default"
)

// Advanced record keys for AWS configuration.
const (
	// AdvancedKeyAuthMethod specifies the authentication method.
	AdvancedKeyAuthMethod = "Auth Method"

	// AdvancedKeyProfileName specifies the AWS profile name for profile-based auth.
	AdvancedKeyProfileName = "Profile Name"

	// AdvancedKeyEndpoint specifies a custom endpoint for AWS-compatible services.
	AdvancedKeyEndpoint = "Endpoint"
)

// Common errors for AWS credential handling.
var (
	// ErrRegionRequired is returned when no region is specified.
	ErrRegionRequired = errors.New("AWS region is required (set via Hostname field)")

	// ErrStaticCredentialsRequired is returned when static auth is selected but credentials are missing.
	ErrStaticCredentialsRequired = errors.New("static auth requires access key (Username) and secret key (Password)")

	// ErrProfileNameRequired is returned when profile auth is selected but no profile name is provided.
	ErrProfileNameRequired = errors.New("profile auth requires a profile name (set via 'Profile Name' advanced option)")

	// ErrInvalidAuthMethod is returned when an unknown authentication method is specified.
	ErrInvalidAuthMethod = errors.New("invalid auth method: must be one of: static, profile, iam, env, default")
)

// AWSCredentialConfig holds parsed AWS configuration extracted from WhoDB credentials.
type AWSCredentialConfig struct {
	// Region is the AWS region (e.g., "us-west-2").
	Region string

	// AuthMethod is the authentication method to use.
	AuthMethod AuthMethod

	// AccessKeyID is the AWS access key ID (for static auth).
	AccessKeyID string

	// SecretAccessKey is the AWS secret access key (for static auth).
	SecretAccessKey string

	// SessionToken is the AWS session token (for temporary credentials).
	SessionToken string

	// ProfileName is the AWS profile name (for profile auth).
	ProfileName string

	// Endpoint is a custom endpoint URL (for LocalStack, MinIO, etc.).
	Endpoint string
}

// ParseFromWhoDB extracts AWS configuration from WhoDB credentials.
// Returns an error if required fields are missing or invalid.
func ParseFromWhoDB(creds *engine.Credentials) (*AWSCredentialConfig, error) {
	if creds == nil {
		return nil, errors.New("credentials cannot be nil")
	}

	config := &AWSCredentialConfig{
		Region:          strings.TrimSpace(creds.Hostname),
		AccessKeyID:     strings.TrimSpace(creds.Username),
		SecretAccessKey: strings.TrimSpace(creds.Password),
	}

	// Parse session token from AccessToken field
	if creds.AccessToken != nil {
		config.SessionToken = strings.TrimSpace(*creds.AccessToken)
	}

	// Parse advanced options
	authMethodStr := common.GetRecordValueOrDefault(creds.Advanced, AdvancedKeyAuthMethod, string(AuthMethodDefault))
	config.AuthMethod = AuthMethod(strings.ToLower(strings.TrimSpace(authMethodStr)))
	config.ProfileName = common.GetRecordValueOrDefault(creds.Advanced, AdvancedKeyProfileName, "")
	config.Endpoint = common.GetRecordValueOrDefault(creds.Advanced, AdvancedKeyEndpoint, "")

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate checks that the configuration is valid for the selected auth method.
func (c *AWSCredentialConfig) Validate() error {
	// Region is always required
	if c.Region == "" {
		return ErrRegionRequired
	}

	// Validate auth method and its requirements
	switch c.AuthMethod {
	case AuthMethodStatic:
		if c.AccessKeyID == "" || c.SecretAccessKey == "" {
			return ErrStaticCredentialsRequired
		}
	case AuthMethodProfile:
		if c.ProfileName == "" {
			return ErrProfileNameRequired
		}
	case AuthMethodIAM, AuthMethodEnv, AuthMethodDefault:
		// No additional validation needed
	default:
		return ErrInvalidAuthMethod
	}

	return nil
}

// BuildCredentialsProvider creates an AWS credentials provider based on the auth method.
// Returns nil for auth methods that should use the SDK's default credential chain
// (iam, env, default), as those are handled by config.LoadDefaultConfig.
func (c *AWSCredentialConfig) BuildCredentialsProvider() aws.CredentialsProvider {
	switch c.AuthMethod {
	case AuthMethodStatic:
		return credentials.NewStaticCredentialsProvider(
			c.AccessKeyID,
			c.SecretAccessKey,
			c.SessionToken,
		)
	case AuthMethodProfile, AuthMethodIAM, AuthMethodEnv, AuthMethodDefault:
		// These auth methods are handled by config.LoadDefaultConfig options
		// or the SDK's default credential chain
		return nil
	default:
		return nil
	}
}

// HasCustomEndpoint returns true if a custom endpoint is configured.
func (c *AWSCredentialConfig) HasCustomEndpoint() bool {
	return c.Endpoint != ""
}

// IsStaticAuth returns true if using static credentials.
func (c *AWSCredentialConfig) IsStaticAuth() bool {
	return c.AuthMethod == AuthMethodStatic
}

// IsProfileAuth returns true if using profile-based credentials.
func (c *AWSCredentialConfig) IsProfileAuth() bool {
	return c.AuthMethod == AuthMethodProfile
}
