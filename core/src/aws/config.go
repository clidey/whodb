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

func loadConfigFromAWSCredentials(ctx context.Context, awsCreds *AWSCredentialConfig) (aws.Config, error) {
	options := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(awsCreds.Region),
	}

	if provider := awsCreds.BuildCredentialsProvider(); provider != nil {
		options = append(options, awsconfig.WithCredentialsProvider(provider))
	}

	if awsCreds.IsProfileAuth() && awsCreds.ProfileName != "" {
		options = append(options, awsconfig.WithSharedConfigProfile(awsCreds.ProfileName))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, options...)
	if err != nil {
		log.Logger.WithFields(map[string]any{
			"region":     awsCreds.Region,
			"authMethod": awsCreds.AuthMethod,
		}).WithError(err).Error("Failed to load AWS configuration")
		return aws.Config{}, HandleAWSError(err)
	}

	log.Logger.WithFields(map[string]any{
		"region":         awsCreds.Region,
		"authMethod":     awsCreds.AuthMethod,
		"hasProfileName": awsCreds.ProfileName != "",
	}).Debug("AWS configuration loaded successfully")

	return cfg, nil
}
