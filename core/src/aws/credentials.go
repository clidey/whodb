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

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
)

type AuthMethod string

const (
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
	Region      string
	AuthMethod  AuthMethod
	ProfileName string
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

	authMethodStr := common.GetRecordValueOrDefault(creds.Advanced, AdvancedKeyAuthMethod, string(AuthMethodDefault))
	profileName := common.GetRecordValueOrDefault(creds.Advanced, AdvancedKeyProfileName, "")

	config := &AWSCredentialConfig{
		Region:      region,
		AuthMethod:  AuthMethod(strings.ToLower(strings.TrimSpace(authMethodStr))),
		ProfileName: profileName,
	}

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

func (c *AWSCredentialConfig) IsProfileAuth() bool {
	return c.AuthMethod == AuthMethodProfile
}
