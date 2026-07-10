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

package azure

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

// newResponseError creates an azcore.ResponseError for testing.
func newResponseError(code string, statusCode int) *azcore.ResponseError {
	return &azcore.ResponseError{
		ErrorCode:  code,
		StatusCode: statusCode,
		RawResponse: &http.Response{
			StatusCode: statusCode,
		},
	}
}

func TestHandleAzureError_Nil(t *testing.T) {
	if HandleAzureError(nil) != nil {
		t.Error("expected nil for nil error")
	}
}

func TestHandleAzureError_AccessDenied(t *testing.T) {
	testCases := []string{
		"AuthorizationFailed",
		"AuthenticationFailed",
		"AuthenticationFailedInvalidHeader",
	}

	for _, code := range testCases {
		err := newResponseError(code, 403)
		result := HandleAzureError(err)
		if !errors.Is(result, ErrAccessDenied) {
			t.Errorf("expected ErrAccessDenied for %s, got %v", code, result)
		}
	}
}

func TestHandleAzureError_InvalidCredentials(t *testing.T) {
	testCases := []string{
		"InvalidAuthenticationToken",
		"InvalidAuthenticationTokenTenant",
	}

	for _, code := range testCases {
		err := newResponseError(code, 401)
		result := HandleAzureError(err)
		if !errors.Is(result, ErrInvalidCredentials) {
			t.Errorf("expected ErrInvalidCredentials for %s, got %v", code, result)
		}
	}
}

func TestHandleAzureError_ExpiredCredentials(t *testing.T) {
	err := newResponseError("ExpiredAuthenticationToken", 401)
	result := HandleAzureError(err)
	if !errors.Is(result, ErrExpiredCredentials) {
		t.Errorf("expected ErrExpiredCredentials, got %v", result)
	}
}

func TestHandleAzureError_SubscriptionNotFound(t *testing.T) {
	err := newResponseError("SubscriptionNotFound", 404)
	result := HandleAzureError(err)
	if !errors.Is(result, ErrSubscriptionNotFound) {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", result)
	}
}

func TestHandleAzureError_ResourceNotFound(t *testing.T) {
	testCases := []string{
		"ResourceNotFound",
		"ResourceGroupNotFound",
	}

	for _, code := range testCases {
		err := newResponseError(code, 404)
		result := HandleAzureError(err)
		if !errors.Is(result, ErrResourceNotFound) {
			t.Errorf("expected ErrResourceNotFound for %s, got %v", code, result)
		}
	}
}

func TestHandleAzureError_Throttling(t *testing.T) {
	testCases := []string{
		"TooManyRequests",
		"RequestRateLimited",
	}

	for _, code := range testCases {
		err := newResponseError(code, 429)
		result := HandleAzureError(err)
		if !errors.Is(result, ErrThrottling) {
			t.Errorf("expected ErrThrottling for %s, got %v", code, result)
		}
	}
}

func TestHandleAzureError_StatusCode401(t *testing.T) {
	err := newResponseError("SomeUnknownCode", 401)
	result := HandleAzureError(err)
	if !errors.Is(result, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials for status 401, got %v", result)
	}
}

func TestHandleAzureError_StatusCode403(t *testing.T) {
	err := newResponseError("SomeUnknownCode", 403)
	result := HandleAzureError(err)
	if !errors.Is(result, ErrAccessDenied) {
		t.Errorf("expected ErrAccessDenied for status 403, got %v", result)
	}
}

func TestHandleAzureError_StatusCode404(t *testing.T) {
	err := newResponseError("SomeUnknownCode", 404)
	result := HandleAzureError(err)
	if !errors.Is(result, ErrResourceNotFound) {
		t.Errorf("expected ErrResourceNotFound for status 404, got %v", result)
	}
}

func TestHandleAzureError_StatusCode429(t *testing.T) {
	err := newResponseError("SomeUnknownCode", 429)
	result := HandleAzureError(err)
	if !errors.Is(result, ErrThrottling) {
		t.Errorf("expected ErrThrottling for status 429, got %v", result)
	}
}

func TestHandleAzureError_StatusCode500(t *testing.T) {
	err := newResponseError("SomeUnknownCode", 500)
	result := HandleAzureError(err)
	if !errors.Is(result, ErrServiceUnavailable) {
		t.Errorf("expected ErrServiceUnavailable for status 500, got %v", result)
	}
}

func TestHandleAzureError_StatusCode503(t *testing.T) {
	err := newResponseError("SomeUnknownCode", 503)
	result := HandleAzureError(err)
	if !errors.Is(result, ErrServiceUnavailable) {
		t.Errorf("expected ErrServiceUnavailable for status 503, got %v", result)
	}
}

func TestHandleAzureError_UnknownCode(t *testing.T) {
	err := newResponseError("SomeUnknownError", 418)
	result := HandleAzureError(err)
	if result == nil {
		t.Fatal("expected non-nil error for unknown code")
	}
	if !strings.Contains(result.Error(), "SomeUnknownError") {
		t.Errorf("expected error to contain code, got %q", result.Error())
	}
	if !strings.Contains(result.Error(), "418") {
		t.Errorf("expected error to contain status code, got %q", result.Error())
	}
}

func TestHandleAzureError_NoCredentialPattern(t *testing.T) {
	err := errors.New("no credential available")
	result := HandleAzureError(err)
	if !errors.Is(result, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials for no credential error, got %v", result)
	}
}

func TestHandleAzureError_AuthenticationFailedPattern(t *testing.T) {
	err := errors.New("authentication failed for user")
	result := HandleAzureError(err)
	if !errors.Is(result, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials for authentication failed error, got %v", result)
	}
}

func TestHandleAzureError_ConnectionPatterns(t *testing.T) {
	testCases := []string{
		"connection refused",
		"no such host",
		"dial tcp: failed to connect",
	}

	for _, msg := range testCases {
		err := errors.New(msg)
		result := HandleAzureError(err)
		if result == nil {
			t.Errorf("expected non-nil error for %q", msg)
			continue
		}
		if !errors.Is(result, ErrConnectionFailed) {
			t.Errorf("expected ErrConnectionFailed for %q, got %v", msg, result)
		}
	}
}

func TestHandleAzureError_NonAzureError(t *testing.T) {
	err := errors.New("something unrelated happened")
	result := HandleAzureError(err)
	if !errors.Is(result, err) {
		t.Errorf("expected original error to be returned, got %v", result)
	}
}
