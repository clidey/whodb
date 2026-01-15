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
	"testing"
)

func TestDetectStatementType(t *testing.T) {
	cases := []struct {
		name     string
		query    string
		expected StatementType
	}{
		{"select", "SELECT * FROM users", StatementSelect},
		{"select lowercase", "select id from users", StatementSelect},
		{"select with whitespace", "  SELECT id FROM users", StatementSelect},
		{"insert", "INSERT INTO users VALUES (1)", StatementInsert},
		{"update", "UPDATE users SET name='bob'", StatementUpdate},
		{"delete", "DELETE FROM users WHERE id=1", StatementDelete},
		{"drop table", "DROP TABLE users", StatementDrop},
		{"create table", "CREATE TABLE users (id int)", StatementCreate},
		{"alter table", "ALTER TABLE users ADD col int", StatementAlter},
		{"truncate", "TRUNCATE TABLE users", StatementTruncate},
		{"show", "SHOW TABLES", StatementShow},
		{"describe", "DESCRIBE users", StatementDescribe},
		{"desc shorthand", "DESC users", StatementDescribe},
		{"explain", "EXPLAIN SELECT * FROM users", StatementExplain},
		{"with cte", "WITH cte AS (SELECT 1) SELECT * FROM cte", StatementWith},
		{"unknown", "GRANT ALL ON users TO admin", StatementUnknown},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DetectStatementType(tc.query)
			if got != tc.expected {
				t.Errorf("DetectStatementType(%q) = %v, want %v", tc.query, got, tc.expected)
			}
		})
	}
}

func TestValidateSQLStatement_ReadOnly(t *testing.T) {
	cases := []struct {
		name      string
		query     string
		expectErr bool
	}{
		{"select allowed", "SELECT * FROM users", false},
		{"show allowed", "SHOW TABLES", false},
		{"describe allowed", "DESCRIBE users", false},
		{"explain allowed", "EXPLAIN SELECT * FROM users", false},
		{"insert blocked", "INSERT INTO users VALUES (1)", true},
		{"update blocked", "UPDATE users SET name='bob'", true},
		{"delete blocked", "DELETE FROM users", true},
		{"drop blocked", "DROP TABLE users", true},
		{"create blocked", "CREATE TABLE foo (id int)", true},
		{"alter blocked", "ALTER TABLE users ADD col int", true},
		{"truncate blocked", "TRUNCATE TABLE users", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSQLStatement(tc.query, false, SecurityLevelStandard, false, false)
			if tc.expectErr && err == nil {
				t.Errorf("expected error for query %q, got nil", tc.query)
			}
			if !tc.expectErr && err != nil {
				t.Errorf("unexpected error for query %q: %v", tc.query, err)
			}
		})
	}
}

func TestValidateSQLStatement_AllowWrite(t *testing.T) {
	cases := []struct {
		name      string
		query     string
		expectErr bool
	}{
		{"select allowed", "SELECT * FROM users", false},
		{"insert allowed", "INSERT INTO users VALUES (1)", false},
		{"update allowed", "UPDATE users SET name='bob'", false},
		{"delete allowed", "DELETE FROM users WHERE id=1", false},
		{"create allowed", "CREATE TABLE foo (id int)", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSQLStatement(tc.query, true, SecurityLevelStandard, false, false)
			if tc.expectErr && err == nil {
				t.Errorf("expected error for query %q, got nil", tc.query)
			}
			if !tc.expectErr && err != nil {
				t.Errorf("unexpected error for query %q: %v", tc.query, err)
			}
		})
	}
}

func TestValidateSQLStatement_MultiStatement(t *testing.T) {
	cases := []struct {
		name       string
		query      string
		allowMulti bool
		expectErr  bool
	}{
		{"single statement", "SELECT * FROM users", false, false},
		{"trailing semicolon ok", "SELECT * FROM users;", false, false},
		{"multi blocked", "SELECT 1; DROP TABLE users", false, true},
		{"multi allowed", "SELECT 1; SELECT 2", true, false},
		{"semicolon in string blocked", "SELECT 'a;b' FROM users", false, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSQLStatement(tc.query, false, SecurityLevelStandard, tc.allowMulti, false)
			if tc.expectErr && err == nil {
				t.Errorf("expected error for query %q, got nil", tc.query)
			}
			if !tc.expectErr && err != nil {
				t.Errorf("unexpected error for query %q: %v", tc.query, err)
			}
		})
	}
}

func TestValidateSQLStatement_StrictMode(t *testing.T) {
	cases := []struct {
		name      string
		query     string
		expectErr bool
	}{
		{"normal select ok", "SELECT * FROM users", false},
		{"pg_terminate_backend blocked", "SELECT pg_terminate_backend(123)", true},
		{"pg_read_file blocked", "SELECT pg_read_file('/etc/passwd')", true},
		{"lo_import blocked", "SELECT lo_import('/tmp/file')", true},
		{"LOAD_FILE blocked", "SELECT LOAD_FILE('/etc/passwd')", true},
		{"INTO OUTFILE blocked", "SELECT * INTO OUTFILE '/tmp/out' FROM users", true},
		{"COPY blocked", "COPY users TO '/tmp/out'", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSQLStatement(tc.query, false, SecurityLevelStrict, false, false)
			if tc.expectErr && err == nil {
				t.Errorf("expected error for query %q, got nil", tc.query)
			}
			if !tc.expectErr && err != nil {
				t.Errorf("unexpected error for query %q: %v", tc.query, err)
			}
		})
	}
}

func TestValidateSQLStatement_MinimalMode(t *testing.T) {
	cases := []struct {
		name       string
		query      string
		allowWrite bool
		expectErr  bool
	}{
		{"select allowed", "SELECT * FROM users", false, false},
		{"drop blocked even with write", "DROP TABLE users", true, true},
		{"truncate blocked even with write", "TRUNCATE users", true, true},
		{"delete without where blocked", "DELETE FROM users", true, true},
		{"delete with where allowed", "DELETE FROM users WHERE id=1", true, false},
		{"insert allowed with write", "INSERT INTO users VALUES (1)", true, false},
		{"update allowed with write", "UPDATE users SET name='x'", true, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSQLStatement(tc.query, tc.allowWrite, SecurityLevelMinimal, false, false)
			if tc.expectErr && err == nil {
				t.Errorf("expected error for query %q, got nil", tc.query)
			}
			if !tc.expectErr && err != nil {
				t.Errorf("unexpected error for query %q: %v", tc.query, err)
			}
		})
	}
}

func TestValidateSQLStatement_Comments(t *testing.T) {
	cases := []struct {
		name      string
		query     string
		expectErr bool
	}{
		{"single line comment blocked", "SELECT * FROM users -- drop table", true},
		{"block comment blocked", "SELECT * /* drop */ FROM users", true},
		{"no comment ok", "SELECT * FROM users", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSQLStatement(tc.query, false, SecurityLevelStandard, false, false)
			if tc.expectErr && err == nil {
				t.Errorf("expected error for query %q, got nil", tc.query)
			}
			if !tc.expectErr && err != nil {
				t.Errorf("unexpected error for query %q: %v", tc.query, err)
			}
		})
	}
}

func TestIsReadOnlyStatement(t *testing.T) {
	if !IsReadOnlyStatement(StatementSelect) {
		t.Error("SELECT should be read-only")
	}
	if !IsReadOnlyStatement(StatementShow) {
		t.Error("SHOW should be read-only")
	}
	if IsReadOnlyStatement(StatementInsert) {
		t.Error("INSERT should not be read-only")
	}
	if IsReadOnlyStatement(StatementDelete) {
		t.Error("DELETE should not be read-only")
	}
}

func TestIsWriteStatement(t *testing.T) {
	if IsWriteStatement(StatementSelect) {
		t.Error("SELECT should not be a write statement")
	}
	if !IsWriteStatement(StatementInsert) {
		t.Error("INSERT should be a write statement")
	}
	if !IsWriteStatement(StatementUpdate) {
		t.Error("UPDATE should be a write statement")
	}
	if !IsWriteStatement(StatementDelete) {
		t.Error("DELETE should be a write statement")
	}
	if !IsWriteStatement(StatementDrop) {
		t.Error("DROP should be a write statement")
	}
}

func TestValidateSQLStatement_AllowDestructive(t *testing.T) {
	cases := []struct {
		name             string
		query            string
		allowWrite       bool
		allowDestructive bool
		expectErr        bool
	}{
		{"drop blocked without flag", "DROP TABLE users", true, false, true},
		{"truncate blocked without flag", "TRUNCATE TABLE users", true, false, true},
		{"drop allowed with flag", "DROP TABLE users", true, true, false},
		{"truncate allowed with flag", "TRUNCATE TABLE users", true, true, false},
		{"drop lowercase blocked", "drop table users", true, false, true},
		{"truncate mixed case blocked", "Truncate Table users", true, false, true},
		{"drop injection blocked", "SELECT 1; DROP TABLE x", true, false, true},
		{"truncate injection blocked", "INSERT INTO x VALUES(1); TRUNCATE y", true, false, true},
		{"insert allowed without destructive", "INSERT INTO users VALUES (1)", true, false, false},
		{"update allowed without destructive", "UPDATE users SET x=1", true, false, false},
		{"delete allowed without destructive", "DELETE FROM users WHERE id=1", true, false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSQLStatement(tc.query, tc.allowWrite, SecurityLevelMinimal, true, tc.allowDestructive)
			if tc.expectErr && err == nil {
				t.Errorf("expected error for query %q, got nil", tc.query)
			}
			if !tc.expectErr && err != nil {
				t.Errorf("unexpected error for query %q: %v", tc.query, err)
			}
		})
	}
}

func TestValidateSQLStatement_DestructiveSafetyNet(t *testing.T) {
	cases := []struct {
		name  string
		query string
	}{
		{"drop at start", "DROP TABLE users"},
		{"drop after semicolon", "SELECT 1; DROP TABLE users"},
		{"drop in subquery attempt", "SELECT * FROM (DROP TABLE users)"},
		{"truncate at start", "TRUNCATE TABLE users"},
		{"truncate after semicolon", "UPDATE x SET y=1; TRUNCATE z"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSQLStatement(tc.query, true, SecurityLevelMinimal, true, false)
			if err == nil {
				t.Errorf("expected error for query %q (safety net should catch)", tc.query)
			}
		})
	}
}
