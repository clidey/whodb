//go:build ignore

// Seeds the DuckDB E2E fixture using the same duckdb-go driver the server
// links, so the file's storage format always matches what the server can read.
//
// Usage (from core/ so the driver resolves from core/go.mod):
//
//	go run ../dev/sample-data/duckdb/seed_duckdb.go <db-path> <sql-path>
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("usage: %s <db-path> <sql-path>", os.Args[0])
	}
	dbPath, sqlPath := os.Args[1], os.Args[2]

	script, err := os.ReadFile(sqlPath)
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(string(script)); err != nil {
		log.Fatalf("seeding %s failed: %v", dbPath, err)
	}

	fmt.Printf("✅ Created %s with seed data\n", dbPath)
}
