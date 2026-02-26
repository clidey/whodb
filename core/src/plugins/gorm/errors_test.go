package gorm_plugin

import (
	"errors"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestErrorHandlerHandleErrorMapsCommonCases(t *testing.T) {
	h := NewErrorHandler(nil)

	cases := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "duplicate key",
			err:      errors.New("duplicate key value violates unique constraint \"users_email_key\""),
			expected: "duplicate key: a record with these values already exists",
		},
		{
			name:     "foreign key",
			err:      errors.New("violates foreign key constraint"),
			expected: "foreign key constraint: referenced record does not exist or is in use",
		},
		{
			name:     "check constraint",
			err:      errors.New("check constraint violation"),
			expected: "check constraint violation: value does not meet requirements",
		},
		{
			name:     "not null",
			err:      errors.New("cannot be null"),
			expected: "required field cannot be empty",
		},
		{
			name:     "connection error",
			err:      errors.New("dial tcp: no such host"),
			expected: "database connection error: please check your connection settings",
		},
		{
			name:     "timeout",
			err:      errors.New("context deadline exceeded"),
			expected: "operation timed out: the database took too long to respond",
		},
		{
			name:     "permission",
			err:      errors.New("permission denied"),
			expected: "permission denied: insufficient privileges for this operation",
		},
		{
			name:     "missing where clause",
			err:      gorm.ErrMissingWhereClause,
			expected: "WHERE clause required for this operation",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := h.HandleError(tc.err, "op", map[string]any{"k": "v"})
			if got == nil {
				t.Fatalf("expected error, got nil")
			}
			if got.Error() != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got.Error())
			}
		})
	}
}

func TestErrorHandlerSanitizeErrorMessageRedactsSecrets(t *testing.T) {
	h := NewErrorHandler(nil)

	raw := strings.Join([]string{
		"connect failed:",
		"password=supersecret",
		"token=abcd",
		"api_key=key123",
		"postgres://user:pass@localhost:5432/db",
	}, " ")

	sanitized := h.sanitizeErrorMessage(errors.New(raw))

	for _, leaked := range []string{"supersecret", "abcd", "key123", "user:pass", "@localhost:5432"} {
		if strings.Contains(sanitized, leaked) {
			t.Fatalf("expected sanitized message to redact %q, got %q", leaked, sanitized)
		}
	}
	if !strings.Contains(sanitized, "[REDACTED]") {
		t.Fatalf("expected sanitized message to contain redaction marker, got %q", sanitized)
	}
}

func TestErrorHandlerSanitizeErrorMessageTruncatesLongStrings(t *testing.T) {
	h := NewErrorHandler(nil)

	raw := strings.Repeat("a", 600)
	sanitized := h.sanitizeErrorMessage(errors.New(raw))

	if len(sanitized) != 503 {
		t.Fatalf("expected 503 chars (500 + ...), got %d", len(sanitized))
	}
	if !strings.HasSuffix(sanitized, "...") {
		t.Fatalf("expected suffix ..., got %q", sanitized[len(sanitized)-10:])
	}
}
