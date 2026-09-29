package sqlguard

import "testing"

func TestProtectedReadsFailClosed(t *testing.T) {
	for _, query := range []string{
		"SELECT * INTO copy FROM users", "SELECT * FROM users INTO OUTFILE '/tmp/dump'",
		"SELECT data FROM blobs INTO DUMPFILE '/tmp/dump'", "SELECT 1 INTO\nOUTFILE '/tmp/dump'",
		"SELECT pg_read_file('/etc/passwd', 0, 100000)", "SELECT lo_export(1, '/tmp/export')",
		"SELECT LOAD_FILE('/etc/passwd')", "SELECT 1; USE other_database",
		"SELECT 1; FLURB anything", "SELECT 1 /*! INTO OUTFILE '/tmp/dump' */",
		"SELECT 1 /*M! INTO OUTFILE '/tmp/dump' */", "SELECT set_config('search_path', 'evil', false)",
		"SELECT nextval('seq')", "SELECT seq.nextval FROM dual", `SELECT seq."NEXTVAL" FROM dual`, "SELECT pg_advisory_lock(1)", "SELECT readfile('/tmp/secret')",
		"SELECT my_custom_function()", `SELECT "my_custom_function"()`,
		"SELECT * FROM external_table('https://example.invalid')", "PRAGMA query_only(OFF)",
		"PRAGMA wal_checkpoint", "USE analytics", "SELECT 1 FOR SHARE", "SELECT @x := 1",
		"SELECT 1 SETTINGS readonly=0", "SELECT 'unterminated", "SELECT 1 /* unterminated",
		"SELECT 1; SELECT unknown_function()",
	} {
		t.Run(query, func(t *testing.T) {
			if cls := Classify(query); !cls.Mutating {
				t.Errorf("unsafe or unsupported query accepted: %+v", cls)
			}
		})
	}
}

func TestProtectedReadCompatibility(t *testing.T) {
	for _, query := range []string{
		"SELECT * FROM users", "SELECT id, name FROM users WHERE id = $1",
		"SELECT 'DELETE; INTO OUTFILE' AS note", "SELECT count(*) FROM users",
		"SELECT u.id, sum(o.total) AS total FROM users AS u LEFT JOIN orders AS o ON o.user_id = u.id WHERE u.id > 0 GROUP BY u.id HAVING count(*) > 1 ORDER BY u.id DESC LIMIT 10 OFFSET 2",
		"WITH c AS (SELECT id FROM users WHERE id = ?) SELECT * FROM c",
		"SELECT * FROM users WHERE id IN (SELECT user_id FROM orders)",
		"SELECT 1 UNION ALL SELECT 2", "SELECT 1; SELECT 2", "VALUES (1), (2)",
		"SELECT \"delete\" FROM \"users\"", "/* comment */ SELECT 1 -- comment\n",
	} {
		if cls := Classify(query); cls.Mutating {
			t.Errorf("read rejected: %s: %s", query, cls.Reason)
		}
	}
}

func FuzzProtectedRead(f *testing.F) {
	for _, seed := range []string{"SELECT 1", "SELECT 'x'", "WITH c AS(DELETE FROM t RETURNING *) SELECT * FROM c", "SELECT 1 /*! INTO OUTFILE 'x' */", "SELECT 1; USE db"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, query string) {
		cls := Classify(query)
		if !cls.Mutating && !Classify(query+"; DELETE FROM guard_table").Mutating {
			t.Fatal("trailing write accepted")
		}
	})
}
