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
	"strings"
	"testing"

	"google.golang.org/api/googleapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandleGCPError_Nil(t *testing.T) {
	if HandleGCPError(nil) != nil {
		t.Error("expected nil for nil error")
	}
}

func TestHandleGCPError_REST401(t *testing.T) {
	err := &googleapi.Error{Code: 401, Message: "invalid credentials"}
	result := HandleGCPError(err)
	if !errors.Is(result, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials for 401, got %v", result)
	}
}

func TestHandleGCPError_REST403_PermissionDenied(t *testing.T) {
	err := &googleapi.Error{Code: 403, Message: "caller does not have permission"}
	result := HandleGCPError(err)
	if !errors.Is(result, ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied for 403, got %v", result)
	}
}

func TestHandleGCPError_REST403_ServiceDisabled(t *testing.T) {
	testCases := []string{
		"API has not been used in project before",
		"sqladmin.googleapis.com is not enabled",
		"API has not been enabled in project",
	}

	for _, msg := range testCases {
		err := &googleapi.Error{Code: 403, Message: msg}
		result := HandleGCPError(err)
		if !errors.Is(result, ErrServiceDisabled) {
			t.Errorf("expected ErrServiceDisabled for %q, got %v", msg, result)
		}
	}
}

func TestHandleGCPError_REST404(t *testing.T) {
	err := &googleapi.Error{Code: 404, Message: "not found"}
	result := HandleGCPError(err)
	if !errors.Is(result, ErrResourceNotFound) {
		t.Errorf("expected ErrResourceNotFound for 404, got %v", result)
	}
}

func TestHandleGCPError_REST429(t *testing.T) {
	err := &googleapi.Error{Code: 429, Message: "rate limit exceeded"}
	result := HandleGCPError(err)
	if !errors.Is(result, ErrQuotaExceeded) {
		t.Errorf("expected ErrQuotaExceeded for 429, got %v", result)
	}
}

func TestHandleGCPError_REST500(t *testing.T) {
	err := &googleapi.Error{Code: 500, Message: "internal error"}
	result := HandleGCPError(err)
	if !errors.Is(result, ErrServiceUnavailable) {
		t.Errorf("expected ErrServiceUnavailable for 500, got %v", result)
	}
}

func TestHandleGCPError_REST502(t *testing.T) {
	err := &googleapi.Error{Code: 502, Message: "bad gateway"}
	result := HandleGCPError(err)
	if !errors.Is(result, ErrServiceUnavailable) {
		t.Errorf("expected ErrServiceUnavailable for 502, got %v", result)
	}
}

func TestHandleGCPError_REST503(t *testing.T) {
	err := &googleapi.Error{Code: 503, Message: "service unavailable"}
	result := HandleGCPError(err)
	if !errors.Is(result, ErrServiceUnavailable) {
		t.Errorf("expected ErrServiceUnavailable for 503, got %v", result)
	}
}

func TestHandleGCPError_RESTUnknownCode(t *testing.T) {
	err := &googleapi.Error{Code: 418, Message: "I'm a teapot"}
	result := HandleGCPError(err)
	if result == nil {
		t.Fatal("expected non-nil error for unknown code")
	}
	if !strings.Contains(result.Error(), "418") {
		t.Errorf("expected error to contain status code, got %q", result.Error())
	}
	if !strings.Contains(result.Error(), "I'm a teapot") {
		t.Errorf("expected error to contain message, got %q", result.Error())
	}
}

func TestHandleGCPError_GRPCUnauthenticated(t *testing.T) {
	err := status.Error(codes.Unauthenticated, "request had invalid credentials")
	result := HandleGCPError(err)
	if !errors.Is(result, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials for Unauthenticated, got %v", result)
	}
}

func TestHandleGCPError_GRPCPermissionDenied(t *testing.T) {
	err := status.Error(codes.PermissionDenied, "caller does not have permission")
	result := HandleGCPError(err)
	if !errors.Is(result, ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied for PermissionDenied, got %v", result)
	}
}

func TestHandleGCPError_GRPCPermissionDenied_ServiceDisabled(t *testing.T) {
	testCases := []string{
		"alloydb.googleapis.com has not been used in project",
		"Cloud Redis API is not enabled",
	}

	for _, msg := range testCases {
		err := status.Error(codes.PermissionDenied, msg)
		result := HandleGCPError(err)
		if !errors.Is(result, ErrServiceDisabled) {
			t.Errorf("expected ErrServiceDisabled for %q, got %v", msg, result)
		}
	}
}

func TestHandleGCPError_GRPCNotFound(t *testing.T) {
	err := status.Error(codes.NotFound, "resource not found")
	result := HandleGCPError(err)
	if !errors.Is(result, ErrResourceNotFound) {
		t.Errorf("expected ErrResourceNotFound for NotFound, got %v", result)
	}
}

func TestHandleGCPError_GRPCResourceExhausted(t *testing.T) {
	err := status.Error(codes.ResourceExhausted, "quota exceeded")
	result := HandleGCPError(err)
	if !errors.Is(result, ErrQuotaExceeded) {
		t.Errorf("expected ErrQuotaExceeded for ResourceExhausted, got %v", result)
	}
}

func TestHandleGCPError_GRPCUnavailable(t *testing.T) {
	err := status.Error(codes.Unavailable, "service unavailable")
	result := HandleGCPError(err)
	if !errors.Is(result, ErrServiceUnavailable) {
		t.Errorf("expected ErrServiceUnavailable for Unavailable, got %v", result)
	}
}

func TestHandleGCPError_GRPCInternal(t *testing.T) {
	err := status.Error(codes.Internal, "internal error")
	result := HandleGCPError(err)
	if !errors.Is(result, ErrServiceUnavailable) {
		t.Errorf("expected ErrServiceUnavailable for Internal, got %v", result)
	}
}

func TestHandleGCPError_GRPCInvalidArgument(t *testing.T) {
	err := status.Error(codes.InvalidArgument, "bad request")
	result := HandleGCPError(err)
	if result == nil {
		t.Fatal("expected non-nil error for InvalidArgument")
	}
	if !strings.Contains(result.Error(), "bad request") {
		t.Errorf("expected error to contain message, got %q", result.Error())
	}
}

func TestHandleGCPError_GRPCDeadlineExceeded(t *testing.T) {
	err := status.Error(codes.DeadlineExceeded, "deadline exceeded")
	result := HandleGCPError(err)
	if !errors.Is(result, ErrConnectionFailed) {
		t.Errorf("expected ErrConnectionFailed for DeadlineExceeded, got %v", result)
	}
}

func TestHandleGCPError_GRPCUnknownCode(t *testing.T) {
	err := status.Error(codes.DataLoss, "data corruption detected")
	result := HandleGCPError(err)
	if result == nil {
		t.Fatal("expected non-nil error for unknown gRPC code")
	}
	if !strings.Contains(result.Error(), "data corruption detected") {
		t.Errorf("expected error to contain message, got %q", result.Error())
	}
}

func TestHandleGCPError_NoCredentialPattern(t *testing.T) {
	err := errors.New("google: could not find default credentials")
	result := HandleGCPError(err)
	if !errors.Is(result, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials for no default credentials, got %v", result)
	}
}

func TestHandleGCPError_DefaultCredentialsPattern(t *testing.T) {
	err := errors.New("could not find default credentials for API")
	result := HandleGCPError(err)
	if !errors.Is(result, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials for missing default credentials, got %v", result)
	}
}

func TestHandleGCPError_ConnectionPatterns(t *testing.T) {
	testCases := []string{
		"connection refused",
		"no such host",
		"dial tcp: failed to connect",
	}

	for _, msg := range testCases {
		err := errors.New(msg)
		result := HandleGCPError(err)
		if result == nil {
			t.Errorf("expected non-nil error for %q", msg)
			continue
		}
		if !errors.Is(result, ErrConnectionFailed) {
			t.Errorf("expected ErrConnectionFailed for %q, got %v", msg, result)
		}
	}
}

func TestHandleGCPError_NonGCPError(t *testing.T) {
	err := errors.New("something unrelated happened")
	result := HandleGCPError(err)
	if !errors.Is(result, err) {
		t.Errorf("expected original error to be returned, got %v", result)
	}
}
