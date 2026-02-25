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

package aws

import (
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
)

type AuthMethod string

const (
	// AuthMethodStatic uses explicit access key and secret key credentials.
	AuthMethodStatic AuthMethod = "static"

	// AuthMethodProfile uses a named profile from the AWS shared credentials file.
	AuthMethodProfile AuthMethod = "profile"

	// AuthMethodDefault uses the AWS SDK's default credential chain.
	// This is the recommended method as it automatically handles:
	// environment variables, shared credentials file, IAM roles, etc.
	AuthMethodDefault AuthMethod = "default"
)

const (
	AdvancedKeyAuthMethod  = "Auth Method"
	AdvancedKeyProfileName = "Profile Name"
)

// AWSCredentialConfig holds parsed AWS configuration extracted from WhoDB credentials.
type AWSCredentialConfig struct {
	Region          string
	AuthMethod      AuthMethod
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	ProfileName     string
}

// ParseFromWhoDB extracts AWS configuration from WhoDB credentials.
// Returns an error if required fields are missing or invalid.
func ParseFromWhoDB(creds *engine.Credentials) (*AWSCredentialConfig, error) {
	if creds == nil {
		return nil, errors.New("credentials cannot be nil")
	}

	region := strings.TrimSpace(creds.Hostname)
	if region == "" {
		region = common.GetRecordValueOrDefault(creds.Advanced, "Region", "")
	}

	config := &AWSCredentialConfig{
		Region:          region,
		AccessKeyID:     strings.TrimSpace(creds.Username),
		SecretAccessKey: strings.TrimSpace(creds.Password),
	}

	if creds.AccessToken != nil {
		config.SessionToken = strings.TrimSpace(*creds.AccessToken)
	}

	authMethodStr := common.GetRecordValueOrDefault(creds.Advanced, AdvancedKeyAuthMethod, string(AuthMethodDefault))
	config.AuthMethod = AuthMethod(strings.ToLower(strings.TrimSpace(authMethodStr)))
	config.ProfileName = common.GetRecordValueOrDefault(creds.Advanced, AdvancedKeyProfileName, "")

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate checks that the configuration is valid for the selected auth method.
func (c *AWSCredentialConfig) Validate() error {
	if c.Region == "" {
		return ErrRegionRequired
	}

	switch c.AuthMethod {
	case AuthMethodStatic:
		if c.AccessKeyID == "" || c.SecretAccessKey == "" {
			return ErrStaticCredentialsRequired
		}
	case AuthMethodProfile:
		if c.ProfileName == "" {
			return ErrProfileNameRequired
		}
	case AuthMethodDefault:
	default:
		return ErrInvalidAuthMethod
	}

	return nil
}

// BuildCredentialsProvider creates an AWS credentials provider based on the auth method.
// Returns nil for auth methods that should use the SDK's default credential chain
// (profile, default), as those are handled by config.LoadDefaultConfig.
func (c *AWSCredentialConfig) BuildCredentialsProvider() aws.CredentialsProvider {
	switch c.AuthMethod {
	case AuthMethodStatic:
		return credentials.NewStaticCredentialsProvider(
			c.AccessKeyID,
			c.SecretAccessKey,
			c.SessionToken,
		)
	case AuthMethodProfile, AuthMethodDefault:
		return nil
	default:
		return nil
	}
}

func (c *AWSCredentialConfig) IsProfileAuth() bool {
	return c.AuthMethod == AuthMethodProfile
}
