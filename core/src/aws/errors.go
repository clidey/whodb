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
	"fmt"
	"strings"

	"github.com/aws/smithy-go"
)

// Common AWS errors with user-friendly messages.
var (
	// ErrAccessDenied indicates the credentials don't have permission for the operation.
	ErrAccessDenied = errors.New("access denied: check IAM permissions for this operation")

	// ErrInvalidCredentials indicates the AWS credentials are invalid.
	ErrInvalidCredentials = errors.New("invalid AWS credentials: check access key and secret key")

	// ErrExpiredCredentials indicates the AWS credentials have expired.
	ErrExpiredCredentials = errors.New("AWS credentials have expired: refresh session token or re-authenticate")

	// ErrResourceNotFound indicates the requested AWS resource doesn't exist.
	ErrResourceNotFound = errors.New("resource not found: check the resource name and region")

	// ErrServiceUnavailable indicates the AWS service is temporarily unavailable.
	ErrServiceUnavailable = errors.New("AWS service temporarily unavailable: try again later")

	// ErrThrottling indicates the request was throttled by AWS.
	ErrThrottling = errors.New("request throttled: too many requests, try again later")

	// ErrConnectionFailed indicates a network connection failure.
	ErrConnectionFailed = errors.New("connection failed: check network connectivity and endpoint")

	// ErrInvalidRegion indicates the specified region is invalid or inaccessible.
	ErrInvalidRegion = errors.New("invalid or inaccessible region: check the region name")
)

// HandleAWSError converts AWS SDK errors to user-friendly messages.
// This follows the pattern of gorm_plugin.ErrorHandler.
func HandleAWSError(err error) error {
	if err == nil {
		return nil
	}

	// Check for smithy API errors (AWS SDK v2 error type)
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return handleAPIError(apiErr)
	}

	// Check for smithy operation errors (wraps API errors with operation context)
	var opErr *smithy.OperationError
	if errors.As(err, &opErr) {
		// Check if the underlying error is an API error
		if errors.As(opErr.Err, &apiErr) {
			return handleAPIError(apiErr)
		}
		// Otherwise, check for connection-related errors
		if isConnectionError(opErr.Err) {
			return fmt.Errorf("%w: %s", ErrConnectionFailed, opErr.Err.Error())
		}
	}

	// Check for common error message patterns
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

	// Return original error if no mapping found
	return err
}

// handleAPIError maps AWS API error codes to user-friendly errors.
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

// isConnectionError checks if an error is related to network connectivity.
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

// IsAccessDenied checks if an error is an access denied error.
func IsAccessDenied(err error) bool {
	return errors.Is(err, ErrAccessDenied)
}

// IsInvalidCredentials checks if an error is an invalid credentials error.
func IsInvalidCredentials(err error) bool {
	return errors.Is(err, ErrInvalidCredentials)
}

// IsResourceNotFound checks if an error is a resource not found error.
func IsResourceNotFound(err error) bool {
	return errors.Is(err, ErrResourceNotFound)
}

// IsThrottling checks if an error is a throttling error.
func IsThrottling(err error) bool {
	return errors.Is(err, ErrThrottling)
}

// IsConnectionError checks if an error is a connection error.
func IsConnectionError(err error) bool {
	return errors.Is(err, ErrConnectionFailed) || isConnectionError(err)
}

// IsRetryable checks if an error is retryable.
// Throttling, service unavailable, and connection errors are typically retryable.
func IsRetryable(err error) bool {
	return IsThrottling(err) ||
		errors.Is(err, ErrServiceUnavailable) ||
		IsConnectionError(err)
}
