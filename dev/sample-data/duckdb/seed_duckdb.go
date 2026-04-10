//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	db, err := sql.Open("duckdb", "tmp/e2e_test.duckdb")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	stmts := []string{
		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY,
			username VARCHAR NOT NULL,
			email VARCHAR NOT NULL,
			password VARCHAR NOT NULL,
			created_at TIMESTAMP DEFAULT current_timestamp
		)`,
		`DELETE FROM users`,
		`INSERT INTO users (id, username, email, password) VALUES
			(1, 'john_doe', 'john@example.com', 'securepassword1'),
			(2, 'jane_smith', 'jane@example.com', 'securepassword2'),
			(3, 'admin_user', 'admin@example.com', 'adminpass1')`,

		// Products table
		`CREATE TABLE IF NOT EXISTS products (
			id INTEGER PRIMARY KEY,
			name VARCHAR NOT NULL,
			price DOUBLE NOT NULL,
			category VARCHAR
		)`,
		`DELETE FROM products`,
		`INSERT INTO products (id, name, price, category) VALUES
			(1, 'Widget', 9.99, 'gadgets'),
			(2, 'Gizmo', 19.99, 'gadgets'),
			(3, 'Thingamajig', 29.99, 'tools')`,

		// Orders table (FK to users)
		`CREATE TABLE IF NOT EXISTS orders (
			id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id),
			total DOUBLE NOT NULL,
			status VARCHAR DEFAULT 'pending',
			created_at TIMESTAMP DEFAULT current_timestamp
		)`,
		`DELETE FROM orders`,
		`INSERT INTO orders (id, user_id, total, status) VALUES
			(1, 1, 29.98, 'completed'),
			(2, 2, 19.99, 'pending'),
			(3, 1, 59.97, 'completed')`,

		// Order items table (FK to orders and products)
		`CREATE TABLE IF NOT EXISTS order_items (
			id INTEGER PRIMARY KEY,
			order_id INTEGER NOT NULL REFERENCES orders(id),
			product_id INTEGER NOT NULL REFERENCES products(id),
			quantity INTEGER NOT NULL,
			price DOUBLE NOT NULL
		)`,
		`DELETE FROM order_items`,
		`INSERT INTO order_items (id, order_id, product_id, quantity, price) VALUES
			(1, 1, 1, 2, 9.99),
			(2, 1, 2, 1, 19.99),
			(3, 2, 2, 1, 19.99),
			(4, 3, 3, 3, 29.99)`,

		// Payments table (FK to orders)
		`CREATE TABLE IF NOT EXISTS payments (
			id INTEGER PRIMARY KEY,
			order_id INTEGER NOT NULL REFERENCES orders(id),
			amount DOUBLE NOT NULL,
			method VARCHAR NOT NULL,
			paid_at TIMESTAMP DEFAULT current_timestamp
		)`,
		`DELETE FROM payments`,
		`INSERT INTO payments (id, order_id, amount, method) VALUES
			(1, 1, 29.98, 'credit_card'),
			(2, 3, 59.97, 'paypal')`,

		// Order summary view
		`CREATE OR REPLACE VIEW order_summary AS
		SELECT o.id AS order_id, u.username, o.total, o.status,
			COUNT(oi.id) AS item_count
		FROM orders o
		JOIN users u ON o.user_id = u.id
		JOIN order_items oi ON oi.order_id = o.id
		GROUP BY o.id, u.username, o.total, o.status`,

		// Test casting table
		`CREATE TABLE IF NOT EXISTS test_casting (
			id INTEGER PRIMARY KEY,
			bigint_col BIGINT,
			integer_col INTEGER,
			smallint_col SMALLINT,
			numeric_col DOUBLE,
			description VARCHAR
		)`,
		`DELETE FROM test_casting`,
		`INSERT INTO test_casting (id, bigint_col, integer_col, smallint_col, numeric_col, description) VALUES
			(1, 1000000000, 100, 10, 123.45, 'test row')`,

		// Data types table
		`CREATE TABLE IF NOT EXISTS data_types (
			id INTEGER PRIMARY KEY,
			col_tinyint TINYINT,
			col_smallint SMALLINT,
			col_integer INTEGER,
			col_bigint BIGINT,
			col_hugeint HUGEINT,
			col_utinyint UTINYINT,
			col_usmallint USMALLINT,
			col_uinteger UINTEGER,
			col_ubigint UBIGINT,
			col_float FLOAT,
			col_double DOUBLE,
			col_decimal DECIMAL(10,2),
			col_varchar VARCHAR,
			col_blob BLOB,
			col_boolean BOOLEAN,
			col_date DATE,
			col_time TIME,
			col_timestamp TIMESTAMP,
			col_timestamptz TIMESTAMP WITH TIME ZONE,
			col_interval INTERVAL,
			col_json JSON,
			col_uuid UUID
		)`,
		`DELETE FROM data_types`,
		`INSERT INTO data_types VALUES (
			1, 127, 32000, 1000000, 9000000000000, 170141183460469231731687303715884105727,
			255, 65000, 4000000000, 18000000000000000000,
			3.14, 3.14159265358979, 12345.67,
			'text_value', '\x48454C4C4F'::BLOB,
			true, '2025-01-01', '12:30:00', '2025-01-01 12:00:00',
			'2025-01-01 12:00:00+00', INTERVAL '1 year 2 months 3 days',
			'{"key": "value"}'::JSON,
			'550e8400-e29b-41d4-a716-446655440000'::UUID
		)`,
	}

	for i, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			log.Fatalf("Statement %d failed: %v\nSQL: %s", i, err, stmt)
		}
	}

	fmt.Println("✅ Created tmp/e2e_test.duckdb with seed data")
}
