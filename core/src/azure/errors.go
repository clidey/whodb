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
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

var (
	ErrAccessDenied          = errors.New("access denied: check Azure RBAC permissions for this operation")
	ErrInvalidCredentials    = errors.New("invalid Azure credentials: check tenant ID, client ID, and secret")
	ErrExpiredCredentials    = errors.New("azure credentials have expired: re-authenticate or refresh token")
	ErrSubscriptionNotFound  = errors.New("subscription not found: check the subscription ID and permissions")
	ErrResourceNotFound      = errors.New("resource not found: check the resource name and subscription")
	ErrServiceUnavailable    = errors.New("azure service temporarily unavailable: try again later")
	ErrThrottling            = errors.New("request throttled: too many requests, try again later")
	ErrConnectionFailed      = errors.New("connection failed: check network connectivity and endpoint")
	ErrSubscriptionRequired  = errors.New("azure subscription ID is required")
	ErrTenantIDRequired      = errors.New("service principal auth requires a Tenant ID")
	ErrClientIDRequired      = errors.New("service principal auth requires a Client ID")
	ErrClientSecretRequired  = errors.New("service principal auth requires a Client Secret")
	ErrInvalidAuthMethod     = errors.New("invalid auth method: must be one of: default, service-principal")
	ErrAzureProviderDisabled = errors.New("azure provider is disabled")
)

// HandleAzureError maps Azure SDK errors to user-friendly sentinel errors.
func HandleAzureError(err error) error {
	if err == nil {
		return nil
	}

	if respErr, ok := errors.AsType[*azcore.ResponseError](err); ok {
		return handleResponseError(respErr)
	}

	errMsg := err.Error()
	if strings.Contains(errMsg, "no credential") ||
		strings.Contains(errMsg, "authentication failed") {
		return ErrInvalidCredentials
	}
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "no such host") ||
		strings.Contains(errMsg, "dial tcp") {
		return fmt.Errorf("%w: %s", ErrConnectionFailed, errMsg)
	}

	return err
}

func handleResponseError(respErr *azcore.ResponseError) error {
	code := respErr.ErrorCode
	status := respErr.StatusCode

	// Map by error code first
	switch code {
	case "AuthorizationFailed", "AuthenticationFailed", "AuthenticationFailedInvalidHeader":
		return ErrAccessDenied
	case "InvalidAuthenticationToken", "InvalidAuthenticationTokenTenant":
		return ErrInvalidCredentials
	case "ExpiredAuthenticationToken":
		return ErrExpiredCredentials
	case "SubscriptionNotFound":
		return ErrSubscriptionNotFound
	case "ResourceNotFound", "ResourceGroupNotFound":
		return ErrResourceNotFound
	case "TooManyRequests", "RequestRateLimited":
		return ErrThrottling
	}

	// Fall back to HTTP status codes
	switch {
	case status == 401:
		return ErrInvalidCredentials
	case status == 403:
		return ErrAccessDenied
	case status == 404:
		return ErrResourceNotFound
	case status == 429:
		return ErrThrottling
	case status >= 500:
		return ErrServiceUnavailable
	}

	return fmt.Errorf("azure error [%s] (HTTP %d): %s", code, status, respErr.Error())
}
