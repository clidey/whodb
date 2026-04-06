-- Copyright 2026 Clidey, Inc.
--
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--     http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

-- Drop tables in reverse dependency order
DROP VIEW IF EXISTS order_summary;

DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS test_casting;
DROP TABLE IF EXISTS data_types;

-- Drop sequences
DROP SEQUENCE IF EXISTS users_id_seq;
DROP SEQUENCE IF EXISTS products_id_seq;
DROP SEQUENCE IF EXISTS orders_id_seq;
DROP SEQUENCE IF EXISTS order_items_id_seq;
DROP SEQUENCE IF EXISTS payments_id_seq;
DROP SEQUENCE IF EXISTS test_casting_id_seq;
DROP SEQUENCE IF EXISTS data_types_id_seq;

-- Create sequences
CREATE SEQUENCE users_id_seq;
CREATE SEQUENCE products_id_seq;
CREATE SEQUENCE orders_id_seq;
CREATE SEQUENCE order_items_id_seq;
CREATE SEQUENCE payments_id_seq;
CREATE SEQUENCE test_casting_id_seq;
CREATE SEQUENCE data_types_id_seq;

-- Users Table
CREATE TABLE users (
    id INTEGER PRIMARY KEY DEFAULT nextval('users_id_seq'),
    username VARCHAR UNIQUE NOT NULL,
    email VARCHAR UNIQUE NOT NULL,
    password VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT current_timestamp,
    CHECK (LENGTH(password) >= 8)
);

-- Products Table
CREATE TABLE products (
    id INTEGER PRIMARY KEY DEFAULT nextval('products_id_seq'),
    name VARCHAR NOT NULL UNIQUE,
    description VARCHAR,
    price DOUBLE NOT NULL CHECK (price >= 0),
    stock_quantity INTEGER NOT NULL DEFAULT 0 CHECK (stock_quantity >= 0),
    created_at TIMESTAMP DEFAULT current_timestamp
);

-- Orders Table
CREATE TABLE orders (
    id INTEGER PRIMARY KEY DEFAULT nextval('orders_id_seq'),
    user_id INTEGER NOT NULL,
    order_date TIMESTAMP DEFAULT current_timestamp,
    total_amount DOUBLE NOT NULL DEFAULT 0,
    status VARCHAR DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'canceled')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE NO ACTION
);

-- Order Items Table
CREATE TABLE order_items (
    id INTEGER PRIMARY KEY DEFAULT nextval('order_items_id_seq'),
    order_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    price_at_purchase DOUBLE NOT NULL CHECK (price_at_purchase >= 0),
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE NO ACTION,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE NO ACTION
);

-- Payments Table
CREATE TABLE payments (
    id INTEGER PRIMARY KEY DEFAULT nextval('payments_id_seq'),
    order_id INTEGER NOT NULL,
    payment_date TIMESTAMP DEFAULT current_timestamp,
    amount DOUBLE NOT NULL CHECK (amount >= 0),
    payment_method VARCHAR CHECK (payment_method IN ('credit_card', 'paypal', 'bank_transfer')),
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE NO ACTION
);

-- Indexes for faster queries
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_payments_order_id ON payments(order_id);

-- View for Order Summary
CREATE VIEW order_summary AS
SELECT
    o.id AS order_id,
    u.username,
    o.order_date,
    o.status,
    o.total_amount
FROM orders o
JOIN users u ON o.user_id = u.id;

-- Sample Data for Users
INSERT INTO users (username, email, password) VALUES
('john_doe', 'john@example.com', 'securepassword1'),
('jane_smith', 'jane@example.com', 'securepassword2'),
('admin_user', 'admin@example.com', 'adminpass1');

-- Sample Data for Products
INSERT INTO products (name, description, price, stock_quantity) VALUES
('Laptop', 'High-performance laptop', 1200.00, 10),
('Smartphone', 'Latest model smartphone', 800.00, 20),
('Headphones', 'Noise-canceling headphones', 150.00, 50),
('Monitor', '4K UHD Monitor', 400.00, 15);

-- Sample Orders
INSERT INTO orders (user_id, total_amount, status) VALUES
(1, 0, 'completed'),
(2, 0, 'pending');

-- Sample Order Items
INSERT INTO order_items (order_id, product_id, quantity, price_at_purchase) VALUES
(1, 1, 1, 1200.00),
(1, 2, 1, 800.00),
(2, 3, 1, 150.00);

-- Update order totals manually (DuckDB doesn't support triggers)
UPDATE orders SET total_amount = 2000.00 WHERE id = 1;
UPDATE orders SET total_amount = 150.00 WHERE id = 2;

-- Sample Payments
INSERT INTO payments (order_id, amount, payment_method) VALUES
(1, 2000.00, 'credit_card'),
(2, 150.00, 'paypal');

-- Test Casting Table for type casting validation
CREATE TABLE test_casting (
    id INTEGER PRIMARY KEY DEFAULT nextval('test_casting_id_seq'),
    bigint_col BIGINT NOT NULL,
    integer_col INTEGER NOT NULL,
    smallint_col SMALLINT NOT NULL,
    numeric_col DOUBLE,
    description VARCHAR
);

-- Insert sample data for test_casting
INSERT INTO test_casting (bigint_col, integer_col, smallint_col, numeric_col, description) VALUES
(9223372036854775807, 2147483647, 32767, 99999999.99, 'Maximum values'),
(1000000, 1000, 100, 1234.56, 'Regular values'),
(-9223372036854775808, -2147483648, -32768, -99999999.99, 'Minimum values');

-- Data Types Table for exhaustive type testing
CREATE TABLE data_types (
    id INTEGER PRIMARY KEY DEFAULT nextval('data_types_id_seq'),
    -- Signed integers
    col_tinyint TINYINT,
    col_smallint SMALLINT,
    col_integer INTEGER,
    col_bigint BIGINT,
    col_hugeint HUGEINT,
    -- Unsigned integers
    col_utinyint UTINYINT,
    col_usmallint USMALLINT,
    col_uinteger UINTEGER,
    col_ubigint UBIGINT,
    -- Floating point
    col_float FLOAT,
    col_double DOUBLE,
    col_decimal DECIMAL(10,2),
    -- Text & Binary
    col_varchar VARCHAR,
    col_blob BLOB,
    -- Boolean
    col_boolean BOOLEAN,
    -- Date/Time
    col_date DATE,
    col_time TIME,
    col_timestamp TIMESTAMP,
    col_timestamptz TIMESTAMP WITH TIME ZONE,
    col_interval INTERVAL,
    -- JSON & UUID
    col_json JSON,
    col_uuid UUID
);

-- Insert seed data for data_types
INSERT INTO data_types (
    col_tinyint, col_smallint, col_integer, col_bigint, col_hugeint,
    col_utinyint, col_usmallint, col_uinteger, col_ubigint,
    col_float, col_double, col_decimal,
    col_varchar, col_blob,
    col_boolean,
    col_date, col_time, col_timestamp, col_timestamptz, col_interval,
    col_json, col_uuid
) VALUES (
    42, 1000, 1000000, 9223372036854775807, 170141183460469231731687303715884105727,
    200, 60000, 4000000000, 18000000000000000000,
    3.14, 3.14159265358979, 12345.67,
    'text_value', '\xDEADBEEF'::BLOB,
    true,
    '2025-01-01', '14:30:00', '2025-01-01 12:00:00', '2025-01-01 12:00:00+00', INTERVAL 1 DAY,
    '{"key": "value", "num": 42}'::JSON, 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11'::UUID
);
