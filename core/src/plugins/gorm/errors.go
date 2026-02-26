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

package gorm_plugin

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/clidey/whodb/core/src/log"
	"gorm.io/gorm"
)

// ErrorHandler provides centralized error handling for GORM operations
type ErrorHandler struct {
	plugin GormPluginFunctions
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(plugin GormPluginFunctions) *ErrorHandler {
	return &ErrorHandler{plugin: plugin}
}

// HandleError processes GORM errors and returns user-friendly messages
func (h *ErrorHandler) HandleError(err error, operation string, details map[string]any) error {
	if err == nil {
		return nil
	}

	// Log the original error with context
	logger := log.WithError(err).WithField("operation", operation)
	for k, v := range details {
		logger = logger.WithField(k, v)
	}

	// Handle specific GORM errors
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		logger.Debug("Record not found")
		return fmt.Errorf("record not found")

	case errors.Is(err, gorm.ErrInvalidTransaction):
		logger.Error("Invalid transaction")
		return fmt.Errorf("transaction error: please retry the operation")

	case errors.Is(err, gorm.ErrNotImplemented):
		logger.Error("Feature not implemented")
		return fmt.Errorf("this feature is not supported for the current database")

	case errors.Is(err, gorm.ErrMissingWhereClause):
		logger.Warn("Missing WHERE clause in destructive operation")
		return fmt.Errorf("WHERE clause required for this operation")

	case errors.Is(err, gorm.ErrUnsupportedRelation):
		logger.Error("Unsupported relation")
		return fmt.Errorf("relationship operation not supported")

	case errors.Is(err, gorm.ErrPrimaryKeyRequired):
		logger.Error("Primary key required")
		return fmt.Errorf("primary key is required for this operation")

	case errors.Is(err, gorm.ErrModelValueRequired):
		logger.Error("Model value required")
		return fmt.Errorf("value is required for this operation")

	case errors.Is(err, gorm.ErrUnsupportedDriver):
		logger.Error("Unsupported database driver")
		return fmt.Errorf("database driver not supported")

	case h.isDuplicateKeyError(err):
		logger.Warn("Duplicate key violation")
		return fmt.Errorf("duplicate key: a record with these values already exists")

	case h.isForeignKeyError(err):
		logger.Warn("Foreign key constraint violation")
		return fmt.Errorf("foreign key constraint: referenced record does not exist or is in use")

	case h.isCheckConstraintError(err):
		logger.Warn("Check constraint violation")
		return fmt.Errorf("check constraint violation: value does not meet requirements")

	case h.isNotNullError(err):
		logger.Warn("NOT NULL constraint violation")
		return fmt.Errorf("required field cannot be empty")

	case h.isConnectionError(err):
		logger.Error("Database connection error")
		return fmt.Errorf("database connection error: please check your connection settings")

	case h.isTimeoutError(err):
		logger.Error("Operation timeout")
		return fmt.Errorf("operation timed out: the database took too long to respond")

	case h.isPermissionError(err):
		logger.Error("Permission denied")
		return fmt.Errorf("permission denied: insufficient privileges for this operation")

	default:
		// Log full error for debugging but return sanitized message
		logger.Error("Unhandled database error")
		return fmt.Errorf("database operation failed: %s", h.sanitizeErrorMessage(err))
	}
}

// isDuplicateKeyError checks if error is a duplicate key violation
func (h *ErrorHandler) isDuplicateKeyError(err error) bool {
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "duplicate") ||
		strings.Contains(errStr, "unique constraint") ||
		strings.Contains(errStr, "unique_violation") ||
		strings.Contains(errStr, "23505") // PostgreSQL error code
}

// isForeignKeyError checks if error is a foreign key violation
func (h *ErrorHandler) isForeignKeyError(err error) bool {
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "foreign key") ||
		strings.Contains(errStr, "fk_") ||
		strings.Contains(errStr, "23503") || // PostgreSQL error code
		strings.Contains(errStr, "1452") // MySQL error code
}

// isCheckConstraintError checks if error is a check constraint violation
func (h *ErrorHandler) isCheckConstraintError(err error) bool {
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "check constraint") ||
		strings.Contains(errStr, "chk_") ||
		strings.Contains(errStr, "23514") // PostgreSQL error code
}

// isNotNullError checks if error is a NOT NULL violation
func (h *ErrorHandler) isNotNullError(err error) bool {
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "not null") ||
		strings.Contains(errStr, "cannot be null") ||
		strings.Contains(errStr, "23502") || // PostgreSQL error code
		strings.Contains(errStr, "1048") // MySQL error code
}

// isConnectionError checks if error is connection-related
func (h *ErrorHandler) isConnectionError(err error) bool {
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "connect") ||
		strings.Contains(errStr, "refused") ||
		strings.Contains(errStr, "closed") ||
		strings.Contains(errStr, "dial") ||
		strings.Contains(errStr, "no such host")
}

// isTimeoutError checks if error is timeout-related
func (h *ErrorHandler) isTimeoutError(err error) bool {
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline") ||
		strings.Contains(errStr, "context canceled")
}

// isPermissionError checks if error is permission-related
func (h *ErrorHandler) isPermissionError(err error) bool {
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "permission") ||
		strings.Contains(errStr, "denied") ||
		strings.Contains(errStr, "privilege") ||
		strings.Contains(errStr, "access denied") ||
		strings.Contains(errStr, "1044") || // MySQL error code
		strings.Contains(errStr, "1045") || // MySQL error code
		strings.Contains(errStr, "42501") // PostgreSQL error code
}

// sanitizePattern pairs a pre-compiled regex with its replacement string.
type sanitizePattern struct {
	re          *regexp.Regexp
	replacement string
}

// sanitizePatterns are pre-compiled patterns for removing sensitive information
// from error messages. These patterns are intentionally broad to avoid leaking
// credentials from driver/SDK error strings (DSNs, query params, etc.).
var sanitizePatterns = []sanitizePattern{
	{regexp.MustCompile(`(?i)\bpassword=\S+`), "[REDACTED]"},
	{regexp.MustCompile(`(?i)\bpasswd=\S+`), "[REDACTED]"},
	{regexp.MustCompile(`(?i)\bpwd=\S+`), "[REDACTED]"},
	{regexp.MustCompile(`(?i)\btoken=\S+`), "[REDACTED]"},
	{regexp.MustCompile(`(?i)\bapi_key=\S+`), "[REDACTED]"},
	{regexp.MustCompile(`(?i)\baccess_key_id=\S+`), "[REDACTED]"},
	{regexp.MustCompile(`(?i)\bsecret_access_key=\S+`), "[REDACTED]"},
	{regexp.MustCompile(`(?i)\bsession_token=\S+`), "[REDACTED]"},
	{regexp.MustCompile(`(?i)\bkey=\S+`), "[REDACTED]"},
	{regexp.MustCompile(`(?i)\bsecret=\S+`), "[REDACTED]"},
	{regexp.MustCompile(`(?i)://[^/\s]+@`), "://[REDACTED]@"},
	{regexp.MustCompile(`@[\w.\-]+:\d+`), "[REDACTED]"},
}

// sanitizeErrorMessage removes sensitive information from error messages
func (h *ErrorHandler) sanitizeErrorMessage(err error) string {
	msg := err.Error()

	for _, p := range sanitizePatterns {
		msg = p.re.ReplaceAllString(msg, p.replacement)
	}

	// Truncate very long error messages
	if len(msg) > 500 {
		msg = msg[:500] + "..."
	}

	return msg
}
