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
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

// LoadAWSConfig creates an AWS SDK configuration from WhoDB credentials.
// This is the primary entry point for AWS plugins.
//
// The function parses WhoDB credentials, builds the appropriate credential
// provider based on the auth method, and returns a configured aws.Config
// that can be used to create service clients.
//
// Example:
//
//	cfg, err := aws.LoadAWSConfig(ctx, pluginConfig.Credentials)
//	if err != nil {
//	    return nil, err
//	}
//	client := dynamodb.NewFromConfig(cfg)
func LoadAWSConfig(ctx context.Context, creds *engine.Credentials) (aws.Config, error) {
	awsCreds, err := ParseFromWhoDB(creds)
	if err != nil {
		return aws.Config{}, err
	}

	return loadConfigFromAWSCredentials(ctx, awsCreds)
}

// loadConfigFromAWSCredentials builds an AWS config from parsed credentials.
func loadConfigFromAWSCredentials(ctx context.Context, awsCreds *AWSCredentialConfig) (aws.Config, error) {
	options := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(awsCreds.Region),
	}

	// Add credentials provider for static auth
	if provider := awsCreds.BuildCredentialsProvider(); provider != nil {
		options = append(options, awsconfig.WithCredentialsProvider(provider))
	}

	// Add profile for profile-based auth
	if awsCreds.IsProfileAuth() && awsCreds.ProfileName != "" {
		options = append(options, awsconfig.WithSharedConfigProfile(awsCreds.ProfileName))
	}

	// Load the configuration
	cfg, err := awsconfig.LoadDefaultConfig(ctx, options...)
	if err != nil {
		log.Logger.WithFields(map[string]any{
			"region":     awsCreds.Region,
			"authMethod": awsCreds.AuthMethod,
		}).WithError(err).Error("Failed to load AWS configuration")
		return aws.Config{}, HandleAWSError(err)
	}

	// Log success (without sensitive data)
	log.Logger.WithFields(map[string]any{
		"region":         awsCreds.Region,
		"authMethod":     awsCreds.AuthMethod,
		"hasEndpoint":    awsCreds.HasCustomEndpoint(),
		"hasProfileName": awsCreds.ProfileName != "",
	}).Debug("AWS configuration loaded successfully")

	return cfg, nil
}

// WithAWSConfig is a helper that creates AWS config and passes it to an operation.
// This follows the pattern of plugins.WithConnection for consistent connection lifecycle.
//
// Example:
//
//	result, err := aws.WithAWSConfig(ctx, creds, func(cfg aws.Config) (*Result, error) {
//	    client := dynamodb.NewFromConfig(cfg)
//	    return client.ListTables(ctx, &dynamodb.ListTablesInput{})
//	})
func WithAWSConfig[T any](ctx context.Context, creds *engine.Credentials, op func(aws.Config) (T, error)) (T, error) {
	cfg, err := GetOrCreateConfig(ctx, creds)
	if err != nil {
		var zero T
		return zero, err
	}
	return op(cfg)
}

// GetAWSCredentialConfig returns the parsed AWS credential configuration
// without loading the full AWS SDK config. Useful for validation or logging.
func GetAWSCredentialConfig(creds *engine.Credentials) (*AWSCredentialConfig, error) {
	return ParseFromWhoDB(creds)
}

// EndpointResolverFunc creates an endpoint resolver function for custom endpoints.
// Use this when creating service clients that need to connect to LocalStack, MinIO, etc.
//
// Example:
//
//	awsCfg, err := aws.LoadAWSConfig(ctx, creds)
//	if err != nil {
//	    return nil, err
//	}
//	awsCreds, _ := aws.GetAWSCredentialConfig(creds)
//	client := dynamodb.NewFromConfig(awsCfg, func(o *dynamodb.Options) {
//	    if awsCreds.HasCustomEndpoint() {
//	        o.BaseEndpoint = aws.String(awsCreds.Endpoint)
//	    }
//	})
func EndpointResolverForCredentials(creds *engine.Credentials) (string, bool) {
	awsCreds, err := ParseFromWhoDB(creds)
	if err != nil {
		return "", false
	}
	if awsCreds.HasCustomEndpoint() {
		return awsCreds.Endpoint, true
	}
	return "", false
}
