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
	"testing"

	"github.com/aws/smithy-go"
)

// mockAPIError implements smithy.APIError for testing.
type mockAPIError struct {
	code    string
	message string
	fault   smithy.ErrorFault
}

func (e *mockAPIError) ErrorCode() string {
	return e.code
}

func (e *mockAPIError) ErrorMessage() string {
	return e.message
}

func (e *mockAPIError) ErrorFault() smithy.ErrorFault {
	return e.fault
}

func (e *mockAPIError) Error() string {
	return e.code + ": " + e.message
}

// Ensure mockAPIError implements smithy.APIError
var _ smithy.APIError = (*mockAPIError)(nil)

func TestHandleAWSError_Nil(t *testing.T) {
	if HandleAWSError(nil) != nil {
		t.Error("expected nil for nil error")
	}
}

func TestHandleAWSError_AccessDenied(t *testing.T) {
	testCases := []string{
		"AccessDeniedException",
		"AccessDenied",
		"UnauthorizedAccess",
	}

	for _, code := range testCases {
		err := &mockAPIError{code: code, message: "test message"}
		result := HandleAWSError(err)
		if !errors.Is(result, ErrAccessDenied) {
			t.Errorf("expected ErrAccessDenied for %s, got %v", code, result)
		}
	}
}

func TestHandleAWSError_InvalidCredentials(t *testing.T) {
	testCases := []string{
		"InvalidSignatureException",
		"SignatureDoesNotMatch",
		"UnrecognizedClientException",
		"InvalidClientTokenId",
		"IncompleteSignature",
	}

	for _, code := range testCases {
		err := &mockAPIError{code: code, message: "test message"}
		result := HandleAWSError(err)
		if !errors.Is(result, ErrInvalidCredentials) {
			t.Errorf("expected ErrInvalidCredentials for %s, got %v", code, result)
		}
	}
}

func TestHandleAWSError_ExpiredCredentials(t *testing.T) {
	testCases := []string{
		"ExpiredTokenException",
		"TokenRefreshRequired",
	}

	for _, code := range testCases {
		err := &mockAPIError{code: code, message: "test message"}
		result := HandleAWSError(err)
		if !errors.Is(result, ErrExpiredCredentials) {
			t.Errorf("expected ErrExpiredCredentials for %s, got %v", code, result)
		}
	}
}

func TestHandleAWSError_ResourceNotFound(t *testing.T) {
	testCases := []string{
		"ResourceNotFoundException",
		"TableNotFoundException",
		"ItemNotFoundException",
	}

	for _, code := range testCases {
		err := &mockAPIError{code: code, message: "test message"}
		result := HandleAWSError(err)
		if !errors.Is(result, ErrResourceNotFound) {
			t.Errorf("expected ErrResourceNotFound for %s, got %v", code, result)
		}
	}
}

func TestHandleAWSError_Throttling(t *testing.T) {
	testCases := []string{
		"ThrottlingException",
		"Throttling",
		"ProvisionedThroughputExceededException",
		"RequestLimitExceeded",
		"TooManyRequestsException",
	}

	for _, code := range testCases {
		err := &mockAPIError{code: code, message: "test message"}
		result := HandleAWSError(err)
		if !errors.Is(result, ErrThrottling) {
			t.Errorf("expected ErrThrottling for %s, got %v", code, result)
		}
	}
}

func TestHandleAWSError_ServiceUnavailable(t *testing.T) {
	testCases := []string{
		"ServiceUnavailable",
		"InternalServerError",
		"InternalError",
		"ServiceException",
	}

	for _, code := range testCases {
		err := &mockAPIError{code: code, message: "test message"}
		result := HandleAWSError(err)
		if !errors.Is(result, ErrServiceUnavailable) {
			t.Errorf("expected ErrServiceUnavailable for %s, got %v", code, result)
		}
	}
}

func TestHandleAWSError_InvalidRegion(t *testing.T) {
	testCases := []string{
		"InvalidRegion",
		"RegionDisabledException",
	}

	for _, code := range testCases {
		err := &mockAPIError{code: code, message: "test message"}
		result := HandleAWSError(err)
		if !errors.Is(result, ErrInvalidRegion) {
			t.Errorf("expected ErrInvalidRegion for %s, got %v", code, result)
		}
	}
}

func TestHandleAWSError_ValidationError(t *testing.T) {
	err := &mockAPIError{code: "ValidationException", message: "field X is invalid"}
	result := HandleAWSError(err)
	if result == nil || result.Error() != "validation error: field X is invalid" {
		t.Errorf("unexpected error message: %v", result)
	}
}

func TestHandleAWSError_UnknownCode(t *testing.T) {
	err := &mockAPIError{code: "SomeUnknownError", message: "unknown error occurred"}
	result := HandleAWSError(err)
	if result == nil {
		t.Error("expected non-nil error for unknown code")
	}
	expected := "AWS error [SomeUnknownError]: unknown error occurred"
	if result.Error() != expected {
		t.Errorf("expected %q, got %q", expected, result.Error())
	}
}

func TestHandleAWSError_ConnectionPatterns(t *testing.T) {
	testCases := []string{
		"no credentials found",
		"NoCredentialProviders: no valid providers",
		"connection refused",
		"no such host",
		"dial tcp: failed",
	}

	for _, msg := range testCases {
		err := errors.New(msg)
		result := HandleAWSError(err)
		if result == nil {
			t.Errorf("expected non-nil error for %s", msg)
		}
	}
}

func TestHandleAWSError_RequestTimeout(t *testing.T) {
	err := &mockAPIError{code: "RequestTimeoutException", message: "timed out"}
	result := HandleAWSError(err)
	if !errors.Is(result, ErrConnectionFailed) {
		t.Errorf("expected ErrConnectionFailed for RequestTimeoutException, got %v", result)
	}
}

func TestHandleAWSError_LimitExceeded(t *testing.T) {
	err := &mockAPIError{code: "LimitExceededException", message: "limit exceeded"}
	result := HandleAWSError(err)
	if !errors.Is(result, ErrThrottling) {
		t.Errorf("expected ErrThrottling for LimitExceededException, got %v", result)
	}
}

func TestIsConnectionError(t *testing.T) {
	testCases := []struct {
		msg      string
		expected bool
	}{
		{"connection refused", true},
		{"no such host", true},
		{"dial tcp 1.2.3.4:443", true},
		{"i/o timeout", true},
		{"network is unreachable", true},
		{"connection reset by peer", true},
		{"EOF", true},
		{"tls handshake timeout", true},
		{"context deadline exceeded", true},
		{"TLS Handshake Timeout", true}, // case-insensitive
		{"Context Deadline Exceeded", true},
		{"something else entirely", false},
		{"", false},
	}

	for _, tc := range testCases {
		result := isConnectionError(errors.New(tc.msg))
		if result != tc.expected {
			t.Errorf("isConnectionError(%q): expected %v, got %v", tc.msg, tc.expected, result)
		}
	}

	// nil error
	if isConnectionError(nil) {
		t.Error("isConnectionError(nil): expected false")
	}
}
