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

package gcp

import (
	"errors"
	"fmt"
	"strings"

	"google.golang.org/api/googleapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrPermissionDenied              = errors.New("permission denied: check IAM permissions for this operation")
	ErrInvalidCredentials            = errors.New("invalid GCP credentials: check service account key or application default credentials")
	ErrProjectNotFound               = errors.New("project not found: check the project ID")
	ErrServiceDisabled               = errors.New("API not enabled: enable the required API in the Google Cloud Console")
	ErrQuotaExceeded                 = errors.New("quota exceeded: too many requests, try again later")
	ErrServiceUnavailable            = errors.New("GCP service temporarily unavailable: try again later")
	ErrResourceNotFound              = errors.New("resource not found: check the resource name and region")
	ErrConnectionFailed              = errors.New("connection failed: check network connectivity and endpoint")
	ErrRegionRequired                = errors.New("GCP region is required")
	ErrProjectIDRequired             = errors.New("GCP project ID is required")
	ErrServiceAccountKeyPathRequired = errors.New("service account key auth requires a key file path")
	ErrInvalidAuthMethod             = errors.New("invalid auth method: must be one of: default, service-account-key")
	ErrGCPProviderDisabled           = errors.New("GCP provider is disabled")
)

// HandleGCPError maps GCP SDK errors to user-friendly messages.
// Handles both REST API errors (googleapi.Error) and gRPC errors (status.Status).
func HandleGCPError(err error) error {
	if err == nil {
		return nil
	}

	// Handle REST API errors (googleapi.Error)
	if apiErr, ok := errors.AsType[*googleapi.Error](err); ok {
		return handleRESTError(apiErr)
	}

	// Handle gRPC errors
	if st, ok := status.FromError(err); ok && st.Code() != codes.OK {
		return handleGRPCError(st)
	}

	// Handle common error message patterns
	errMsg := err.Error()
	if strings.Contains(errMsg, "could not find default credentials") ||
		strings.Contains(errMsg, "google: could not find default credentials") {
		return fmt.Errorf("%w: set GOOGLE_APPLICATION_CREDENTIALS or run 'gcloud auth application-default login'", ErrInvalidCredentials)
	}
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "no such host") ||
		strings.Contains(errMsg, "dial tcp") {
		return fmt.Errorf("%w: %s", ErrConnectionFailed, errMsg)
	}

	return err
}

func handleRESTError(apiErr *googleapi.Error) error {
	switch apiErr.Code {
	case 401:
		return ErrInvalidCredentials
	case 403:
		if strings.Contains(apiErr.Message, "has not been used") ||
			strings.Contains(apiErr.Message, "is not enabled") ||
			strings.Contains(apiErr.Message, "API has not been enabled") {
			return fmt.Errorf("%w: %s", ErrServiceDisabled, apiErr.Message)
		}
		return fmt.Errorf("%w: %s", ErrPermissionDenied, apiErr.Message)
	case 404:
		return ErrResourceNotFound
	case 429:
		return ErrQuotaExceeded
	case 500, 502, 503:
		return ErrServiceUnavailable
	default:
		return fmt.Errorf("GCP error [HTTP %d]: %s", apiErr.Code, apiErr.Message)
	}
}

func handleGRPCError(st *status.Status) error {
	switch st.Code() {
	case codes.Unauthenticated:
		return ErrInvalidCredentials
	case codes.PermissionDenied:
		msg := st.Message()
		if strings.Contains(msg, "has not been used") ||
			strings.Contains(msg, "is not enabled") {
			return fmt.Errorf("%w: %s", ErrServiceDisabled, msg)
		}
		return fmt.Errorf("%w: %s", ErrPermissionDenied, msg)
	case codes.NotFound:
		return ErrResourceNotFound
	case codes.ResourceExhausted:
		return ErrQuotaExceeded
	case codes.Unavailable:
		return ErrServiceUnavailable
	case codes.Internal:
		return ErrServiceUnavailable
	case codes.InvalidArgument:
		return fmt.Errorf("invalid argument: %s", st.Message())
	case codes.DeadlineExceeded:
		return ErrConnectionFailed
	default:
		return fmt.Errorf("GCP error [%s]: %s", st.Code(), st.Message())
	}
}
