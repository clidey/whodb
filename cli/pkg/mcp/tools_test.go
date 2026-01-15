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

package mcp

import (
	"strings"
	"testing"
	"time"
)

// TestHandleQuery_ReadOnlyBlocksWrites tests that HandleQuery blocks write operations
// in read-only mode by checking the validation before any database connection is attempted.
func TestHandleQuery_ReadOnlyBlocksWrites(t *testing.T) {
	ctx := t.Context()

	// Set up read-only security options
	secOpts := &SecurityOptions{
		ReadOnly:            true,
		ConfirmWrites:       false,
		SecurityLevel:       SecurityLevelStandard,
		QueryTimeout:        30 * time.Second,
		MaxRows:             1000,
		AllowMultiStatement: false,
	}

	// Test cases for blocked queries
	blockedQueries := []struct {
		name  string
		query string
	}{
		{"INSERT blocked", "INSERT INTO users VALUES (1, 'test')"},
		{"UPDATE blocked", "UPDATE users SET name='x' WHERE id=1"},
		{"DELETE blocked", "DELETE FROM users WHERE id=1"},
		{"DROP blocked", "DROP TABLE users"},
		{"CREATE blocked", "CREATE TABLE foo (id int)"},
		{"ALTER blocked", "ALTER TABLE users ADD col int"},
		{"TRUNCATE blocked", "TRUNCATE TABLE users"},
	}

	for _, tc := range blockedQueries {
		t.Run(tc.name, func(t *testing.T) {
			input := QueryInput{
				Query:      tc.query,
				Connection: "nonexistent", // Won't reach DB connection
			}

			_, output, err := HandleQuery(ctx, nil, input, secOpts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if output.Error == "" {
				t.Errorf("expected error for query %q, got none", tc.query)
			}

			if !strings.Contains(output.Error, "blocked") && !strings.Contains(output.Error, "not allowed") {
				t.Errorf("expected 'blocked' or 'not allowed' in error, got: %s", output.Error)
			}
		})
	}
}

// TestHandleQuery_ReadOnlyAllowsSelects tests that SELECT queries pass validation in read-only mode
func TestHandleQuery_ReadOnlyAllowsSelects(t *testing.T) {
	ctx := t.Context()

	secOpts := &SecurityOptions{
		ReadOnly:            true,
		ConfirmWrites:       false,
		SecurityLevel:       SecurityLevelStandard,
		QueryTimeout:        30 * time.Second,
		MaxRows:             1000,
		AllowMultiStatement: false,
	}

	// These queries should pass validation (they'll fail at connection, not validation)
	allowedQueries := []string{
		"SELECT * FROM users",
		"SELECT id, name FROM users WHERE id = 1",
		"SHOW TABLES",
		"DESCRIBE users",
		"EXPLAIN SELECT * FROM users",
	}

	for _, query := range allowedQueries {
		t.Run(query, func(t *testing.T) {
			input := QueryInput{
				Query:      query,
				Connection: "nonexistent",
			}

			_, output, err := HandleQuery(ctx, nil, input, secOpts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Should fail at connection resolution, NOT at query validation
			if strings.Contains(output.Error, "blocked") || strings.Contains(output.Error, "not allowed") {
				t.Errorf("SELECT should not be blocked, got error: %s", output.Error)
			}
		})
	}
}

// TestHandleQuery_ConfirmWritesMode tests that write operations return confirmation requests
func TestHandleQuery_ConfirmWritesMode(t *testing.T) {
	ctx := t.Context()

	secOpts := &SecurityOptions{
		ReadOnly:            false, // Writes allowed, but...
		ConfirmWrites:       true,  // ...require confirmation
		SecurityLevel:       SecurityLevelStandard,
		QueryTimeout:        30 * time.Second,
		MaxRows:             1000,
		AllowMultiStatement: false,
	}

	// Write queries should return confirmation request, not error
	writeQueries := []string{
		"INSERT INTO users VALUES (1, 'test')",
		"UPDATE users SET name='x' WHERE id=1",
		"DELETE FROM users WHERE id=1",
	}

	for _, query := range writeQueries {
		t.Run(query, func(t *testing.T) {
			input := QueryInput{
				Query:      query,
				Connection: "test_conn",
			}

			_, output, err := HandleQuery(ctx, nil, input, secOpts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !output.ConfirmationRequired {
				t.Errorf("expected ConfirmationRequired=true for write query")
			}

			if output.ConfirmationToken == "" {
				t.Error("expected ConfirmationToken to be set")
			}

			if output.ConfirmationQuery != query {
				t.Errorf("ConfirmationQuery mismatch: got %q, want %q", output.ConfirmationQuery, query)
			}
		})
	}
}

// TestHandleQuery_MultiStatementBlocked tests that multi-statement queries are blocked
func TestHandleQuery_MultiStatementBlocked(t *testing.T) {
	ctx := t.Context()

	secOpts := &SecurityOptions{
		ReadOnly:            true,
		ConfirmWrites:       false,
		SecurityLevel:       SecurityLevelStandard,
		QueryTimeout:        30 * time.Second,
		MaxRows:             1000,
		AllowMultiStatement: false, // Multi-statement blocked
	}

	input := QueryInput{
		Query:      "SELECT 1; SELECT 2", // Multi-statement without DROP
		Connection: "nonexistent",
	}

	_, output, err := HandleQuery(ctx, nil, input, secOpts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Error == "" {
		t.Error("expected error for multi-statement query")
	}

	if !strings.Contains(output.Error, "multiple") {
		t.Errorf("expected 'multiple' in error message, got: %s", output.Error)
	}
}

// TestHandleQuery_DropBlockedWithoutFlag tests that DROP is blocked in allow-write mode
func TestHandleQuery_DropBlockedWithoutFlag(t *testing.T) {
	ctx := t.Context()

	// allow-write but NOT allow-drop: DROP should be blocked
	secOpts := &SecurityOptions{
		ReadOnly:            false,
		ConfirmWrites:       false,
		AllowDrop:           false, // No --allow-drop flag
		SecurityLevel:       SecurityLevelMinimal,
		QueryTimeout:        30 * time.Second,
		MaxRows:             1000,
		AllowMultiStatement: true,
	}

	dangerousQueries := []string{
		"DROP TABLE users",
		"TRUNCATE TABLE users",
	}

	for _, query := range dangerousQueries {
		t.Run(query, func(t *testing.T) {
			input := QueryInput{
				Query:      query,
				Connection: "nonexistent",
			}

			_, output, err := HandleQuery(ctx, nil, input, secOpts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if output.Error == "" {
				t.Errorf("expected error for destructive query %q", query)
			}

			if !strings.Contains(output.Error, "destructive") {
				t.Errorf("expected 'destructive' in error, got: %s", output.Error)
			}
		})
	}
}

// TestHandleQuery_DropConfirmationWithConfirmWrites tests DROP goes through confirmation
func TestHandleQuery_DropConfirmationWithConfirmWrites(t *testing.T) {
	ctx := t.Context()

	// With --confirm-writes, DROP should require confirmation (not be blocked)
	secOpts := &SecurityOptions{
		ReadOnly:            false,
		ConfirmWrites:       true, // Human-in-the-loop
		SecurityLevel:       SecurityLevelStandard,
		QueryTimeout:        30 * time.Second,
		MaxRows:             1000,
		AllowMultiStatement: false,
	}

	input := QueryInput{
		Query:      "DROP TABLE users",
		Connection: "test_conn",
	}

	_, output, err := HandleQuery(ctx, nil, input, secOpts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should require confirmation, not return error
	if output.Error != "" {
		t.Errorf("DROP should go through confirmation, not be blocked: %s", output.Error)
	}

	if !output.ConfirmationRequired {
		t.Error("expected ConfirmationRequired=true for DROP with --confirm-writes")
	}

	if output.ConfirmationToken == "" {
		t.Error("expected ConfirmationToken to be set")
	}
}

// TestHandleQuery_StrictModeBlocksDangerousFunctions tests strict security level
func TestHandleQuery_StrictModeBlocksDangerousFunctions(t *testing.T) {
	ctx := t.Context()

	secOpts := &SecurityOptions{
		ReadOnly:            true,
		ConfirmWrites:       false,
		SecurityLevel:       SecurityLevelStrict, // Strict mode
		QueryTimeout:        30 * time.Second,
		MaxRows:             1000,
		AllowMultiStatement: false,
	}

	dangerousQueries := []string{
		"SELECT pg_terminate_backend(123)",
		"SELECT pg_read_file('/etc/passwd')",
		"SELECT LOAD_FILE('/etc/passwd')",
	}

	for _, query := range dangerousQueries {
		t.Run(query, func(t *testing.T) {
			input := QueryInput{
				Query:      query,
				Connection: "nonexistent",
			}

			_, output, err := HandleQuery(ctx, nil, input, secOpts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if output.Error == "" {
				t.Errorf("expected error for dangerous function in query %q", query)
			}
		})
	}
}

// TestHandleQuery_CommentsBlocked tests that SQL comments are blocked
func TestHandleQuery_CommentsBlocked(t *testing.T) {
	ctx := t.Context()

	secOpts := &SecurityOptions{
		ReadOnly:            true,
		ConfirmWrites:       false,
		SecurityLevel:       SecurityLevelStandard,
		QueryTimeout:        30 * time.Second,
		MaxRows:             1000,
		AllowMultiStatement: false,
	}

	commentQueries := []string{
		"SELECT * FROM users -- this is a comment",
		"SELECT * FROM users /* block comment */",
	}

	for _, query := range commentQueries {
		t.Run(query, func(t *testing.T) {
			input := QueryInput{
				Query:      query,
				Connection: "nonexistent",
			}

			_, output, err := HandleQuery(ctx, nil, input, secOpts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if output.Error == "" {
				t.Errorf("expected error for comment in query %q", query)
			}

			if !strings.Contains(output.Error, "comment") {
				t.Errorf("expected 'comment' in error message, got: %s", output.Error)
			}
		})
	}
}

// TestPendingConfirmation tests the confirmation token storage and retrieval
func TestPendingConfirmation(t *testing.T) {
	query := "INSERT INTO test VALUES (1)"
	connection := "test_conn"

	// Store a pending confirmation
	token := storePendingConfirmation(query, connection)
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Retrieve it
	pending, err := getPendingConfirmation(token)
	if err != nil {
		t.Fatalf("getPendingConfirmation failed: %v", err)
	}

	if pending.Query != query {
		t.Errorf("Query mismatch: got %q, want %q", pending.Query, query)
	}

	if pending.Connection != connection {
		t.Errorf("Connection mismatch: got %q, want %q", pending.Connection, connection)
	}

	// Second retrieval should fail (one-time use)
	_, err = getPendingConfirmation(token)
	if err == nil {
		t.Error("expected error on second retrieval (token should be consumed)")
	}
}

// TestPendingConfirmation_InvalidToken tests that invalid tokens are rejected
func TestPendingConfirmation_InvalidToken(t *testing.T) {
	_, err := getPendingConfirmation("invalid_token_12345")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

// TestHandleConfirm_InvalidToken tests that HandleConfirm rejects invalid tokens
func TestHandleConfirm_InvalidToken(t *testing.T) {
	ctx := t.Context()
	secOpts := &SecurityOptions{
		ReadOnly:      false,
		ConfirmWrites: true,
		QueryTimeout:  30 * time.Second,
	}

	input := ConfirmInput{
		Token: "invalid_token_xyz",
	}

	_, output, err := HandleConfirm(ctx, nil, input, secOpts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Error == "" {
		t.Error("expected error for invalid token")
	}

	if output.Message != "" {
		t.Error("Message should be empty for invalid token")
	}
}

// TestHandleQuery_DropAllowedWithFlag tests that DROP passes when --allow-drop is set
func TestHandleQuery_DropAllowedWithFlag(t *testing.T) {
	ctx := t.Context()

	// allow-write AND allow-drop: DROP should pass validation (fail at connection)
	secOpts := &SecurityOptions{
		ReadOnly:            false,
		ConfirmWrites:       false,
		AllowDrop:           true, // --allow-drop flag set
		SecurityLevel:       SecurityLevelMinimal,
		QueryTimeout:        30 * time.Second,
		MaxRows:             1000,
		AllowMultiStatement: false,
	}

	dangerousQueries := []string{
		"DROP TABLE users",
		"TRUNCATE TABLE users",
	}

	for _, query := range dangerousQueries {
		t.Run(query, func(t *testing.T) {
			input := QueryInput{
				Query:      query,
				Connection: "nonexistent",
			}

			_, output, err := HandleQuery(ctx, nil, input, secOpts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Should NOT be blocked by destructive check (will fail at connection resolution instead)
			if strings.Contains(output.Error, "destructive") {
				t.Errorf("DROP should pass with --allow-drop, got: %s", output.Error)
			}
		})
	}
}

// TestHandleQuery_SelectInConfirmWritesMode tests that SELECT doesn't require confirmation
func TestHandleQuery_SelectInConfirmWritesMode(t *testing.T) {
	ctx := t.Context()

	secOpts := &SecurityOptions{
		ReadOnly:            false,
		ConfirmWrites:       true, // confirm-writes enabled
		SecurityLevel:       SecurityLevelStandard,
		QueryTimeout:        30 * time.Second,
		MaxRows:             1000,
		AllowMultiStatement: false,
	}

	// SELECT should NOT require confirmation
	input := QueryInput{
		Query:      "SELECT * FROM users",
		Connection: "nonexistent",
	}

	_, output, err := HandleQuery(ctx, nil, input, secOpts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.ConfirmationRequired {
		t.Error("SELECT should not require confirmation")
	}

	if output.ConfirmationToken != "" {
		t.Error("SELECT should not have a confirmation token")
	}
}

// TestHandleQuery_TruncateConfirmationWithConfirmWrites tests TRUNCATE goes through confirmation
func TestHandleQuery_TruncateConfirmationWithConfirmWrites(t *testing.T) {
	ctx := t.Context()

	secOpts := &SecurityOptions{
		ReadOnly:            false,
		ConfirmWrites:       true, // Human-in-the-loop
		SecurityLevel:       SecurityLevelStandard,
		QueryTimeout:        30 * time.Second,
		MaxRows:             1000,
		AllowMultiStatement: false,
	}

	input := QueryInput{
		Query:      "TRUNCATE TABLE users",
		Connection: "test_conn",
	}

	_, output, err := HandleQuery(ctx, nil, input, secOpts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Error != "" {
		t.Errorf("TRUNCATE should go through confirmation, not be blocked: %s", output.Error)
	}

	if !output.ConfirmationRequired {
		t.Error("expected ConfirmationRequired=true for TRUNCATE with --confirm-writes")
	}
}

// TestHandleQuery_DropCaseInsensitive tests that DROP detection is case-insensitive
func TestHandleQuery_DropCaseInsensitive(t *testing.T) {
	ctx := t.Context()

	secOpts := &SecurityOptions{
		ReadOnly:            false,
		ConfirmWrites:       false,
		AllowDrop:           false, // DROP should be blocked
		SecurityLevel:       SecurityLevelMinimal,
		QueryTimeout:        30 * time.Second,
		MaxRows:             1000,
		AllowMultiStatement: false,
	}

	// Test various case combinations
	caseVariations := []string{
		"drop table users",
		"Drop Table users",
		"DROP TABLE users",
		"dRoP tAbLe users",
	}

	for _, query := range caseVariations {
		t.Run(query, func(t *testing.T) {
			input := QueryInput{
				Query:      query,
				Connection: "nonexistent",
			}

			_, output, err := HandleQuery(ctx, nil, input, secOpts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if output.Error == "" {
				t.Errorf("expected error for case variation %q", query)
			}

			if !strings.Contains(output.Error, "destructive") {
				t.Errorf("expected 'destructive' in error for %q, got: %s", query, output.Error)
			}
		})
	}
}

// TestHandleQuery_DropInjectionInMultiStatement tests DROP is caught even in multi-statement mode
func TestHandleQuery_DropInjectionInMultiStatement(t *testing.T) {
	ctx := t.Context()

	secOpts := &SecurityOptions{
		ReadOnly:            false,
		ConfirmWrites:       false,
		AllowDrop:           false,
		SecurityLevel:       SecurityLevelMinimal,
		QueryTimeout:        30 * time.Second,
		MaxRows:             1000,
		AllowMultiStatement: true, // Multi-statement allowed, but DROP should still be caught
	}

	injectionAttempts := []string{
		"SELECT 1; DROP TABLE users",
		"INSERT INTO log VALUES (1); DROP TABLE users",
		"UPDATE x SET y=1; TRUNCATE TABLE users",
	}

	for _, query := range injectionAttempts {
		t.Run(query, func(t *testing.T) {
			input := QueryInput{
				Query:      query,
				Connection: "nonexistent",
			}

			_, output, err := HandleQuery(ctx, nil, input, secOpts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if output.Error == "" {
				t.Errorf("expected error for injection attempt %q", query)
			}

			// Should be caught by the DROP/TRUNCATE safety net
			if !strings.Contains(output.Error, "destructive") && !strings.Contains(output.Error, "DROP") && !strings.Contains(output.Error, "TRUNCATE") {
				t.Errorf("expected destructive operation error for %q, got: %s", query, output.Error)
			}
		})
	}
}

// TestHandleQuery_EmptyQuery tests that empty queries are rejected
func TestHandleQuery_EmptyQuery(t *testing.T) {
	ctx := t.Context()

	secOpts := &SecurityOptions{
		ReadOnly:            true,
		ConfirmWrites:       false,
		SecurityLevel:       SecurityLevelStandard,
		QueryTimeout:        30 * time.Second,
		MaxRows:             1000,
		AllowMultiStatement: false,
	}

	emptyQueries := []string{
		"",
		"   ",
		"\t\n",
	}

	for _, query := range emptyQueries {
		t.Run("empty", func(t *testing.T) {
			input := QueryInput{
				Query:      query,
				Connection: "nonexistent",
			}

			_, output, err := HandleQuery(ctx, nil, input, secOpts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if output.Error == "" {
				t.Error("expected error for empty query")
			}
		})
	}
}
