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

// Package aws provides foundational AWS infrastructure for WhoDB.
//
// This package is NOT a database plugin. It provides shared infrastructure
// that AWS database plugins (DynamoDB, RDS, etc.) import to handle AWS
// authentication and SDK configuration.
//
// # Credential Mapping
//
// WhoDB credentials map to AWS configuration as follows:
//   - Hostname: AWS Region (e.g., "us-west-2")
//   - Username: AWS Access Key ID (for static auth)
//   - Password: AWS Secret Access Key (for static auth)
//   - Database: Service-specific identifier (table name, DB identifier)
//   - AccessToken: AWS Session Token (for temporary credentials)
//   - Advanced: Auth method, profile name, endpoint override
//
// # Authentication Methods
//
// Supported authentication methods via the "Auth Method" Advanced record:
//   - "static": Uses Username/Password as Access Key/Secret Key
//   - "profile": Uses AWS shared credentials file (~/.aws/credentials)
//   - "iam": Uses EC2/ECS/Lambda instance roles
//   - "env": Uses AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY environment variables
//   - "default": Uses AWS SDK's automatic credential chain (recommended)
//
// # Usage
//
// AWS plugins should use [LoadAWSConfig] to create an AWS SDK configuration:
//
//	cfg, err := aws.LoadAWSConfig(ctx, pluginConfig.Credentials)
//	if err != nil {
//	    return nil, err
//	}
//	client := dynamodb.NewFromConfig(cfg)
//
// Or use the helper [WithAWSConfig] for operations:
//
//	result, err := aws.WithAWSConfig(ctx, creds, func(cfg aws.Config) (*Result, error) {
//	    client := dynamodb.NewFromConfig(cfg)
//	    return client.ListTables(ctx, &dynamodb.ListTablesInput{})
//	})
//
// # Caching
//
// AWS configurations are cached with a 10-minute TTL to avoid repeated
// credential resolution. The cache is shared across all AWS service types.
//
// # Custom Endpoints
//
// For LocalStack, MinIO, or other AWS-compatible services, set the "Endpoint"
// Advanced record:
//
//	Advanced: [{Key: "Endpoint", Value: "http://localhost:4566"}]
package aws
