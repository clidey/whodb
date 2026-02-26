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
	"fmt"
	"strings"

	"github.com/aws/smithy-go"
)

var (
	ErrAccessDenied              = errors.New("access denied: check IAM permissions for this operation")
	ErrInvalidCredentials        = errors.New("invalid AWS credentials: check access key and secret key")
	ErrExpiredCredentials        = errors.New("AWS credentials have expired: refresh session token or re-authenticate")
	ErrResourceNotFound          = errors.New("resource not found: check the resource name and region")
	ErrServiceUnavailable        = errors.New("AWS service temporarily unavailable: try again later")
	ErrThrottling                = errors.New("request throttled: too many requests, try again later")
	ErrConnectionFailed          = errors.New("connection failed: check network connectivity and endpoint")
	ErrInvalidRegion             = errors.New("invalid or inaccessible region: check the region name")
	ErrRegionRequired      = errors.New("AWS region is required (set via Hostname field)")
	ErrProfileNameRequired = errors.New("profile auth requires a profile name (set via 'Profile Name' advanced option)")
	ErrInvalidAuthMethod   = errors.New("invalid auth method: must be one of: default, profile")
	ErrAWSProviderDisabled = errors.New("AWS provider is disabled")
)

func HandleAWSError(err error) error {
	if err == nil {
		return nil
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return handleAPIError(apiErr)
	}

	var opErr *smithy.OperationError
	if errors.As(err, &opErr) {
		if errors.As(opErr.Err, &apiErr) {
			return handleAPIError(apiErr)
		}
		if isConnectionError(opErr.Err) {
			return fmt.Errorf("%w: %s", ErrConnectionFailed, opErr.Err.Error())
		}
	}

	errMsg := err.Error()
	if strings.Contains(errMsg, "no credentials") ||
		strings.Contains(errMsg, "NoCredentialProviders") {
		return ErrInvalidCredentials
	}
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "no such host") ||
		strings.Contains(errMsg, "dial tcp") {
		return fmt.Errorf("%w: %s", ErrConnectionFailed, errMsg)
	}

	return err
}

func handleAPIError(apiErr smithy.APIError) error {
	code := apiErr.ErrorCode()

	switch code {
	// Authentication and authorization errors
	case "AccessDeniedException", "AccessDenied", "UnauthorizedAccess":
		return ErrAccessDenied
	case "InvalidSignatureException", "SignatureDoesNotMatch":
		return ErrInvalidCredentials
	case "UnrecognizedClientException", "InvalidClientTokenId":
		return ErrInvalidCredentials
	case "ExpiredTokenException", "TokenRefreshRequired":
		return ErrExpiredCredentials
	case "IncompleteSignature":
		return ErrInvalidCredentials

	// Resource errors
	case "ResourceNotFoundException", "TableNotFoundException", "ItemNotFoundException":
		return ErrResourceNotFound
	case "ResourceInUseException":
		return fmt.Errorf("resource is in use: %s", apiErr.ErrorMessage())

	// Throttling and rate limiting
	case "ThrottlingException", "Throttling", "ProvisionedThroughputExceededException":
		return ErrThrottling
	case "RequestLimitExceeded", "TooManyRequestsException":
		return ErrThrottling

	// Service availability
	case "ServiceUnavailable", "InternalServerError", "InternalError":
		return ErrServiceUnavailable
	case "ServiceException":
		return ErrServiceUnavailable

	// Validation errors
	case "ValidationException", "ValidationError":
		return fmt.Errorf("validation error: %s", apiErr.ErrorMessage())
	case "InvalidParameterValue", "InvalidParameterException":
		return fmt.Errorf("invalid parameter: %s", apiErr.ErrorMessage())
	case "MissingRequiredParameterException":
		return fmt.Errorf("missing required parameter: %s", apiErr.ErrorMessage())

	// Region errors
	case "InvalidRegion", "RegionDisabledException":
		return ErrInvalidRegion

	// Conditional check failures (DynamoDB)
	case "ConditionalCheckFailedException":
		return fmt.Errorf("conditional check failed: %s", apiErr.ErrorMessage())
	case "TransactionConflictException":
		return fmt.Errorf("transaction conflict: %s", apiErr.ErrorMessage())

	default:
		// Return a formatted error with the AWS error code and message
		return fmt.Errorf("AWS error [%s]: %s", code, apiErr.ErrorMessage())
	}
}

func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	connectionPatterns := []string{
		"connection refused",
		"no such host",
		"dial tcp",
		"i/o timeout",
		"network is unreachable",
		"connection reset",
		"EOF",
	}
	for _, pattern := range connectionPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}
	return false
}
